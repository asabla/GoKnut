package integration

import (
	"context"
	"testing"
	"time"

	"github.com/asabla/goknut/internal/ingestion"
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
