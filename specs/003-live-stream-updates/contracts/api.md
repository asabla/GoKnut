# API Contracts: Live Stream Updates

## Scope
- Define WebSocket-based live updates with graceful fallback to existing HTMX polling endpoints.
- Views covered: home metrics/messages, `/messages`, `/channels`, `/users`, `/users/{username}`.

## Transport
- **Primary**: WebSocket (`nhooyr.io/websocket` server). Client connects via `GET /live` with query parameters indicating view and cursor.
- **Fallback**: Existing HTMX polling endpoints remain available; clients may switch to polling if WebSocket handshake fails or disconnect persists.

### Endpoint
- `GET /live?view=<view>&after_id=<cursor>&channel=<name>&user=<username>`
  - `view`: `home|messages|channels|users|user_profile`
  - `after_id` (optional): last seen message id for catch-up; default 0
  - `channel` (optional): for channel-scoped streams
  - `user` (optional): for user/profile-scoped streams
  - Response: WebSocket upgrade or HTTP 400/503 with JSON error body `{ "error": "..." }` when upgrade refused.

### Message Envelopes (server → client)
- Common fields: `type`, `cursor` (messages.id), `sent_at` when relevant.
- `metrics`: `{ "type": "metrics", "cursor": 123, "total_messages": 1000, "total_channels": 42, "total_users": 17 }
- `message`: `{ "type": "message", "cursor": 124, "id": 124, "channel_id": 1, "user_id": 2, "text": "...", "sent_at": "2025-12-07T12:00:00Z" }
- `channel_count`: `{ "type": "channel_count", "cursor": 125, "channel_id": 1, "total_messages": 250, "last_message_at": "2025-12-07T12:00:01Z" }
- `user_count`: `{ "type": "user_count", "cursor": 126, "user_id": 2, "total_messages": 75, "last_seen_at": "2025-12-07T12:00:02Z" }
- `user_profile`: `{ "type": "user_profile", "cursor": 127, "user_id": 2, "total_messages": 75, "last_seen_at": "2025-12-07T12:00:02Z", "last_message_at": "2025-12-07T12:00:01Z", "message_id": 124 }
- `status`: `{ "type": "status", "state": "connected|idle|reconnecting|fallback|error", "reason": "...", "retry_after_ms": 1000 }
- `error`: `{ "type": "error", "message": "...", "retry_after_ms?": 1000 }`

### Client → Server (optional)
- On subscribe/reconnect: implicit via query string `after_id`.
- No other client messages required for MVP; future extension could allow acks or pings.

## Error Handling
- Upgrade failures: HTTP 400 for bad params, 503 for capacity/backpressure; body `{ "error": "..." }`.
- Stream errors: send `status` with `state="error"` and close; client should fallback or retry.
- Capacity/backpressure: send `status` with `state="fallback"` and `reason`, close connection.

## Backoff & Reconnect Expectations
- Clients retry with exponential backoff starting at ~1s, capped at ~30s; show status during retry.
- On reconnect, include `after_id` to request backlog; if backlog too large, server may reply with `status=fallback`.

## Security / Access Control
- No auth currently; endpoints are public. If auth is added later, upgrade must validate session before accepting.

## Compatibility
- Existing polling endpoints remain unchanged to preserve current HTMX flows when WebSocket is disabled or blocked.
