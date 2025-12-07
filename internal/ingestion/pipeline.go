// Package ingestion provides message ingestion pipeline with batching.
package ingestion

import (
	"context"
	"sync"
	"time"
)

// Message represents a chat message to be ingested.
type Message struct {
	ChannelName string
	Username    string
	DisplayName string
	Text        string
	Tags        map[string]string
	ReceivedAt  time.Time
}

// MessageStore is the interface for storing messages.
type MessageStore interface {
	// StoreBatch stores a batch of messages.
	StoreBatch(ctx context.Context, messages []Message) error
}

// UserResolver is the interface for resolving user IDs.
type UserResolver interface {
	// GetOrCreateUser returns the user ID for a username, creating if necessary.
	GetOrCreateUser(ctx context.Context, username, displayName string) (int64, error)
}

// ChannelResolver is the interface for resolving channel IDs.
type ChannelResolver interface {
	// GetChannelByName returns the channel ID for a channel name.
	GetChannelByName(ctx context.Context, name string) (int64, error)
}

// Metrics provides hooks for observability.
type Metrics interface {
	RecordBatchSize(size int)
	RecordBatchLatency(d time.Duration)
	RecordDroppedMessages(count int)
}

// Logger provides logging for the pipeline.
type Logger interface {
	Error(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
}

// PipelineConfig holds ingestion pipeline configuration.
type PipelineConfig struct {
	BatchSize    int
	FlushTimeout time.Duration
	BufferSize   int
	Metrics      Metrics
	Logger       Logger
}

// DefaultPipelineConfig returns default pipeline configuration.
func DefaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		BatchSize:    100,
		FlushTimeout: 100 * time.Millisecond,
		BufferSize:   10000,
	}
}

// Pipeline handles message ingestion with batching.
type Pipeline struct {
	cfg      PipelineConfig
	store    MessageStore
	messages chan Message

	mu      sync.Mutex
	batch   []Message
	timer   *time.Timer
	running bool

	done chan struct{}
	wg   sync.WaitGroup
}

// NewPipeline creates a new ingestion pipeline.
func NewPipeline(cfg PipelineConfig, store MessageStore) *Pipeline {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = DefaultPipelineConfig().BatchSize
	}
	if cfg.FlushTimeout <= 0 {
		cfg.FlushTimeout = DefaultPipelineConfig().FlushTimeout
	}
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = DefaultPipelineConfig().BufferSize
	}

	return &Pipeline{
		cfg:      cfg,
		store:    store,
		messages: make(chan Message, cfg.BufferSize),
		batch:    make([]Message, 0, cfg.BatchSize),
		done:     make(chan struct{}),
	}
}

// Start begins processing incoming messages.
func (p *Pipeline) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = true
	p.timer = time.NewTimer(p.cfg.FlushTimeout)
	p.mu.Unlock()

	p.wg.Add(1)
	go p.processLoop(ctx)

	return nil
}

// Stop stops the pipeline and flushes remaining messages.
func (p *Pipeline) Stop() error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = false
	p.mu.Unlock()

	close(p.done)
	p.wg.Wait()

	// Flush any remaining messages
	p.flush(context.Background())

	return nil
}

// Ingest adds a message to the ingestion queue.
func (p *Pipeline) Ingest(msg Message) {
	select {
	case p.messages <- msg:
	default:
		// Buffer full, drop message and record metric
		if p.cfg.Logger != nil {
			p.cfg.Logger.Warn("ingestion buffer full, dropping message",
				"channel", msg.ChannelName,
				"username", msg.Username,
			)
		}
		if p.cfg.Metrics != nil {
			p.cfg.Metrics.RecordDroppedMessages(1)
		}
	}
}

func (p *Pipeline) processLoop(ctx context.Context) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.done:
			return
		case msg := <-p.messages:
			p.addToBatch(ctx, msg)
		case <-p.timer.C:
			p.flush(ctx)
			p.resetTimer()
		}
	}
}

func (p *Pipeline) addToBatch(ctx context.Context, msg Message) {
	p.mu.Lock()
	p.batch = append(p.batch, msg)
	shouldFlush := len(p.batch) >= p.cfg.BatchSize
	p.mu.Unlock()

	if shouldFlush {
		p.flush(ctx)
		p.resetTimer()
	}
}

func (p *Pipeline) flush(ctx context.Context) {
	p.mu.Lock()
	if len(p.batch) == 0 {
		p.mu.Unlock()
		return
	}

	batch := p.batch
	p.batch = make([]Message, 0, p.cfg.BatchSize)
	p.mu.Unlock()

	start := time.Now()

	if err := p.store.StoreBatch(ctx, batch); err != nil {
		// Log error - messages are lost
		if p.cfg.Logger != nil {
			p.cfg.Logger.Error("failed to store message batch",
				"batch_size", len(batch),
				"error", err,
			)
		}
		// Record dropped messages metric
		if p.cfg.Metrics != nil {
			p.cfg.Metrics.RecordDroppedMessages(len(batch))
		}
		return
	}

	if p.cfg.Metrics != nil {
		p.cfg.Metrics.RecordBatchSize(len(batch))
		p.cfg.Metrics.RecordBatchLatency(time.Since(start))
	}
}

func (p *Pipeline) resetTimer() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.timer != nil {
		if !p.timer.Stop() {
			select {
			case <-p.timer.C:
			default:
			}
		}
		p.timer.Reset(p.cfg.FlushTimeout)
	}
}

// BufferLen returns the current buffer length.
func (p *Pipeline) BufferLen() int {
	return len(p.messages)
}

// BatchLen returns the current batch length.
func (p *Pipeline) BatchLen() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.batch)
}
