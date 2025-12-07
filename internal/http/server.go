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
	addr           string
	server         *http.Server
	mux            *http.ServeMux
	templates      *template.Template
	logger         *observability.Logger
	metrics        *observability.Metrics
	channelService *services.ChannelService
	searchService  *services.SearchService
	channelRepo    *repository.ChannelRepository
	messageRepo    *repository.MessageRepository
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Addr           string
	Logger         *observability.Logger
	Metrics        *observability.Metrics
	ChannelService *services.ChannelService
	SearchService  *services.SearchService
	ChannelRepo    *repository.ChannelRepository
	MessageRepo    *repository.MessageRepository
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
		addr:           cfg.Addr,
		mux:            http.NewServeMux(),
		logger:         cfg.Logger,
		metrics:        cfg.Metrics,
		channelService: cfg.ChannelService,
		searchService:  cfg.SearchService,
		channelRepo:    cfg.ChannelRepo,
		messageRepo:    cfg.MessageRepo,
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
		Addr:         s.addr,
		Handler:      s.middleware(s.mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
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
	return s.server.Shutdown(ctx)
}

func (s *Server) registerRoutes() {
	// Health check
	s.mux.HandleFunc("GET /healthz", s.handleHealth)

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
}

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	s.templates.ExecuteTemplate(w, "home", nil)
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
