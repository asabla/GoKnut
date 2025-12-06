# API Contracts – Twitch Chat Archiver & Explorer

## Channels

### List channels
- `GET /channels`
- Response: 200 JSON or HTML fragment
  - channels: [ { id, name, display_name, enabled, last_message_at, total_messages } ]

### Create channel
- `POST /channels`
- Body: `name`, `display_name?`, `enabled` (bool), `retain_history_on_delete` (bool)
- Response: 201 with channel summary; joins channel if enabled

### Update channel
- `POST /channels/{id}`
- Body: `display_name?`, `enabled`, `retain_history_on_delete`
- Response: 200 with updated channel; join/part based on enabled

### Delete channel
- `POST /channels/{id}/delete`
- Body: `retain_history` (bool)
- Response: 200; leaves channel; deletes messages if retain_history=false

## Messages (per channel)

### Channel view (page + stream)
- `GET /channels/{id}` -> HTML page recent messages
- `GET /channels/{id}/messages` -> HTML fragment; query: `page`/`page_size` or `before_id`
- `GET /channels/{id}/messages/stream` -> HTML fragment; query: `after_id`
- Responses: 200 with rendered list items (timestamp, username, text)

## Users & Search

### Search users
- `GET /users` -> HTML form + results
- Query: `q` (username fragment), `page`, `page_size`
- Response: 200 list with counts and channel tally

### User profile
- `GET /users/{id}`
- Response: 200 page with first_seen_at, last_seen_at, totals, paginated messages (filters: channel, time range)

### Search messages
- `GET /search/messages`
- Query: `q`, `channel_id?`, `user_id?`, `start?`, `end?`, `page`, `page_size`
- Response: 200 list; reverse chronological; highlight terms when FTS enabled

## Config & Health

### Health
- `GET /healthz`
- Response: 200 status

### Config validation (startup)
- Load env/flags; validate Twitch creds, DB path writable, HTTP addr format; fail-fast with errors.
