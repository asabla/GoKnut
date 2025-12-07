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
	"github.com/asabla/goknut/internal/services"
)

// SearchHandler handles search-related HTTP requests.
type SearchHandler struct {
	service   *services.SearchService
	templates *template.Template
	logger    *observability.Logger
}

// NewSearchHandler creates a new search handler.
func NewSearchHandler(
	service *services.SearchService,
	templates *template.Template,
	logger *observability.Logger,
) *SearchHandler {
	return &SearchHandler{
		service:   service,
		templates: templates,
		logger:    logger,
	}
}

// RegisterRoutes registers search routes on the mux.
func (h *SearchHandler) RegisterRoutes(mux *http.ServeMux) {
	// User search and profiles
	mux.HandleFunc("GET /users", h.handleSearchUsers)
	mux.HandleFunc("GET /users/{username}", h.handleUserProfile)
	mux.HandleFunc("GET /users/{username}/messages", h.handleUserMessages)

	// Message search
	mux.HandleFunc("GET /search/messages", h.handleSearchMessages)
}

func (h *SearchHandler) handleSearchUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	// If no query, show empty search form
	if query == "" {
		h.renderSearchUsersPage(w, r, nil, query, 0, 0, 0)
		return
	}

	req := dto.SearchUsersRequest{
		Query: query,
		PaginationRequest: dto.PaginationRequest{
			Page:     page,
			PageSize: pageSize,
		},
	}

	result, err := h.service.SearchUsers(ctx, req)
	if err != nil {
		h.logger.Error("failed to search users", "query", query, "error", err)
		h.renderError(w, r, "Failed to search users", http.StatusInternalServerError)
		return
	}

	h.renderSearchUsersPage(w, r, result, query, result.Page, result.TotalPages, result.TotalCount)
}

func (h *SearchHandler) renderSearchUsersPage(w http.ResponseWriter, r *http.Request, result *services.UserSearchResult, query string, page, totalPages, totalCount int) {
	var users []dto.User
	if result != nil {
		for _, u := range result.Users {
			users = append(users, dto.User{
				ID:            u.ID,
				Username:      u.Username,
				DisplayName:   u.DisplayName,
				FirstSeenAt:   u.FirstSeenAt,
				LastSeenAt:    u.LastSeenAt,
				TotalMessages: u.TotalMessages,
			})
		}
	}

	data := map[string]any{
		"Query":      query,
		"Users":      users,
		"IsEmpty":    len(users) == 0 && query != "",
		"HasQuery":   query != "",
		"Page":       page,
		"TotalPages": totalPages,
		"TotalCount": totalCount,
		"HasNext":    result != nil && result.HasNext,
		"HasPrev":    result != nil && result.HasPrev,
		"NextPage":   page + 1,
		"PrevPage":   page - 1,
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if h.isHTMXRequest(r) {
		h.templates.ExecuteTemplate(w, "users_results.html", data)
	} else {
		h.templates.ExecuteTemplate(w, "search/users", data)
	}
}

func (h *SearchHandler) handleUserProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	username := r.PathValue("username")
	if username == "" {
		h.renderError(w, r, "Invalid username", http.StatusBadRequest)
		return
	}

	profile, err := h.service.GetUserProfileByUsername(ctx, username)
	if err != nil {
		if err == services.ErrUserNotFound {
			h.renderError(w, r, "User not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to get user profile", "username", username, "error", err)
		h.renderError(w, r, "Failed to load user profile", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"User": dto.User{
			ID:            profile.ID,
			Username:      profile.Username,
			DisplayName:   profile.DisplayName,
			FirstSeenAt:   profile.FirstSeenAt,
			LastSeenAt:    profile.LastSeenAt,
			TotalMessages: profile.TotalMessages,
		},
		"Channels": profile.Channels,
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.templates.ExecuteTemplate(w, "search/user_profile", data)
}

func (h *SearchHandler) handleUserMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	username := r.PathValue("username")
	if username == "" {
		h.renderError(w, r, "Invalid username", http.StatusBadRequest)
		return
	}

	// Channel filter by name instead of ID
	var channelName *string
	if chName := r.URL.Query().Get("channel"); chName != "" {
		channelName = &chName
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	result, err := h.service.GetUserMessagesByUsername(ctx, username, channelName, page, pageSize)
	if err != nil {
		h.logger.Error("failed to get user messages", "username", username, "error", err)
		h.renderError(w, r, "Failed to load messages", http.StatusInternalServerError)
		return
	}

	var messages []dto.Message
	for _, m := range result.Messages {
		messages = append(messages, dto.Message{
			ID:          m.ID,
			ChannelID:   m.ChannelID,
			ChannelName: m.ChannelName,
			UserID:      m.UserID,
			Username:    m.Username,
			DisplayName: m.DisplayName,
			Text:        m.Text,
			SentAt:      m.SentAt,
		})
	}

	data := map[string]any{
		"Messages":    messages,
		"IsEmpty":     len(messages) == 0,
		"Page":        result.Page,
		"TotalPages":  result.TotalPages,
		"TotalCount":  result.TotalCount,
		"HasNext":     result.HasNext,
		"HasPrev":     result.HasPrev,
		"NextPage":    result.Page + 1,
		"PrevPage":    result.Page - 1,
		"Username":    username,
		"ChannelName": channelName,
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.templates.ExecuteTemplate(w, "user_messages.html", data)
}

func (h *SearchHandler) handleSearchMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	// If no query, show empty search form
	if query == "" {
		h.renderSearchMessagesPage(w, r, nil, query, nil, nil, nil, nil)
		return
	}

	// Parse optional filters
	var channelID, userID *int64
	var startTime, endTime *time.Time

	if chID := r.URL.Query().Get("channel_id"); chID != "" {
		id, err := strconv.ParseInt(chID, 10, 64)
		if err == nil {
			channelID = &id
		}
	}

	if uID := r.URL.Query().Get("user_id"); uID != "" {
		id, err := strconv.ParseInt(uID, 10, 64)
		if err == nil {
			userID = &id
		}
	}

	if start := r.URL.Query().Get("start"); start != "" {
		if t, err := time.Parse("2006-01-02", start); err == nil {
			startTime = &t
		}
	}

	if end := r.URL.Query().Get("end"); end != "" {
		if t, err := time.Parse("2006-01-02", end); err == nil {
			// Include the entire day
			endOfDay := t.Add(24*time.Hour - time.Second)
			endTime = &endOfDay
		}
	}

	req := dto.SearchMessagesRequest{
		Query:     query,
		ChannelID: channelID,
		UserID:    userID,
		StartTime: startTime,
		EndTime:   endTime,
		PaginationRequest: dto.PaginationRequest{
			Page:     page,
			PageSize: pageSize,
		},
	}

	// Validate query length
	if len(query) < 2 {
		h.renderError(w, r, "Search query must be at least 2 characters", http.StatusBadRequest)
		return
	}

	result, err := h.service.SearchMessages(ctx, req)
	if err != nil {
		h.logger.Error("failed to search messages", "query", query, "error", err)
		h.renderError(w, r, "Failed to search messages", http.StatusInternalServerError)
		return
	}

	h.renderSearchMessagesPage(w, r, result, query, channelID, userID, startTime, endTime)
}

func (h *SearchHandler) renderSearchMessagesPage(w http.ResponseWriter, r *http.Request, result *services.MessageSearchResult, query string, channelID, userID *int64, startTime, endTime *time.Time) {
	var messages []MessageWithHighlight
	if result != nil {
		for _, m := range result.Messages {
			messages = append(messages, MessageWithHighlight{
				Message: dto.Message{
					ID:          m.ID,
					ChannelID:   m.ChannelID,
					ChannelName: m.ChannelName,
					UserID:      m.UserID,
					Username:    m.Username,
					DisplayName: m.DisplayName,
					Text:        m.Text,
					SentAt:      m.SentAt,
				},
				HighlightedText: template.HTML(m.HighlightedText),
			})
		}
	}

	page := 0
	totalPages := 0
	totalCount := 0
	hasNext := false
	hasPrev := false
	if result != nil {
		page = result.Page
		totalPages = result.TotalPages
		totalCount = result.TotalCount
		hasNext = result.HasNext
		hasPrev = result.HasPrev
	}

	data := map[string]any{
		"Query":      query,
		"Messages":   messages,
		"IsEmpty":    len(messages) == 0 && query != "",
		"HasQuery":   query != "",
		"Page":       page,
		"TotalPages": totalPages,
		"TotalCount": totalCount,
		"HasNext":    hasNext,
		"HasPrev":    hasPrev,
		"NextPage":   page + 1,
		"PrevPage":   page - 1,
		"ChannelID":  channelID,
		"UserID":     userID,
		"StartTime":  startTime,
		"EndTime":    endTime,
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if h.isHTMXRequest(r) {
		h.templates.ExecuteTemplate(w, "messages_results.html", data)
	} else {
		h.templates.ExecuteTemplate(w, "search/messages", data)
	}
}

// MessageWithHighlight wraps a message with its highlighted text.
type MessageWithHighlight struct {
	Message         dto.Message
	HighlightedText template.HTML
}

func (h *SearchHandler) renderError(w http.ResponseWriter, r *http.Request, message string, status int) {
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

func (h *SearchHandler) wantsJSON(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "application/json")
}

func (h *SearchHandler) isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}
