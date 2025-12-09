package contract

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/asabla/goknut/internal/http/handlers"
	"github.com/asabla/goknut/internal/observability"
	"github.com/asabla/goknut/internal/repository"
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

// TestChannelsSSEStream tests the SSE stream for the channels view.
// This test verifies:
// - SSE connection is established successfully with view=channels
// - Initial channel_count events are received for all channels
// - Events include channel_id, channel_name, total_messages
func TestChannelsSSEStream(t *testing.T) {
	ctx := context.Background()

	// Set up test database
	db, err := repository.Open(repository.DBConfig{
		Path:      ":memory:",
		EnableFTS: true,
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	// Create repositories
	channelRepo := repository.NewChannelRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	userRepo := repository.NewUserRepository(db)

	// Create test channels
	channels := []struct {
		name    string
		display string
	}{
		{"channel1", "Channel One"},
		{"channel2", "Channel Two"},
		{"channel3", "Channel Three"},
	}

	for _, ch := range channels {
		channel := &repository.Channel{
			Name:        ch.name,
			DisplayName: ch.display,
			Enabled:     true,
		}
		if err := channelRepo.Create(ctx, channel); err != nil {
			t.Fatalf("failed to create channel: %v", err)
		}
	}

	// Create a user and some messages for first channel
	user, err := userRepo.GetOrCreate(ctx, "testuser", "TestUser")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Get channel1 ID
	channel1, err := channelRepo.GetByName(ctx, "channel1")
	if err != nil {
		t.Fatalf("failed to get channel: %v", err)
	}

	// Add 5 messages to channel1
	for i := 0; i < 5; i++ {
		msg := &repository.Message{
			ChannelID: channel1.ID,
			UserID:    user.ID,
			Text:      "Test message",
			SentAt:    time.Now(),
		}
		if err := messageRepo.Create(ctx, msg); err != nil {
			t.Fatalf("failed to create message: %v", err)
		}
	}

	// Create SSE handler
	logger := observability.NewLogger("test")
	metrics := observability.NewMetrics()
	sseHandler := handlers.NewSSEHandler(channelRepo, messageRepo, userRepo, nil, logger, metrics, nil)

	// Create test server
	mux := http.NewServeMux()
	sseHandler.RegisterRoutes(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	// Connect to SSE stream for channels view
	req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/live?view=channels", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to connect to SSE: %v", err)
	}
	defer resp.Body.Close()

	// Verify SSE headers
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %s", ct)
	}

	// Read events
	scanner := bufio.NewScanner(resp.Body)
	timeout := time.After(3 * time.Second)
	var channelCountEvents []map[string]any
	gotStatus := false

	for len(channelCountEvents) < 3 { // Expecting 3 channel_count events
		select {
		case <-timeout:
			t.Fatalf("timeout waiting for events, got %d channel_count events", len(channelCountEvents))
		default:
			if scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "data: ") {
					data := strings.TrimPrefix(line, "data: ")
					var event map[string]any
					if err := json.Unmarshal([]byte(data), &event); err != nil {
						continue
					}
					if event["type"] == "status" {
						gotStatus = true
					} else if event["type"] == "channel_count" {
						channelCountEvents = append(channelCountEvents, event)
					}
				}
			}
		}
	}

	// Verify we got status event
	if !gotStatus {
		t.Error("expected to receive status event")
	}

	// Verify we got channel_count events for all channels
	if len(channelCountEvents) != 3 {
		t.Errorf("expected 3 channel_count events, got %d", len(channelCountEvents))
	}

	// Find channel1 event and verify message count
	var foundChannel1 bool
	for _, evt := range channelCountEvents {
		if evt["channel_name"] == "channel1" {
			foundChannel1 = true
			if total, ok := evt["total_messages"].(float64); !ok || total != 5 {
				t.Errorf("expected channel1 total_messages=5, got %v", evt["total_messages"])
			}
		}
	}
	if !foundChannel1 {
		t.Error("expected to find channel1 in channel_count events")
	}
}
