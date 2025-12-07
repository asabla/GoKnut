# Quickstart: Live Stream Updates

## Prerequisites
- Go 1.22 toolchain
- SQLite available (WAL enabled by default in repo setup)
- Web browser with HTMX support; WebSocket allowed

## Run the server
```bash
make run
# or
go run ./cmd/server
```

Server starts on the configured port (see existing config). No extra env needed for live mode.

## Connect to live stream
- Primary transport: WebSocket at `/live`.
- Example (messages view):
  - URL: `ws://localhost:8080/live?view=messages&after_id=0`
  - Receives JSON envelopes:
    - `metrics`, `message`, `channel_count`, `user_count`, `user_profile`, `status`
- Example (home view): `ws://localhost:8080/live?view=home&after_id=0`

### Using `wscat` (optional)
```bash
npx wscat -c "ws://localhost:8080/live?view=messages&after_id=0"
```

### HTMX integration
- Templates will use `ws:` to bind fragments; when WebSocket unavailable, existing polling endpoints continue to function.

## Reconnect & fallback behavior
- Client retries with backoff (start ~1s, cap ~30s).
- On reconnect, client passes `after_id` to catch up.
- If server responds with `status=fallback`, client should switch to polling.

## Validation steps
1. Start server.
2. Open `http://localhost:8080/` (home) and `/messages` in browser.
3. Ingest or create new messages (existing ingestion pipeline).
4. Observe live updates (metrics/messages) without reload; if WebSocket is blocked, polling continues.

## Troubleshooting
- If WebSocket upgrade fails: check port, proxies, or CSP blocking WS; use polling as fallback.
- If backlog is large and server closes connection with `fallback`: reload page to continue via polling.
