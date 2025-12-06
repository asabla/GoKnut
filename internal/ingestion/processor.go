// Package ingestion provides the message processor for normalizing and storing messages.
package ingestion

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/asabla/goknut/internal/observability"
	"github.com/asabla/goknut/internal/repository"
)

// ProcessorConfig holds processor configuration.
type ProcessorConfig struct {
	Logger  *observability.Logger
	Metrics *observability.Metrics
}

// Processor normalizes incoming messages and stores them in the database.
type Processor struct {
	messageRepo *repository.MessageRepository
	userRepo    *repository.UserRepository
	channelRepo *repository.ChannelRepository
	logger      *observability.Logger
	metrics     *observability.Metrics

	// Cache for channel name -> ID mapping
	channelCache   map[string]int64
	channelCacheMu sync.RWMutex

	// Cache for username -> user ID mapping
	userCache   map[string]int64
	userCacheMu sync.RWMutex
}

// NewProcessor creates a new message processor.
func NewProcessor(
	messageRepo *repository.MessageRepository,
	userRepo *repository.UserRepository,
	channelRepo *repository.ChannelRepository,
	cfg ProcessorConfig,
) *Processor {
	return &Processor{
		messageRepo:  messageRepo,
		userRepo:     userRepo,
		channelRepo:  channelRepo,
		logger:       cfg.Logger,
		metrics:      cfg.Metrics,
		channelCache: make(map[string]int64),
		userCache:    make(map[string]int64),
	}
}

// StoreBatch implements the MessageStore interface for the ingestion pipeline.
func (p *Processor) StoreBatch(ctx context.Context, messages []Message) error {
	if len(messages) == 0 {
		return nil
	}

	start := time.Now()

	// Convert ingestion messages to repository messages
	repoMessages := make([]repository.Message, 0, len(messages))

	for _, msg := range messages {
		// Normalize channel name
		channelName := normalizeChannelName(msg.ChannelName)
		if channelName == "" {
			continue
		}

		// Get or create channel ID
		channelID, err := p.getChannelID(ctx, channelName)
		if err != nil {
			if p.logger != nil {
				p.logger.Error("failed to get channel ID",
					"channel", channelName,
					"error", err,
				)
			}
			continue
		}
		if channelID == 0 {
			// Channel doesn't exist, skip
			continue
		}

		// Normalize username
		username := normalizeUsername(msg.Username)
		if username == "" {
			continue
		}

		// Get or create user ID
		userID, err := p.getOrCreateUserID(ctx, username, msg.DisplayName)
		if err != nil {
			if p.logger != nil {
				p.logger.Error("failed to get user ID",
					"username", username,
					"error", err,
				)
			}
			continue
		}

		// Create repository message
		repoMsg := repository.Message{
			ChannelID: channelID,
			UserID:    userID,
			Text:      msg.Text,
			SentAt:    msg.ReceivedAt,
			Tags:      msg.Tags,
		}
		repoMessages = append(repoMessages, repoMsg)
	}

	if len(repoMessages) == 0 {
		return nil
	}

	// Store batch
	if err := p.messageRepo.CreateBatch(ctx, repoMessages); err != nil {
		if p.logger != nil {
			p.logger.Error("failed to store message batch",
				"count", len(repoMessages),
				"error", err,
			)
		}
		return err
	}

	// Record metrics
	latency := time.Since(start)
	if p.metrics != nil {
		p.metrics.RecordBatchSize(len(repoMessages))
		p.metrics.RecordBatchLatency(latency)
	}

	if p.logger != nil {
		p.logger.Ingestion("stored message batch",
			"count", len(repoMessages),
			"latency_ms", latency.Milliseconds(),
			"dropped", len(messages)-len(repoMessages),
		)
	}

	return nil
}

// getChannelID returns the channel ID for a channel name from cache or database.
func (p *Processor) getChannelID(ctx context.Context, channelName string) (int64, error) {
	// Check cache first
	p.channelCacheMu.RLock()
	if id, ok := p.channelCache[channelName]; ok {
		p.channelCacheMu.RUnlock()
		return id, nil
	}
	p.channelCacheMu.RUnlock()

	// Query database
	channel, err := p.channelRepo.GetByName(ctx, channelName)
	if err != nil {
		return 0, err
	}
	if channel == nil {
		return 0, nil // Channel doesn't exist
	}

	// Update cache
	p.channelCacheMu.Lock()
	p.channelCache[channelName] = channel.ID
	p.channelCacheMu.Unlock()

	return channel.ID, nil
}

// getOrCreateUserID returns the user ID for a username, creating if necessary.
func (p *Processor) getOrCreateUserID(ctx context.Context, username, displayName string) (int64, error) {
	// Check cache first
	p.userCacheMu.RLock()
	if id, ok := p.userCache[username]; ok {
		p.userCacheMu.RUnlock()
		return id, nil
	}
	p.userCacheMu.RUnlock()

	// Get or create user
	user, err := p.userRepo.GetOrCreate(ctx, username, displayName)
	if err != nil {
		return 0, err
	}

	// Update cache
	p.userCacheMu.Lock()
	p.userCache[username] = user.ID
	p.userCacheMu.Unlock()

	return user.ID, nil
}

// ClearCaches clears the channel and user caches.
func (p *Processor) ClearCaches() {
	p.channelCacheMu.Lock()
	p.channelCache = make(map[string]int64)
	p.channelCacheMu.Unlock()

	p.userCacheMu.Lock()
	p.userCache = make(map[string]int64)
	p.userCacheMu.Unlock()
}

// InvalidateChannelCache removes a channel from the cache.
func (p *Processor) InvalidateChannelCache(channelName string) {
	p.channelCacheMu.Lock()
	delete(p.channelCache, normalizeChannelName(channelName))
	p.channelCacheMu.Unlock()
}

// normalizeChannelName normalizes a channel name (removes # prefix, lowercase).
func normalizeChannelName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.TrimPrefix(name, "#")
	return name
}

// normalizeUsername normalizes a username (lowercase, trim).
func normalizeUsername(username string) string {
	return strings.TrimSpace(strings.ToLower(username))
}
