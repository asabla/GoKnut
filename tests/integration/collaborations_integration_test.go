package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/asabla/goknut/internal/http/dto"
	"github.com/asabla/goknut/internal/http/handlers"
	"github.com/asabla/goknut/internal/observability"
	"github.com/asabla/goknut/internal/repository"
	"github.com/asabla/goknut/internal/services"
)

func TestCollaborationsCreateAndManageParticipants(t *testing.T) {
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
	collaborationRepo := repository.NewCollaborationRepository(db)
	collaborationService := services.NewCollaborationService(collaborationRepo, profileRepo)

	profileOne := &repository.Profile{Name: "Profile One"}
	if err := profileRepo.Create(ctx, profileOne); err != nil {
		t.Fatalf("failed to create Profile One: %v", err)
	}
	profileTwo := &repository.Profile{Name: "Profile Two"}
	if err := profileRepo.Create(ctx, profileTwo); err != nil {
		t.Fatalf("failed to create Profile Two: %v", err)
	}

	mux := http.NewServeMux()
	templates := testCollaborationTemplates(t)
	collaborationHandler := handlers.NewCollaborationHandler(collaborationService, profileRepo, templates, logger, metrics)
	collaborationHandler.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := &http.Client{}

	createReqBody, _ := json.Marshal(map[string]any{
		"name":        "Collab One",
		"description": "Testing",
		"shared_chat": true,
	})
	createReq, _ := http.NewRequest("POST", srv.URL+"/collaborations", bytes.NewReader(createReqBody))
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

	var created dto.Collaboration
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	if created.ID == 0 {
		t.Fatalf("expected created collaboration to have ID")
	}

	addReqBody, _ := json.Marshal(map[string]any{"profile_id": profileOne.ID})
	addReq, _ := http.NewRequest("POST", srv.URL+"/collaborations/"+int64ToString(created.ID)+"/participants", bytes.NewReader(addReqBody))
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Accept", "application/json")
	addResp, err := client.Do(addReq)
	if err != nil {
		t.Fatalf("add participant request failed: %v", err)
	}
	_ = addResp.Body.Close()
	if addResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", addResp.StatusCode)
	}

	addTwoReqBody, _ := json.Marshal(map[string]any{"profile_id": profileTwo.ID})
	addTwoReq, _ := http.NewRequest("POST", srv.URL+"/collaborations/"+int64ToString(created.ID)+"/participants", bytes.NewReader(addTwoReqBody))
	addTwoReq.Header.Set("Content-Type", "application/json")
	addTwoReq.Header.Set("Accept", "application/json")
	addTwoResp, err := client.Do(addTwoReq)
	if err != nil {
		t.Fatalf("add second participant request failed: %v", err)
	}
	_ = addTwoResp.Body.Close()
	if addTwoResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", addTwoResp.StatusCode)
	}

	getReq, _ := http.NewRequest("GET", srv.URL+"/collaborations/"+int64ToString(created.ID), nil)
	getReq.Header.Set("Accept", "application/json")
	getResp, err := client.Do(getReq)
	if err != nil {
		t.Fatalf("get request failed: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 on get, got %d", getResp.StatusCode)
	}
	var getData map[string]any
	if err := json.NewDecoder(getResp.Body).Decode(&getData); err != nil {
		t.Fatalf("failed to decode get response: %v", err)
	}
	participants, ok := getData["Participants"].([]any)
	if !ok {
		t.Fatalf("expected Participants to be an array")
	}
	if len(participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(participants))
	}

	removeReq, _ := http.NewRequest("POST", srv.URL+"/collaborations/"+int64ToString(created.ID)+"/participants/"+int64ToString(profileOne.ID)+"/remove", nil)
	removeReq.Header.Set("Accept", "application/json")
	removeResp, err := client.Do(removeReq)
	if err != nil {
		t.Fatalf("remove request failed: %v", err)
	}
	_ = removeResp.Body.Close()
	if removeResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 on remove, got %d", removeResp.StatusCode)
	}
}

func testCollaborationTemplates(t *testing.T) *template.Template {
	t.Helper()

	tmpl, err := template.New("").Parse(`
		{{define "collaborations/index"}}index{{end}}
		{{define "collaborations/new"}}new{{end}}
		{{define "collaborations/detail"}}detail{{end}}
		{{define "error.html"}}error{{end}}
	`)
	if err != nil {
		t.Fatalf("failed to parse test templates: %v", err)
	}
	return tmpl
}
