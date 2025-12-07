# Quickstart: Live Stream Updates

## Prerequisites
- Go 1.22 toolchain
- SQLite available (WAL enabled by default in repo setup)
- Web browser with SSE support (all modern browsers)

## Run the server
```bash
make run
# or
go run ./cmd/server
```

Server starts on the configured port (see existing config). SSE is enabled by default.

## Configuration

SSE live updates are controlled via the `ENABLE_SSE` environment variable:
```bash
# Enable SSE (default)
ENABLE_SSE=true go run ./cmd/server

# Disable SSE (fallback to polling)
ENABLE_SSE=false go run ./cmd/server
```

## Connect to live stream

- **Primary transport**: Server-Sent Events (SSE) at `/live`.
- Example (home view):
  - URL: `http://localhost:8080/live?view=home&after_id=0`
  - Receives JSON envelopes:
    - `metrics`, `message`, `status`
- Example (messages view): `http://localhost:8080/live?view=messages&after_id=0`
- Example (channels view): `http://localhost:8080/live?view=channels`
- Example (users view): `http://localhost:8080/live?view=users`
- Example (user profile): `http://localhost:8080/live?view=user_profile&user=username`

### Using `curl` for SSE testing
```bash
# Connect to home view SSE stream
curl -N -H "Accept: text/event-stream" "http://localhost:8080/live?view=home&after_id=0"

# Connect to messages view SSE stream
curl -N -H "Accept: text/event-stream" "http://localhost:8080/live?view=messages&after_id=0"
```

### HTMX integration
- Templates use `hx-sse` to bind SSE events to page elements
- When SSE unavailable, existing polling endpoints continue to function
- Status indicators show connection state (connected/idle/reconnecting/fallback)

## Reconnect & fallback behavior
- Client retries with exponential backoff (start ~1s, cap ~30s).
- On reconnect, client passes `after_id` to catch up.
- If server responds with `status=fallback`, client should switch to polling.
- If SSE is disabled via config, existing polling endpoints remain available.

## Validation steps
1. Start server: `make run`
2. Open `http://localhost:8080/` (home) and `/messages` in browser.
3. Ingest or create new messages (existing ingestion pipeline).
4. Observe live updates (metrics/messages) without reload.
5. Check browser dev tools Network tab for SSE connection to `/live`.
6. If SSE blocked, verify polling continues via `/channels/{name}/messages/stream`.

## Troubleshooting
- If SSE connection fails: check port, proxies, or CSP blocking EventSource; use polling as fallback.
- If backlog is large and server closes connection with `fallback`: reload page to continue via polling.
- Check server logs for SSE connection events and errors.
- Verify `ENABLE_SSE=true` in environment if SSE seems disabled.

## SSE Event Types

The `/live` endpoint emits the following event types depending on the view:

| Event Type | Views | Description |
|------------|-------|-------------|
| `status` | All | Connection state updates (connected, idle, error, fallback) |
| `metrics` | home | Aggregate counts (total messages, channels, users) |
| `message` | home, messages | New message data with deduplication cursor |
| `channel_count` | channels | Per-channel message count and last activity |
| `user_count` | users | Per-user message count and last seen time |
| `user_profile` | user_profile | Individual user stats (total messages, last seen) |

## Full Test Suite

Run all tests to validate SSE implementation:
```bash
go test ./... -v
```

Expected result: All tests pass including:
- `tests/integration/live_view_integration_test.go` - SSE stream tests for home, messages, user profile
- `tests/contract/channels_test.go` - Channel SSE contract tests
- `tests/integration/search_integration_test.go` - User search SSE tests
