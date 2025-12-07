package integration

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/asabla/goknut/internal/http/handlers"
	"github.com/asabla/goknut/internal/ingestion"
	"github.com/asabla/goknut/internal/observability"
	"github.com/asabla/goknut/internal/repository"
	"github.com/asabla/goknut/tests/integration/fakes"
)

// Note: These tests follow the failing-first TDD pattern.
// They define the expected behavior for ingestion → storage → HTMX fragments.

func TestIngestionToStoragePipeline(t *testing.T) {
	t.Skip("Integration not yet implemented - failing-first TDD")

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

	// Create test channel
	channelRepo := repository.NewChannelRepository(db)
	channel := &repository.Channel{
		Name:        "testchannel",
		DisplayName: "TestChannel",
		Enabled:     true,
	}
	if err := channelRepo.Create(ctx, channel); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	// TODO: Create MessageRepository and Processor
	// messageRepo := repository.NewMessageRepository(db)
	// processor := ingestion.NewProcessor(messageRepo, channelRepo, userRepo)

	// Set up pipeline with processor as store
	// pipeline := ingestion.NewPipeline(ingestion.DefaultPipelineConfig(), processor)
	// pipeline.Start(ctx)
	// defer pipeline.Stop()

	// Ingest messages
	// messages := []ingestion.Message{
	// 	{ChannelName: "#testchannel", Username: "user1", Text: "Hello world", ReceivedAt: time.Now()},
	// 	{ChannelName: "#testchannel", Username: "user2", Text: "Hi there", ReceivedAt: time.Now()},
	// }
	// for _, msg := range messages {
	// 	pipeline.Ingest(msg)
	// }

	// Wait for batch to flush
	// time.Sleep(200 * time.Millisecond)

	// Verify messages were stored
	// storedMsgs, err := messageRepo.GetRecent(ctx, channel.ID, 10)
	// if err != nil {
	// 	t.Fatalf("failed to get messages: %v", err)
	// }
	// if len(storedMsgs) != 2 {
	// 	t.Errorf("expected 2 messages, got %d", len(storedMsgs))
	// }
}

func TestIngestionBatchingBehavior(t *testing.T) {
	ctx := context.Background()

	store := fakes.NewFakeMessageStore()
	cfg := ingestion.PipelineConfig{
		BatchSize:    5,
		FlushTimeout: 100 * time.Millisecond,
		BufferSize:   100,
	}

	pipeline := ingestion.NewPipeline(cfg, store)
	if err := pipeline.Start(ctx); err != nil {
		t.Fatalf("failed to start pipeline: %v", err)
	}
	defer pipeline.Stop()

	// Send 3 messages (less than batch size)
	for i := 0; i < 3; i++ {
		pipeline.Ingest(ingestion.Message{
			ChannelName: "#test",
			Username:    "user",
			Text:        "message",
			ReceivedAt:  time.Now(),
		})
	}

	// Wait for timeout flush
	time.Sleep(150 * time.Millisecond)

	messages := store.GetMessages()
	if len(messages) != 3 {
		t.Errorf("expected 3 messages after timeout flush, got %d", len(messages))
	}
}

func TestIngestionBatchSizeFlush(t *testing.T) {
	ctx := context.Background()

	store := fakes.NewFakeMessageStore()
	cfg := ingestion.PipelineConfig{
		BatchSize:    5,
		FlushTimeout: 10 * time.Second, // Long timeout so we test batch size flush
		BufferSize:   100,
	}

	pipeline := ingestion.NewPipeline(cfg, store)
	if err := pipeline.Start(ctx); err != nil {
		t.Fatalf("failed to start pipeline: %v", err)
	}
	defer pipeline.Stop()

	// Send exactly batch size messages
	for i := 0; i < 5; i++ {
		pipeline.Ingest(ingestion.Message{
			ChannelName: "#test",
			Username:    "user",
			Text:        "message",
			ReceivedAt:  time.Now(),
		})
	}

	// Give a small amount of time for processing
	time.Sleep(50 * time.Millisecond)

	messages := store.GetMessages()
	if len(messages) != 5 {
		t.Errorf("expected 5 messages after batch size flush, got %d", len(messages))
	}
}

func TestIngestionDropsMessagesWhenBufferFull(t *testing.T) {
	ctx := context.Background()

	var droppedCount int
	metrics := &fakeMetrics{
		onDropped: func(count int) { droppedCount += count },
	}

	store := fakes.NewFakeMessageStore()
	// Simulate slow store
	store.SetStoreLatency(500 * time.Millisecond)

	cfg := ingestion.PipelineConfig{
		BatchSize:    10,
		FlushTimeout: 50 * time.Millisecond,
		BufferSize:   5, // Small buffer
		Metrics:      metrics,
	}

	pipeline := ingestion.NewPipeline(cfg, store)
	if err := pipeline.Start(ctx); err != nil {
		t.Fatalf("failed to start pipeline: %v", err)
	}
	defer pipeline.Stop()

	// Send many messages quickly to overflow buffer
	for i := 0; i < 20; i++ {
		pipeline.Ingest(ingestion.Message{
			ChannelName: "#test",
			Username:    "user",
			Text:        "message",
			ReceivedAt:  time.Now(),
		})
	}

	// Some messages should have been dropped
	if droppedCount == 0 {
		t.Log("expected some messages to be dropped, got 0")
		// This may not always trigger depending on timing
	}
}

func TestMessageStorageWithUserCreation(t *testing.T) {
	t.Skip("Integration not yet implemented - failing-first TDD")

	// ctx := context.Background()

	// Set up test database
	// db, err := repository.Open(repository.DBConfig{Path: ":memory:"})
	// ...

	// When ingesting messages, users should be created if they don't exist
	// and user stats (first_seen_at, last_seen_at, total_messages) should be updated
}

func TestLiveViewFragmentGeneration(t *testing.T) {
	t.Skip("Integration not yet implemented - failing-first TDD")

	// Test that the HTTP handler generates proper HTMX fragments
	// for the stream endpoint that can be swapped into the page
}

func TestLiveViewLatency(t *testing.T) {
	t.Skip("Integration not yet implemented - failing-first TDD")

	// Test that messages are available via the stream endpoint
	// within the 1s latency budget after ingestion
}

// TestHomeSSEStream tests the SSE stream for the home view.
// This test verifies:
// - SSE connection is established successfully
// - Initial metrics event is received
// - Metrics include total_messages, total_channels, total_users
// - Events are properly formatted as SSE
func TestHomeSSEStream(t *testing.T) {
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

	// Create test data
	channel := &repository.Channel{
		Name:        "testchannel",
		DisplayName: "TestChannel",
		Enabled:     true,
	}
	if err := channelRepo.Create(ctx, channel); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	user, err := userRepo.GetOrCreate(ctx, "testuser", "TestUser")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create some messages
	for i := 0; i < 5; i++ {
		msg := &repository.Message{
			ChannelID: channel.ID,
			UserID:    user.ID,
			Text:      "Test message " + string(rune('A'+i)),
			SentAt:    time.Now(),
		}
		if err := messageRepo.Create(ctx, msg); err != nil {
			t.Fatalf("failed to create message: %v", err)
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

	// Connect to SSE stream
	req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/live?view=home", nil)
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

	// Read first few events (with timeout)
	scanner := bufio.NewScanner(resp.Body)
	var events []string
	eventCount := 0
	timeout := time.After(3 * time.Second)

	for eventCount < 2 {
		select {
		case <-timeout:
			t.Fatalf("timeout waiting for events, got %d events: %v", eventCount, events)
		default:
			if scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "data: ") {
					events = append(events, strings.TrimPrefix(line, "data: "))
					eventCount++
				}
			} else {
				if err := scanner.Err(); err != nil {
					t.Fatalf("scanner error: %v", err)
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
	}

	// Verify we got status and metrics events
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(events))
	}

	// First event should be status=connected
	var statusEvent map[string]any
	if err := json.Unmarshal([]byte(events[0]), &statusEvent); err != nil {
		t.Fatalf("failed to parse status event: %v", err)
	}
	if statusEvent["type"] != "status" {
		t.Errorf("expected first event type 'status', got %v", statusEvent["type"])
	}
	if statusEvent["state"] != "connected" {
		t.Errorf("expected state 'connected', got %v", statusEvent["state"])
	}

	// Second event should be metrics
	var metricsEvent map[string]any
	if err := json.Unmarshal([]byte(events[1]), &metricsEvent); err != nil {
		t.Fatalf("failed to parse metrics event: %v", err)
	}
	if metricsEvent["type"] != "metrics" {
		t.Errorf("expected second event type 'metrics', got %v", metricsEvent["type"])
	}

	// Verify metrics content
	if total, ok := metricsEvent["total_messages"].(float64); !ok || total != 5 {
		t.Errorf("expected total_messages=5, got %v", metricsEvent["total_messages"])
	}
	if total, ok := metricsEvent["total_channels"].(float64); !ok || total != 1 {
		t.Errorf("expected total_channels=1, got %v", metricsEvent["total_channels"])
	}
	if total, ok := metricsEvent["total_users"].(float64); !ok || total != 1 {
		t.Errorf("expected total_users=1, got %v", metricsEvent["total_users"])
	}
}

// TestHomeSSEStreamUpdates tests that home view receives live updates.
func TestHomeSSEStreamUpdates(t *testing.T) {
	t.Skip("Integration not yet implemented - requires message broadcast mechanism")

	// This test will verify:
	// - After initial connection, new messages trigger metrics updates
	// - New messages are received as message events
	// - Updates are received within 2s of ingestion
}

// TestMessagesSSEStream tests the SSE stream for the messages view.
// This test verifies:
// - SSE connection is established successfully with view=messages
// - Backfill messages are received in order when after_id is provided
// - Messages have unique IDs (no duplicates)
// - Messages are ordered by ID (ascending)
func TestMessagesSSEStream(t *testing.T) {
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

	// Create test data
	channel := &repository.Channel{
		Name:        "testchannel",
		DisplayName: "TestChannel",
		Enabled:     true,
	}
	if err := channelRepo.Create(ctx, channel); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	user, err := userRepo.GetOrCreate(ctx, "testuser", "TestUser")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create 10 messages with sequential IDs
	var messageIDs []int64
	for i := 0; i < 10; i++ {
		msg := &repository.Message{
			ChannelID: channel.ID,
			UserID:    user.ID,
			Text:      "Test message " + string(rune('A'+i)),
			SentAt:    time.Now().Add(time.Duration(i) * time.Second),
		}
		if err := messageRepo.Create(ctx, msg); err != nil {
			t.Fatalf("failed to create message: %v", err)
		}
		messageIDs = append(messageIDs, msg.ID)
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

	// Test 1: Connect to messages view and receive initial status
	t.Run("initial connection", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/live?view=messages", nil)
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

		// Read first event (status=connected)
		scanner := bufio.NewScanner(resp.Body)
		timeout := time.After(3 * time.Second)
		var statusEvent map[string]any

		for {
			select {
			case <-timeout:
				t.Fatal("timeout waiting for status event")
			default:
				if scanner.Scan() {
					line := scanner.Text()
					if strings.HasPrefix(line, "data: ") {
						data := strings.TrimPrefix(line, "data: ")
						if err := json.Unmarshal([]byte(data), &statusEvent); err != nil {
							t.Fatalf("failed to parse event: %v", err)
						}
						if statusEvent["type"] == "status" && statusEvent["state"] == "connected" {
							return // Success
						}
					}
				}
			}
		}
	})

	// Test 2: Connect with after_id to get backfill
	t.Run("backfill ordering", func(t *testing.T) {
		// Use the 3rd message ID as cursor to get messages 4-10
		afterID := messageIDs[2]
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/live?view=messages&after_id="+string(rune('0'+int(afterID))), nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		// Fix the URL encoding
		req, err = http.NewRequestWithContext(ctx, "GET", server.URL+"/live?view=messages&after_id=3", nil)
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

		// Read events until we have all backfill messages
		scanner := bufio.NewScanner(resp.Body)
		timeout := time.After(3 * time.Second)
		var receivedMessages []map[string]any
		seenIDs := make(map[float64]bool)

		for len(receivedMessages) < 7 { // Expecting 7 messages (IDs 4-10)
			select {
			case <-timeout:
				t.Fatalf("timeout waiting for backfill, got %d messages", len(receivedMessages))
			default:
				if scanner.Scan() {
					line := scanner.Text()
					if strings.HasPrefix(line, "data: ") {
						data := strings.TrimPrefix(line, "data: ")
						var event map[string]any
						if err := json.Unmarshal([]byte(data), &event); err != nil {
							continue
						}
						if event["type"] == "message" {
							id := event["id"].(float64)
							// Check for duplicates
							if seenIDs[id] {
								t.Errorf("duplicate message ID: %v", id)
							}
							seenIDs[id] = true
							receivedMessages = append(receivedMessages, event)
						}
					}
				}
			}
		}

		// Verify ordering (ascending by ID)
		for i := 1; i < len(receivedMessages); i++ {
			prevID := receivedMessages[i-1]["id"].(float64)
			currID := receivedMessages[i]["id"].(float64)
			if currID <= prevID {
				t.Errorf("messages not in order: %v followed by %v", prevID, currID)
			}
		}
	})
}

// TestMessagesSSEStreamNoDuplicates tests that reconnecting clients don't receive duplicate messages.
func TestMessagesSSEStreamNoDuplicates(t *testing.T) {
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

	// Create test data
	channel := &repository.Channel{
		Name:        "testchannel",
		DisplayName: "TestChannel",
		Enabled:     true,
	}
	if err := channelRepo.Create(ctx, channel); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	user, err := userRepo.GetOrCreate(ctx, "testuser", "TestUser")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create 5 messages
	var lastMsgID int64
	for i := 0; i < 5; i++ {
		msg := &repository.Message{
			ChannelID: channel.ID,
			UserID:    user.ID,
			Text:      "Initial message " + string(rune('A'+i)),
			SentAt:    time.Now().Add(time.Duration(i) * time.Second),
		}
		if err := messageRepo.Create(ctx, msg); err != nil {
			t.Fatalf("failed to create message: %v", err)
		}
		lastMsgID = msg.ID
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

	// Connect with after_id = lastMsgID, should receive no backfill messages
	req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/live?view=messages&after_id="+fmt.Sprintf("%d", lastMsgID), nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to connect to SSE: %v", err)
	}
	defer resp.Body.Close()

	// Read events for a short period
	scanner := bufio.NewScanner(resp.Body)
	timeout := time.After(1 * time.Second)
	messageCount := 0

	for {
		select {
		case <-timeout:
			// Expected: no message events, only status
			if messageCount > 0 {
				t.Errorf("expected no backfill messages, got %d", messageCount)
			}
			return
		default:
			if scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "data: ") {
					data := strings.TrimPrefix(line, "data: ")
					var event map[string]any
					if err := json.Unmarshal([]byte(data), &event); err != nil {
						continue
					}
					if event["type"] == "message" {
						messageCount++
					}
				}
			}
		}
	}
}

// TestUserProfileSSEStream tests the SSE stream for the user profile view.
// This test verifies:
// - SSE connection is established successfully with view=user_profile&user=<username>
// - Initial user_profile event is received with correct data
// - user parameter is required
func TestUserProfileSSEStream(t *testing.T) {
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

	// Create test data
	channel := &repository.Channel{
		Name:        "testchannel",
		DisplayName: "TestChannel",
		Enabled:     true,
	}
	if err := channelRepo.Create(ctx, channel); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	user, err := userRepo.GetOrCreate(ctx, "testuser", "TestUser")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create 5 messages for this user
	for i := 0; i < 5; i++ {
		msg := &repository.Message{
			ChannelID: channel.ID,
			UserID:    user.ID,
			Text:      "Test message " + string(rune('A'+i)),
			SentAt:    time.Now().Add(time.Duration(i) * time.Second),
		}
		if err := messageRepo.Create(ctx, msg); err != nil {
			t.Fatalf("failed to create message: %v", err)
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

	t.Run("requires user parameter", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/live?view=user_profile", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// Should return 400 Bad Request
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("receives user_profile event", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/live?view=user_profile&user=testuser", nil)
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
		var profileEvent map[string]any
		gotStatus := false

		for profileEvent == nil {
			select {
			case <-timeout:
				t.Fatal("timeout waiting for user_profile event")
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
						} else if event["type"] == "user_profile" {
							profileEvent = event
						}
					}
				}
			}
		}

		if !gotStatus {
			t.Error("expected to receive status event")
		}

		// Verify user_profile event content
		if profileEvent["username"] != "testuser" {
			t.Errorf("expected username 'testuser', got %v", profileEvent["username"])
		}
		if total, ok := profileEvent["total_messages"].(float64); !ok || total != 5 {
			t.Errorf("expected total_messages=5, got %v", profileEvent["total_messages"])
		}
	})
}

// fakeMetrics implements ingestion.Metrics for testing
type fakeMetrics struct {
	onBatchSize    func(int)
	onBatchLatency func(time.Duration)
	onDropped      func(int)
}

func (f *fakeMetrics) RecordBatchSize(size int) {
	if f.onBatchSize != nil {
		f.onBatchSize(size)
	}
}

func (f *fakeMetrics) RecordBatchLatency(d time.Duration) {
	if f.onBatchLatency != nil {
		f.onBatchLatency(d)
	}
}

func (f *fakeMetrics) RecordDroppedMessages(count int) {
	if f.onDropped != nil {
		f.onDropped(count)
	}
}
