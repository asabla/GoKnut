// Package observability provides structured logging and metrics for the application.
package observability

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"
)

// Logger wraps slog with component-specific context.
type Logger struct {
	*slog.Logger
}

// NewLogger creates a new structured logger with the given component name.
func NewLogger(component string) *Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(handler).With("component", component)
	return &Logger{Logger: logger}
}

// WithContext returns a logger with request-scoped context.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	// Add any context values (trace ID, request ID, etc.)
	return l
}

// IRC logs an IRC-related event.
func (l *Logger) IRC(msg string, args ...any) {
	l.Info(msg, append([]any{"subsystem", "irc"}, args...)...)
}

// Ingestion logs an ingestion-related event.
func (l *Logger) Ingestion(msg string, args ...any) {
	l.Info(msg, append([]any{"subsystem", "ingestion"}, args...)...)
}

// Search logs a search-related event.
func (l *Logger) Search(msg string, args ...any) {
	l.Info(msg, append([]any{"subsystem", "search"}, args...)...)
}

// HTTP logs an HTTP-related event.
func (l *Logger) HTTP(msg string, args ...any) {
	l.Info(msg, append([]any{"subsystem", "http"}, args...)...)
}

// Metrics provides application metrics collection.
type Metrics struct {
	mu sync.RWMutex

	// IRC metrics
	ircConnections    int64
	ircDisconnections int64
	ircMessagesRecv   int64

	// Ingestion metrics
	batchesProcessed  int64
	messagesIngested  int64
	droppedMessages   int64
	totalBatchLatency time.Duration
	batchCount        int64

	// Search metrics
	searchQueries    int64
	searchLatencySum time.Duration
	searchQueryCount int64

	// HTTP metrics
	httpRequests     int64
	httpLatencySum   time.Duration
	httpRequestCount int64

	// Stream polling metrics
	streamPollRequests int64
	streamLatencySum   time.Duration
	streamPollCount    int64

	// SSE metrics
	sseConnections       int64
	sseDisconnections    int64
	sseBackpressure      int64
	sseEventsSent        int64
	sseConnectionsByView map[string]int64

	// CRUD metrics
	profileCreatesSuccess int64
	profileCreatesError   int64
	profileUpdatesSuccess int64
	profileUpdatesError   int64
	profileDeletesSuccess int64
	profileDeletesError   int64
	profileLinksSuccess   int64
	profileLinksError     int64
	profileUnlinksSuccess int64
	profileUnlinksError   int64

	organizationCreatesSuccess int64
	organizationCreatesError   int64
	organizationUpdatesSuccess int64
	organizationUpdatesError   int64
	organizationDeletesSuccess int64
	organizationDeletesError   int64
	organizationLinksSuccess   int64
	organizationLinksError     int64
	organizationUnlinksSuccess int64
	organizationUnlinksError   int64

	eventCreatesSuccess int64
	eventCreatesError   int64
	eventUpdatesSuccess int64
	eventUpdatesError   int64
	eventDeletesSuccess int64
	eventDeletesError   int64
	eventLinksSuccess   int64
	eventLinksError     int64
	eventUnlinksSuccess int64
	eventUnlinksError   int64

	collaborationCreatesSuccess int64
	collaborationCreatesError   int64
	collaborationUpdatesSuccess int64
	collaborationUpdatesError   int64
	collaborationDeletesSuccess int64
	collaborationDeletesError   int64
	collaborationLinksSuccess   int64
	collaborationLinksError     int64
	collaborationUnlinksSuccess int64
	collaborationUnlinksError   int64
}

// NewMetrics creates a new Metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{
		sseConnectionsByView: make(map[string]int64),
	}
}

// RecordIRCConnection records an IRC connection event.
func (m *Metrics) RecordIRCConnection() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ircConnections++
}

// RecordIRCDisconnection records an IRC disconnection event.
func (m *Metrics) RecordIRCDisconnection() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ircDisconnections++
}

// RecordIRCMessage records an IRC message received.
func (m *Metrics) RecordIRCMessage() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ircMessagesRecv++
}

// RecordBatchSize records the size of a processed batch.
func (m *Metrics) RecordBatchSize(size int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.batchesProcessed++
	m.messagesIngested += int64(size)
}

// RecordBatchLatency records the latency of batch processing.
func (m *Metrics) RecordBatchLatency(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalBatchLatency += d
	m.batchCount++
}

// RecordDroppedMessages records dropped messages.
func (m *Metrics) RecordDroppedMessages(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.droppedMessages += int64(count)
}

// RecordSearchQuery records a search query.
func (m *Metrics) RecordSearchQuery(latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.searchQueries++
	m.searchLatencySum += latency
	m.searchQueryCount++
}

// RecordSearchRequest records a search request with type information.
func (m *Metrics) RecordSearchRequest(searchType string, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.searchQueries++
	m.searchLatencySum += latency
	m.searchQueryCount++
}

// RecordHTTPRequest records an HTTP request.
func (m *Metrics) RecordHTTPRequest(latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.httpRequests++
	m.httpLatencySum += latency
	m.httpRequestCount++
}

// Stats returns a snapshot of current metrics.
func (m *Metrics) Stats() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var avgBatchLatency, avgSearchLatency, avgHTTPLatency, avgStreamLatency time.Duration
	if m.batchCount > 0 {
		avgBatchLatency = m.totalBatchLatency / time.Duration(m.batchCount)
	}
	if m.searchQueryCount > 0 {
		avgSearchLatency = m.searchLatencySum / time.Duration(m.searchQueryCount)
	}
	if m.httpRequestCount > 0 {
		avgHTTPLatency = m.httpLatencySum / time.Duration(m.httpRequestCount)
	}
	if m.streamPollCount > 0 {
		avgStreamLatency = m.streamLatencySum / time.Duration(m.streamPollCount)
	}

	return MetricsSnapshot{
		IRCConnections:     m.ircConnections,
		IRCDisconnections:  m.ircDisconnections,
		IRCMessagesRecv:    m.ircMessagesRecv,
		BatchesProcessed:   m.batchesProcessed,
		MessagesIngested:   m.messagesIngested,
		DroppedMessages:    m.droppedMessages,
		AvgBatchLatency:    avgBatchLatency,
		SearchQueries:      m.searchQueries,
		AvgSearchLatency:   avgSearchLatency,
		HTTPRequests:       m.httpRequests,
		AvgHTTPLatency:     avgHTTPLatency,
		StreamPollRequests: m.streamPollRequests,
		AvgStreamLatency:   avgStreamLatency,

		ProfileCreatesSuccess: m.profileCreatesSuccess,
		ProfileCreatesError:   m.profileCreatesError,
		ProfileUpdatesSuccess: m.profileUpdatesSuccess,
		ProfileUpdatesError:   m.profileUpdatesError,
		ProfileDeletesSuccess: m.profileDeletesSuccess,
		ProfileDeletesError:   m.profileDeletesError,
		ProfileLinksSuccess:   m.profileLinksSuccess,
		ProfileLinksError:     m.profileLinksError,
		ProfileUnlinksSuccess: m.profileUnlinksSuccess,
		ProfileUnlinksError:   m.profileUnlinksError,

		OrganizationCreatesSuccess: m.organizationCreatesSuccess,
		OrganizationCreatesError:   m.organizationCreatesError,
		OrganizationUpdatesSuccess: m.organizationUpdatesSuccess,
		OrganizationUpdatesError:   m.organizationUpdatesError,
		OrganizationDeletesSuccess: m.organizationDeletesSuccess,
		OrganizationDeletesError:   m.organizationDeletesError,
		OrganizationLinksSuccess:   m.organizationLinksSuccess,
		OrganizationLinksError:     m.organizationLinksError,
		OrganizationUnlinksSuccess: m.organizationUnlinksSuccess,
		OrganizationUnlinksError:   m.organizationUnlinksError,

		EventCreatesSuccess: m.eventCreatesSuccess,
		EventCreatesError:   m.eventCreatesError,
		EventUpdatesSuccess: m.eventUpdatesSuccess,
		EventUpdatesError:   m.eventUpdatesError,
		EventDeletesSuccess: m.eventDeletesSuccess,
		EventDeletesError:   m.eventDeletesError,
		EventLinksSuccess:   m.eventLinksSuccess,
		EventLinksError:     m.eventLinksError,
		EventUnlinksSuccess: m.eventUnlinksSuccess,
		EventUnlinksError:   m.eventUnlinksError,

		CollaborationCreatesSuccess: m.collaborationCreatesSuccess,
		CollaborationCreatesError:   m.collaborationCreatesError,
		CollaborationUpdatesSuccess: m.collaborationUpdatesSuccess,
		CollaborationUpdatesError:   m.collaborationUpdatesError,
		CollaborationDeletesSuccess: m.collaborationDeletesSuccess,
		CollaborationDeletesError:   m.collaborationDeletesError,
		CollaborationLinksSuccess:   m.collaborationLinksSuccess,
		CollaborationLinksError:     m.collaborationLinksError,
		CollaborationUnlinksSuccess: m.collaborationUnlinksSuccess,
		CollaborationUnlinksError:   m.collaborationUnlinksError,
	}
}

// MetricsSnapshot is a point-in-time snapshot of metrics.
type MetricsSnapshot struct {
	IRCConnections     int64
	IRCDisconnections  int64
	IRCMessagesRecv    int64
	BatchesProcessed   int64
	MessagesIngested   int64
	DroppedMessages    int64
	AvgBatchLatency    time.Duration
	SearchQueries      int64
	AvgSearchLatency   time.Duration
	HTTPRequests       int64
	AvgHTTPLatency     time.Duration
	StreamPollRequests int64
	AvgStreamLatency   time.Duration

	ProfileCreatesSuccess int64
	ProfileCreatesError   int64
	ProfileUpdatesSuccess int64
	ProfileUpdatesError   int64
	ProfileDeletesSuccess int64
	ProfileDeletesError   int64
	ProfileLinksSuccess   int64
	ProfileLinksError     int64
	ProfileUnlinksSuccess int64
	ProfileUnlinksError   int64

	OrganizationCreatesSuccess int64
	OrganizationCreatesError   int64
	OrganizationUpdatesSuccess int64
	OrganizationUpdatesError   int64
	OrganizationDeletesSuccess int64
	OrganizationDeletesError   int64
	OrganizationLinksSuccess   int64
	OrganizationLinksError     int64
	OrganizationUnlinksSuccess int64
	OrganizationUnlinksError   int64

	EventCreatesSuccess int64
	EventCreatesError   int64
	EventUpdatesSuccess int64
	EventUpdatesError   int64
	EventDeletesSuccess int64
	EventDeletesError   int64
	EventLinksSuccess   int64
	EventLinksError     int64
	EventUnlinksSuccess int64
	EventUnlinksError   int64

	CollaborationCreatesSuccess int64
	CollaborationCreatesError   int64
	CollaborationUpdatesSuccess int64
	CollaborationUpdatesError   int64
	CollaborationDeletesSuccess int64
	CollaborationDeletesError   int64
	CollaborationLinksSuccess   int64
	CollaborationLinksError     int64
	CollaborationUnlinksSuccess int64
	CollaborationUnlinksError   int64
}

// RecordStreamPoll records a stream poll request.
func (m *Metrics) RecordStreamPoll(latency time.Duration, messageCount int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.streamPollRequests++
	m.streamLatencySum += latency
	m.streamPollCount++
}

// RecordSSEConnect records an SSE connection event.
func (m *Metrics) RecordSSEConnect(view string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sseConnections++
	if m.sseConnectionsByView == nil {
		m.sseConnectionsByView = make(map[string]int64)
	}
	m.sseConnectionsByView[view]++
}

// RecordSSEDisconnect records an SSE disconnection event.
func (m *Metrics) RecordSSEDisconnect(view string, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sseDisconnections++
}

// RecordSSEBackpressure records an SSE backpressure event (buffer full).
func (m *Metrics) RecordSSEBackpressure(view string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sseBackpressure++
}

// RecordSSEEventSent records an SSE event sent.
func (m *Metrics) RecordSSEEventSent() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sseEventsSent++
}

func (m *Metrics) RecordProfileCreate(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if success {
		m.profileCreatesSuccess++
		return
	}
	m.profileCreatesError++
}

func (m *Metrics) RecordProfileUpdate(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if success {
		m.profileUpdatesSuccess++
		return
	}
	m.profileUpdatesError++
}

func (m *Metrics) RecordProfileDelete(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if success {
		m.profileDeletesSuccess++
		return
	}
	m.profileDeletesError++
}

func (m *Metrics) RecordProfileLink(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if success {
		m.profileLinksSuccess++
		return
	}
	m.profileLinksError++
}

func (m *Metrics) RecordProfileUnlink(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if success {
		m.profileUnlinksSuccess++
		return
	}
	m.profileUnlinksError++
}

func (m *Metrics) RecordOrganizationCreate(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if success {
		m.organizationCreatesSuccess++
		return
	}
	m.organizationCreatesError++
}

func (m *Metrics) RecordOrganizationUpdate(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if success {
		m.organizationUpdatesSuccess++
		return
	}
	m.organizationUpdatesError++
}

func (m *Metrics) RecordOrganizationDelete(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if success {
		m.organizationDeletesSuccess++
		return
	}
	m.organizationDeletesError++
}

func (m *Metrics) RecordOrganizationLink(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if success {
		m.organizationLinksSuccess++
		return
	}
	m.organizationLinksError++
}

func (m *Metrics) RecordOrganizationUnlink(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if success {
		m.organizationUnlinksSuccess++
		return
	}
	m.organizationUnlinksError++
}

func (m *Metrics) RecordEventCreate(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if success {
		m.eventCreatesSuccess++
		return
	}
	m.eventCreatesError++
}

func (m *Metrics) RecordEventUpdate(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if success {
		m.eventUpdatesSuccess++
		return
	}
	m.eventUpdatesError++
}

func (m *Metrics) RecordEventDelete(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if success {
		m.eventDeletesSuccess++
		return
	}
	m.eventDeletesError++
}

func (m *Metrics) RecordEventLink(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if success {
		m.eventLinksSuccess++
		return
	}
	m.eventLinksError++
}

func (m *Metrics) RecordEventUnlink(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if success {
		m.eventUnlinksSuccess++
		return
	}
	m.eventUnlinksError++
}

func (m *Metrics) RecordCollaborationCreate(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if success {
		m.collaborationCreatesSuccess++
		return
	}
	m.collaborationCreatesError++
}

func (m *Metrics) RecordCollaborationUpdate(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if success {
		m.collaborationUpdatesSuccess++
		return
	}
	m.collaborationUpdatesError++
}

func (m *Metrics) RecordCollaborationDelete(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if success {
		m.collaborationDeletesSuccess++
		return
	}
	m.collaborationDeletesError++
}

func (m *Metrics) RecordCollaborationLink(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if success {
		m.collaborationLinksSuccess++
		return
	}
	m.collaborationLinksError++
}

func (m *Metrics) RecordCollaborationUnlink(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if success {
		m.collaborationUnlinksSuccess++
		return
	}
	m.collaborationUnlinksError++
}

// SSEStats returns SSE-specific metrics.
func (m *Metrics) SSEStats() SSEMetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	viewCounts := make(map[string]int64)
	for k, v := range m.sseConnectionsByView {
		viewCounts[k] = v
	}

	return SSEMetricsSnapshot{
		Connections:       m.sseConnections,
		Disconnections:    m.sseDisconnections,
		Backpressure:      m.sseBackpressure,
		EventsSent:        m.sseEventsSent,
		ConnectionsByView: viewCounts,
	}
}

// SSEMetricsSnapshot is a point-in-time snapshot of SSE metrics.
type SSEMetricsSnapshot struct {
	Connections       int64
	Disconnections    int64
	Backpressure      int64
	EventsSent        int64
	ConnectionsByView map[string]int64
}
