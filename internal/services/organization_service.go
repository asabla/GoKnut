// Package services provides business logic for the Twitch Chat Archiver.
package services

import (
	"context"
	"errors"
	"strings"

	"github.com/asabla/goknut/internal/repository"
)

var (
	ErrOrganizationNotFound    = errors.New("organization not found")
	ErrInvalidOrganizationName = errors.New("invalid organization name")
	ErrMembershipAlreadyExists = errors.New("membership already exists")
	ErrMembershipNotFound      = errors.New("membership not found")
)

// OrganizationService manages organizations and memberships.
type OrganizationService struct {
	orgs     *repository.OrganizationRepository
	profiles *repository.ProfileRepository
}

func NewOrganizationService(
	orgs *repository.OrganizationRepository,
	profiles *repository.ProfileRepository,
) *OrganizationService {
	return &OrganizationService{orgs: orgs, profiles: profiles}
}

func (s *OrganizationService) List(ctx context.Context) ([]repository.Organization, error) {
	return s.orgs.List(ctx)
}

func (s *OrganizationService) Get(ctx context.Context, id int64) (*repository.Organization, error) {
	org, err := s.orgs.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, ErrOrganizationNotFound
	}
	return org, nil
}

func (s *OrganizationService) Create(ctx context.Context, name, description string) (*repository.Organization, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidOrganizationName
	}
	org := &repository.Organization{Name: name, Description: strings.TrimSpace(description)}
	if err := s.orgs.Create(ctx, org); err != nil {
		return nil, err
	}
	return org, nil
}

func (s *OrganizationService) Update(ctx context.Context, id int64, name, description string) (*repository.Organization, error) {
	org, err := s.orgs.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, ErrOrganizationNotFound
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidOrganizationName
	}

	org.Name = name
	org.Description = strings.TrimSpace(description)
	if err := s.orgs.Update(ctx, org); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrOrganizationNotFound
		}
		return nil, err
	}
	return org, nil
}

func (s *OrganizationService) Delete(ctx context.Context, id int64) error {
	err := s.orgs.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrOrganizationNotFound
		}
		return err
	}
	return nil
}

func (s *OrganizationService) AddMember(ctx context.Context, organizationID, profileID int64) error {
	org, err := s.orgs.GetByID(ctx, organizationID)
	if err != nil {
		return err
	}
	if org == nil {
		return ErrOrganizationNotFound
	}

	p, err := s.profiles.GetByID(ctx, profileID)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrProfileNotFound
	}

	if err := s.orgs.AddMember(ctx, organizationID, profileID); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return ErrMembershipAlreadyExists
		}
		return err
	}
	return nil
}

func (s *OrganizationService) RemoveMember(ctx context.Context, organizationID, profileID int64) error {
	if err := s.orgs.RemoveMember(ctx, organizationID, profileID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrMembershipNotFound
		}
		return err
	}
	return nil
}

func (s *OrganizationService) ListMembers(ctx context.Context, organizationID int64) ([]repository.Profile, error) {
	org, err := s.orgs.GetByID(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, ErrOrganizationNotFound
	}
	return s.orgs.ListMembers(ctx, organizationID)
}
