# Data Model: Live Stream Updates

## Existing Entities (authoritative sources)
- **messages**: `id` (autoincrement), `channel_id`, `user_id`, `text`, `sent_at`, `tags`
- **channels**: `id`, `name`, `display_name`, `total_messages`, `last_message_at`
- **users**: `id`, `username`, `display_name`, `total_messages`, `last_seen_at`
- **messages_fts**: FTS mirror of `messages.text` for search

## Live Stream Envelopes (no schema changes required)
- **MessageEvent**: `{ type: "message", message_id, channel_id, user_id, text, sent_at, cursor }`
- **MetricsEvent**: `{ type: "metrics", total_messages, total_channels, total_users, cursor }`
- **ChannelCountEvent**: `{ type: "channel_count", channel_id, total_messages, last_message_at, cursor }`
- **UserCountEvent**: `{ type: "user_count", user_id, total_messages, last_seen_at, cursor }`
- **UserProfileEvent**: `{ type: "user_profile", user_id, total_messages, last_seen_at, last_message_at?, message_id?, cursor }`
- **StatusEvent**: `{ type: "status", state: connected|idle|reconnecting|fallback|error, reason?, retry_after_ms? }`

All events include `cursor` derived from `messages.id` (monotonic) to support ordering/deduplication and reconnect catch-up.

## Cursors and Ordering
- Use `messages.id` as the primary cursor for ordering and deduplication across streams that deal with messages.
- For count-only updates (channels/users), reuse latest related `messages.id` as cursor to align cross-stream ordering; if unavailable, fall back to timestamp (`last_message_at`/`last_seen_at`).

## Reconnect & Backfill
- Clients send last seen `cursor` on (re)subscribe; server returns backlog in bounded batches.
- If backlog exceeds safe threshold, server instructs client to fallback to polling or hard refresh.

## Storage Impact
- No migrations required; leverage existing tables and triggers that maintain `total_messages`, `last_message_at`, and `last_seen_at`.

## Open Questions
- Backfill batch size limit? (proposal needed)
- Should channel/user count streams include top-N filtering server-side to limit payload?
