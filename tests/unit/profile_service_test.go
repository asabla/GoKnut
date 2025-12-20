package unit

import (
	"context"
	"errors"
	"testing"

	"github.com/asabla/goknut/internal/repository"
	"github.com/asabla/goknut/internal/services"
)

func TestProfileServiceLinkChannel(t *testing.T) {
	ctx := context.Background()

	db, err := repository.Open(repository.DBConfig{Path: ":memory:", EnableFTS: true})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	channelRepo := repository.NewChannelRepository(db)
	profileRepo := repository.NewProfileRepository(db)
	service := services.NewProfileService(profileRepo, channelRepo)

	channel := &repository.Channel{Name: "channel1", DisplayName: "Channel One", Enabled: true}
	if err := channelRepo.Create(ctx, channel); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	profile := &repository.Profile{Name: "Profile One"}
	if err := profileRepo.Create(ctx, profile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	t.Run("links channel", func(t *testing.T) {
		if err := service.LinkChannel(ctx, profile.ID, channel.ID); err != nil {
			t.Fatalf("LinkChannel returned error: %v", err)
		}

		linked, err := service.ListLinkedChannels(ctx, profile.ID)
		if err != nil {
			t.Fatalf("ListLinkedChannels returned error: %v", err)
		}
		if len(linked) != 1 {
			t.Fatalf("expected 1 linked channel, got %d", len(linked))
		}
		if linked[0].ID != channel.ID {
			t.Fatalf("expected linked channel ID %d, got %d", channel.ID, linked[0].ID)
		}
	})

	t.Run("returns ErrProfileNotFound when profile missing", func(t *testing.T) {
		err := service.LinkChannel(ctx, 999999, channel.ID)
		if err != services.ErrProfileNotFound {
			t.Fatalf("expected %v, got %v", services.ErrProfileNotFound, err)
		}
	})

	t.Run("returns repository.ErrNotFound when channel missing", func(t *testing.T) {
		otherProfile := &repository.Profile{Name: "Profile Two"}
		if err := profileRepo.Create(ctx, otherProfile); err != nil {
			t.Fatalf("failed to create profile: %v", err)
		}

		err := service.LinkChannel(ctx, otherProfile.ID, 999999)
		if !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf("expected errors.Is(err, repository.ErrNotFound) to be true; got err=%v", err)
		}
	})

	t.Run("returns ErrChannelAlreadyLinked on conflict", func(t *testing.T) {
		otherProfile := &repository.Profile{Name: "Profile Three"}
		if err := profileRepo.Create(ctx, otherProfile); err != nil {
			t.Fatalf("failed to create profile: %v", err)
		}

		err := service.LinkChannel(ctx, otherProfile.ID, channel.ID)
		if err != services.ErrChannelAlreadyLinked {
			t.Fatalf("expected %v, got %v", services.ErrChannelAlreadyLinked, err)
		}
	})
}

func TestProfileServiceUnlinkChannel(t *testing.T) {
	ctx := context.Background()

	db, err := repository.Open(repository.DBConfig{Path: ":memory:", EnableFTS: true})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	channelRepo := repository.NewChannelRepository(db)
	profileRepo := repository.NewProfileRepository(db)
	service := services.NewProfileService(profileRepo, channelRepo)

	channel := &repository.Channel{Name: "channel1", DisplayName: "Channel One", Enabled: true}
	if err := channelRepo.Create(ctx, channel); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	profile := &repository.Profile{Name: "Profile One"}
	if err := profileRepo.Create(ctx, profile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	if err := service.LinkChannel(ctx, profile.ID, channel.ID); err != nil {
		t.Fatalf("LinkChannel returned error: %v", err)
	}

	if err := service.UnlinkChannel(ctx, profile.ID, channel.ID); err != nil {
		t.Fatalf("UnlinkChannel returned error: %v", err)
	}

	linked, err := service.ListLinkedChannels(ctx, profile.ID)
	if err != nil {
		t.Fatalf("ListLinkedChannels returned error: %v", err)
	}
	if len(linked) != 0 {
		t.Fatalf("expected 0 linked channels after unlink, got %d", len(linked))
	}

	if err := service.UnlinkChannel(ctx, profile.ID, channel.ID); err != services.ErrProfileChannelNotFound {
		t.Fatalf("expected %v, got %v", services.ErrProfileChannelNotFound, err)
	}
}
