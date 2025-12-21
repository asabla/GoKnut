package handlers

import (
	"html/template"
	"net/http"
	"time"

	"github.com/asabla/goknut/internal/observability"
)

// HomeDashboardHandler serves the dashboard fragments embedded on the home page.
//
// In the foundational phase this handler only returns placeholder HTML.
// Later phases add data aggregation and Prometheus queries.
type HomeDashboardHandler struct {
	templates *template.Template
	logger    *observability.Logger

	prometheusBaseURL string
	prometheusTimeout time.Duration
}

func NewHomeDashboardHandler(templates *template.Template, logger *observability.Logger, prometheusBaseURL string, prometheusTimeout time.Duration) *HomeDashboardHandler {
	return &HomeDashboardHandler{
		templates:         templates,
		logger:            logger,
		prometheusBaseURL: prometheusBaseURL,
		prometheusTimeout: prometheusTimeout,
	}
}

func (h *HomeDashboardHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /dashboard/home/summary", h.handleSummary)
	mux.HandleFunc("GET /dashboard/home/diagrams", h.handleDiagrams)
}

func (h *HomeDashboardHandler) handleSummary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "dashboard/home_summary", nil); err != nil {
		h.logger.Error("failed to execute dashboard summary template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *HomeDashboardHandler) handleDiagrams(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "dashboard/home_diagrams", nil); err != nil {
		h.logger.Error("failed to execute dashboard diagrams template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
