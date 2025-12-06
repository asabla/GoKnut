// Package fakes provides test fakes for integration testing.
package fakes

import (
	"context"
	"sync"
	"time"

	"github.com/asabla/goknut/internal/ingestion"
)

// FakeMessageStore is a fake message store for testing.
type FakeMessageStore struct {
	mu sync.RWMutex

	messages     []ingestion.Message
	storeCalls   int
	storeError   error
	storeLatency time.Duration
}

// NewFakeMessageStore creates a new fake message store.
func NewFakeMessageStore() *FakeMessageStore {
	return &FakeMessageStore{
		messages: make([]ingestion.Message, 0),
	}
}

// StoreBatch stores a batch of messages.
func (f *FakeMessageStore) StoreBatch(ctx context.Context, messages []ingestion.Message) error {
	if f.storeLatency > 0 {
		time.Sleep(f.storeLatency)
	}

	if f.storeError != nil {
		return f.storeError
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages = append(f.messages, messages...)
	f.storeCalls++
	return nil
}

// SetStoreError sets an error to return from StoreBatch.
func (f *FakeMessageStore) SetStoreError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.storeError = err
}

// SetStoreLatency sets latency for StoreBatch.
func (f *FakeMessageStore) SetStoreLatency(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.storeLatency = d
}

// GetMessages returns all stored messages.
func (f *FakeMessageStore) GetMessages() []ingestion.Message {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return append([]ingestion.Message{}, f.messages...)
}

// GetStoreCalls returns the number of StoreBatch calls.
func (f *FakeMessageStore) GetStoreCalls() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.storeCalls
}

// ClearMessages clears stored messages.
func (f *FakeMessageStore) ClearMessages() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages = make([]ingestion.Message, 0)
	f.storeCalls = 0
}
