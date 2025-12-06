# Data Model – Twitch Chat Archiver & Explorer

## Entities

### Channel
- id (integer, PK)
- name (string, unique, lowercase)
- display_name (string)
- enabled (boolean)
- retain_history_on_delete (boolean)
- created_at (timestamp)
- updated_at (timestamp)
- last_message_at (timestamp, nullable)
- total_messages (integer, derived/cached)

### User
- id (integer, PK)
- username (string, unique, lowercase)
- display_name (string, optional)
- first_seen_at (timestamp)
- last_seen_at (timestamp)
- total_messages (integer)

### Message
- id (integer, PK)
- channel_id (fk -> channels.id)
- user_id (fk -> users.id)
- text (string)
- sent_at (timestamp)
- tags (json/raw text, optional)

### MessageSearch (optional FTS5)
- content (fts5)
- message_id (UNINDEXED, fk -> messages.id)

## Relationships & Rules

- Channel has many Messages; User has many Messages; Message belongs to one Channel and one User.
- Channel name/usernames normalized to lowercase on ingest; display names preserved as provided.
- Enabling/disabling a channel toggles IRC join/part; deleting may keep history based on retain flag.
- Message inserts batched; sent_at stored from IRC event receipt; last_message_at and totals updated on ingest.
- FTS table maintained on message insert for search; fallback to LIKE queries if FTS disabled.
- Validation: channel name pattern, username non-empty, text length bounded; timestamps in UTC.

## State Transitions

- Channel: created -> enabled/disabled toggles; delete (with optional history retention).
- User: first_seen_at set on first message; last_seen_at updated on each message.
- Message: append-only; no updates; deletion only via retention policy when channel removed and delete history chosen.
