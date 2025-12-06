// Package fakes provides test fakes for integration testing.
package fakes

import (
	"context"
	"sync"
	"time"

	"github.com/asabla/goknut/internal/irc"
)

// FakeIRCClient is a fake IRC client for testing.
type FakeIRCClient struct {
	mu sync.RWMutex

	connected bool
	anonymous bool
	authMode  irc.AuthMode
	channels  map[string]bool
	messages  []irc.Message

	onMessage       irc.MessageHandler
	onChannelChange irc.ChannelChangeHandler

	// Control channels for testing
	ConnectError   error
	JoinError      error
	PartError      error
	SimulatedDelay time.Duration
}

// NewFakeIRCClient creates a new fake IRC client.
func NewFakeIRCClient() *FakeIRCClient {
	return &FakeIRCClient{
		channels: make(map[string]bool),
		messages: make([]irc.Message, 0),
		authMode: irc.AuthModeAuthenticated,
	}
}

// NewFakeAnonymousIRCClient creates a new fake anonymous IRC client.
func NewFakeAnonymousIRCClient() *FakeIRCClient {
	return &FakeIRCClient{
		channels:  make(map[string]bool),
		messages:  make([]irc.Message, 0),
		authMode:  irc.AuthModeAnonymous,
		anonymous: true,
	}
}

// SetOnMessage sets the message handler.
func (f *FakeIRCClient) SetOnMessage(handler irc.MessageHandler) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onMessage = handler
}

// SetOnChannelChange sets the channel change handler.
func (f *FakeIRCClient) SetOnChannelChange(handler irc.ChannelChangeHandler) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onChannelChange = handler
}

// Connect simulates connecting to IRC.
func (f *FakeIRCClient) Connect(ctx context.Context) error {
	if f.SimulatedDelay > 0 {
		time.Sleep(f.SimulatedDelay)
	}

	if f.ConnectError != nil {
		return f.ConnectError
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.connected = true
	return nil
}

// Disconnect simulates disconnecting from IRC.
func (f *FakeIRCClient) Disconnect() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.connected = false
	return nil
}

// Join simulates joining a channel.
func (f *FakeIRCClient) Join(channel string) error {
	if f.JoinError != nil {
		return f.JoinError
	}

	f.mu.Lock()
	f.channels[channel] = true
	handler := f.onChannelChange
	f.mu.Unlock()

	if handler != nil {
		handler(channel, true)
	}

	return nil
}

// Part simulates leaving a channel.
func (f *FakeIRCClient) Part(channel string) error {
	if f.PartError != nil {
		return f.PartError
	}

	f.mu.Lock()
	delete(f.channels, channel)
	handler := f.onChannelChange
	f.mu.Unlock()

	if handler != nil {
		handler(channel, false)
	}

	return nil
}

// IsConnected returns whether the client is connected.
func (f *FakeIRCClient) IsConnected() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.connected
}

// Channels returns the list of joined channels.
func (f *FakeIRCClient) Channels() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	channels := make([]string, 0, len(f.channels))
	for ch := range f.channels {
		channels = append(channels, ch)
	}
	return channels
}

// IsAnonymous returns true if the client is using anonymous mode.
func (f *FakeIRCClient) IsAnonymous() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.anonymous
}

// AuthMode returns the current authentication mode.
func (f *FakeIRCClient) AuthMode() irc.AuthMode {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.authMode
}

// SimulateMessage simulates receiving a message.
func (f *FakeIRCClient) SimulateMessage(msg irc.Message) {
	f.mu.Lock()
	f.messages = append(f.messages, msg)
	handler := f.onMessage
	f.mu.Unlock()

	if handler != nil {
		handler(msg)
	}
}

// GetReceivedMessages returns all received messages.
func (f *FakeIRCClient) GetReceivedMessages() []irc.Message {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return append([]irc.Message{}, f.messages...)
}

// ClearMessages clears received messages.
func (f *FakeIRCClient) ClearMessages() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages = make([]irc.Message, 0)
}
