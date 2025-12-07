package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Note: These tests follow the failing-first TDD pattern.
// They define the expected contract and will fail until handlers are implemented.

func TestChannelsListGET(t *testing.T) {
	t.Skip("Handler not yet implemented - failing-first TDD")

	srv := httptest.NewServer(nil) // TODO: Wire up actual server
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/channels")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") && !strings.Contains(contentType, "text/html") {
		t.Errorf("expected JSON or HTML content type, got %s", contentType)
	}
}

func TestChannelsCreatePOST(t *testing.T) {
	t.Skip("Handler not yet implemented - failing-first TDD")

	srv := httptest.NewServer(nil) // TODO: Wire up actual server
	defer srv.Close()

	body := `{"name": "testchannel", "enabled": true, "retain_history_on_delete": true}`
	resp, err := http.Post(srv.URL+"/channels", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result["name"] != "testchannel" {
		t.Errorf("expected name 'testchannel', got %v", result["name"])
	}
}

func TestChannelsCreateValidation(t *testing.T) {
	t.Skip("Handler not yet implemented - failing-first TDD")

	srv := httptest.NewServer(nil) // TODO: Wire up actual server
	defer srv.Close()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "empty name",
			body:       `{"name": "", "enabled": true}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid characters",
			body:       `{"name": "test@channel!", "enabled": true}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "name too long",
			body:       `{"name": "` + strings.Repeat("a", 30) + `", "enabled": true}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Post(srv.URL+"/channels", "application/json", strings.NewReader(tt.body))
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
		})
	}
}

func TestChannelsUpdatePOST(t *testing.T) {
	t.Skip("Handler not yet implemented - failing-first TDD")

	srv := httptest.NewServer(nil) // TODO: Wire up actual server
	defer srv.Close()

	// First create a channel
	createBody := `{"name": "testchannel", "enabled": true}`
	createResp, err := http.Post(srv.URL+"/channels", "application/json", strings.NewReader(createBody))
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	createResp.Body.Close()

	// Then update it
	updateBody := `{"enabled": false}`
	resp, err := http.Post(srv.URL+"/channels/testchannel", "application/json", strings.NewReader(updateBody))
	if err != nil {
		t.Fatalf("update request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestChannelsDeletePOST(t *testing.T) {
	t.Skip("Handler not yet implemented - failing-first TDD")

	srv := httptest.NewServer(nil) // TODO: Wire up actual server
	defer srv.Close()

	// First create a channel
	createBody := `{"name": "testchannel", "enabled": true}`
	createResp, err := http.Post(srv.URL+"/channels", "application/json", strings.NewReader(createBody))
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	createResp.Body.Close()

	// Then delete it
	deleteBody := `{"retain_history": false}`
	resp, err := http.Post(srv.URL+"/channels/testchannel/delete", "application/json", strings.NewReader(deleteBody))
	if err != nil {
		t.Fatalf("delete request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestChannelsDeleteRetainsHistory(t *testing.T) {
	t.Skip("Handler not yet implemented - failing-first TDD")

	srv := httptest.NewServer(nil) // TODO: Wire up actual server
	defer srv.Close()

	// Create channel, add messages, then delete with retain_history=true
	// Messages should still be queryable after deletion

	createBody := `{"name": "testchannel", "enabled": true, "retain_history_on_delete": true}`
	createResp, err := http.Post(srv.URL+"/channels", "application/json", strings.NewReader(createBody))
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	createResp.Body.Close()

	// Delete with history retention
	deleteBody := `{"retain_history": true}`
	resp, err := http.Post(srv.URL+"/channels/testchannel/delete", "application/json", strings.NewReader(deleteBody))
	if err != nil {
		t.Fatalf("delete request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// TODO: Verify messages are still accessible via search
}
