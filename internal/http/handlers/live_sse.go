// Package handlers provides HTTP handlers for the web UI.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/asabla/goknut/internal/observability"
	"github.com/asabla/goknut/internal/repository"
)

// SSE Event Types
const (
	EventTypeMetrics      = "metrics"
	EventTypeMessage      = "message"
	EventTypeChannelCount = "channel_count"
	EventTypeUserCount    = "user_count"
	EventTypeUserProfile  = "user_profile"
	EventTypeStatus       = "status"
	EventTypeError        = "error"
)

// SSE Status States
const (
	StatusConnected    = "connected"
	StatusIdle         = "idle"
	StatusReconnecting = "reconnecting"
	StatusFallback     = "fallback"
	StatusError        = "error"
)

// SSE Configuration
const (
	SSEMaxBackfill     = 500 // Maximum events to backfill on reconnect
	SSEHeartbeatPeriod = 30 * time.Second
	SSEWriteTimeout    = 10 * time.Second
	SSEBufferSize      = 100 // Per-connection event buffer
)

// SSEEvent represents a Server-Sent Event envelope.
type SSEEvent struct {
	Type   string `json:"type"`
	Cursor int64  `json:"cursor"`
}

// MetricsEvent represents a metrics update event.
type MetricsEvent struct {
	SSEEvent
	TotalMessages int64 `json:"total_messages"`
	TotalChannels int64 `json:"total_channels"`
	TotalUsers    int64 `json:"total_users"`
}

// MessageEvent represents a new message event.
type MessageEvent struct {
	SSEEvent
	ID          int64     `json:"id"`
	ChannelID   int64     `json:"channel_id"`
	ChannelName string    `json:"channel_name,omitempty"`
	UserID      int64     `json:"user_id"`
	Username    string    `json:"username,omitempty"`
	DisplayName string    `json:"display_name,omitempty"`
	Text        string    `json:"text"`
	SentAt      time.Time `json:"sent_at"`
}

// ChannelCountEvent represents a channel message count update.
type ChannelCountEvent struct {
	SSEEvent
	ChannelID     int64     `json:"channel_id"`
	ChannelName   string    `json:"channel_name,omitempty"`
	TotalMessages int64     `json:"total_messages"`
	LastMessageAt time.Time `json:"last_message_at,omitempty"`
}

// UserCountEvent represents a user message count update.
type UserCountEvent struct {
	SSEEvent
	UserID        int64     `json:"user_id"`
	Username      string    `json:"username,omitempty"`
	TotalMessages int64     `json:"total_messages"`
	LastSeenAt    time.Time `json:"last_seen_at,omitempty"`
}

// UserProfileEvent represents a user profile update event.
type UserProfileEvent struct {
	SSEEvent
	UserID        int64      `json:"user_id"`
	Username      string     `json:"username,omitempty"`
	TotalMessages int64      `json:"total_messages"`
	LastSeenAt    time.Time  `json:"last_seen_at,omitempty"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
	MessageID     *int64     `json:"message_id,omitempty"`
}

// StatusEvent represents a connection status event.
type StatusEvent struct {
	SSEEvent
	State        string `json:"state"`
	Reason       string `json:"reason,omitempty"`
	RetryAfterMs int    `json:"retry_after_ms,omitempty"`
}

// ErrorEvent represents an error event.
type ErrorEvent struct {
	SSEEvent
	Message      string `json:"message"`
	RetryAfterMs int    `json:"retry_after_ms,omitempty"`
}

// SSEClient represents a connected SSE client.
type SSEClient struct {
	ID          string
	View        string
	AfterID     int64
	Channel     string // For channel-scoped views
	User        string // For user-scoped views
	Events      chan []byte
	Done        chan struct{}
	ConnectedAt time.Time
}

// SSEHandler handles Server-Sent Events for live updates.
type SSEHandler struct {
	channelRepo  *repository.ChannelRepository
	messageRepo  *repository.MessageRepository
	userRepo     *repository.UserRepository
	templates    *template.Template
	logger       *observability.Logger
	metrics      *observability.Metrics
	otelProvider *observability.OTelProvider

	// Client management
	mu      sync.RWMutex
	clients map[string]*SSEClient

	// Shutdown management
	shutdownOnce sync.Once
	shutdownCh   chan struct{}
}

// NewSSEHandler creates a new SSE handler.
func NewSSEHandler(
	channelRepo *repository.ChannelRepository,
	messageRepo *repository.MessageRepository,
	userRepo *repository.UserRepository,
	templates *template.Template,
	logger *observability.Logger,
	metrics *observability.Metrics,
	otelProvider *observability.OTelProvider,
) *SSEHandler {
	return &SSEHandler{
		channelRepo:  channelRepo,
		messageRepo:  messageRepo,
		userRepo:     userRepo,
		templates:    templates,
		logger:       logger,
		metrics:      metrics,
		otelProvider: otelProvider,
		clients:      make(map[string]*SSEClient),
		shutdownCh:   make(chan struct{}),
	}
}

// RegisterRoutes registers SSE routes on the mux.
func (h *SSEHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /live", h.HandleSSE)
}

// Shutdown gracefully closes all SSE client connections.
func (h *SSEHandler) Shutdown() {
	h.shutdownOnce.Do(func() {
		h.logger.HTTP("shutting down SSE handler", "clients", h.GetClientCount())
		close(h.shutdownCh)
	})
}

// HandleSSE handles SSE connections for live updates.
func (h *SSEHandler) HandleSSE(w http.ResponseWriter, r *http.Request) {
	// Check if the client supports SSE
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.sendJSONError(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Parse query parameters
	view := r.URL.Query().Get("view")
	if view == "" {
		view = "home" // Default to home view
	}

	afterID, _ := strconv.ParseInt(r.URL.Query().Get("after_id"), 10, 64)
	channel := r.URL.Query().Get("channel")
	user := r.URL.Query().Get("user")

	// Validate view
	validViews := map[string]bool{
		"home":         true,
		"messages":     true,
		"channels":     true,
		"users":        true,
		"user_profile": true,
	}
	if !validViews[view] {
		h.sendJSONError(w, fmt.Sprintf("invalid view: %s", view), http.StatusBadRequest)
		return
	}

	// Validate user_profile requires user parameter
	if view == "user_profile" && user == "" {
		h.sendJSONError(w, "user parameter required for user_profile view", http.StatusBadRequest)
		return
	}

	// Create client
	clientID := fmt.Sprintf("%s-%d", view, time.Now().UnixNano())
	client := &SSEClient{
		ID:          clientID,
		View:        view,
		AfterID:     afterID,
		Channel:     channel,
		User:        user,
		Events:      make(chan []byte, SSEBufferSize),
		Done:        make(chan struct{}),
		ConnectedAt: time.Now(),
	}

	// Register client
	h.registerClient(client)
	defer h.unregisterClient(client)

	// Record connection metrics
	if h.metrics != nil {
		h.metrics.RecordSSEConnect(view)
	}
	if h.otelProvider != nil {
		h.otelProvider.RecordSSEConnect(r.Context(), view)
	}
	h.logger.HTTP("SSE client connected", "client_id", clientID, "view", view, "after_id", afterID)

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Send initial connected status
	h.sendEvent(client, h.createStatusEvent(StatusConnected, "", 0))
	flusher.Flush()

	// Send backfill if after_id provided
	if afterID > 0 {
		if err := h.sendBackfill(r.Context(), client, flusher); err != nil {
			h.logger.Error("failed to send backfill", "error", err, "client_id", clientID)
		}
	}

	// Send initial data based on view
	if err := h.sendInitialData(r.Context(), client, flusher); err != nil {
		h.logger.Error("failed to send initial data", "error", err, "client_id", clientID)
	}

	// Create heartbeat ticker
	heartbeat := time.NewTicker(SSEHeartbeatPeriod)
	defer heartbeat.Stop()

	// Event loop
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			h.logger.HTTP("SSE client disconnected", "client_id", clientID, "reason", "context_done")
			if h.metrics != nil {
				h.metrics.RecordSSEDisconnect(view, "context_done")
			}
			if h.otelProvider != nil {
				h.otelProvider.RecordSSEDisconnect(r.Context(), view)
			}
			return

		case <-h.shutdownCh:
			// Server shutting down
			h.logger.HTTP("SSE client disconnected", "client_id", clientID, "reason", "server_shutdown")
			if h.metrics != nil {
				h.metrics.RecordSSEDisconnect(view, "server_shutdown")
			}
			if h.otelProvider != nil {
				h.otelProvider.RecordSSEDisconnect(r.Context(), view)
			}
			return

		case <-client.Done:
			// Server closing connection
			h.logger.HTTP("SSE client disconnected", "client_id", clientID, "reason", "server_close")
			if h.metrics != nil {
				h.metrics.RecordSSEDisconnect(view, "server_close")
			}
			if h.otelProvider != nil {
				h.otelProvider.RecordSSEDisconnect(r.Context(), view)
			}
			return

		case event := <-client.Events:
			// Send event to client
			if _, err := w.Write(event); err != nil {
				h.logger.Error("failed to write SSE event", "error", err, "client_id", clientID)
				return
			}
			flusher.Flush()

		case <-heartbeat.C:
			// Send heartbeat (comment line)
			if _, err := fmt.Fprintf(w, ": heartbeat %d\n\n", time.Now().Unix()); err != nil {
				h.logger.Error("failed to write heartbeat", "error", err, "client_id", clientID)
				return
			}
			flusher.Flush()
		}
	}
}

// registerClient adds a client to the active clients map.
func (h *SSEHandler) registerClient(client *SSEClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client.ID] = client
}

// unregisterClient removes a client from the active clients map.
func (h *SSEHandler) unregisterClient(client *SSEClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, client.ID)
	close(client.Done)
}

// GetClientCount returns the number of connected clients.
func (h *SSEHandler) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetClientsByView returns clients subscribed to a specific view.
func (h *SSEHandler) GetClientsByView(view string) []*SSEClient {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var clients []*SSEClient
	for _, client := range h.clients {
		if client.View == view {
			clients = append(clients, client)
		}
	}
	return clients
}

// BroadcastToView sends an event to all clients subscribed to a view.
func (h *SSEHandler) BroadcastToView(view string, event any) {
	clients := h.GetClientsByView(view)
	for _, client := range clients {
		h.sendEvent(client, event)
	}
}

// BroadcastToAll sends an event to all connected clients.
func (h *SSEHandler) BroadcastToAll(event any) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		h.sendEvent(client, event)
	}
}

// BroadcastMessage broadcasts a new message to all relevant SSE clients.
// This is called when a new message is stored in the database.
func (h *SSEHandler) BroadcastMessage(id, channelID int64, channelName string,
	userID int64, username, displayName, text string, sentAt time.Time) {

	event := MessageEvent{
		SSEEvent:    SSEEvent{Type: EventTypeMessage, Cursor: id},
		ID:          id,
		ChannelID:   channelID,
		ChannelName: channelName,
		UserID:      userID,
		Username:    username,
		DisplayName: displayName,
		Text:        text,
		SentAt:      sentAt,
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		// Send to home view (shows all messages)
		if client.View == "home" {
			h.sendEvent(client, event)
			continue
		}

		// Send to messages view (shows all messages)
		if client.View == "messages" {
			h.sendEvent(client, event)
			continue
		}

		// Send to channel-specific views
		if client.View == "channels" && client.Channel == channelName {
			h.sendEvent(client, event)
			continue
		}

		// Send to user profile views
		if client.View == "user_profile" && client.User == username {
			h.sendEvent(client, event)
		}
	}
}

// sendEvent sends a formatted SSE event to a client.
func (h *SSEHandler) sendEvent(client *SSEClient, event any) {
	data, err := json.Marshal(event)
	if err != nil {
		h.logger.Error("failed to marshal SSE event", "error", err)
		return
	}

	// Format as SSE event
	sseData := fmt.Sprintf("data: %s\n\n", data)

	// Non-blocking send with backpressure handling
	select {
	case client.Events <- []byte(sseData):
		// Event sent successfully
		if h.otelProvider != nil {
			h.otelProvider.RecordSSEEventSent(context.Background())
		}
	default:
		// Buffer full, record backpressure
		if h.metrics != nil {
			h.metrics.RecordSSEBackpressure(client.View)
		}
		if h.otelProvider != nil {
			h.otelProvider.RecordSSEBackpressure(context.Background(), client.View)
		}
		h.logger.Warn("SSE client buffer full, dropping event", "client_id", client.ID, "view", client.View)
	}
}

// sendBackfill sends missed events to a reconnecting client.
func (h *SSEHandler) sendBackfill(ctx context.Context, client *SSEClient, flusher http.Flusher) error {
	if client.AfterID <= 0 {
		return nil
	}

	// Get messages after the cursor
	messages, err := h.messageRepo.GetGlobalAfterID(ctx, client.AfterID, SSEMaxBackfill)
	if err != nil {
		return fmt.Errorf("failed to get backfill messages: %w", err)
	}

	// Check if backlog is too large
	if len(messages) >= SSEMaxBackfill {
		// Too many messages, suggest fallback
		h.sendEvent(client, h.createStatusEvent(StatusFallback, "backlog too large, please refresh", 0))
		flusher.Flush()
		return nil
	}

	// Send backfill messages
	for _, msg := range messages {
		event := h.messageToEvent(&msg)
		h.sendEvent(client, event)
	}

	if len(messages) > 0 {
		flusher.Flush()
		h.logger.HTTP("SSE backfill sent", "client_id", client.ID, "count", len(messages))
	}

	return nil
}

// sendInitialData sends initial data based on the view type.
func (h *SSEHandler) sendInitialData(ctx context.Context, client *SSEClient, flusher http.Flusher) error {
	switch client.View {
	case "home":
		return h.sendHomeData(ctx, client, flusher)
	case "messages":
		// Messages view starts with empty state, waits for new messages
		return nil
	case "channels":
		return h.sendChannelsData(ctx, client, flusher)
	case "users":
		return h.sendUsersData(ctx, client, flusher)
	case "user_profile":
		return h.sendUserProfileData(ctx, client, flusher)
	default:
		return nil
	}
}

// sendHomeData sends initial home view data (metrics).
func (h *SSEHandler) sendHomeData(ctx context.Context, client *SSEClient, flusher http.Flusher) error {
	var totalMessages, totalChannels, totalUsers int64

	if h.messageRepo != nil {
		totalMessages, _ = h.messageRepo.GetTotalCount(ctx)
	}
	if h.channelRepo != nil {
		totalChannels, _ = h.channelRepo.GetCount(ctx)
	}
	if h.userRepo != nil {
		totalUsers, _ = h.userRepo.GetCount(ctx)
	}

	// Get latest message ID as cursor
	var cursor int64
	if h.messageRepo != nil {
		cursor, _ = h.messageRepo.GetGlobalLatestID(ctx)
	}

	event := MetricsEvent{
		SSEEvent:      SSEEvent{Type: EventTypeMetrics, Cursor: cursor},
		TotalMessages: totalMessages,
		TotalChannels: totalChannels,
		TotalUsers:    totalUsers,
	}

	h.sendEvent(client, event)
	flusher.Flush()
	return nil
}

// sendChannelsData sends initial channel counts.
func (h *SSEHandler) sendChannelsData(ctx context.Context, client *SSEClient, flusher http.Flusher) error {
	if h.channelRepo == nil {
		return nil
	}

	channels, err := h.channelRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to get channels: %w", err)
	}

	for _, ch := range channels {
		var lastMessageAt time.Time
		if ch.LastMessageAt != nil {
			lastMessageAt = *ch.LastMessageAt
		}
		event := ChannelCountEvent{
			SSEEvent:      SSEEvent{Type: EventTypeChannelCount, Cursor: 0},
			ChannelID:     ch.ID,
			ChannelName:   ch.Name,
			TotalMessages: ch.TotalMessages,
			LastMessageAt: lastMessageAt,
		}
		h.sendEvent(client, event)
	}

	flusher.Flush()
	return nil
}

// sendUsersData sends initial user counts.
func (h *SSEHandler) sendUsersData(ctx context.Context, client *SSEClient, flusher http.Flusher) error {
	// Users data will be sent on demand
	return nil
}

// sendUserProfileData sends initial user profile data.
func (h *SSEHandler) sendUserProfileData(ctx context.Context, client *SSEClient, flusher http.Flusher) error {
	if h.userRepo == nil || client.User == "" {
		return nil
	}

	user, err := h.userRepo.GetByUsername(ctx, client.User)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil
	}

	event := UserProfileEvent{
		SSEEvent:      SSEEvent{Type: EventTypeUserProfile, Cursor: 0},
		UserID:        user.ID,
		Username:      user.Username,
		TotalMessages: user.TotalMessages,
		LastSeenAt:    user.LastSeenAt,
	}

	h.sendEvent(client, event)
	flusher.Flush()
	return nil
}

// createStatusEvent creates a status event.
func (h *SSEHandler) createStatusEvent(state, reason string, retryAfterMs int) StatusEvent {
	return StatusEvent{
		SSEEvent:     SSEEvent{Type: EventTypeStatus, Cursor: 0},
		State:        state,
		Reason:       reason,
		RetryAfterMs: retryAfterMs,
	}
}

// messageToEvent converts a repository message to an SSE event.
func (h *SSEHandler) messageToEvent(msg *repository.Message) MessageEvent {
	return MessageEvent{
		SSEEvent:    SSEEvent{Type: EventTypeMessage, Cursor: msg.ID},
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

// sendJSONError sends a JSON error response.
func (h *SSEHandler) sendJSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
