# Contract Tests

This directory contains HTTP/API contract tests for the Twitch Chat Archiver.

## Overview

Contract tests verify that HTTP endpoints conform to the API specification defined in `specs/001-spec-reference-spec/contracts/api.md`.

## Running Tests

```bash
# Run all contract tests
go test ./tests/contract/...

# Run with verbose output
go test -v ./tests/contract/...

# Run specific test
go test -v ./tests/contract/... -run TestChannels
```

## Test Structure

Each endpoint has corresponding tests that verify:
- Response status codes
- Response body structure
- Error handling
- Content-Type headers
- HTMX fragment responses

## Test Files

- `channels_test.go` - Channel CRUD endpoints
- `channel_view_test.go` - Live channel view and message streaming
- `search_test.go` - User and message search endpoints

## Writing Contract Tests

1. Use `httptest.NewServer` to create a test server
2. Make HTTP requests against the test server
3. Assert response matches the contract specification
4. Test both success and error cases

### Example

```go
func TestChannelsList(t *testing.T) {
    srv := setupTestServer(t)
    defer srv.Close()

    resp, err := http.Get(srv.URL + "/channels")
    if err != nil {
        t.Fatalf("request failed: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        t.Errorf("expected status 200, got %d", resp.StatusCode)
    }

    // Verify response structure matches contract
    // ...
}
```

## Failing-First Pattern

Contract tests follow the TDD failing-first pattern:

1. Write tests that define expected behavior
2. Run tests (they should fail initially)
3. Implement the endpoint
4. Run tests (they should pass)
5. Refactor as needed
