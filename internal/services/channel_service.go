// Package services provides business logic for the Twitch Chat Archiver.
package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/asabla/goknut/internal/observability"
	"github.com/asabla/goknut/internal/repository"
)

// ChannelService errors
var (
	ErrChannelNotFound      = errors.New("channel not found")
	ErrChannelAlreadyExists = errors.New("channel already exists")
	ErrInvalidChannelName   = errors.New("invalid channel name")
)

// IRCController is the interface for IRC operations.
type IRCController interface {
	Join(channel string) error
	Part(channel string) error
	IsConnected() bool
}

// ChannelService manages channel lifecycle operations.
type ChannelService struct {
	repo    *repository.ChannelRepository
	irc     IRCController
	logger  *observability.Logger
	metrics *observability.Metrics

	mu       sync.RWMutex
	channels map[string]*repository.Channel // name -> channel
}

// NewChannelService creates a new channel service.
func NewChannelService(
	repo *repository.ChannelRepository,
	irc IRCController,
	logger *observability.Logger,
	metrics *observability.Metrics,
) *ChannelService {
	return &ChannelService{
		repo:     repo,
		irc:      irc,
		logger:   logger,
		metrics:  metrics,
		channels: make(map[string]*repository.Channel),
	}
}

// Initialize loads existing channels and joins enabled ones.
func (s *ChannelService) Initialize(ctx context.Context) error {
	channels, err := s.repo.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to load channels: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range channels {
		ch := &channels[i]
		s.channels[ch.Name] = ch

		if ch.Enabled && s.irc != nil && s.irc.IsConnected() {
			if err := s.irc.Join("#" + ch.Name); err != nil {
				s.logger.Error("failed to join channel", "channel", ch.Name, "error", err)
			} else {
				s.logger.IRC("joined channel", "channel", ch.Name)
			}
		}
	}

	s.logger.Info("initialized channel service", "count", len(channels))
	return nil
}

// List returns all channels.
func (s *ChannelService) List(ctx context.Context) ([]repository.Channel, error) {
	return s.repo.List(ctx)
}

// Get returns a channel by ID.
func (s *ChannelService) Get(ctx context.Context, id int64) (*repository.Channel, error) {
	ch, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, ErrChannelNotFound
	}
	return ch, nil
}

// Create creates a new channel.
func (s *ChannelService) Create(ctx context.Context, name, displayName string, enabled, retainHistory bool) (*repository.Channel, error) {
	name = normalizeChannelName(name)
	if name == "" {
		return nil, ErrInvalidChannelName
	}

	// Check if channel already exists
	existing, err := s.repo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrChannelAlreadyExists
	}

	if displayName == "" {
		displayName = name
	}

	ch := &repository.Channel{
		Name:                  name,
		DisplayName:           displayName,
		Enabled:               enabled,
		RetainHistoryOnDelete: retainHistory,
	}

	if err := s.repo.Create(ctx, ch); err != nil {
		return nil, err
	}

	// Update cache
	s.mu.Lock()
	s.channels[name] = ch
	s.mu.Unlock()

	s.logger.Info("created channel", "id", ch.ID, "name", name, "enabled", enabled)

	// Join if enabled and connected
	if enabled && s.irc != nil && s.irc.IsConnected() {
		if err := s.irc.Join("#" + name); err != nil {
			s.logger.Error("failed to join channel after create", "channel", name, "error", err)
		} else {
			s.logger.IRC("joined channel", "channel", name)
		}
	}

	return ch, nil
}

// Update updates a channel.
func (s *ChannelService) Update(ctx context.Context, id int64, displayName *string, enabled *bool, retainHistory *bool) (*repository.Channel, error) {
	ch, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, ErrChannelNotFound
	}

	wasEnabled := ch.Enabled

	// Apply updates
	if displayName != nil {
		ch.DisplayName = *displayName
	}
	if enabled != nil {
		ch.Enabled = *enabled
	}
	if retainHistory != nil {
		ch.RetainHistoryOnDelete = *retainHistory
	}

	if err := s.repo.Update(ctx, ch); err != nil {
		return nil, err
	}

	// Update cache
	s.mu.Lock()
	s.channels[ch.Name] = ch
	s.mu.Unlock()

	s.logger.Info("updated channel", "id", id, "name", ch.Name)

	// Handle IRC join/part based on enabled state change
	if s.irc != nil && s.irc.IsConnected() {
		if !wasEnabled && ch.Enabled {
			// Was disabled, now enabled -> join
			if err := s.irc.Join("#" + ch.Name); err != nil {
				s.logger.Error("failed to join channel after enable", "channel", ch.Name, "error", err)
			} else {
				s.logger.IRC("joined channel", "channel", ch.Name)
			}
		} else if wasEnabled && !ch.Enabled {
			// Was enabled, now disabled -> part
			if err := s.irc.Part("#" + ch.Name); err != nil {
				s.logger.Error("failed to part channel after disable", "channel", ch.Name, "error", err)
			} else {
				s.logger.IRC("left channel", "channel", ch.Name)
			}
		}
	}

	return ch, nil
}

// Delete deletes a channel.
func (s *ChannelService) Delete(ctx context.Context, id int64, retainHistory bool) error {
	ch, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if ch == nil {
		return ErrChannelNotFound
	}

	// Part from IRC if connected and was enabled
	if ch.Enabled && s.irc != nil && s.irc.IsConnected() {
		if err := s.irc.Part("#" + ch.Name); err != nil {
			s.logger.Error("failed to part channel before delete", "channel", ch.Name, "error", err)
		} else {
			s.logger.IRC("left channel", "channel", ch.Name)
		}
	}

	if err := s.repo.Delete(ctx, id, retainHistory); err != nil {
		return err
	}

	// Remove from cache
	s.mu.Lock()
	delete(s.channels, ch.Name)
	s.mu.Unlock()

	s.logger.Info("deleted channel", "id", id, "name", ch.Name, "retain_history", retainHistory)

	return nil
}

// GetByName returns a channel by name from cache or database.
func (s *ChannelService) GetByName(ctx context.Context, name string) (*repository.Channel, error) {
	name = normalizeChannelName(name)

	s.mu.RLock()
	if ch, ok := s.channels[name]; ok {
		s.mu.RUnlock()
		return ch, nil
	}
	s.mu.RUnlock()

	return s.repo.GetByName(ctx, name)
}

// GetChannelID returns the channel ID for a channel name, for use by ingestion.
func (s *ChannelService) GetChannelID(ctx context.Context, name string) (int64, error) {
	ch, err := s.GetByName(ctx, name)
	if err != nil {
		return 0, err
	}
	if ch == nil {
		return 0, ErrChannelNotFound
	}
	return ch.ID, nil
}

func normalizeChannelName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.TrimPrefix(name, "#")
	return name
}
