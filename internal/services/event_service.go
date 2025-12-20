// Package services provides business logic for the Twitch Chat Archiver.
package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/asabla/goknut/internal/repository"
)

var (
	ErrEventNotFound      = errors.New("event not found")
	ErrInvalidEventTitle  = errors.New("invalid event title")
	ErrInvalidEventDates  = errors.New("invalid event dates")
	ErrParticipantExists  = errors.New("participant already exists")
	ErrParticipantMissing = errors.New("participant not found")
)

// EventService manages events and their participants.
type EventService struct {
	events   *repository.EventRepository
	profiles *repository.ProfileRepository
}

func NewEventService(
	events *repository.EventRepository,
	profiles *repository.ProfileRepository,
) *EventService {
	return &EventService{events: events, profiles: profiles}
}

func (s *EventService) List(ctx context.Context) ([]repository.Event, error) {
	return s.events.List(ctx)
}

func (s *EventService) Get(ctx context.Context, id int64) (*repository.Event, error) {
	evt, err := s.events.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if evt == nil {
		return nil, ErrEventNotFound
	}
	return evt, nil
}

func (s *EventService) Create(ctx context.Context, title, description string, startAt time.Time, endAt *time.Time) (*repository.Event, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrInvalidEventTitle
	}
	if startAt.IsZero() {
		return nil, ErrInvalidEventDates
	}
	if endAt != nil && !endAt.IsZero() && endAt.Before(startAt) {
		return nil, ErrInvalidEventDates
	}

	evt := &repository.Event{Title: title, Description: strings.TrimSpace(description), StartAt: startAt}
	if endAt != nil && !endAt.IsZero() {
		copy := *endAt
		evt.EndAt = &copy
	}

	if err := s.events.Create(ctx, evt); err != nil {
		return nil, err
	}
	return evt, nil
}

func (s *EventService) Update(ctx context.Context, id int64, title, description string, startAt time.Time, endAt *time.Time) (*repository.Event, error) {
	evt, err := s.events.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if evt == nil {
		return nil, ErrEventNotFound
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrInvalidEventTitle
	}
	if startAt.IsZero() {
		return nil, ErrInvalidEventDates
	}
	if endAt != nil && !endAt.IsZero() && endAt.Before(startAt) {
		return nil, ErrInvalidEventDates
	}

	evt.Title = title
	evt.Description = strings.TrimSpace(description)
	evt.StartAt = startAt
	if endAt != nil && !endAt.IsZero() {
		copy := *endAt
		evt.EndAt = &copy
	} else {
		evt.EndAt = nil
	}

	if err := s.events.Update(ctx, evt); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}
	return evt, nil
}

func (s *EventService) Delete(ctx context.Context, id int64) error {
	err := s.events.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrEventNotFound
		}
		return err
	}
	return nil
}

func (s *EventService) AddParticipant(ctx context.Context, eventID, profileID int64) error {
	evt, err := s.events.GetByID(ctx, eventID)
	if err != nil {
		return err
	}
	if evt == nil {
		return ErrEventNotFound
	}

	p, err := s.profiles.GetByID(ctx, profileID)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrProfileNotFound
	}

	if err := s.events.AddParticipant(ctx, eventID, profileID); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return ErrParticipantExists
		}
		return err
	}
	return nil
}

func (s *EventService) RemoveParticipant(ctx context.Context, eventID, profileID int64) error {
	if err := s.events.RemoveParticipant(ctx, eventID, profileID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrParticipantMissing
		}
		return err
	}
	return nil
}

func (s *EventService) ListParticipants(ctx context.Context, eventID int64) ([]repository.Profile, error) {
	evt, err := s.events.GetByID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if evt == nil {
		return nil, ErrEventNotFound
	}
	return s.events.ListParticipants(ctx, eventID)
}
