// Package services provides business logic for the Twitch Chat Archiver.
package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/asabla/goknut/internal/repository"
)

var (
	ErrProfileNotFound          = errors.New("profile not found")
	ErrInvalidProfileName       = errors.New("invalid profile name")
	ErrChannelAlreadyLinked     = errors.New("channel already linked to a profile")
	ErrProfileChannelNotFound   = errors.New("profile channel link not found")
	ErrProfileOrganizationQuery = errors.New("failed to query profile organizations")
)

// ProfileService manages profiles and channel linking.
type ProfileService struct {
	profiles *repository.ProfileRepository
	channels *repository.ChannelRepository
}

// NewProfileService creates a new profile service.
func NewProfileService(
	profiles *repository.ProfileRepository,
	channels *repository.ChannelRepository,
) *ProfileService {
	return &ProfileService{profiles: profiles, channels: channels}
}

func (s *ProfileService) List(ctx context.Context) ([]repository.Profile, error) {
	return s.profiles.List(ctx)
}

func (s *ProfileService) Get(ctx context.Context, id int64) (*repository.Profile, error) {
	p, err := s.profiles.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrProfileNotFound
	}
	return p, nil
}

func (s *ProfileService) Create(ctx context.Context, name, description string) (*repository.Profile, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidProfileName
	}

	p := &repository.Profile{Name: name, Description: strings.TrimSpace(description)}
	if err := s.profiles.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *ProfileService) Update(ctx context.Context, id int64, name, description string) (*repository.Profile, error) {
	p, err := s.profiles.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrProfileNotFound
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidProfileName
	}

	p.Name = name
	p.Description = strings.TrimSpace(description)
	if err := s.profiles.Update(ctx, p); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}
	return p, nil
}

func (s *ProfileService) Delete(ctx context.Context, id int64) error {
	err := s.profiles.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrProfileNotFound
		}
		return err
	}
	return nil
}

// LinkChannel links an existing channel (by ID) to a profile.
func (s *ProfileService) LinkChannel(ctx context.Context, profileID, channelID int64) error {
	p, err := s.profiles.GetByID(ctx, profileID)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrProfileNotFound
	}

	ch, err := s.channels.GetByID(ctx, channelID)
	if err != nil {
		return err
	}
	if ch == nil {
		return fmt.Errorf("%w: channel not found", repository.ErrNotFound)
	}

	if err := s.profiles.LinkChannel(ctx, profileID, channelID); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return ErrChannelAlreadyLinked
		}
		return err
	}
	return nil
}

func (s *ProfileService) UnlinkChannel(ctx context.Context, profileID, channelID int64) error {
	if err := s.profiles.UnlinkChannel(ctx, profileID, channelID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrProfileChannelNotFound
		}
		return err
	}
	return nil
}

func (s *ProfileService) ListLinkedChannels(ctx context.Context, profileID int64) ([]repository.Channel, error) {
	p, err := s.profiles.GetByID(ctx, profileID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrProfileNotFound
	}
	return s.profiles.ListLinkedChannels(ctx, profileID)
}
