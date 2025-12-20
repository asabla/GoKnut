// Package services provides business logic for the Twitch Chat Archiver.
package services

import (
	"context"
	"errors"
	"strings"

	"github.com/asabla/goknut/internal/repository"
)

var (
	ErrCollaborationNotFound      = errors.New("collaboration not found")
	ErrInvalidCollaborationName   = errors.New("invalid collaboration name")
	ErrCollaborationTooFewPeople  = errors.New("collaboration must have at least two participants")
	ErrCollaborationParticipant   = errors.New("participant already exists")
	ErrCollaborationNoParticipant = errors.New("participant not found")
)

// CollaborationService manages collaborations and their participants.
type CollaborationService struct {
	collabs  *repository.CollaborationRepository
	profiles *repository.ProfileRepository
}

func NewCollaborationService(
	collabs *repository.CollaborationRepository,
	profiles *repository.ProfileRepository,
) *CollaborationService {
	return &CollaborationService{collabs: collabs, profiles: profiles}
}

func (s *CollaborationService) List(ctx context.Context) ([]repository.Collaboration, error) {
	return s.collabs.List(ctx)
}

func (s *CollaborationService) Get(ctx context.Context, id int64) (*repository.Collaboration, error) {
	c, err := s.collabs.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCollaborationNotFound
	}
	return c, nil
}

func (s *CollaborationService) Create(ctx context.Context, name, description string, sharedChat bool) (*repository.Collaboration, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidCollaborationName
	}

	c := &repository.Collaboration{Name: name, Description: strings.TrimSpace(description), SharedChat: sharedChat}
	if err := s.collabs.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CollaborationService) Update(ctx context.Context, id int64, name, description string, sharedChat bool) (*repository.Collaboration, error) {
	c, err := s.collabs.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCollaborationNotFound
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidCollaborationName
	}

	c.Name = name
	c.Description = strings.TrimSpace(description)
	c.SharedChat = sharedChat
	if err := s.collabs.Update(ctx, c); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrCollaborationNotFound
		}
		return nil, err
	}

	return c, nil
}

func (s *CollaborationService) Delete(ctx context.Context, id int64) error {
	err := s.collabs.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrCollaborationNotFound
		}
		return err
	}
	return nil
}

func (s *CollaborationService) AddParticipant(ctx context.Context, collaborationID, profileID int64) error {
	c, err := s.collabs.GetByID(ctx, collaborationID)
	if err != nil {
		return err
	}
	if c == nil {
		return ErrCollaborationNotFound
	}

	p, err := s.profiles.GetByID(ctx, profileID)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrProfileNotFound
	}

	if err := s.collabs.AddParticipant(ctx, collaborationID, profileID); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return ErrCollaborationParticipant
		}
		return err
	}
	return nil
}

func (s *CollaborationService) RemoveParticipant(ctx context.Context, collaborationID, profileID int64) error {
	if err := s.collabs.RemoveParticipant(ctx, collaborationID, profileID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrCollaborationNoParticipant
		}
		return err
	}
	return nil
}

func (s *CollaborationService) ListParticipants(ctx context.Context, collaborationID int64) ([]repository.Profile, error) {
	c, err := s.collabs.GetByID(ctx, collaborationID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCollaborationNotFound
	}
	return s.collabs.ListParticipants(ctx, collaborationID)
}

// ValidateMinimumParticipants checks the minimum participants rule (2+).
// Callers can use this after loading participants (e.g., in handlers).
func (s *CollaborationService) ValidateMinimumParticipants(participants []repository.Profile) error {
	if len(participants) < 2 {
		return ErrCollaborationTooFewPeople
	}
	return nil
}
