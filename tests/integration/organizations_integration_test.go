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

func TestOrganizationsCreateAndAddMember(t *testing.T) {
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
	orgRepo := repository.NewOrganizationRepository(db)
	orgService := services.NewOrganizationService(orgRepo, profileRepo)

	profile := &repository.Profile{Name: "Profile One"}
	if err := profileRepo.Create(ctx, profile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	mux := http.NewServeMux()
	templates := testOrganizationTemplates(t)
	orgHandler := handlers.NewOrganizationHandler(orgService, profileRepo, templates, logger, metrics)
	orgHandler.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := &http.Client{}

	createReqBody, _ := json.Marshal(map[string]any{
		"name":        "Org One",
		"description": "Testing",
	})
	createReq, _ := http.NewRequest("POST", srv.URL+"/organizations", bytes.NewReader(createReqBody))
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

	var createdOrg dto.Organization
	if err := json.NewDecoder(createResp.Body).Decode(&createdOrg); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	if createdOrg.ID == 0 {
		t.Fatalf("expected created org to have ID")
	}

	addReqBody, _ := json.Marshal(map[string]any{"profile_id": profile.ID})
	addReq, _ := http.NewRequest("POST", srv.URL+"/organizations/"+int64ToString(createdOrg.ID)+"/members", bytes.NewReader(addReqBody))
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Accept", "application/json")

	addResp, err := client.Do(addReq)
	if err != nil {
		t.Fatalf("add member request failed: %v", err)
	}
	defer addResp.Body.Close()

	if addResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", addResp.StatusCode)
	}

	members, err := orgService.ListMembers(ctx, createdOrg.ID)
	if err != nil {
		t.Fatalf("ListMembers returned error: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(members))
	}
	if members[0].ID != profile.ID {
		t.Fatalf("expected member profile ID %d, got %d", profile.ID, members[0].ID)
	}

	conflictReqBody, _ := json.Marshal(map[string]any{"profile_id": profile.ID})
	conflictReq, _ := http.NewRequest("POST", srv.URL+"/organizations/"+int64ToString(createdOrg.ID)+"/members", bytes.NewReader(conflictReqBody))
	conflictReq.Header.Set("Content-Type", "application/json")
	conflictReq.Header.Set("Accept", "application/json")

	conflictResp, err := client.Do(conflictReq)
	if err != nil {
		t.Fatalf("conflict member request failed: %v", err)
	}
	defer conflictResp.Body.Close()

	if conflictResp.StatusCode != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", conflictResp.StatusCode)
	}
}

func testOrganizationTemplates(t *testing.T) *template.Template {
	t.Helper()

	tmpl, err := template.New("").Parse(`
		{{define "organizations/index"}}index{{end}}
		{{define "organizations/new"}}new{{end}}
		{{define "organizations/detail"}}detail{{end}}
		{{define "error.html"}}error{{end}}
	`)
	if err != nil {
		t.Fatalf("failed to parse test templates: %v", err)
	}
	return tmpl
}

func int64ToString(v int64) string {
	// Keep tests simple and dependency-free.
	buf := make([]byte, 0, 20)
	if v == 0 {
		return "0"
	}

	neg := v < 0
	if neg {
		v = -v
	}
	for v > 0 {
		d := v % 10
		buf = append(buf, byte('0'+d))
		v /= 10
	}
	if neg {
		buf = append(buf, '-')
	}

	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
