# Data Model: Message Search View

## Entities

### Message
- **Fields**: `id`, `channel_id`, `user_id`, `text`, `sent_at`, `tags` (optional), `ingested_at` (implicit via `sent_at` + trigger timestamps), `channel_name` (via join), `username`, `display_name`.
- **Relationships**: Belongs to `Channel`; belongs to `User`.
- **Indexes/FTS**: `messages` has indices on `channel_id`, `user_id`, `sent_at`, and FTS5 virtual table `messages_fts` for `text`.

### Channel
- **Fields**: `id`, `name`, `display_name`, `enabled`, `retain_history_on_delete`, `created_at`, `updated_at`, `last_message_at`, `total_messages`.
- **Relationships**: Has many `Message` records.
- **Indexes**: `enabled`, `name`, `name` uniqueness.

### User
- **Fields**: `id`, `username`, `display_name`, `first_seen_at`, `last_seen_at`, `total_messages`.
- **Relationships**: Has many `Message` records.
- **Indexes**: `username` (unique), `last_seen_at`.

## Validation Rules

- **Search query**: Minimum length 2 characters; trim whitespace; reject/flag empty queries.
- **Filters**: Optional `channel_id`, `user_id`; parse integers safely, ignore invalid values rather than crashing.
- **Time range**: `start` and `end` parsed as `YYYY-MM-DD`; reject if `end` precedes `start`; expand `end` to end-of-day for inclusive range.
- **Pagination**: Default reasonable page/page_size (e.g., page>=1, page_size bounded to prevent large payloads, defaults reused from service).

## State Transitions

- Messages are immutable after ingestion for search purposes; FTS triggers keep `messages_fts` in sync on insert/update/delete.
- Channel/user counters update via triggers on message insert.

## Derived Data & Querying

- Message search uses FTS5 content with optional filters on channel/user/time; results ordered by ingestion timestamp descending by default.
- Highlighted snippets provided by repository for display in templates.

## Notes

- No new tables required; reuse existing schema and FTS.
- Ensure search results include navigation context (channel link, message timestamp) to satisfy FR-009.
