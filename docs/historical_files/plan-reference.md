# Technical Plan – Twitch Chat Archiver & Explorer

## Architecture Overview

The application is a single Go binary with these main components:

1. Configuration loader.
2. Twitch IRC client and message ingestion pipeline.
3. SQLite persistence layer with WAL and batching.
4. Domain services (channels, users, messages/search).
5. HTTP server and HTMX-enabled UI.

The design must support approximately 100–150 messages/sec ingestion and expose ingested messages to the UI within ≤1s.

## Components

### 1. Configuration

- Load from environment variables and optionally a config file.
- Key values:
  - `TWITCH_USERNAME`
  - `TWITCH_OAUTH_TOKEN`
  - `TWITCH_CHANNELS` (optional, comma-separated)
  - `DB_PATH` (e.g., `./twitch.db`)
  - `HTTP_ADDR` (e.g., `:8080`)
- Provide a typed `Config` struct.
- Validate:
  - Credentials present.
  - DB path writable.
  - HTTP address valid.

### 2. SQLite Persistence

#### Connection and pragmas

- Use a single shared `*sql.DB` with connection pooling.
- On startup:
  - `PRAGMA journal_mode=WAL;`
  - `PRAGMA synchronous=NORMAL;` (or configurable, `FULL` if durability is more important than throughput).
  - `PRAGMA foreign_keys=ON;`
- Keep max open connections small but non-zero (e.g. 10) to separate readers from the single writer goroutine.

#### Schema

Tables:

- `channels`  
- `users`  
- `messages`  
- Optional `message_search` FTS5 table.

(See spec for fields.)

#### Repositories

Implement repository interfaces:

- `ChannelRepository`
  - `GetByID`, `GetByName`, `List`, `Create`, `Update`, `Delete`, `ListEnabled`.
- `UserRepository`
  - `GetByID`, `GetByUsername`, `Upsert`, `Search`.
- `MessageRepository`
  - `InsertMany` (batched insert).
  - `ListByChannel` (with pagination and “since last ID” mode).
  - `ListByUser` (with pagination).
  - `SearchMessages` (with filters and pagination).

All write operations use prepared statements and transactions where appropriate.

### 3. Twitch IRC Client and Ingestion Pipeline

#### IRCClient

- Connect to `irc.chat.twitch.tv:6697` using TLS.
- Authenticate with `PASS oauth:<token>` and `NICK <username>`.
- On startup:
  - Load enabled channels.
  - JOIN each channel (`#name`).
- Maintain:
  - A reader loop that:
    - Reads lines from the connection.
    - Parses them into IRC message structs.
    - Sends chat events to an internal Go channel.
  - A small write handler for JOIN/PART, PONG replies, etc.

#### Event types

Define a simple event model:

- `ChatMessageEvent`:
  - `ChannelName`
  - `Username`
  - `Text`
  - `Tags` (raw map)
  - `ReceivedAt` (time)
- `ConnectionEvent`:
  - Type (connected, disconnected, error, joined, parted).
  - Context info.

#### Ingestion worker

- Use a buffered Go channel (capacity e.g. 10,000) for `ChatMessageEvent`.
- Dedicated ingestion goroutine(s):

  - Validate and normalize:
    - Normalize channel name to lowercase without `#`.
    - Normalize username to lowercase.
  - Upsert channel and user:
    - Use in-memory caches for IDs to avoid repeated DB lookups on the hot path:
      - `map[string]int64` for `channelName -> channelID`.
      - `map[string]int64` for `username -> userID`.
    - Cache miss:
      - Lookup and create as needed via repositories.
  - Append the message to an in-memory buffer:
    - A slice of `MessageRecord` ready for bulk insert.

- Batching strategy:
  - Accumulate messages in a batch until either:
    - Batch size reaches N (e.g. 100–200 messages), or
    - A timeout occurs (e.g. 50–100 ms).
  - On flush:
    - Begin transaction.
    - Insert all messages via `InsertMany`.
    - Commit transaction.
  - This keeps fsync frequency low while keeping end-to-end latency well below 1s.

- Error handling:
  - Log any failed inserts.
  - Do not block further ingestion on single message failures.
  - If the database is unavailable for extended time, apply backpressure (the buffered channel will fill) and log accordingly.

### 4. Domain Services

#### ChannelService

- Wraps `ChannelRepository` and `IRCClient`.
- Responsibilities:
  - Add channel:
    - Create DB record.
    - Join channel if enabled.
  - Update channel:
    - Update DB.
    - If `enabled` flag changes:
      - Join or part the channel through the IRC client.
  - Delete channel:
    - Update DB and optionally delete messages.
    - Part channel if currently joined.

#### UserService

- Wraps `UserRepository`.
- Provides:
  - Lookup and upsert semantics.
  - Helpers to update `first_seen_at`, `last_seen_at`, and message counts.

#### SearchService

- Wraps `MessageRepository`, `UserRepository`, and `ChannelRepository`.
- Provides:
  - User search by username (LIKE / FTS).
  - Channel search by name/display name.
  - Message search:
    - Plain `LIKE` or FTS5-based `MATCH`.
    - Filters for channel, user, date range.

### 5. HTTP / UI Layer

#### Routing

Example routes:

- `GET /`  
  Dashboard with high-level stats and quick links.

- Channel management:
  - `GET /channels`
  - `POST /channels` (create)
  - `POST /channels/{id}` (update)
  - `POST /channels/{id}/delete` (delete)

- Channel view:
  - `GET /channels/{id}`
    - Renders the page with latest messages.
  - `GET /channels/{id}/messages`
    - Returns message list fragment (HTML) for:
      - Initial load (latest N).
      - “Load more” pagination (via `before_id` or `page`).
  - `GET /channels/{id}/messages/stream`
    - HTMX polling endpoint returning “new messages since last ID” as an HTML fragment.

- Users:
  - `GET /users` (search form + results).
  - `GET /users/{id}` (profile with messages).

- Search:
  - `GET /search/messages` (form + results).
  - Optionally:
    - `GET /search/channels`
    - `GET /search/users` if separated from `/users`.

#### Real-time behavior (≤1s)

- Use HTMX polling on the channel view:
  - The channel page keeps the ID of the newest rendered message.
  - HTMX attribute example (conceptually):
    - `hx-get="/channels/{id}/messages/stream?after_id=XYZ"`
    - `hx-trigger="every 500ms"` (polling interval ~0.5–1s).
    - `hx-swap="afterend"` or similar to append new messages.
- Server logic:
  - Query messages where `id > after_id` and `channel_id = ...`.
  - Return HTML fragment with new messages.
- Combined with the ingestion batching strategy (≤100ms flush), this satisfies the ≤1s requirement in practice.

#### Templates and styling

- Use `html/template` for layout and partials.
- Tailwind CSS:
  - Compile at build time into `app.css`.
  - Use a simple responsive layout:
    - Navigation bar.
    - Left sidebar (optional) for channels.
    - Main content for views.

### 6. Search Implementation Details

#### User search

- SQL example:
  - `SELECT ... FROM users WHERE username LIKE ? ORDER BY message_count DESC LIMIT ? OFFSET ?`
- Normalize search input to lowercase.

#### Channel search

- SQL example:
  - `SELECT ... FROM channels WHERE name LIKE ? OR display_name LIKE ? ORDER BY last_message_at DESC NULLS LAST LIMIT ? OFFSET ?`

#### Message search

- Option 1 (initial, simpler):
  - `SELECT ... FROM messages WHERE text LIKE ? [AND channel_id = ?] [AND user_id = ?] [AND sent_at BETWEEN ? AND ?] ORDER BY sent_at DESC LIMIT ? OFFSET ?`
- Option 2 (FTS5):
  - FTS table `message_search(content, message_id UNINDEXED)`.
  - `SELECT m.* FROM message_search ms JOIN messages m ON ms.message_id = m.id WHERE ms MATCH ? [filters...] ORDER BY m.sent_at DESC LIMIT ? OFFSET ?`
- All search endpoints take `page` and `page_size` query parameters with sane defaults and upper bounds.

### 7. Configuration and Deployment

#### Configuration

- Implement `LoadConfig()`:
  - Reads environment variables and/or config file.
  - Validates mandatory fields.
- Optional CLI flags:
  - `--config=path`
  - `--db-path=...`
  - `--http-addr=...`

#### Deployment

- Development:
  - `go run ./cmd/server`.
- Production:
  - Build static binary.
  - Run under systemd or similar.
  - Optionally put behind a reverse proxy for TLS and auth.

### 8. Logging and Observability

- Structured logging with fields:
  - Component (`irc`, `db`, `http`, `ingest`).
  - Level (`INFO`, `WARN`, `ERROR`).
  - Context (channel name, user, etc.).
- Log:
  - IRC connection lifecycle and reconnect attempts.
  - Join/part operations.
  - Database errors.
  - HTTP 5xx errors.

- Optional metrics (later iteration):
  - Ingested messages/sec.
  - Per-channel message rate.
  - P99 insert latency.

### 9. Error Handling and Edge Cases

- IRC disconnects:
  - Automatic reconnect with exponential backoff.
  - Re-join enabled channels upon reconnect.
- Database unavailable:
  - Log error and attempt periodic reconnect.
  - If ingestion channel is full, log backpressure warning.
- Invalid user input:
  - Validate channel names (simple pattern).
  - Validate pagination parameters (bounded).
- Graceful shutdown:
  - On SIGINT/SIGTERM:
    - Stop HTTP accepting new requests.
    - Stop IRC client and ingestion worker.
    - Flush any remaining batched messages.
    - Close DB.

### 10. Iteration Plan

1. **Bootstrap**
   - Config loading.
   - Basic HTTP server with a placeholder page.
   - SQLite connection and migrations.

2. **IRC Client and Simple Ingestion**
   - Connect to Twitch.
   - Join a hard-coded channel.
   - Parse and log messages to stdout.

3. **SQLite-backed Ingestion (No UI)**
   - Implement repositories.
   - Implement ingestion worker with batching and WAL.
   - Confirm throughput (load tests or simple simulated inputs).

4. **Channel and User Models**
   - Implement `ChannelService` and `UserService`.
   - Replace hard-coded channel with DB-driven channels.

5. **Basic UI**
   - Dashboard with channel list.
   - Channel detail view with last N messages (no polling).

6. **Real-time Updates**
   - HTMX polling endpoint for new messages.
   - Ensure end-to-end latency under expected load stays ≤1s.

7. **User Profiles**
   - Implement user search and profile views.
   - Cross-channel message listing.

8. **Search**
   - Message search with filters and pagination.
   - Optionally enable FTS5 for better performance.

9. **Polish and Hardening**
   - Input validation, error pages.
   - Logging and basic metrics.
   - Configuration validation and documentation.

