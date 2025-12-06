package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/asabla/goknut/internal/repository"
)

// Note: These tests follow the failing-first TDD pattern.
// They will be enabled as search repository and service are implemented.

func TestMessageSearchFTS(t *testing.T) {
	t.Skip("Search repository not yet implemented - failing-first TDD")

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

	// TODO: Create search repository and test FTS search
	// searchRepo := search.NewSearchRepository(db)
	// results, total, err := searchRepo.SearchMessages(ctx, search.MessageSearchParams{
	// 	Query:    "test",
	// 	Page:     1,
	// 	PageSize: 20,
	// })
	// if err != nil {
	// 	t.Fatalf("failed to search messages: %v", err)
	// }
	// if total != 3 {
	// 	t.Errorf("expected 3 results, got %d", total)
	// }
}

func TestMessageSearchLIKE(t *testing.T) {
	t.Skip("Search repository not yet implemented - failing-first TDD")

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

	// TODO: Create test data and verify LIKE-based search works
}

func TestUserSearch(t *testing.T) {
	t.Skip("Search repository not yet implemented - failing-first TDD")

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

	// TODO: Create search repository and test user search
	// searchRepo := search.NewSearchRepository(db)
	// results, total, err := searchRepo.SearchUsers(ctx, search.UserSearchParams{
	// 	Query:    "test",
	// 	Page:     1,
	// 	PageSize: 20,
	// })
	// if err != nil {
	// 	t.Fatalf("failed to search users: %v", err)
	// }
	// if total != 2 {
	// 	t.Errorf("expected 2 results, got %d", total)
	// }
}

func TestUserSearchWithMessageCount(t *testing.T) {
	t.Skip("Search repository not yet implemented - failing-first TDD")

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
	_ = channelRepo.Create(ctx, channel)

	userRepo := repository.NewUserRepository(db)
	user, _ := userRepo.GetOrCreate(ctx, "testuser", "TestUser")

	msgRepo := repository.NewMessageRepository(db)
	for i := 0; i < 5; i++ {
		_ = msgRepo.Create(ctx, &repository.Message{
			ChannelID: channel.ID,
			UserID:    user.ID,
			Text:      "test message",
			SentAt:    time.Now(),
		})
	}

	// TODO: Verify user search returns correct message count
}

func TestSearchFiltersChannelAndTimeRange(t *testing.T) {
	t.Skip("Search repository not yet implemented - failing-first TDD")

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
	_ = channelRepo.Create(ctx, channel1)
	channel2 := &repository.Channel{
		Name:        "channel2",
		DisplayName: "Channel 2",
		Enabled:     true,
	}
	_ = channelRepo.Create(ctx, channel2)

	userRepo := repository.NewUserRepository(db)
	user, _ := userRepo.GetOrCreate(ctx, "testuser", "TestUser")

	msgRepo := repository.NewMessageRepository(db)

	// Messages in channel1
	_ = msgRepo.Create(ctx, &repository.Message{
		ChannelID: channel1.ID,
		UserID:    user.ID,
		Text:      "hello channel1",
		SentAt:    time.Now().Add(-48 * time.Hour), // 2 days ago
	})

	// Messages in channel2
	_ = msgRepo.Create(ctx, &repository.Message{
		ChannelID: channel2.ID,
		UserID:    user.ID,
		Text:      "hello channel2",
		SentAt:    time.Now().Add(-1 * time.Hour), // 1 hour ago
	})

	// TODO: Test channel filter
	// TODO: Test time range filter
	// TODO: Test combined filters
}

func TestSearchResultsReverseChronological(t *testing.T) {
	t.Skip("Search repository not yet implemented - failing-first TDD")

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
	_ = channelRepo.Create(ctx, channel)

	userRepo := repository.NewUserRepository(db)
	user, _ := userRepo.GetOrCreate(ctx, "testuser", "TestUser")

	msgRepo := repository.NewMessageRepository(db)

	// Insert messages at different times
	_ = msgRepo.Create(ctx, &repository.Message{
		ChannelID: channel.ID,
		UserID:    user.ID,
		Text:      "oldest test message",
		SentAt:    time.Now().Add(-3 * time.Hour),
	})
	_ = msgRepo.Create(ctx, &repository.Message{
		ChannelID: channel.ID,
		UserID:    user.ID,
		Text:      "middle test message",
		SentAt:    time.Now().Add(-2 * time.Hour),
	})
	_ = msgRepo.Create(ctx, &repository.Message{
		ChannelID: channel.ID,
		UserID:    user.ID,
		Text:      "newest test message",
		SentAt:    time.Now().Add(-1 * time.Hour),
	})

	// TODO: Search for "test" and verify results are in reverse chronological order
	// (newest first)
}

func TestUserProfileData(t *testing.T) {
	t.Skip("Search repository not yet implemented - failing-first TDD")

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
	_ = channelRepo.Create(ctx, channel1)
	channel2 := &repository.Channel{
		Name:        "channel2",
		DisplayName: "Channel 2",
		Enabled:     true,
	}
	_ = channelRepo.Create(ctx, channel2)

	userRepo := repository.NewUserRepository(db)
	user, _ := userRepo.GetOrCreate(ctx, "testuser", "TestUser")

	msgRepo := repository.NewMessageRepository(db)

	// Add messages across channels
	_ = msgRepo.Create(ctx, &repository.Message{
		ChannelID: channel1.ID,
		UserID:    user.ID,
		Text:      "message in channel1",
		SentAt:    time.Now().Add(-2 * time.Hour),
	})
	_ = msgRepo.Create(ctx, &repository.Message{
		ChannelID: channel2.ID,
		UserID:    user.ID,
		Text:      "message in channel2",
		SentAt:    time.Now().Add(-1 * time.Hour),
	})

	// TODO: Get user profile and verify:
	// - first_seen_at and last_seen_at are correct
	// - total_messages is correct
	// - channels list includes both channels
}
