package unit

import (
	"testing"
	"time"

	"github.com/asabla/goknut/internal/http/dto"
	"github.com/asabla/goknut/internal/search"
)

// Note: These tests follow the failing-first TDD pattern.
// They will be enabled as search service is implemented.

func TestSearchUsersRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     dto.SearchUsersRequest
		wantErr error
	}{
		{
			name:    "valid search",
			req:     dto.SearchUsersRequest{Query: "testuser", PaginationRequest: dto.PaginationRequest{Page: 1, PageSize: 20}},
			wantErr: nil,
		},
		{
			name:    "empty query",
			req:     dto.SearchUsersRequest{Query: "", PaginationRequest: dto.PaginationRequest{Page: 1, PageSize: 20}},
			wantErr: dto.ErrUsernameRequired,
		},
		{
			name:    "whitespace only query",
			req:     dto.SearchUsersRequest{Query: "   ", PaginationRequest: dto.PaginationRequest{Page: 1, PageSize: 20}},
			wantErr: dto.ErrUsernameRequired,
		},
		{
			name:    "query normalized to lowercase",
			req:     dto.SearchUsersRequest{Query: "TestUser", PaginationRequest: dto.PaginationRequest{Page: 1, PageSize: 20}},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			// If valid, check normalization
			if err == nil && tt.req.Query != "" {
				for _, r := range tt.req.Query {
					if r >= 'A' && r <= 'Z' {
						t.Errorf("expected lowercase query, got %q", tt.req.Query)
						break
					}
				}
			}
		})
	}
}

func TestHighlightSearchTerm(t *testing.T) {
	tests := []struct {
		name string
		text string
		term string
		want string
	}{
		{
			name: "single match",
			text: "hello world",
			term: "world",
			want: "hello <mark>world</mark>",
		},
		{
			name: "multiple matches",
			text: "test one test two test three",
			term: "test",
			want: "<mark>test</mark> one <mark>test</mark> two <mark>test</mark> three",
		},
		{
			name: "case insensitive",
			text: "Hello World HELLO",
			term: "hello",
			want: "<mark>Hello</mark> World <mark>HELLO</mark>",
		},
		{
			name: "no match",
			text: "hello world",
			term: "xyz",
			want: "hello world",
		},
		{
			name: "html escaped",
			text: "<script>alert('xss')</script>",
			term: "script",
			want: "&lt;<mark>script</mark>&gt;alert(&#39;xss&#39;)&lt;/<mark>script</mark>&gt;",
		},
		{
			name: "empty term",
			text: "hello world",
			term: "",
			want: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := search.HighlightTerm(tt.text, tt.term)
			if got != tt.want {
				t.Errorf("HighlightTerm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildFTSQuery(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple word",
			input: "hello",
			want:  "hello*",
		},
		{
			name:  "multiple words",
			input: "hello world",
			want:  "hello* world*",
		},
		{
			name:  "with quotes (phrase search)",
			input: `"hello world"`,
			want:  `"hello world"`,
		},
		{
			name:  "special characters escaped",
			input: "test@user",
			want:  "testuser*",
		},
		{
			name:  "mixed phrase and words",
			input: `"exact phrase" other words`,
			want:  `"exact phrase" other* words*`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := search.BuildFTSQuery(tt.input)
			if got != tt.want {
				t.Errorf("BuildFTSQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildLIKEPattern(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple word",
			input: "hello",
			want:  "%hello%",
		},
		{
			name:  "escape percent",
			input: "100%",
			want:  "%100\\%%",
		},
		{
			name:  "escape underscore",
			input: "user_name",
			want:  "%user\\_name%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := search.BuildLIKEPattern(tt.input)
			if got != tt.want {
				t.Errorf("BuildLIKEPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserSearchResult(t *testing.T) {
	// Test that UserSearchResult has expected fields
	result := search.UserSearchResult{
		ID:            1,
		Username:      "testuser",
		DisplayName:   "Test User",
		TotalMessages: 100,
		ChannelCount:  5,
	}

	if result.ID != 1 {
		t.Errorf("expected ID 1, got %d", result.ID)
	}
	if result.Username != "testuser" {
		t.Errorf("expected Username 'testuser', got %s", result.Username)
	}
	if result.DisplayName != "Test User" {
		t.Errorf("expected DisplayName 'Test User', got %s", result.DisplayName)
	}
	if result.TotalMessages != 100 {
		t.Errorf("expected TotalMessages 100, got %d", result.TotalMessages)
	}
	if result.ChannelCount != 5 {
		t.Errorf("expected ChannelCount 5, got %d", result.ChannelCount)
	}
}

func TestMessageSearchResult(t *testing.T) {
	// Test that MessageSearchResult has expected fields
	result := search.MessageSearchResult{
		ID:              1,
		ChannelID:       2,
		ChannelName:     "testchannel",
		UserID:          3,
		Username:        "testuser",
		DisplayName:     "Test User",
		Text:            "Hello world",
		HighlightedText: "Hello <mark>world</mark>",
	}

	if result.ID != 1 {
		t.Errorf("expected ID 1, got %d", result.ID)
	}
	if result.ChannelID != 2 {
		t.Errorf("expected ChannelID 2, got %d", result.ChannelID)
	}
	if result.ChannelName != "testchannel" {
		t.Errorf("expected ChannelName 'testchannel', got %s", result.ChannelName)
	}
	if result.UserID != 3 {
		t.Errorf("expected UserID 3, got %d", result.UserID)
	}
	if result.Username != "testuser" {
		t.Errorf("expected Username 'testuser', got %s", result.Username)
	}
	if result.Text != "Hello world" {
		t.Errorf("expected Text 'Hello world', got %s", result.Text)
	}
	if result.HighlightedText != "Hello <mark>world</mark>" {
		t.Errorf("expected HighlightedText with mark, got %s", result.HighlightedText)
	}
}

func TestSearchPaginationDefaults(t *testing.T) {
	tests := []struct {
		name         string
		page         int
		pageSize     int
		wantPage     int
		wantPageSize int
	}{
		{
			name:         "zero values get defaults",
			page:         0,
			pageSize:     0,
			wantPage:     1,
			wantPageSize: 20,
		},
		{
			name:         "negative page becomes 1",
			page:         -1,
			pageSize:     10,
			wantPage:     1,
			wantPageSize: 10,
		},
		{
			name:         "valid values preserved",
			page:         5,
			pageSize:     50,
			wantPage:     5,
			wantPageSize: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := dto.SearchMessagesRequest{
				Query: "test",
				PaginationRequest: dto.PaginationRequest{
					Page:     tt.page,
					PageSize: tt.pageSize,
				},
			}

			err := req.Validate()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if req.Page != tt.wantPage {
				t.Errorf("Page = %d, want %d", req.Page, tt.wantPage)
			}
			if req.PageSize != tt.wantPageSize {
				t.Errorf("PageSize = %d, want %d", req.PageSize, tt.wantPageSize)
			}
		})
	}
}

func TestTimeRangeFilter(t *testing.T) {
	// Time range filtering is handled in the DTO layer via SearchMessagesRequest
	// and applied in the search repository. This test validates the time parsing behavior.

	// Valid time ranges work with the search (tested via integration tests)
	// This unit test validates that the DTO accepts time pointers correctly.
	req := dto.SearchMessagesRequest{
		Query:             "test",
		PaginationRequest: dto.PaginationRequest{Page: 1, PageSize: 20},
	}

	err := req.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// StartTime and EndTime are optional (nil by default)
	if req.StartTime != nil {
		t.Error("expected StartTime to be nil by default")
	}
	if req.EndTime != nil {
		t.Error("expected EndTime to be nil by default")
	}
}

func TestTimeRangeValidation(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)

	tests := []struct {
		name      string
		startTime *time.Time
		endTime   *time.Time
		wantErr   error
	}{
		{
			name:      "no time range",
			startTime: nil,
			endTime:   nil,
			wantErr:   nil,
		},
		{
			name:      "only start time",
			startTime: &yesterday,
			endTime:   nil,
			wantErr:   nil,
		},
		{
			name:      "only end time",
			startTime: nil,
			endTime:   &tomorrow,
			wantErr:   nil,
		},
		{
			name:      "valid range (start before end)",
			startTime: &yesterday,
			endTime:   &tomorrow,
			wantErr:   nil,
		},
		{
			name:      "same day (start equals end)",
			startTime: &now,
			endTime:   &now,
			wantErr:   nil,
		},
		{
			name:      "invalid range (end before start)",
			startTime: &tomorrow,
			endTime:   &yesterday,
			wantErr:   dto.ErrTimeRangeInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := dto.SearchMessagesRequest{
				Query:             "test",
				StartTime:         tt.startTime,
				EndTime:           tt.endTime,
				PaginationRequest: dto.PaginationRequest{Page: 1, PageSize: 20},
			}

			err := req.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
