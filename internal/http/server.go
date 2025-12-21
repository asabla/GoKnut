// Package http provides the HTTP server and handlers for the web UI.
package http

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/asabla/goknut/internal/http/handlers"
	"github.com/asabla/goknut/internal/observability"
	"github.com/asabla/goknut/internal/repository"
	"github.com/asabla/goknut/internal/services"
)

//go:embed templates
var templatesFS embed.FS

// Server is the HTTP server for the web UI.
type Server struct {
	addr                 string
	server               *http.Server
	mux                  *http.ServeMux
	templates            *template.Template
	logger               *observability.Logger
	metrics              *observability.Metrics
	otelProvider         *observability.OTelProvider
	channelService       *services.ChannelService
	searchService        *services.SearchService
	profileService       *services.ProfileService
	organizationService  *services.OrganizationService
	organizationRepo     *repository.OrganizationRepository
	eventService         *services.EventService
	eventRepo            *repository.EventRepository
	collaborationService *services.CollaborationService
	collaborationRepo    *repository.CollaborationRepository
	channelRepo          *repository.ChannelRepository
	messageRepo          *repository.MessageRepository
	userRepo             *repository.UserRepository
	profileRepo          *repository.ProfileRepository
	enableSSE            bool
	sseHandler           *handlers.SSEHandler

	prometheusBaseURL string
	prometheusTimeout time.Duration
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Addr                 string
	Logger               *observability.Logger
	Metrics              *observability.Metrics
	OTelProvider         *observability.OTelProvider
	ChannelService       *services.ChannelService
	SearchService        *services.SearchService
	ProfileService       *services.ProfileService
	OrganizationService  *services.OrganizationService
	OrganizationRepo     *repository.OrganizationRepository
	EventService         *services.EventService
	EventRepo            *repository.EventRepository
	CollaborationService *services.CollaborationService
	CollaborationRepo    *repository.CollaborationRepository
	ChannelRepo          *repository.ChannelRepository
	MessageRepo          *repository.MessageRepository
	UserRepo             *repository.UserRepository
	ProfileRepo          *repository.ProfileRepository
	EnableSSE            bool

	PrometheusBaseURL string
	PrometheusTimeout time.Duration
}

// templateFuncs returns the custom template functions.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"formatNumber": func(n int64) string {
			if n < 1000 {
				return fmt.Sprintf("%d", n)
			}
			if n < 1000000 {
				return fmt.Sprintf("%.1fK", float64(n)/1000)
			}
			return fmt.Sprintf("%.1fM", float64(n)/1000000)
		},
		"formatDate": func(t time.Time) string {
			if t.IsZero() {
				return "Never"
			}
			return t.Format("Jan 2, 2006")
		},
		"formatTime": func(t *time.Time) string {
			if t == nil || t.IsZero() {
				return "Never"
			}
			return t.Format("Jan 2, 2006 3:04 PM")
		},
		"dict": func(values ...any) map[string]any {
			if len(values)%2 != 0 {
				return nil
			}
			m := make(map[string]any, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					continue
				}
				m[key] = values[i+1]
			}
			return m
		},
	}
}

// NewServer creates a new HTTP server.
func NewServer(cfg ServerConfig) (*Server, error) {
	s := &Server{
		addr:                 cfg.Addr,
		mux:                  http.NewServeMux(),
		logger:               cfg.Logger,
		metrics:              cfg.Metrics,
		otelProvider:         cfg.OTelProvider,
		channelService:       cfg.ChannelService,
		searchService:        cfg.SearchService,
		profileService:       cfg.ProfileService,
		organizationService:  cfg.OrganizationService,
		organizationRepo:     cfg.OrganizationRepo,
		eventService:         cfg.EventService,
		eventRepo:            cfg.EventRepo,
		collaborationService: cfg.CollaborationService,
		collaborationRepo:    cfg.CollaborationRepo,
		channelRepo:          cfg.ChannelRepo,
		messageRepo:          cfg.MessageRepo,
		userRepo:             cfg.UserRepo,
		profileRepo:          cfg.ProfileRepo,
		enableSSE:            cfg.EnableSSE,
		prometheusBaseURL:    cfg.PrometheusBaseURL,
		prometheusTimeout:    cfg.PrometheusTimeout,
	}

	// Parse templates with custom functions
	tmplFS, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		return nil, err
	}

	s.templates, err = template.New("").Funcs(templateFuncs()).ParseFS(tmplFS, "*.html", "*/*.html")
	if err != nil {
		return nil, err
	}

	// Register routes
	s.registerRoutes()

	s.server = &http.Server{
		Addr:        s.addr,
		Handler:     s.middleware(s.mux),
		ReadTimeout: 15 * time.Second,
		// WriteTimeout disabled (0) to support SSE long-lived connections.
		// SSE handler manages its own timeouts via heartbeats.
		IdleTimeout: 60 * time.Second,
	}

	return s, nil
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	s.logger.HTTP("starting HTTP server", "addr", s.addr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.HTTP("shutting down HTTP server")

	// Close all SSE connections first so they don't block shutdown
	if s.sseHandler != nil {
		s.sseHandler.Shutdown()
	}

	return s.server.Shutdown(ctx)
}

// SSEHandler returns the SSE handler for broadcasting events.
// Returns nil if SSE is not enabled.
func (s *Server) SSEHandler() *handlers.SSEHandler {
	return s.sseHandler
}

func (s *Server) registerRoutes() {
	// Health check
	s.mux.HandleFunc("GET /healthz", s.handleHealth)

	// Prometheus metrics endpoint
	if s.otelProvider != nil {
		s.mux.Handle("GET /metrics", s.otelProvider.PrometheusHandler())
		s.logger.HTTP("Prometheus metrics endpoint enabled", "path", "/metrics")
	}

	// Home
	s.mux.HandleFunc("GET /", s.handleHome)

	// Register channel handler routes
	if s.channelService != nil {
		channelHandler := handlers.NewChannelHandler(s.channelService, s.templates, s.logger)
		channelHandler.RegisterRoutes(s.mux)
	}

	// Register channel view handler routes (live view)
	if s.channelRepo != nil && s.messageRepo != nil {
		channelViewHandler := handlers.NewChannelViewHandler(
			s.channelRepo, s.messageRepo, s.templates, s.logger, s.metrics)
		channelViewHandler.RegisterRoutes(s.mux)
	}

	// Register search handler routes
	if s.searchService != nil {
		searchHandler := handlers.NewSearchHandler(s.searchService, s.templates, s.logger)
		searchHandler.RegisterRoutes(s.mux)
	}

	// Register profile handler routes
	if s.profileService != nil && s.channelRepo != nil {
		profileHandler := handlers.NewProfileHandler(s.profileService, s.channelRepo, s.organizationRepo, s.eventRepo, s.collaborationRepo, s.templates, s.logger, s.metrics)
		profileHandler.RegisterRoutes(s.mux)
	}

	// Register organization handler routes
	if s.organizationService != nil && s.profileRepo != nil {
		organizationHandler := handlers.NewOrganizationHandler(s.organizationService, s.profileRepo, s.templates, s.logger, s.metrics)
		organizationHandler.RegisterRoutes(s.mux)
	}

	// Register event handler routes
	if s.eventService != nil && s.profileRepo != nil {
		eventHandler := handlers.NewEventHandler(s.eventService, s.profileRepo, s.templates, s.logger, s.metrics)
		eventHandler.RegisterRoutes(s.mux)
	}

	// Register collaboration handler routes
	if s.collaborationService != nil && s.profileRepo != nil {
		collaborationHandler := handlers.NewCollaborationHandler(s.collaborationService, s.profileRepo, s.templates, s.logger, s.metrics)
		collaborationHandler.RegisterRoutes(s.mux)
	}

	// Register home dashboard fragments
	if s.templates != nil {
		homeDashboardHandler := handlers.NewHomeDashboardHandler(
			s.templates,
			s.logger,
			s.messageRepo,
			s.channelRepo,
			s.userRepo,
			s.prometheusBaseURL,
			s.prometheusTimeout,
		)
		homeDashboardHandler.RegisterRoutes(s.mux)
	}

	// Register SSE live updates handler
	if s.enableSSE {
		s.sseHandler = handlers.NewSSEHandler(
			s.channelRepo, s.messageRepo, s.userRepo, s.templates, s.logger, s.metrics, s.otelProvider)
		s.sseHandler.RegisterRoutes(s.mux)
		s.logger.HTTP("SSE live updates enabled", "path", "/live")
	}
}

func (s *Server) middleware(next http.Handler) http.Handler {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		// Record metrics
		latency := time.Since(start)
		if s.metrics != nil {
			s.metrics.RecordHTTPRequest(latency)
		}

		s.logger.HTTP("request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"latency_ms", latency.Milliseconds(),
		)
	})

	// Wrap with OTel HTTP middleware if enabled
	if s.otelProvider != nil {
		return s.otelProvider.HTTPMiddleware(handler)
	}
	return handler
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	ctx := r.Context()

	// Prepare data for the template
	data := struct {
		TotalMessages   int64
		TotalChannels   int64
		EnabledChannels int64
		TotalUsers      int64
		RecentMessages  []repository.Message
	}{}

	// Fetch statistics (ignore errors, show 0 if unavailable)
	if s.messageRepo != nil {
		data.TotalMessages, _ = s.messageRepo.GetTotalCount(ctx)
		data.RecentMessages, _ = s.messageRepo.GetRecentGlobal(ctx, 20)
	}
	if s.channelRepo != nil {
		data.TotalChannels, _ = s.channelRepo.GetCount(ctx)
		data.EnabledChannels, _ = s.channelRepo.GetEnabledCount(ctx)
	}
	if s.userRepo != nil {
		data.TotalUsers, _ = s.userRepo.GetCount(ctx)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, "home", data); err != nil {
		s.logger.Error("failed to execute home template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Flush implements http.Flusher to support SSE streaming.
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
