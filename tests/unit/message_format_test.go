package unit

import (
	"testing"
	"time"

	"github.com/asabla/goknut/internal/http/dto"
)

func TestMessagePaginationValidation(t *testing.T) {
	tests := []struct {
		name     string
		req      dto.PaginationRequest
		wantErr  bool
		wantPage int
		wantSize int
	}{
		{
			name:     "valid pagination",
			req:      dto.PaginationRequest{Page: 1, PageSize: 20},
			wantErr:  false,
			wantPage: 1,
			wantSize: 20,
		},
		{
			name:     "zero page defaults to 1",
			req:      dto.PaginationRequest{Page: 0, PageSize: 20},
			wantErr:  false,
			wantPage: 1,
			wantSize: 20,
		},
		{
			name:     "zero page size defaults to 20",
			req:      dto.PaginationRequest{Page: 1, PageSize: 0},
			wantErr:  false,
			wantPage: 1,
			wantSize: 20,
		},
		{
			name:    "page size too large",
			req:     dto.PaginationRequest{Page: 1, PageSize: 500},
			wantErr: true,
		},
		{
			name:    "negative page size",
			req:     dto.PaginationRequest{Page: 1, PageSize: -1},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr {
				if tt.req.Page != tt.wantPage {
					t.Errorf("expected page %d, got %d", tt.wantPage, tt.req.Page)
				}
				if tt.req.PageSize != tt.wantSize {
					t.Errorf("expected page_size %d, got %d", tt.wantSize, tt.req.PageSize)
				}
			}
		})
	}
}

func TestPaginationOffset(t *testing.T) {
	tests := []struct {
		page     int
		pageSize int
		want     int
	}{
		{page: 1, pageSize: 20, want: 0},
		{page: 2, pageSize: 20, want: 20},
		{page: 3, pageSize: 10, want: 20},
		{page: 5, pageSize: 50, want: 200},
	}

	for _, tt := range tests {
		req := dto.PaginationRequest{Page: tt.page, PageSize: tt.pageSize}
		got := req.Offset()
		if got != tt.want {
			t.Errorf("Page=%d, PageSize=%d: expected offset %d, got %d",
				tt.page, tt.pageSize, tt.want, got)
		}
	}
}

func TestMessageDTO(t *testing.T) {
	now := time.Now()
	msg := dto.Message{
		ID:          1,
		ChannelID:   10,
		ChannelName: "testchannel",
		UserID:      100,
		Username:    "testuser",
		DisplayName: "TestUser",
		Text:        "Hello, world!",
		SentAt:      now,
	}

	if msg.ID != 1 {
		t.Errorf("expected ID 1, got %d", msg.ID)
	}
	if msg.ChannelName != "testchannel" {
		t.Errorf("expected channel 'testchannel', got %s", msg.ChannelName)
	}
	if msg.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %s", msg.Username)
	}
	if msg.Text != "Hello, world!" {
		t.Errorf("expected text 'Hello, world!', got %s", msg.Text)
	}
}

func TestTimeFormatting(t *testing.T) {
	// Test that message timestamps are formatted correctly for display
	now := time.Date(2025, 12, 6, 15, 30, 45, 0, time.UTC)

	// Format for display (HH:MM:SS)
	timeStr := now.Format("15:04:05")
	if timeStr != "15:30:45" {
		t.Errorf("expected '15:30:45', got %s", timeStr)
	}

	// Format for full timestamp
	fullStr := now.Format(time.RFC3339)
	if fullStr != "2025-12-06T15:30:45Z" {
		t.Errorf("expected '2025-12-06T15:30:45Z', got %s", fullStr)
	}
}

func TestPaginatedResponse(t *testing.T) {
	items := []dto.Message{
		{ID: 1, Text: "msg1"},
		{ID: 2, Text: "msg2"},
		{ID: 3, Text: "msg3"},
	}

	resp := dto.PaginatedResponse[dto.Message]{
		Items:      items,
		Page:       1,
		PageSize:   20,
		TotalCount: 100,
		TotalPages: 5,
		HasNext:    true,
		HasPrev:    false,
	}

	if len(resp.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(resp.Items))
	}
	if resp.TotalPages != 5 {
		t.Errorf("expected 5 total pages, got %d", resp.TotalPages)
	}
	if !resp.HasNext {
		t.Error("expected HasNext to be true")
	}
	if resp.HasPrev {
		t.Error("expected HasPrev to be false")
	}
}

func TestCalculateTotalPages(t *testing.T) {
	tests := []struct {
		totalCount int
		pageSize   int
		want       int
	}{
		{totalCount: 0, pageSize: 20, want: 0},
		{totalCount: 10, pageSize: 20, want: 1},
		{totalCount: 20, pageSize: 20, want: 1},
		{totalCount: 21, pageSize: 20, want: 2},
		{totalCount: 100, pageSize: 20, want: 5},
		{totalCount: 101, pageSize: 20, want: 6},
	}

	for _, tt := range tests {
		got := calculateTotalPages(tt.totalCount, tt.pageSize)
		if got != tt.want {
			t.Errorf("totalCount=%d, pageSize=%d: expected %d pages, got %d",
				tt.totalCount, tt.pageSize, tt.want, got)
		}
	}
}

// Helper function to calculate total pages
func calculateTotalPages(totalCount, pageSize int) int {
	if totalCount == 0 || pageSize == 0 {
		return 0
	}
	return (totalCount + pageSize - 1) / pageSize
}

func TestMessageTextBounds(t *testing.T) {
	// Test message text length validation
	shortText := "hi"
	longText := string(make([]byte, 501)) // 501 characters

	if len(shortText) > 500 {
		t.Error("short text should be valid")
	}
	if len(longText) <= 500 {
		t.Error("long text should exceed limit")
	}
}

func TestUserDTOFields(t *testing.T) {
	now := time.Now()
	user := dto.User{
		ID:            1,
		Username:      "testuser",
		DisplayName:   "Test User",
		FirstSeenAt:   now.Add(-24 * time.Hour),
		LastSeenAt:    now,
		TotalMessages: 100,
	}

	if user.ID != 1 {
		t.Errorf("expected ID 1, got %d", user.ID)
	}
	if user.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %s", user.Username)
	}
	if user.TotalMessages != 100 {
		t.Errorf("expected 100 messages, got %d", user.TotalMessages)
	}
	if !user.LastSeenAt.After(user.FirstSeenAt) {
		t.Error("last_seen_at should be after first_seen_at")
	}
}
