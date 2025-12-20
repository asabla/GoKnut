package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/asabla/goknut/internal/http/dto"
	"github.com/asabla/goknut/internal/http/handlers"
	"github.com/asabla/goknut/internal/observability"
	"github.com/asabla/goknut/internal/repository"
	"github.com/asabla/goknut/internal/services"
)

func TestEventsCreateAndAddParticipant(t *testing.T) {
	ctx := context.Background()

	db, err := repository.Open(repository.DBConfig{Path: ":memory:", EnableFTS: true})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	logger := observability.NewLogger("test")
	metrics := observability.NewMetrics()

	profileRepo := repository.NewProfileRepository(db)
	profile := &repository.Profile{Name: "Profile One"}
	if err := profileRepo.Create(ctx, profile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	eventRepo := repository.NewEventRepository(db)
	eventService := services.NewEventService(eventRepo, profileRepo)

	mux := http.NewServeMux()
	templates := testEventTemplates(t)
	eventHandler := handlers.NewEventHandler(eventService, profileRepo, templates, logger, metrics)
	eventHandler.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := &http.Client{}

	startAt := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	endBefore := startAt.Add(-time.Hour)
	badCreateBody, _ := json.Marshal(map[string]any{
		"title":       "My Event",
		"description": "Testing",
		"start_at":    startAt,
		"end_at":      endBefore,
	})
	badCreateReq, _ := http.NewRequest("POST", srv.URL+"/events", bytes.NewReader(badCreateBody))
	badCreateReq.Header.Set("Content-Type", "application/json")
	badCreateReq.Header.Set("Accept", "application/json")
	badCreateResp, err := client.Do(badCreateReq)
	if err != nil {
		t.Fatalf("bad create request failed: %v", err)
	}
	defer badCreateResp.Body.Close()
	if badCreateResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", badCreateResp.StatusCode)
	}

	endAt := startAt.Add(time.Hour)
	createReqBody, _ := json.Marshal(map[string]any{
		"title":       "My Event",
		"description": "Testing",
		"start_at":    startAt,
		"end_at":      endAt,
	})
	createReq, _ := http.NewRequest("POST", srv.URL+"/events", bytes.NewReader(createReqBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Accept", "application/json")

	createResp, err := client.Do(createReq)
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", createResp.StatusCode)
	}

	var created dto.Event
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	if created.ID == 0 {
		t.Fatalf("expected created event to have ID")
	}

	addReqBody, _ := json.Marshal(map[string]any{"profile_id": profile.ID})
	addReq, _ := http.NewRequest("POST", srv.URL+"/events/"+int64ToString(created.ID)+"/participants", bytes.NewReader(addReqBody))
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Accept", "application/json")
	addResp, err := client.Do(addReq)
	if err != nil {
		t.Fatalf("add participant request failed: %v", err)
	}
	defer addResp.Body.Close()
	if addResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", addResp.StatusCode)
	}

	participants, err := eventService.ListParticipants(ctx, created.ID)
	if err != nil {
		t.Fatalf("ListParticipants returned error: %v", err)
	}
	if len(participants) != 1 {
		t.Fatalf("expected 1 participant, got %d", len(participants))
	}
	if participants[0].ID != profile.ID {
		t.Fatalf("expected participant profile ID %d, got %d", profile.ID, participants[0].ID)
	}
}

func testEventTemplates(t *testing.T) *template.Template {
	t.Helper()

	tmpl, err := template.New("").Parse(`
		{{define "events/index"}}index{{end}}
		{{define "events/new"}}new{{end}}
		{{define "events/detail"}}detail{{end}}
		{{define "error.html"}}error{{end}}
	`)
	if err != nil {
		t.Fatalf("failed to parse test templates: %v", err)
	}
	return tmpl
}
