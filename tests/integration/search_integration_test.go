package integration

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/asabla/goknut/internal/http/handlers"
	"github.com/asabla/goknut/internal/observability"
	"github.com/asabla/goknut/internal/repository"
	"github.com/asabla/goknut/internal/search"
)

// Note: These tests follow the failing-first TDD pattern.
// They will be enabled as search repository and service are implemented.

func TestMessageSearchFTS(t *testing.T) {
	// Setup temporary database with FTS enabled
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := repository.Open(repository.DBConfig{
		Path:      tmpFile.Name(),
		EnableFTS: true,
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Create test data
	channelRepo := repository.NewChannelRepository(db)
	channel := &repository.Channel{
		Name:        "testchannel",
		DisplayName: "Test Channel",
		Enabled:     true,
	}
	if err := channelRepo.Create(ctx, channel); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	user, err := userRepo.GetOrCreate(ctx, "testuser", "TestUser")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	msgRepo := repository.NewMessageRepository(db)

	// Insert test messages
	messages := []repository.Message{
		{ChannelID: channel.ID, UserID: user.ID, Text: "hello world this is a test", SentAt: time.Now().Add(-5 * time.Minute)},
		{ChannelID: channel.ID, UserID: user.ID, Text: "another message about testing", SentAt: time.Now().Add(-4 * time.Minute)},
		{ChannelID: channel.ID, UserID: user.ID, Text: "goodbye world", SentAt: time.Now().Add(-3 * time.Minute)},
		{ChannelID: channel.ID, UserID: user.ID, Text: "random chat message", SentAt: time.Now().Add(-2 * time.Minute)},
		{ChannelID: channel.ID, UserID: user.ID, Text: "testing one two three", SentAt: time.Now().Add(-1 * time.Minute)},
	}

	if err := msgRepo.CreateBatch(ctx, messages); err != nil {
		t.Fatalf("failed to create messages: %v", err)
	}

	// Create search repository and test FTS search
	searchRepo := search.NewSearchRepository(db, true)
	results, totalCount, err := searchRepo.SearchMessages(ctx, search.MessageSearchParams{
		Query:    "test",
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("failed to search messages: %v", err)
	}
	// Should match: "hello world this is a test", "another message about testing", "testing one two three"
	if totalCount < 2 {
		t.Errorf("expected at least 2 results, got %d", totalCount)
	}
	if len(results) < 2 {
		t.Errorf("expected at least 2 result items, got %d", len(results))
	}
}

func TestMessageSearchLIKE(t *testing.T) {
	// Setup temporary database with FTS DISABLED
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := repository.Open(repository.DBConfig{
		Path:      tmpFile.Name(),
		EnableFTS: false, // Disable FTS to test LIKE fallback
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Create test data
	channelRepo := repository.NewChannelRepository(db)
	channel := &repository.Channel{
		Name:        "testchannel",
		DisplayName: "Test Channel",
		Enabled:     true,
	}
	if err := channelRepo.Create(ctx, channel); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	user, err := userRepo.GetOrCreate(ctx, "testuser", "TestUser")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	msgRepo := repository.NewMessageRepository(db)

	// Insert test messages
	messages := []repository.Message{
		{ChannelID: channel.ID, UserID: user.ID, Text: "hello world", SentAt: time.Now().Add(-2 * time.Minute)},
		{ChannelID: channel.ID, UserID: user.ID, Text: "goodbye world", SentAt: time.Now().Add(-1 * time.Minute)},
	}
	if err := msgRepo.CreateBatch(ctx, messages); err != nil {
		t.Fatalf("failed to create messages: %v", err)
	}

	// Create search repository with FTS disabled
	searchRepo := search.NewSearchRepository(db, false)
	results, totalCount, err := searchRepo.SearchMessages(ctx, search.MessageSearchParams{
		Query:    "world",
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("failed to search messages: %v", err)
	}
	if totalCount != 2 {
		t.Errorf("expected 2 results, got %d", totalCount)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 result items, got %d", len(results))
	}
}

func TestUserSearch(t *testing.T) {
	// Setup temporary database
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := repository.Open(repository.DBConfig{
		Path:      tmpFile.Name(),
		EnableFTS: true,
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Create test users
	userRepo := repository.NewUserRepository(db)
	_, err = userRepo.GetOrCreate(ctx, "testuser1", "TestUser1")
	if err != nil {
		t.Fatalf("failed to create user1: %v", err)
	}
	_, err = userRepo.GetOrCreate(ctx, "testuser2", "TestUser2")
	if err != nil {
		t.Fatalf("failed to create user2: %v", err)
	}
	_, err = userRepo.GetOrCreate(ctx, "otheruser", "OtherUser")
	if err != nil {
		t.Fatalf("failed to create otheruser: %v", err)
	}

	// Create search repository and test user search
	searchRepo := search.NewSearchRepository(db, true)
	results, totalCount, err := searchRepo.SearchUsers(ctx, search.UserSearchParams{
		Query:    "test",
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("failed to search users: %v", err)
	}
	if totalCount != 2 {
		t.Errorf("expected 2 results, got %d", totalCount)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 result items, got %d", len(results))
	}
}

func TestUserSearchWithMessageCount(t *testing.T) {
	// Setup temporary database
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := repository.Open(repository.DBConfig{
		Path:      tmpFile.Name(),
		EnableFTS: true,
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Create test data with messages
	channelRepo := repository.NewChannelRepository(db)
	channel := &repository.Channel{
		Name:        "testchannel",
		DisplayName: "Test Channel",
		Enabled:     true,
	}
	if err := channelRepo.Create(ctx, channel); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	user, err := userRepo.GetOrCreate(ctx, "testuser", "TestUser")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	msgRepo := repository.NewMessageRepository(db)
	for i := 0; i < 5; i++ {
		if err := msgRepo.Create(ctx, &repository.Message{
			ChannelID: channel.ID,
			UserID:    user.ID,
			Text:      "test message",
			SentAt:    time.Now(),
		}); err != nil {
			t.Fatalf("failed to create message: %v", err)
		}
	}

	// Verify user search returns correct message count
	searchRepo := search.NewSearchRepository(db, true)
	results, _, err := searchRepo.SearchUsers(ctx, search.UserSearchParams{
		Query:    "test",
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("failed to search users: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 user, got %d", len(results))
	}
	if results[0].TotalMessages != 5 {
		t.Errorf("expected 5 messages, got %d", results[0].TotalMessages)
	}
}

func TestSearchFiltersChannelAndTimeRange(t *testing.T) {
	// Setup temporary database
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := repository.Open(repository.DBConfig{
		Path:      tmpFile.Name(),
		EnableFTS: true,
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Create test data across multiple channels and time ranges
	channelRepo := repository.NewChannelRepository(db)
	channel1 := &repository.Channel{
		Name:        "channel1",
		DisplayName: "Channel 1",
		Enabled:     true,
	}
	if err := channelRepo.Create(ctx, channel1); err != nil {
		t.Fatalf("failed to create channel1: %v", err)
	}
	channel2 := &repository.Channel{
		Name:        "channel2",
		DisplayName: "Channel 2",
		Enabled:     true,
	}
	if err := channelRepo.Create(ctx, channel2); err != nil {
		t.Fatalf("failed to create channel2: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	user, err := userRepo.GetOrCreate(ctx, "testuser", "TestUser")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	msgRepo := repository.NewMessageRepository(db)

	// Messages in channel1 (2 days ago)
	oldTime := time.Now().Add(-48 * time.Hour)
	if err := msgRepo.Create(ctx, &repository.Message{
		ChannelID: channel1.ID,
		UserID:    user.ID,
		Text:      "hello channel1",
		SentAt:    oldTime,
	}); err != nil {
		t.Fatalf("failed to create message: %v", err)
	}

	// Messages in channel2 (1 hour ago)
	recentTime := time.Now().Add(-1 * time.Hour)
	if err := msgRepo.Create(ctx, &repository.Message{
		ChannelID: channel2.ID,
		UserID:    user.ID,
		Text:      "hello channel2",
		SentAt:    recentTime,
	}); err != nil {
		t.Fatalf("failed to create message: %v", err)
	}

	searchRepo := search.NewSearchRepository(db, true)

	// Test channel filter
	channel1Name := "channel1"
	results, totalCount, err := searchRepo.SearchMessages(ctx, search.MessageSearchParams{
		Query:       "hello",
		ChannelName: &channel1Name,
		Page:        1,
		PageSize:    20,
	})
	if err != nil {
		t.Fatalf("failed to search messages: %v", err)
	}
	if totalCount != 1 {
		t.Errorf("expected 1 result for channel filter, got %d", totalCount)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result item, got %d", len(results))
	}

	// Test time range filter (only recent)
	startTime := time.Now().Add(-2 * time.Hour)
	endTime := time.Now()
	results, totalCount, err = searchRepo.SearchMessages(ctx, search.MessageSearchParams{
		Query:     "hello",
		StartTime: &startTime,
		EndTime:   &endTime,
		Page:      1,
		PageSize:  20,
	})
	if err != nil {
		t.Fatalf("failed to search messages: %v", err)
	}
	if totalCount != 1 {
		t.Errorf("expected 1 result for time range filter, got %d", totalCount)
	}
}

func TestSearchResultsReverseChronological(t *testing.T) {
	// Setup temporary database
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := repository.Open(repository.DBConfig{
		Path:      tmpFile.Name(),
		EnableFTS: true,
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Create test messages with different timestamps
	channelRepo := repository.NewChannelRepository(db)
	channel := &repository.Channel{
		Name:        "testchannel",
		DisplayName: "Test Channel",
		Enabled:     true,
	}
	if err := channelRepo.Create(ctx, channel); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	user, err := userRepo.GetOrCreate(ctx, "testuser", "TestUser")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	msgRepo := repository.NewMessageRepository(db)

	// Insert messages at different times
	if err := msgRepo.Create(ctx, &repository.Message{
		ChannelID: channel.ID,
		UserID:    user.ID,
		Text:      "oldest test message",
		SentAt:    time.Now().Add(-3 * time.Hour),
	}); err != nil {
		t.Fatalf("failed to create message: %v", err)
	}
	if err := msgRepo.Create(ctx, &repository.Message{
		ChannelID: channel.ID,
		UserID:    user.ID,
		Text:      "middle test message",
		SentAt:    time.Now().Add(-2 * time.Hour),
	}); err != nil {
		t.Fatalf("failed to create message: %v", err)
	}
	if err := msgRepo.Create(ctx, &repository.Message{
		ChannelID: channel.ID,
		UserID:    user.ID,
		Text:      "newest test message",
		SentAt:    time.Now().Add(-1 * time.Hour),
	}); err != nil {
		t.Fatalf("failed to create message: %v", err)
	}

	// Search for "test" and verify results are in reverse chronological order (newest first)
	searchRepo := search.NewSearchRepository(db, true)
	results, _, err := searchRepo.SearchMessages(ctx, search.MessageSearchParams{
		Query:    "test",
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("failed to search messages: %v", err)
	}
	if len(results) < 3 {
		t.Fatalf("expected at least 3 results, got %d", len(results))
	}

	// Verify order: newest first
	if results[0].Text != "newest test message" {
		t.Errorf("expected newest message first, got: %s", results[0].Text)
	}
	if results[len(results)-1].Text != "oldest test message" {
		t.Errorf("expected oldest message last, got: %s", results[len(results)-1].Text)
	}
}

func TestUserProfileData(t *testing.T) {
	// Setup temporary database
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := repository.Open(repository.DBConfig{
		Path:      tmpFile.Name(),
		EnableFTS: true,
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Create test data
	channelRepo := repository.NewChannelRepository(db)
	channel1 := &repository.Channel{
		Name:        "channel1",
		DisplayName: "Channel 1",
		Enabled:     true,
	}
	if err := channelRepo.Create(ctx, channel1); err != nil {
		t.Fatalf("failed to create channel1: %v", err)
	}
	channel2 := &repository.Channel{
		Name:        "channel2",
		DisplayName: "Channel 2",
		Enabled:     true,
	}
	if err := channelRepo.Create(ctx, channel2); err != nil {
		t.Fatalf("failed to create channel2: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	user, err := userRepo.GetOrCreate(ctx, "testuser", "TestUser")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	msgRepo := repository.NewMessageRepository(db)

	// Add messages across channels
	if err := msgRepo.Create(ctx, &repository.Message{
		ChannelID: channel1.ID,
		UserID:    user.ID,
		Text:      "message in channel1",
		SentAt:    time.Now().Add(-2 * time.Hour),
	}); err != nil {
		t.Fatalf("failed to create message: %v", err)
	}
	if err := msgRepo.Create(ctx, &repository.Message{
		ChannelID: channel2.ID,
		UserID:    user.ID,
		Text:      "message in channel2",
		SentAt:    time.Now().Add(-1 * time.Hour),
	}); err != nil {
		t.Fatalf("failed to create message: %v", err)
	}

	// Get user profile and verify
	searchRepo := search.NewSearchRepository(db, true)
	profile, err := searchRepo.GetUserProfileByUsername(ctx, "testuser")
	if err != nil {
		t.Fatalf("failed to get user profile: %v", err)
	}
	if profile == nil {
		t.Fatal("expected profile, got nil")
	}

	// Verify total_messages is correct
	if profile.TotalMessages != 2 {
		t.Errorf("expected 2 total messages, got %d", profile.TotalMessages)
	}

	// Verify channels list includes both channels
	if len(profile.Channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(profile.Channels))
	}
}

// TestUsersSSEStream tests the SSE stream for the users view.
// This test verifies:
// - SSE connection is established successfully with view=users
// - Status event is received on connect
func TestUsersSSEStream(t *testing.T) {
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

	// Create test users with messages
	channel := &repository.Channel{
		Name:        "testchannel",
		DisplayName: "TestChannel",
		Enabled:     true,
	}
	if err := channelRepo.Create(ctx, channel); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	// Create 3 users with different message counts
	users := []struct {
		username    string
		displayName string
		msgCount    int
	}{
		{"user1", "User One", 10},
		{"user2", "User Two", 5},
		{"user3", "User Three", 2},
	}

	for _, u := range users {
		user, err := userRepo.GetOrCreate(ctx, u.username, u.displayName)
		if err != nil {
			t.Fatalf("failed to create user: %v", err)
		}
		for i := 0; i < u.msgCount; i++ {
			msg := &repository.Message{
				ChannelID: channel.ID,
				UserID:    user.ID,
				Text:      "Test message",
				SentAt:    time.Now(),
			}
			if err := messageRepo.Create(ctx, msg); err != nil {
				t.Fatalf("failed to create message: %v", err)
			}
		}
	}

	// Create SSE handler
	logger := observability.NewLogger("test")
	metrics := observability.NewMetrics()
	sseHandler := handlers.NewSSEHandler(channelRepo, messageRepo, userRepo, nil, logger, metrics)

	// Create test server
	mux := http.NewServeMux()
	sseHandler.RegisterRoutes(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	// Connect to SSE stream for users view
	req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/live?view=users", nil)
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

	// Read status event
	scanner := bufio.NewScanner(resp.Body)
	timeout := time.After(3 * time.Second)
	gotStatus := false

	for !gotStatus {
		select {
		case <-timeout:
			t.Fatal("timeout waiting for status event")
		default:
			if scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "data: ") {
					data := strings.TrimPrefix(line, "data: ")
					var event map[string]any
					if err := json.Unmarshal([]byte(data), &event); err != nil {
						continue
					}
					if event["type"] == "status" && event["state"] == "connected" {
						gotStatus = true
					}
				}
			}
		}
	}

	if !gotStatus {
		t.Error("expected to receive status event")
	}
}
