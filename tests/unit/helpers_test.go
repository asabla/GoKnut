package unit

import (
	"os"
	"testing"

	"github.com/asabla/goknut/internal/config"
	"github.com/asabla/goknut/internal/http/dto"
)

// TestConfigValidation tests configuration validation
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &config.Config{
				DBPath:           "./test.db",
				HTTPAddr:         ":8080",
				TwitchUsername:   "testuser",
				TwitchOAuthToken: "oauth:token",
				BatchSize:        100,
				FlushTimeout:     100,
			},
			wantErr: false,
		},
		{
			name: "missing db path",
			cfg: &config.Config{
				DBPath:           "",
				HTTPAddr:         ":8080",
				TwitchUsername:   "testuser",
				TwitchOAuthToken: "oauth:token",
				BatchSize:        100,
				FlushTimeout:     100,
			},
			wantErr: true,
		},
		{
			name: "missing http addr",
			cfg: &config.Config{
				DBPath:           "./test.db",
				HTTPAddr:         "",
				TwitchUsername:   "testuser",
				TwitchOAuthToken: "oauth:token",
				BatchSize:        100,
				FlushTimeout:     100,
			},
			wantErr: true,
		},
		{
			name: "missing twitch username",
			cfg: &config.Config{
				DBPath:           "./test.db",
				HTTPAddr:         ":8080",
				TwitchUsername:   "",
				TwitchOAuthToken: "oauth:token",
				BatchSize:        100,
				FlushTimeout:     100,
			},
			wantErr: true,
		},
		{
			name: "missing oauth token",
			cfg: &config.Config{
				DBPath:           "./test.db",
				HTTPAddr:         ":8080",
				TwitchUsername:   "testuser",
				TwitchOAuthToken: "",
				BatchSize:        100,
				FlushTimeout:     100,
			},
			wantErr: true,
		},
		{
			name: "invalid batch size",
			cfg: &config.Config{
				DBPath:           "./test.db",
				HTTPAddr:         ":8080",
				TwitchUsername:   "testuser",
				TwitchOAuthToken: "oauth:token",
				BatchSize:        0,
				FlushTimeout:     100,
			},
			wantErr: true,
		},
		{
			name: "negative batch size",
			cfg: &config.Config{
				DBPath:           "./test.db",
				HTTPAddr:         ":8080",
				TwitchUsername:   "testuser",
				TwitchOAuthToken: "oauth:token",
				BatchSize:        -1,
				FlushTimeout:     100,
			},
			wantErr: true,
		},
		{
			name: "invalid flush timeout",
			cfg: &config.Config{
				DBPath:           "./test.db",
				HTTPAddr:         ":8080",
				TwitchUsername:   "testuser",
				TwitchOAuthToken: "oauth:token",
				BatchSize:        100,
				FlushTimeout:     0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestDefaultConfig tests default configuration values
func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	if cfg.DBPath != "./twitch.db" {
		t.Errorf("expected default DBPath './twitch.db', got %s", cfg.DBPath)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("expected default HTTPAddr ':8080', got %s", cfg.HTTPAddr)
	}
	if cfg.BatchSize != 100 {
		t.Errorf("expected default BatchSize 100, got %d", cfg.BatchSize)
	}
	if cfg.FlushTimeout != 100 {
		t.Errorf("expected default FlushTimeout 100, got %d", cfg.FlushTimeout)
	}
	if !cfg.EnableFTS {
		t.Error("expected default EnableFTS to be true")
	}
}

// TestConfigEnvOverrides tests that environment variables override defaults
func TestConfigEnvOverrides(t *testing.T) {
	// Save original env vars
	origDBPath := os.Getenv("DB_PATH")
	origHTTPAddr := os.Getenv("HTTP_ADDR")
	origUsername := os.Getenv("TWITCH_USERNAME")
	origToken := os.Getenv("TWITCH_OAUTH_TOKEN")

	// Restore after test
	defer func() {
		os.Setenv("DB_PATH", origDBPath)
		os.Setenv("HTTP_ADDR", origHTTPAddr)
		os.Setenv("TWITCH_USERNAME", origUsername)
		os.Setenv("TWITCH_OAUTH_TOKEN", origToken)
	}()

	// Set test values
	os.Setenv("DB_PATH", "/custom/path.db")
	os.Setenv("HTTP_ADDR", ":9090")
	os.Setenv("TWITCH_USERNAME", "envuser")
	os.Setenv("TWITCH_OAUTH_TOKEN", "oauth:envtoken")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DBPath != "/custom/path.db" {
		t.Errorf("expected DBPath '/custom/path.db', got %s", cfg.DBPath)
	}
	if cfg.HTTPAddr != ":9090" {
		t.Errorf("expected HTTPAddr ':9090', got %s", cfg.HTTPAddr)
	}
	if cfg.TwitchUsername != "envuser" {
		t.Errorf("expected TwitchUsername 'envuser', got %s", cfg.TwitchUsername)
	}
	if cfg.TwitchOAuthToken != "oauth:envtoken" {
		t.Errorf("expected TwitchOAuthToken 'oauth:envtoken', got %s", cfg.TwitchOAuthToken)
	}
}

// TestValidateUsername tests the username validation helper
func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  error
	}{
		{
			name:     "valid username",
			username: "testuser123",
			wantErr:  nil,
		},
		{
			name:     "valid with underscore",
			username: "test_user_123",
			wantErr:  nil,
		},
		{
			name:     "empty username",
			username: "",
			wantErr:  dto.ErrUsernameRequired,
		},
		{
			name:     "whitespace only",
			username: "   ",
			wantErr:  dto.ErrUsernameRequired,
		},
		{
			name:     "invalid characters - space",
			username: "test user",
			wantErr:  dto.ErrUsernameInvalid,
		},
		{
			name:     "invalid characters - special",
			username: "test@user",
			wantErr:  dto.ErrUsernameInvalid,
		},
		{
			name:     "invalid characters - hyphen",
			username: "test-user",
			wantErr:  dto.ErrUsernameInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := dto.ValidateUsername(tt.username)
			if err != tt.wantErr {
				t.Errorf("ValidateUsername(%q) = %v, want %v", tt.username, err, tt.wantErr)
			}
		})
	}
}

// TestChannelNameNormalization tests channel name normalization
func TestChannelNameNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase preserved",
			input:    "testchannel",
			expected: "testchannel",
		},
		{
			name:     "uppercase converted",
			input:    "TestChannel",
			expected: "testchannel",
		},
		{
			name:     "mixed case converted",
			input:    "TeSt_ChAnNeL",
			expected: "test_channel",
		},
		{
			name:     "with whitespace trimmed",
			input:    "  testchannel  ",
			expected: "testchannel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := dto.CreateChannelRequest{Name: tt.input, Enabled: true}
			err := req.Validate()
			if err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
			if req.Name != tt.expected {
				t.Errorf("expected name %q, got %q", tt.expected, req.Name)
			}
		})
	}
}

// TestUpdateChannelRequestOptionalFields tests that update request handles optional fields
func TestUpdateChannelRequestOptionalFields(t *testing.T) {
	// Test with all nil fields (no changes)
	req := dto.UpdateChannelRequest{}
	if req.DisplayName != nil {
		t.Error("expected nil DisplayName")
	}
	if req.Enabled != nil {
		t.Error("expected nil Enabled")
	}
	if req.RetainHistoryOnDelete != nil {
		t.Error("expected nil RetainHistoryOnDelete")
	}

	// Test with some fields set
	displayName := "New Name"
	enabled := true
	req2 := dto.UpdateChannelRequest{
		DisplayName: &displayName,
		Enabled:     &enabled,
	}
	if *req2.DisplayName != "New Name" {
		t.Errorf("expected DisplayName 'New Name', got %s", *req2.DisplayName)
	}
	if *req2.Enabled != true {
		t.Error("expected Enabled to be true")
	}
	if req2.RetainHistoryOnDelete != nil {
		t.Error("expected nil RetainHistoryOnDelete")
	}
}

// TestSearchQueryEdgeCases tests edge cases for search query validation
func TestSearchQueryEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr error
	}{
		{
			name:    "exactly 2 chars",
			query:   "ab",
			wantErr: nil,
		},
		{
			name:    "exactly 100 chars",
			query:   string(make([]byte, 100)),
			wantErr: nil,
		},
		{
			name:    "1 char - too short",
			query:   "a",
			wantErr: dto.ErrSearchQueryTooShort,
		},
		{
			name:    "101 chars - too long",
			query:   string(make([]byte, 101)),
			wantErr: dto.ErrSearchQueryTooLong,
		},
		{
			name:    "whitespace trimmed then validated",
			query:   "  ab  ",
			wantErr: nil,
		},
		{
			name:    "only whitespace - too short after trim",
			query:   "     ",
			wantErr: dto.ErrSearchQueryTooShort,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := dto.SearchMessagesRequest{
				Query:             tt.query,
				PaginationRequest: dto.PaginationRequest{Page: 1, PageSize: 20},
			}
			err := req.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestPaginationEdgeCases tests edge cases for pagination
func TestPaginationEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		page         int
		pageSize     int
		wantPage     int
		wantPageSize int
		wantOffset   int
		wantErr      bool
	}{
		{
			name:         "large page number",
			page:         1000,
			pageSize:     20,
			wantPage:     1000,
			wantPageSize: 20,
			wantOffset:   19980,
			wantErr:      false,
		},
		{
			name:         "page size 1",
			page:         5,
			pageSize:     1,
			wantPage:     5,
			wantPageSize: 1,
			wantOffset:   4,
			wantErr:      false,
		},
		{
			name:         "page size 100 (max)",
			page:         1,
			pageSize:     100,
			wantPage:     1,
			wantPageSize: 100,
			wantOffset:   0,
			wantErr:      false,
		},
		{
			name:         "page size 101 (over max)",
			page:         1,
			pageSize:     101,
			wantPage:     1,
			wantPageSize: 101,
			wantOffset:   0,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := dto.PaginationRequest{Page: tt.page, PageSize: tt.pageSize}
			err := req.Validate()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if req.Page != tt.wantPage {
				t.Errorf("Page = %d, want %d", req.Page, tt.wantPage)
			}
			if req.PageSize != tt.wantPageSize {
				t.Errorf("PageSize = %d, want %d", req.PageSize, tt.wantPageSize)
			}
			if req.Offset() != tt.wantOffset {
				t.Errorf("Offset() = %d, want %d", req.Offset(), tt.wantOffset)
			}
		})
	}
}

// TestDeleteChannelRequest tests delete channel request
func TestDeleteChannelRequest(t *testing.T) {
	// Test with retain history true
	req1 := dto.DeleteChannelRequest{RetainHistory: true}
	if !req1.RetainHistory {
		t.Error("expected RetainHistory to be true")
	}

	// Test with retain history false
	req2 := dto.DeleteChannelRequest{RetainHistory: false}
	if req2.RetainHistory {
		t.Error("expected RetainHistory to be false")
	}

	// Test zero value defaults to false
	req3 := dto.DeleteChannelRequest{}
	if req3.RetainHistory {
		t.Error("expected default RetainHistory to be false")
	}
}
