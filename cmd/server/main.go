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

	// Initialize observability
	logger := observability.NewLogger("goknut")
	metrics := observability.NewMetrics()

	logger.Info("starting GoKnut",
		"auth_mode", cfg.TwitchAuthMode,
		"db_path", cfg.DBPath,
		"http_addr", cfg.HTTPAddr,
		"batch_size", cfg.BatchSize,
		"flush_timeout_ms", cfg.FlushTimeout,
		"enable_fts", cfg.EnableFTS,
	)

	// Open database
	db, err := repository.Open(repository.DBConfig{
		Path:      cfg.DBPath,
		EnableFTS: cfg.EnableFTS,
	})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Run migrations
	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	logger.Info("database migrations complete")

	// Create repositories
	channelRepo := repository.NewChannelRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	userRepo := repository.NewUserRepository(db)

	// Create message processor (implements ingestion.MessageStore)
	processor := ingestion.NewProcessor(
		messageRepo,
		userRepo,
		channelRepo,
		ingestion.ProcessorConfig{
			Logger:  logger,
			Metrics: metrics,
		},
	)

	// Create ingestion pipeline
	pipeline := ingestion.NewPipeline(
		ingestion.PipelineConfig{
			BatchSize:    cfg.BatchSize,
			FlushTimeout: time.Duration(cfg.FlushTimeout) * time.Millisecond,
			BufferSize:   10000,
			Metrics:      metrics,
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
	searchService := services.NewSearchService(searchRepo, logger, metrics)

	// Create HTTP server
	httpServer, err := gohttp.NewServer(gohttp.ServerConfig{
		Addr:           cfg.HTTPAddr,
		Logger:         logger,
		Metrics:        metrics,
		ChannelService: channelService,
		SearchService:  searchService,
		ChannelRepo:    channelRepo,
		MessageRepo:    messageRepo,
		UserRepo:       userRepo,
		EnableSSE:      cfg.EnableSSE,
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
