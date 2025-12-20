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

func TestProfilesCreateAndLinkChannel(t *testing.T) {
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

	channelRepo := repository.NewChannelRepository(db)
	profileRepo := repository.NewProfileRepository(db)
	organizationRepo := repository.NewOrganizationRepository(db)
	profileService := services.NewProfileService(profileRepo, channelRepo)

	channel := &repository.Channel{Name: "channel1", DisplayName: "Channel One", Enabled: true}
	if err := channelRepo.Create(ctx, channel); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	mux := http.NewServeMux()
	templates := testProfileTemplates(t)
	profileHandler := handlers.NewProfileHandler(profileService, channelRepo, organizationRepo, templates, logger)
	profileHandler.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := &http.Client{}

	createReqBody, _ := json.Marshal(map[string]any{
		"name":        "Profile One",
		"description": "Testing",
	})
	createReq, _ := http.NewRequest("POST", srv.URL+"/profiles", bytes.NewReader(createReqBody))
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

	var createdProfile dto.Profile
	if err := json.NewDecoder(createResp.Body).Decode(&createdProfile); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	if createdProfile.ID == 0 {
		t.Fatalf("expected created profile to have ID")
	}

	org := &repository.Organization{Name: "Org One", Description: "Test Org"}
	if err := organizationRepo.Create(ctx, org); err != nil {
		t.Fatalf("failed to create organization: %v", err)
	}
	if err := organizationRepo.AddMember(ctx, org.ID, createdProfile.ID); err != nil {
		t.Fatalf("failed to add organization member: %v", err)
	}

	getReq, _ := http.NewRequest("GET", srv.URL+"/profiles/"+itoa(createdProfile.ID), nil)
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
	orgs, ok := getData["Organizations"].([]any)
	if !ok {
		t.Fatalf("expected Organizations to be an array")
	}
	if len(orgs) != 1 {
		t.Fatalf("expected 1 organization, got %d", len(orgs))
	}

	linkReqBody, _ := json.Marshal(map[string]any{"channel_id": channel.ID})
	linkReq, _ := http.NewRequest("POST", srv.URL+"/profiles/"+itoa(createdProfile.ID)+"/channels", bytes.NewReader(linkReqBody))
	linkReq.Header.Set("Content-Type", "application/json")
	linkReq.Header.Set("Accept", "application/json")

	linkResp, err := client.Do(linkReq)
	if err != nil {
		t.Fatalf("link request failed: %v", err)
	}
	defer linkResp.Body.Close()

	if linkResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", linkResp.StatusCode)
	}

	linkedChannels, err := profileService.ListLinkedChannels(ctx, createdProfile.ID)
	if err != nil {
		t.Fatalf("ListLinkedChannels returned error: %v", err)
	}
	if len(linkedChannels) != 1 {
		t.Fatalf("expected 1 linked channel, got %d", len(linkedChannels))
	}
	if linkedChannels[0].ID != channel.ID {
		t.Fatalf("expected linked channel ID %d, got %d", channel.ID, linkedChannels[0].ID)
	}

	createTwoReqBody, _ := json.Marshal(map[string]any{"name": "Profile Two"})
	createTwoReq, _ := http.NewRequest("POST", srv.URL+"/profiles", bytes.NewReader(createTwoReqBody))
	createTwoReq.Header.Set("Content-Type", "application/json")
	createTwoReq.Header.Set("Accept", "application/json")

	createTwoResp, err := client.Do(createTwoReq)
	if err != nil {
		t.Fatalf("create second profile request failed: %v", err)
	}
	defer createTwoResp.Body.Close()

	if createTwoResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201 for second profile, got %d", createTwoResp.StatusCode)
	}

	var secondProfile dto.Profile
	if err := json.NewDecoder(createTwoResp.Body).Decode(&secondProfile); err != nil {
		t.Fatalf("failed to decode create second profile response: %v", err)
	}

	conflictReqBody, _ := json.Marshal(map[string]any{"channel_id": channel.ID})
	conflictReq, _ := http.NewRequest("POST", srv.URL+"/profiles/"+itoa(secondProfile.ID)+"/channels", bytes.NewReader(conflictReqBody))
	conflictReq.Header.Set("Content-Type", "application/json")
	conflictReq.Header.Set("Accept", "application/json")

	conflictResp, err := client.Do(conflictReq)
	if err != nil {
		t.Fatalf("conflict link request failed: %v", err)
	}
	defer conflictResp.Body.Close()

	if conflictResp.StatusCode != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", conflictResp.StatusCode)
	}
}

func testProfileTemplates(t *testing.T) *template.Template {
	t.Helper()

	tmpl, err := template.New("").Parse(`
		{{define "profiles/index"}}index{{end}}
		{{define "profiles/new"}}new{{end}}
		{{define "profiles/detail"}}detail{{end}}
		{{define "error.html"}}error{{end}}
	`)
	if err != nil {
		t.Fatalf("failed to parse test templates: %v", err)
	}
	return tmpl
}

func itoa(v int64) string {
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

	// reverse
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
