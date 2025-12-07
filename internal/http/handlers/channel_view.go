// Package handlers provides HTTP handlers for the web UI.
package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/asabla/goknut/internal/http/dto"
	"github.com/asabla/goknut/internal/observability"
	"github.com/asabla/goknut/internal/repository"
)

// ChannelViewHandler handles channel view and message stream requests.
type ChannelViewHandler struct {
	channelRepo *repository.ChannelRepository
	messageRepo *repository.MessageRepository
	templates   *template.Template
	logger      *observability.Logger
	metrics     *observability.Metrics
}

// NewChannelViewHandler creates a new channel view handler.
func NewChannelViewHandler(
	channelRepo *repository.ChannelRepository,
	messageRepo *repository.MessageRepository,
	templates *template.Template,
	logger *observability.Logger,
	metrics *observability.Metrics,
) *ChannelViewHandler {
	return &ChannelViewHandler{
		channelRepo: channelRepo,
		messageRepo: messageRepo,
		templates:   templates,
		logger:      logger,
		metrics:     metrics,
	}
}

// RegisterRoutes registers channel view routes on the mux.
func (h *ChannelViewHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /channels/{name}/view", h.handleChannelView)
	mux.HandleFunc("GET /channels/{name}/messages", h.handleMessages)
	mux.HandleFunc("GET /channels/{name}/messages/stream", h.handleMessageStream)
}

// handleChannelView renders the main channel view page with recent messages.
func (h *ChannelViewHandler) handleChannelView(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()

	name := r.PathValue("name")
	if name == "" {
		h.renderError(w, r, "Invalid channel name", http.StatusBadRequest)
		return
	}

	// Get channel
	channel, err := h.channelRepo.GetByName(ctx, name)
	if err != nil {
		h.logger.Error("failed to get channel", "name", name, "error", err)
		h.renderError(w, r, "Failed to load channel", http.StatusInternalServerError)
		return
	}
	if channel == nil {
		h.renderError(w, r, "Channel not found", http.StatusNotFound)
		return
	}

	// Get recent messages
	messages, err := h.messageRepo.GetRecent(ctx, channel.ID, 50)
	if err != nil {
		h.logger.Error("failed to get messages", "channel_id", channel.ID, "error", err)
		h.renderError(w, r, "Failed to load messages", http.StatusInternalServerError)
		return
	}

	// Get latest message ID for streaming
	latestID, _ := h.messageRepo.GetLatestID(ctx, channel.ID)

	// Convert to DTOs
	channelDTO := h.channelToDTO(channel)
	messageDTOs := h.messagesToDTOs(messages)

	// Reverse order for display (oldest first at top)
	for i, j := 0, len(messageDTOs)-1; i < j; i, j = i+1, j-1 {
		messageDTOs[i], messageDTOs[j] = messageDTOs[j], messageDTOs[i]
	}

	data := map[string]any{
		"Channel":   channelDTO,
		"Messages":  messageDTOs,
		"LatestID":  latestID,
		"IsEmpty":   len(messageDTOs) == 0,
		"PollDelay": 1000, // 1 second polling interval
	}

	h.recordLatency(start)

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
		return
	}

	h.templates.ExecuteTemplate(w, "live/channel.html", data)
}

// handleMessages returns paginated messages as HTML fragment or JSON.
func (h *ChannelViewHandler) handleMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()

	name := r.PathValue("name")
	if name == "" {
		h.renderError(w, r, "Invalid channel name", http.StatusBadRequest)
		return
	}

	// Get channel to get its ID
	channel, err := h.channelRepo.GetByName(ctx, name)
	if err != nil {
		h.logger.Error("failed to get channel", "name", name, "error", err)
		h.renderError(w, r, "Failed to load channel", http.StatusInternalServerError)
		return
	}
	if channel == nil {
		h.renderError(w, r, "Channel not found", http.StatusNotFound)
		return
	}

	// Parse pagination params
	page, pageSize := h.parsePagination(r)
	beforeID, _ := strconv.ParseInt(r.URL.Query().Get("before_id"), 10, 64)

	var messages []repository.Message
	var totalCount int

	if beforeID > 0 {
		// Cursor-based pagination
		messages, err = h.messageRepo.GetBeforeID(ctx, channel.ID, beforeID, pageSize)
		if err != nil {
			h.logger.Error("failed to get messages", "channel_id", channel.ID, "error", err)
			h.renderError(w, r, "Failed to load messages", http.StatusInternalServerError)
			return
		}
	} else {
		// Page-based pagination
		messages, totalCount, err = h.messageRepo.GetPaginated(ctx, channel.ID, page, pageSize)
		if err != nil {
			h.logger.Error("failed to get messages", "channel_id", channel.ID, "error", err)
			h.renderError(w, r, "Failed to load messages", http.StatusInternalServerError)
			return
		}
	}

	// Convert to DTOs
	messageDTOs := h.messagesToDTOs(messages)

	h.recordLatency(start)

	if h.wantsJSON(r) {
		totalPages := 0
		if totalCount > 0 && pageSize > 0 {
			totalPages = (totalCount + pageSize - 1) / pageSize
		}

		response := dto.PaginatedResponse[dto.Message]{
			Items:      messageDTOs,
			Page:       page,
			PageSize:   pageSize,
			TotalCount: totalCount,
			TotalPages: totalPages,
			HasNext:    page < totalPages,
			HasPrev:    page > 1,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"messages":    response.Items,
			"page":        response.Page,
			"page_size":   response.PageSize,
			"total_count": response.TotalCount,
			"total_pages": response.TotalPages,
			"has_next":    response.HasNext,
			"has_prev":    response.HasPrev,
		})
		return
	}

	// For HTMX requests, render message list fragment
	data := map[string]any{
		"Messages":    messageDTOs,
		"IsEmpty":     len(messageDTOs) == 0,
		"Page":        page,
		"TotalCount":  totalCount,
		"HasNext":     beforeID == 0 && page < (totalCount+pageSize-1)/pageSize,
		"HasPrev":     page > 1 || (beforeID > 0 && len(messageDTOs) > 0),
		"NextPage":    page + 1,
		"PrevPage":    page - 1,
		"ChannelName": channel.Name,
	}

	// If there are messages and cursor pagination, set next before_id
	if len(messageDTOs) > 0 && beforeID > 0 {
		data["NextBeforeID"] = messageDTOs[len(messageDTOs)-1].ID
	}

	h.templates.ExecuteTemplate(w, "live/messages.html", data)
}

// handleMessageStream returns new messages since the given ID for live polling.
func (h *ChannelViewHandler) handleMessageStream(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()

	name := r.PathValue("name")
	if name == "" {
		h.renderError(w, r, "Invalid channel name", http.StatusBadRequest)
		return
	}

	// Get channel to get its ID
	channel, err := h.channelRepo.GetByName(ctx, name)
	if err != nil {
		h.logger.Error("failed to get channel", "name", name, "error", err)
		h.renderError(w, r, "Failed to load channel", http.StatusInternalServerError)
		return
	}
	if channel == nil {
		h.renderError(w, r, "Channel not found", http.StatusNotFound)
		return
	}

	afterID, _ := strconv.ParseInt(r.URL.Query().Get("after_id"), 10, 64)

	// Get new messages
	messages, err := h.messageRepo.GetAfterID(ctx, channel.ID, afterID, 50)
	if err != nil {
		h.logger.Error("failed to get stream messages", "channel_id", channel.ID, "error", err)
		h.renderError(w, r, "Failed to load messages", http.StatusInternalServerError)
		return
	}

	// Convert to DTOs
	messageDTOs := h.messagesToDTOs(messages)

	// Get latest ID for next poll
	newLatestID := afterID
	if len(messageDTOs) > 0 {
		newLatestID = messageDTOs[len(messageDTOs)-1].ID
	}

	h.recordStreamPoll(start, len(messageDTOs))

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"messages":  messageDTOs,
			"latest_id": newLatestID,
		})
		return
	}

	// Render stream fragment for HTMX
	data := map[string]any{
		"Messages":    messageDTOs,
		"LatestID":    newLatestID,
		"IsEmpty":     len(messageDTOs) == 0,
		"ChannelName": channel.Name,
	}

	h.templates.ExecuteTemplate(w, "live/stream.html", data)
}

func (h *ChannelViewHandler) parsePagination(r *http.Request) (page, pageSize int) {
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ = strconv.Atoi(r.URL.Query().Get("page_size"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	return page, pageSize
}

func (h *ChannelViewHandler) channelToDTO(ch *repository.Channel) dto.Channel {
	return dto.Channel{
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

func (h *ChannelViewHandler) messagesToDTOs(messages []repository.Message) []dto.Message {
	dtos := make([]dto.Message, len(messages))
	for i, msg := range messages {
		dtos[i] = dto.Message{
			ID:          msg.ID,
			ChannelID:   msg.ChannelID,
			ChannelName: msg.ChannelName,
			UserID:      msg.UserID,
			Username:    msg.Username,
			DisplayName: msg.DisplayName,
			Text:        msg.Text,
			SentAt:      msg.SentAt,
		}
	}
	return dtos
}

func (h *ChannelViewHandler) renderError(w http.ResponseWriter, r *http.Request, message string, status int) {
	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]string{"error": message})
		return
	}

	w.WriteHeader(status)
	h.templates.ExecuteTemplate(w, "error.html", map[string]any{
		"Title":   http.StatusText(status),
		"Message": message,
	})
}

func (h *ChannelViewHandler) wantsJSON(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "application/json")
}

func (h *ChannelViewHandler) recordLatency(start time.Time) {
	if h.metrics != nil {
		h.metrics.RecordHTTPRequest(time.Since(start))
	}
}

func (h *ChannelViewHandler) recordStreamPoll(start time.Time, messageCount int) {
	if h.metrics != nil {
		h.metrics.RecordStreamPoll(time.Since(start), messageCount)
	}
}
