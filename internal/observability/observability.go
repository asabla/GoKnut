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
}

// NewMetrics creates a new Metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{}
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

	var avgBatchLatency, avgSearchLatency, avgHTTPLatency time.Duration
	if m.batchCount > 0 {
		avgBatchLatency = m.totalBatchLatency / time.Duration(m.batchCount)
	}
	if m.searchQueryCount > 0 {
		avgSearchLatency = m.searchLatencySum / time.Duration(m.searchQueryCount)
	}
	if m.httpRequestCount > 0 {
		avgHTTPLatency = m.httpLatencySum / time.Duration(m.httpRequestCount)
	}

	return MetricsSnapshot{
		IRCConnections:    m.ircConnections,
		IRCDisconnections: m.ircDisconnections,
		IRCMessagesRecv:   m.ircMessagesRecv,
		BatchesProcessed:  m.batchesProcessed,
		MessagesIngested:  m.messagesIngested,
		DroppedMessages:   m.droppedMessages,
		AvgBatchLatency:   avgBatchLatency,
		SearchQueries:     m.searchQueries,
		AvgSearchLatency:  avgSearchLatency,
		HTTPRequests:      m.httpRequests,
		AvgHTTPLatency:    avgHTTPLatency,
	}
}

// MetricsSnapshot is a point-in-time snapshot of metrics.
type MetricsSnapshot struct {
	IRCConnections    int64
	IRCDisconnections int64
	IRCMessagesRecv   int64
	BatchesProcessed  int64
	MessagesIngested  int64
	DroppedMessages   int64
	AvgBatchLatency   time.Duration
	SearchQueries     int64
	AvgSearchLatency  time.Duration
	HTTPRequests      int64
	AvgHTTPLatency    time.Duration
}
