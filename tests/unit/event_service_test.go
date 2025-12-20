package unit

import (
	"context"
	"testing"
	"time"

	"github.com/asabla/goknut/internal/repository"
	"github.com/asabla/goknut/internal/services"
)

func TestEventServiceDateValidation(t *testing.T) {
	ctx := context.Background()

	db, err := repository.Open(repository.DBConfig{Path: ":memory:", EnableFTS: true})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	eventRepo := repository.NewEventRepository(db)
	profileRepo := repository.NewProfileRepository(db)
	service := services.NewEventService(eventRepo, profileRepo)

	startAt := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	endBefore := startAt.Add(-time.Hour)
	if _, err := service.Create(ctx, "My Event", "", startAt, &endBefore); err != services.ErrInvalidEventDates {
		t.Fatalf("expected %v, got %v", services.ErrInvalidEventDates, err)
	}

	endAt := startAt.Add(time.Hour)
	evt, err := service.Create(ctx, "My Event", "", startAt, &endAt)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if evt.ID == 0 {
		t.Fatalf("expected created event to have ID")
	}

	title := "Updated Event"
	badEnd := startAt.Add(-2 * time.Hour)
	if _, err := service.Update(ctx, evt.ID, title, "", startAt, &badEnd); err != services.ErrInvalidEventDates {
		t.Fatalf("expected %v, got %v", services.ErrInvalidEventDates, err)
	}
}
