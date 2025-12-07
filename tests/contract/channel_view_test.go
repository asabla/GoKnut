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

func TestChannelViewGET(t *testing.T) {
	t.Skip("Handler not yet implemented - failing-first TDD")

	srv := httptest.NewServer(nil) // TODO: Wire up actual server
	defer srv.Close()

	// Channel view should return HTML page with recent messages
	resp, err := http.Get(srv.URL + "/channels/testchannel/view")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected HTML content type, got %s", contentType)
	}
}

func TestChannelViewNotFound(t *testing.T) {
	t.Skip("Handler not yet implemented - failing-first TDD")

	srv := httptest.NewServer(nil) // TODO: Wire up actual server
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/channels/nonexistentchannel/view")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}
}

func TestChannelMessagesGET(t *testing.T) {
	t.Skip("Handler not yet implemented - failing-first TDD")

	srv := httptest.NewServer(nil) // TODO: Wire up actual server
	defer srv.Close()

	// Messages endpoint should return HTML fragment or JSON based on Accept header
	resp, err := http.Get(srv.URL + "/channels/testchannel/messages")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestChannelMessagesPagination(t *testing.T) {
	t.Skip("Handler not yet implemented - failing-first TDD")

	srv := httptest.NewServer(nil) // TODO: Wire up actual server
	defer srv.Close()

	tests := []struct {
		name   string
		query  string
		wantOK bool
	}{
		{
			name:   "page and page_size",
			query:  "?page=1&page_size=20",
			wantOK: true,
		},
		{
			name:   "before_id pagination",
			query:  "?before_id=100",
			wantOK: true,
		},
		{
			name:   "invalid page",
			query:  "?page=-1",
			wantOK: false,
		},
		{
			name:   "page_size too large",
			query:  "?page_size=500",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(srv.URL + "/channels/testchannel/messages" + tt.query)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if tt.wantOK && resp.StatusCode != http.StatusOK {
				t.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			if !tt.wantOK && resp.StatusCode == http.StatusOK {
				t.Errorf("expected error status, got 200")
			}
		})
	}
}

func TestChannelMessagesStreamGET(t *testing.T) {
	t.Skip("Handler not yet implemented - failing-first TDD")

	srv := httptest.NewServer(nil) // TODO: Wire up actual server
	defer srv.Close()

	// Stream endpoint should return new messages after the given ID
	resp, err := http.Get(srv.URL + "/channels/testchannel/messages/stream?after_id=50")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Should return HTML fragment for HTMX
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected HTML content type for stream, got %s", contentType)
	}
}

func TestChannelMessagesStreamWithHTMXHeader(t *testing.T) {
	t.Skip("Handler not yet implemented - failing-first TDD")

	srv := httptest.NewServer(nil) // TODO: Wire up actual server
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/channels/testchannel/messages/stream?after_id=50", nil)
	req.Header.Set("HX-Request", "true")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestChannelMessagesJSONResponse(t *testing.T) {
	t.Skip("Handler not yet implemented - failing-first TDD")

	srv := httptest.NewServer(nil) // TODO: Wire up actual server
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/channels/testchannel/messages", nil)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected JSON content type, got %s", contentType)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	// Should have messages array
	if _, ok := result["messages"]; !ok {
		t.Error("expected 'messages' field in response")
	}
}

func TestChannelMessagesResponseFormat(t *testing.T) {
	t.Skip("Handler not yet implemented - failing-first TDD")

	srv := httptest.NewServer(nil) // TODO: Wire up actual server
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/channels/testchannel/messages?page=1&page_size=10", nil)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Messages []struct {
			ID          int64  `json:"id"`
			ChannelID   int64  `json:"channel_id"`
			UserID      int64  `json:"user_id"`
			Username    string `json:"username"`
			DisplayName string `json:"display_name"`
			Text        string `json:"text"`
			SentAt      string `json:"sent_at"`
		} `json:"messages"`
		Page       int  `json:"page"`
		PageSize   int  `json:"page_size"`
		TotalCount int  `json:"total_count"`
		HasNext    bool `json:"has_next"`
		HasPrev    bool `json:"has_prev"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Validate response structure
	if result.Page != 1 {
		t.Errorf("expected page 1, got %d", result.Page)
	}
	if result.PageSize != 10 {
		t.Errorf("expected page_size 10, got %d", result.PageSize)
	}
}
