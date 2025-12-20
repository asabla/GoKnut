package unit

import (
	"context"
	"testing"

	"github.com/asabla/goknut/internal/repository"
	"github.com/asabla/goknut/internal/services"
)

func TestOrganizationServiceMembershipUniqueness(t *testing.T) {
	ctx := context.Background()

	db, err := repository.Open(repository.DBConfig{Path: ":memory:", EnableFTS: true})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	orgRepo := repository.NewOrganizationRepository(db)
	profileRepo := repository.NewProfileRepository(db)
	service := services.NewOrganizationService(orgRepo, profileRepo)

	org, err := service.Create(ctx, "Org One", "")
	if err != nil {
		t.Fatalf("failed to create organization: %v", err)
	}

	profile := &repository.Profile{Name: "Profile One"}
	if err := profileRepo.Create(ctx, profile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	if err := service.AddMember(ctx, org.ID, profile.ID); err != nil {
		t.Fatalf("AddMember returned error: %v", err)
	}

	if err := service.AddMember(ctx, org.ID, profile.ID); err != services.ErrMembershipAlreadyExists {
		t.Fatalf("expected %v, got %v", services.ErrMembershipAlreadyExists, err)
	}

	members, err := service.ListMembers(ctx, org.ID)
	if err != nil {
		t.Fatalf("ListMembers returned error: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(members))
	}
	if members[0].ID != profile.ID {
		t.Fatalf("expected member profile ID %d, got %d", profile.ID, members[0].ID)
	}

	if err := service.RemoveMember(ctx, org.ID, profile.ID); err != nil {
		t.Fatalf("RemoveMember returned error: %v", err)
	}

	if err := service.RemoveMember(ctx, org.ID, profile.ID); err != services.ErrMembershipNotFound {
		t.Fatalf("expected %v, got %v", services.ErrMembershipNotFound, err)
	}
}
