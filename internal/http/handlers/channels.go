// Package handlers provides HTTP handlers for the web UI.
package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strings"

	"github.com/asabla/goknut/internal/http/dto"
	"github.com/asabla/goknut/internal/observability"
	"github.com/asabla/goknut/internal/services"
)

// ChannelHandler handles channel-related HTTP requests.
type ChannelHandler struct {
	service   *services.ChannelService
	templates *template.Template
	logger    *observability.Logger
}

// NewChannelHandler creates a new channel handler.
func NewChannelHandler(
	service *services.ChannelService,
	templates *template.Template,
	logger *observability.Logger,
) *ChannelHandler {
	return &ChannelHandler{
		service:   service,
		templates: templates,
		logger:    logger,
	}
}

// RegisterRoutes registers channel routes on the mux.
func (h *ChannelHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /channels", h.handleList)
	mux.HandleFunc("POST /channels", h.handleCreate)
	mux.HandleFunc("GET /channels/{name}", h.handleGet)
	mux.HandleFunc("POST /channels/{name}", h.handleUpdate)
	mux.HandleFunc("POST /channels/{name}/delete", h.handleDelete)
}

func (h *ChannelHandler) handleList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	channels, err := h.service.List(ctx)
	if err != nil {
		h.logger.Error("failed to list channels", "error", err)
		h.renderError(w, r, "Failed to load channels", http.StatusInternalServerError)
		return
	}

	// Convert to DTOs
	channelDTOs := make([]dto.Channel, len(channels))
	for i, ch := range channels {
		channelDTOs[i] = dto.Channel{
			ID:                    ch.ID,
			Name:                  ch.Name,
			DisplayName:           ch.DisplayName,
			Enabled:               ch.Enabled,
			RetainHistoryOnDelete: ch.RetainHistoryOnDelete,
			CreatedAt:             ch.CreatedAt,
			UpdatedAt:             ch.UpdatedAt,
			LastMessageAt:         ch.LastMessageAt,
			TotalMessages:         ch.TotalMessages,
		}
	}

	// Respond based on Accept header or HX-Request
	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"channels": channelDTOs})
		return
	}

	// Render HTML template
	data := map[string]any{
		"Channels": channelDTOs,
		"IsEmpty":  len(channelDTOs) == 0,
	}

	if h.isHTMXRequest(r) {
		if err := h.templates.ExecuteTemplate(w, "list.html", data); err != nil {
			h.logger.Error("failed to execute list template", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	} else {
		if err := h.templates.ExecuteTemplate(w, "channels/index", data); err != nil {
			h.logger.Error("failed to execute channels/index template", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

func (h *ChannelHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dto.CreateChannelRequest

	// Handle both JSON and form data
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.renderError(w, r, "Invalid request body", http.StatusBadRequest)
			return
		}
	} else {
		r.ParseForm()
		req.Name = r.FormValue("name")
		req.DisplayName = r.FormValue("display_name")
		req.Enabled = r.FormValue("enabled") == "on" || r.FormValue("enabled") == "true"
		req.RetainHistoryOnDelete = r.FormValue("retain_history_on_delete") == "on" || r.FormValue("retain_history_on_delete") == "true"
	}

	// Validate
	if err := req.Validate(); err != nil {
		h.renderError(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	// Create channel
	ch, err := h.service.Create(ctx, req.Name, req.DisplayName, req.Enabled, req.RetainHistoryOnDelete)
	if err != nil {
		if err == services.ErrChannelAlreadyExists {
			h.renderError(w, r, "Channel already exists", http.StatusConflict)
			return
		}
		h.logger.Error("failed to create channel", "error", err)
		h.renderError(w, r, "Failed to create channel", http.StatusInternalServerError)
		return
	}

	channelDTO := dto.Channel{
		ID:                    ch.ID,
		Name:                  ch.Name,
		DisplayName:           ch.DisplayName,
		Enabled:               ch.Enabled,
		RetainHistoryOnDelete: ch.RetainHistoryOnDelete,
		CreatedAt:             ch.CreatedAt,
		UpdatedAt:             ch.UpdatedAt,
		TotalMessages:         ch.TotalMessages,
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(channelDTO)
		return
	}

	// For HTMX, return the new row to append
	if h.isHTMXRequest(r) {
		if err := h.templates.ExecuteTemplate(w, "row.html", channelDTO); err != nil {
			h.logger.Error("failed to execute row template", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Redirect to channels list
	http.Redirect(w, r, "/channels", http.StatusSeeOther)
}

func (h *ChannelHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	name := r.PathValue("name")
	if name == "" {
		h.renderError(w, r, "Invalid channel name", http.StatusBadRequest)
		return
	}

	ch, err := h.service.GetByName(ctx, name)
	if err != nil {
		h.logger.Error("failed to get channel", "error", err)
		h.renderError(w, r, "Failed to load channel", http.StatusInternalServerError)
		return
	}
	if ch == nil {
		h.renderError(w, r, "Channel not found", http.StatusNotFound)
		return
	}

	channelDTO := dto.Channel{
		ID:                    ch.ID,
		Name:                  ch.Name,
		DisplayName:           ch.DisplayName,
		Enabled:               ch.Enabled,
		RetainHistoryOnDelete: ch.RetainHistoryOnDelete,
		CreatedAt:             ch.CreatedAt,
		UpdatedAt:             ch.UpdatedAt,
		LastMessageAt:         ch.LastMessageAt,
		TotalMessages:         ch.TotalMessages,
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(channelDTO)
		return
	}

	if err := h.templates.ExecuteTemplate(w, "channels/detail", channelDTO); err != nil {
		h.logger.Error("failed to execute channels/detail template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *ChannelHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	name := r.PathValue("name")
	if name == "" {
		h.renderError(w, r, "Invalid channel name", http.StatusBadRequest)
		return
	}

	// First get the channel by name to get the ID
	ch, err := h.service.GetByName(ctx, name)
	if err != nil {
		h.logger.Error("failed to get channel", "error", err)
		h.renderError(w, r, "Failed to load channel", http.StatusInternalServerError)
		return
	}
	if ch == nil {
		h.renderError(w, r, "Channel not found", http.StatusNotFound)
		return
	}

	var req dto.UpdateChannelRequest

	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.renderError(w, r, "Invalid request body", http.StatusBadRequest)
			return
		}
	} else {
		r.ParseForm()
		if dn := r.FormValue("display_name"); dn != "" {
			req.DisplayName = &dn
		}
		if r.Form.Has("enabled") {
			enabled := r.FormValue("enabled") == "on" || r.FormValue("enabled") == "true"
			req.Enabled = &enabled
		}
		if r.Form.Has("retain_history_on_delete") {
			retain := r.FormValue("retain_history_on_delete") == "on" || r.FormValue("retain_history_on_delete") == "true"
			req.RetainHistoryOnDelete = &retain
		}
	}

	ch, err = h.service.Update(ctx, ch.ID, req.DisplayName, req.Enabled, req.RetainHistoryOnDelete)
	if err != nil {
		if err == services.ErrChannelNotFound {
			h.renderError(w, r, "Channel not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to update channel", "error", err)
		h.renderError(w, r, "Failed to update channel", http.StatusInternalServerError)
		return
	}

	channelDTO := dto.Channel{
		ID:                    ch.ID,
		Name:                  ch.Name,
		DisplayName:           ch.DisplayName,
		Enabled:               ch.Enabled,
		RetainHistoryOnDelete: ch.RetainHistoryOnDelete,
		CreatedAt:             ch.CreatedAt,
		UpdatedAt:             ch.UpdatedAt,
		LastMessageAt:         ch.LastMessageAt,
		TotalMessages:         ch.TotalMessages,
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(channelDTO)
		return
	}

	// For HTMX, return the updated row
	if h.isHTMXRequest(r) {
		if err := h.templates.ExecuteTemplate(w, "row.html", channelDTO); err != nil {
			h.logger.Error("failed to execute row template", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, "/channels", http.StatusSeeOther)
}

func (h *ChannelHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	name := r.PathValue("name")
	if name == "" {
		h.renderError(w, r, "Invalid channel name", http.StatusBadRequest)
		return
	}

	// First get the channel by name to get the ID
	ch, err := h.service.GetByName(ctx, name)
	if err != nil {
		h.logger.Error("failed to get channel", "error", err)
		h.renderError(w, r, "Failed to load channel", http.StatusInternalServerError)
		return
	}
	if ch == nil {
		h.renderError(w, r, "Channel not found", http.StatusNotFound)
		return
	}

	var req dto.DeleteChannelRequest

	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		json.NewDecoder(r.Body).Decode(&req)
	} else {
		r.ParseForm()
		req.RetainHistory = r.FormValue("retain_history") == "on" || r.FormValue("retain_history") == "true"
	}

	err = h.service.Delete(ctx, ch.ID, req.RetainHistory)
	if err != nil {
		if err == services.ErrChannelNotFound {
			h.renderError(w, r, "Channel not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to delete channel", "error", err)
		h.renderError(w, r, "Failed to delete channel", http.StatusInternalServerError)
		return
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
		return
	}

	// For HTMX, return empty to remove the row
	if h.isHTMXRequest(r) {
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/channels", http.StatusSeeOther)
}

func (h *ChannelHandler) renderError(w http.ResponseWriter, r *http.Request, message string, status int) {
	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]string{"error": message})
		return
	}

	w.WriteHeader(status)
	if err := h.templates.ExecuteTemplate(w, "error.html", map[string]any{
		"Title":   http.StatusText(status),
		"Message": message,
	}); err != nil {
		h.logger.Error("failed to execute error template", "error", err)
	}
}

func (h *ChannelHandler) wantsJSON(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "application/json")
}

func (h *ChannelHandler) isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}
