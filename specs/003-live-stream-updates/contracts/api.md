# API Contracts: Live Stream Updates

## Scope
- Define Server-Sent Events (SSE) based live updates with graceful fallback to existing HTMX polling endpoints.
- Views covered: home metrics/messages, `/messages`, `/channels`, `/users`, `/users/{username}`.

## Transport
- **Primary**: Server-Sent Events (SSE) using Go's `net/http` with `text/event-stream` content type.
- **Fallback**: Existing HTMX polling endpoints remain available; clients may switch to polling if SSE connection fails or is unavailable.

### Endpoint
- `GET /live?view=<view>&after_id=<cursor>&channel=<name>&user=<username>`
  - `view`: `home|messages|channels|users|user_profile` (default: `home`)
  - `after_id` (optional): last seen message id for catch-up; default 0
  - `channel` (optional): for channel-scoped streams
  - `user` (optional): for user/profile-scoped streams (required for `user_profile` view)
  - Response: SSE stream (`text/event-stream`) or HTTP 400/503 with JSON error body `{ "error": "..." }` when connection refused.

### SSE Event Format
Events follow the standard SSE format:
```
data: {"type":"<type>","cursor":<id>,...}\n\n
```

### Message Envelopes (server → client)
- Common fields: `type`, `cursor` (messages.id for ordering/deduplication).

**metrics** (home view):
```json
{"type":"metrics","cursor":123,"total_messages":1000,"total_channels":42,"total_users":17}
```

**message** (new message event):
```json
{"type":"message","cursor":124,"id":124,"channel_id":1,"channel_name":"example","user_id":2,"username":"user","display_name":"User","text":"Hello world","sent_at":"2025-12-07T12:00:00Z"}
```

**channel_count** (channel message count update):
```json
{"type":"channel_count","cursor":125,"channel_id":1,"channel_name":"example","total_messages":250,"last_message_at":"2025-12-07T12:00:01Z"}
```

**user_count** (user message count update):
```json
{"type":"user_count","cursor":126,"user_id":2,"username":"user","total_messages":75,"last_seen_at":"2025-12-07T12:00:02Z"}
```

**user_profile** (user profile update):
```json
{"type":"user_profile","cursor":127,"user_id":2,"username":"user","total_messages":75,"last_seen_at":"2025-12-07T12:00:02Z","last_message_at":"2025-12-07T12:00:01Z","message_id":124}
```

**status** (connection status):
```json
{"type":"status","cursor":0,"state":"connected|idle|reconnecting|fallback|error","reason":"optional message","retry_after_ms":1000}
```

### Client → Server
- On subscribe/reconnect: implicit via query string `after_id`.
- No other client messages required for SSE (unidirectional protocol).

## Error Handling
- Connection failures: HTTP 400 for bad params, 503 for capacity/backpressure; body `{ "error": "..." }`.
- Stream errors: send `status` with `state="error"` and close; client should fallback or retry.
- Capacity/backpressure: send `status` with `state="fallback"` and `reason`, close connection.

## Backoff & Reconnect Expectations
- Clients retry with exponential backoff starting at ~1s, capped at ~30s; show status during retry.
- On reconnect, include `after_id` to request backlog; if backlog too large (≥500 events), server may reply with `status=fallback`.

## Heartbeat
- Server sends heartbeat comments every 30 seconds to keep connection alive:
  ```
  : heartbeat 1733577600
  ```

## Security / Access Control
- No auth currently; endpoints are public. If auth is added later, SSE endpoint must validate session before accepting.

## Compatibility
- Existing polling endpoints remain unchanged to preserve current HTMX flows when SSE is disabled or blocked.
- SSE can be disabled via `ENABLE_SSE=false` environment variable.

## Configuration
- `SSEMaxBackfill`: 500 events maximum for reconnect catch-up
- `SSEHeartbeatPeriod`: 30 seconds between heartbeats
- `SSEWriteTimeout`: 10 seconds for write operations
- `SSEBufferSize`: 100 events per-connection buffer
