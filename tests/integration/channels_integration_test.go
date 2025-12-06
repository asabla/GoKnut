package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/asabla/goknut/internal/irc"
	"github.com/asabla/goknut/internal/repository"
	"github.com/asabla/goknut/tests/integration/fakes"
)

// Note: These tests follow the failing-first TDD pattern.
// They will be enabled as channel repository and service are implemented.

func TestChannelPersistence(t *testing.T) {
	t.Skip("Repository not yet fully implemented - failing-first TDD")

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

	// TODO: Test channel CRUD operations via repository
}

func TestChannelIRCJoinOnEnable(t *testing.T) {
	t.Skip("Channel service not yet implemented - failing-first TDD")

	fakeIRC := fakes.NewFakeIRCClient()

	// TODO: Create channel service with fake IRC
	// Enable a channel
	// Verify IRC.Join was called

	if err := fakeIRC.Connect(context.Background()); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	if err := fakeIRC.Join("#testchannel"); err != nil {
		t.Fatalf("failed to join: %v", err)
	}

	channels := fakeIRC.Channels()
	found := false
	for _, ch := range channels {
		if ch == "#testchannel" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected channel to be joined")
	}
}

func TestChannelIRCPartOnDisable(t *testing.T) {
	t.Skip("Channel service not yet implemented - failing-first TDD")

	fakeIRC := fakes.NewFakeIRCClient()

	// Setup
	fakeIRC.Connect(context.Background())
	fakeIRC.Join("#testchannel")

	// TODO: Create channel service with fake IRC
	// Disable the channel
	// Verify IRC.Part was called

	if err := fakeIRC.Part("#testchannel"); err != nil {
		t.Fatalf("failed to part: %v", err)
	}

	channels := fakeIRC.Channels()
	for _, ch := range channels {
		if ch == "#testchannel" {
			t.Error("expected channel to be left")
		}
	}
}

func TestChannelMessageRecording(t *testing.T) {
	t.Skip("Ingestion not yet wired - failing-first TDD")

	fakeIRC := fakes.NewFakeIRCClient()
	fakeStore := fakes.NewFakeMessageStore()

	// Setup
	fakeIRC.Connect(context.Background())
	fakeIRC.Join("#testchannel")

	// Simulate a message using the IRC message type
	fakeIRC.SimulateMessage(irc.Message{
		Channel:    "#testchannel",
		Username:   "testuser",
		Text:       "Hello world!",
		ReceivedAt: time.Now(),
	})

	// Allow time for processing
	time.Sleep(200 * time.Millisecond)

	// TODO: Verify message was stored
	_ = fakeStore
}
