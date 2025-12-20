package unit

import (
	"testing"

	"github.com/asabla/goknut/internal/repository"
	"github.com/asabla/goknut/internal/services"
)

func TestCollaborationServiceValidateMinimumParticipants(t *testing.T) {
	svc := &services.CollaborationService{}

	t.Run("zero participants", func(t *testing.T) {
		if err := svc.ValidateMinimumParticipants(nil); err == nil {
			t.Fatalf("expected error, got nil")
		} else if err != services.ErrCollaborationTooFewPeople {
			t.Fatalf("expected %v, got %v", services.ErrCollaborationTooFewPeople, err)
		}
	})

	t.Run("one participant", func(t *testing.T) {
		participants := []repository.Profile{{ID: 1, Name: "p1"}}
		if err := svc.ValidateMinimumParticipants(participants); err == nil {
			t.Fatalf("expected error, got nil")
		} else if err != services.ErrCollaborationTooFewPeople {
			t.Fatalf("expected %v, got %v", services.ErrCollaborationTooFewPeople, err)
		}
	})

	t.Run("two participants", func(t *testing.T) {
		participants := []repository.Profile{{ID: 1, Name: "p1"}, {ID: 2, Name: "p2"}}
		if err := svc.ValidateMinimumParticipants(participants); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})
}
