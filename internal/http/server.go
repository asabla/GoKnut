// Package http provides the HTTP server and handlers for the web UI.
package http

import (
	"context"
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"time"

	"github.com/asabla/goknut/internal/observability"
)

//go:embed templates
var templatesFS embed.FS

// Server is the HTTP server for the web UI.
type Server struct {
	addr      string
	server    *http.Server
	mux       *http.ServeMux
	templates *template.Template
	logger    *observability.Logger
	metrics   *observability.Metrics
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Addr    string
	Logger  *observability.Logger
	Metrics *observability.Metrics
}

// NewServer creates a new HTTP server.
func NewServer(cfg ServerConfig) (*Server, error) {
	s := &Server{
		addr:    cfg.Addr,
		mux:     http.NewServeMux(),
		logger:  cfg.Logger,
		metrics: cfg.Metrics,
	}

	// Parse templates
	tmplFS, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		return nil, err
	}

	s.templates, err = template.ParseFS(tmplFS, "*.html", "*/*.html")
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

	// Static placeholder routes - will be implemented with handlers
	s.mux.HandleFunc("GET /channels", s.handleNotImplemented)
	s.mux.HandleFunc("GET /users", s.handleNotImplemented)
	s.mux.HandleFunc("GET /search/messages", s.handleNotImplemented)
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
	s.templates.ExecuteTemplate(w, "layout.html", nil)
}

func (s *Server) handleNotImplemented(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
