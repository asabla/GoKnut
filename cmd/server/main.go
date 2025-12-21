// Package main provides the entry point for the GoKnut Twitch Chat Archiver.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asabla/goknut/internal/config"
	gohttp "github.com/asabla/goknut/internal/http"
	"github.com/asabla/goknut/internal/ingestion"
	"github.com/asabla/goknut/internal/irc"
	"github.com/asabla/goknut/internal/observability"
	"github.com/asabla/goknut/internal/repository"
	"github.com/asabla/goknut/internal/search"
	"github.com/asabla/goknut/internal/services"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx := context.Background()

	// Initialize observability
	logger := observability.NewLogger("goknut")
	metrics := observability.NewMetrics()

	// Initialize OpenTelemetry if enabled
	var otelProvider *observability.OTelProvider
	if cfg.OTelEnabled {
		otelCfg := observability.OTelConfig{
			ServiceName:    cfg.OTelServiceName,
			OTLPEndpoint:   cfg.OTelExporterOTLP,
			Insecure:       cfg.OTelInsecure,
			SamplerRatio:   cfg.OTelSamplerRatio,
			MetricsEnabled: cfg.OTelMetricsEnabled,
			TracesEnabled:  cfg.OTelTracesEnabled,
			LogsEnabled:    cfg.OTelLogsEnabled,
		}
		otelProvider, err = observability.InitOTel(ctx, otelCfg)
		if err != nil {
			return fmt.Errorf("failed to initialize OpenTelemetry: %w", err)
		}
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := otelProvider.Shutdown(shutdownCtx); err != nil {
				logger.Error("failed to shutdown OpenTelemetry", "error", err)
			}
		}()
		logger.Info("OpenTelemetry initialized",
			"service", cfg.OTelServiceName,
			"endpoint", cfg.OTelExporterOTLP,
			"traces", cfg.OTelTracesEnabled,
			"metrics", cfg.OTelMetricsEnabled,
		)
	}

	logger.Info("starting GoKnut",
		"auth_mode", cfg.TwitchAuthMode,
		"db_driver", cfg.DBDriver,
		"db_path", cfg.DBPath,
		"http_addr", cfg.HTTPAddr,
		"batch_size", cfg.BatchSize,
		"flush_timeout_ms", cfg.FlushTimeout,
		"enable_fts", cfg.EnableFTS,
		"otel_enabled", cfg.OTelEnabled,
	)

	// Open database based on driver
	var db repository.Database
	switch cfg.DBDriver {
	case config.DBDriverSQLite:
		db, err = repository.OpenSQLite(repository.SQLiteDBConfig{
			Path:      cfg.DBPath,
			EnableFTS: cfg.EnableFTS,
		})
		if err != nil {
			return fmt.Errorf("failed to open sqlite database: %w", err)
		}
	case config.DBDriverPostgres:
		db, err = repository.OpenPostgres(repository.PostgresDBConfig{
			Host:     cfg.PGHost,
			Port:     cfg.PGPort,
			User:     cfg.PGUser,
			Password: cfg.PGPassword,
			Database: cfg.PGDatabase,
			SSLMode:  cfg.PGSSLMode,
		})
		if err != nil {
			return fmt.Errorf("failed to open postgres database: %w", err)
		}
	default:
		return fmt.Errorf("unsupported database driver: %s", cfg.DBDriver)
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(ctx); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	logger.Info("database migrations complete")

	// Create repositories
	channelRepo := repository.NewChannelRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	userRepo := repository.NewUserRepository(db)
	profileRepo := repository.NewProfileRepository(db)
	organizationRepo := repository.NewOrganizationRepository(db)
	eventRepo := repository.NewEventRepository(db)
	collaborationRepo := repository.NewCollaborationRepository(db)

	// Register database count callbacks for OTel metrics
	if otelProvider != nil {
		dbCountProvider := &databaseCountProvider{
			messageRepo: messageRepo,
			userRepo:    userRepo,
			channelRepo: channelRepo,
		}
		if err := otelProvider.RegisterDatabaseCountCallbacks(dbCountProvider); err != nil {
			logger.Error("failed to register database count callbacks", "error", err)
		} else {
			logger.Info("database count metrics registered")
		}
	}

	// Create message processor (implements ingestion.MessageStore)
	processor := ingestion.NewProcessor(
		messageRepo,
		userRepo,
		channelRepo,
		ingestion.ProcessorConfig{
			Logger:       logger,
			Metrics:      metrics,
			OTelProvider: otelProvider,
		},
	)

	// Create ingestion pipeline
	pipeline := ingestion.NewPipeline(
		ingestion.PipelineConfig{
			BatchSize:    cfg.BatchSize,
			FlushTimeout: time.Duration(cfg.FlushTimeout) * time.Millisecond,
			BufferSize:   10000,
			Metrics:      metrics,
			OTelProvider: otelProvider,
		},
		processor,
	)

	// Create IRC client
	ircClient := irc.NewClient(irc.ClientConfig{
		AuthMode:   irc.AuthMode(cfg.TwitchAuthMode),
		Username:   cfg.TwitchUsername,
		OAuthToken: cfg.TwitchOAuthToken,
		OnMessage: func(msg irc.Message) {
			metrics.RecordIRCMessage()
			// Record OTel metrics for IRC messages (with channel label)
			if otelProvider != nil {
				otelProvider.RecordIRCMessage(ctx, msg.Channel)
			}
			pipeline.Ingest(ingestion.Message{
				ChannelName: msg.Channel,
				Username:    msg.Username,
				DisplayName: msg.DisplayName,
				Text:        msg.Text,
				Tags:        msg.Tags,
				ReceivedAt:  msg.ReceivedAt,
			})
		},
		OnChannelChange: func(channel string, joined bool) {
			if joined {
				logger.IRC("joined channel", "channel", channel)
			} else {
				logger.IRC("left channel", "channel", channel)
			}
		},
	})

	// Create channel service
	channelService := services.NewChannelService(
		channelRepo,
		ircClient,
		logger,
		metrics,
	)

	// Create search repository and service
	searchRepo := search.NewSearchRepository(db, cfg.EnableFTS)
	searchService := services.NewSearchService(searchRepo, logger, metrics, otelProvider)

	// Create profile/org/event/collaboration services
	profileService := services.NewProfileService(profileRepo, channelRepo)
	organizationService := services.NewOrganizationService(organizationRepo, profileRepo)
	eventService := services.NewEventService(eventRepo, profileRepo)
	collaborationService := services.NewCollaborationService(collaborationRepo, profileRepo)

	// Create HTTP server
	httpServer, err := gohttp.NewServer(gohttp.ServerConfig{
		Addr:                 cfg.HTTPAddr,
		Logger:               logger,
		Metrics:              metrics,
		OTelProvider:         otelProvider,
		ChannelService:       channelService,
		SearchService:        searchService,
		ProfileService:       profileService,
		OrganizationService:  organizationService,
		EventService:         eventService,
		EventRepo:            eventRepo,
		CollaborationService: collaborationService,
		CollaborationRepo:    collaborationRepo,
		ChannelRepo:          channelRepo,
		MessageRepo:          messageRepo,
		UserRepo:             userRepo,
		ProfileRepo:          profileRepo,
		OrganizationRepo:     organizationRepo,
		EnableSSE:            cfg.EnableSSE,
		PrometheusBaseURL:    cfg.PrometheusBaseURL,
		PrometheusTimeout:    time.Duration(cfg.PrometheusTimeout) * time.Millisecond,
	})

	if err != nil {
		return fmt.Errorf("failed to create HTTP server: %w", err)
	}

	// Wire up SSE broadcasting for new messages
	if sseHandler := httpServer.SSEHandler(); sseHandler != nil {
		processor.SetOnMessageStored(func(msg ingestion.StoredMessage) {
			sseHandler.BroadcastMessage(msg.ID, msg.ChannelID, msg.ChannelName,
				msg.UserID, msg.Username, msg.DisplayName, msg.Text, msg.SentAt)
		})
		logger.Info("SSE message broadcasting enabled")
	}

	// Start ingestion pipeline
	if err := pipeline.Start(ctx); err != nil {
		return fmt.Errorf("failed to start ingestion pipeline: %w", err)
	}
	logger.Info("ingestion pipeline started")

	// Connect IRC client
	if err := ircClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to IRC: %w", err)
	}
	metrics.RecordIRCConnection()
	if otelProvider != nil {
		otelProvider.RecordIRCConnection(ctx)
	}
	logger.IRC("connected to Twitch IRC",
		"mode", cfg.TwitchAuthMode,
		"anonymous", ircClient.IsAnonymous(),
	)

	// Initialize channel service (loads channels from DB and joins enabled ones)
	if err := channelService.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize channel service: %w", err)
	}

	// Join channels from config if provided
	for _, channel := range cfg.TwitchChannels {
		// Create channel in DB if it doesn't exist, then join
		existing, err := channelService.GetByName(ctx, channel)
		if err != nil {
			logger.Error("failed to check channel", "channel", channel, "error", err)
			continue
		}
		if existing == nil {
			// Create the channel as enabled
			_, err := channelService.Create(ctx, channel, channel, true, true)
			if err != nil {
				logger.Error("failed to create channel from config", "channel", channel, "error", err)
				continue
			}
			logger.Info("created channel from config", "channel", channel)
		} else if !existing.Enabled {
			// Enable the channel if it exists but is disabled
			enabled := true
			_, err := channelService.Update(ctx, existing.ID, nil, &enabled, nil)
			if err != nil {
				logger.Error("failed to enable channel from config", "channel", channel, "error", err)
				continue
			}
			logger.Info("enabled channel from config", "channel", channel)
		}
	}

	// Start HTTP server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := httpServer.Start(); err != nil {
			errChan <- err
		}
	}()

	logger.Info("server ready",
		"http_addr", cfg.HTTPAddr,
		"channels", len(cfg.TwitchChannels),
	)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		logger.Info("received shutdown signal", "signal", sig)
	case err := <-errChan:
		return fmt.Errorf("HTTP server error: %w", err)
	}

	// Graceful shutdown
	logger.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed to shutdown HTTP server", "error", err)
	}

	// Disconnect IRC
	if err := ircClient.Disconnect(); err != nil {
		logger.Error("failed to disconnect IRC", "error", err)
	}
	metrics.RecordIRCDisconnection()
	if otelProvider != nil {
		otelProvider.RecordIRCDisconnection(ctx)
	}

	// Stop ingestion pipeline (flushes remaining messages)
	if err := pipeline.Stop(); err != nil {
		logger.Error("failed to stop ingestion pipeline", "error", err)
	}

	// Log final metrics
	stats := metrics.Stats()
	logger.Info("final metrics",
		"irc_messages_recv", stats.IRCMessagesRecv,
		"messages_ingested", stats.MessagesIngested,
		"batches_processed", stats.BatchesProcessed,
		"dropped_messages", stats.DroppedMessages,
		"http_requests", stats.HTTPRequests,
	)

	logger.Info("shutdown complete")
	return nil
}

// databaseCountProvider implements observability.DatabaseCountProvider
// to provide database counts for OTel observable gauges.
type databaseCountProvider struct {
	messageRepo *repository.MessageRepository
	userRepo    *repository.UserRepository
	channelRepo *repository.ChannelRepository
}

func (p *databaseCountProvider) GetMessageCount(ctx context.Context) (int64, error) {
	return p.messageRepo.GetTotalCount(ctx)
}

func (p *databaseCountProvider) GetUserCount(ctx context.Context) (int64, error) {
	return p.userRepo.GetCount(ctx)
}

func (p *databaseCountProvider) GetChannelCount(ctx context.Context) (int64, error) {
	return p.channelRepo.GetCount(ctx)
}
