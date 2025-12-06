# Twitch Chat Archiver & Explorer

## Overview

This application connects to Twitch IRC chat, logs messages from multiple channels into a local SQLite database, and exposes a web interface (Golang + HTMX + Tailwind CSS) for browsing, searching, and managing those messages.

The application runs as a single server process that:

- Maintains IRC connections to Twitch channels.
- Persists chat messages and related metadata into SQLite.
- Serves an HTTP interface to view and search stored data.

Target steady-state throughput is approximately 100–150 chat messages per second across all channels combined. New messages should become visible in the web UI within at most 1 second end-to-end.

## Users and Use Cases

### Primary users

1. Streamer / Channel Owner  
2. Moderator / Community Manager  
3. Developer / Power User  

(See detailed use cases below.)

### Core scenarios

#### 1. Manage connected channels

- View a list of channels currently being tracked:
  - Channel name (e.g., `#somechannel`).
  - Connection status (connected, connecting, error, disabled).
  - Basic stats: total messages logged, last message timestamp.
- Add a new channel:
  - Enter channel name.
  - Enable/disable logging for the channel.
- Remove a channel:
  - Option to keep or delete associated messages must be explicit.
- Update a channel:
  - Toggle enabled/disabled.
  - Adjust a display label.
- When a channel is added and enabled:
  - The backend joins the channel on Twitch IRC and begins logging messages.

#### 2. View a live message stream per channel

- Select a channel from the channel list.
- See:
  - A live-updating stream of messages for that channel.
  - Recent history (e.g., last N messages) when the page loads.
- Messages show:
  - Timestamp.
  - Username.
  - Message text.
  - Optional metadata (moderator, subscriber, etc. when available).
- New messages:
  - Are persisted to SQLite.
  - Become visible in the UI with an end-to-end delay of at most 1 second under normal conditions.

#### 3. View historical messages

- Load older messages for a given channel via paging or “Load previous N messages”.
- Messages are shown in reverse chronological order.
- Clear indication when there are no more messages to load.

#### 4. View user profile and cross-channel history

- Search for or click a username.
- User profile shows:
  - Username.
  - First seen timestamp.
  - Last seen timestamp.
  - Total messages logged.
  - Channels in which the user has appeared.
- The profile page includes:
  - A list of messages sent by this user across all channels.
  - Filters: by channel, optional time range.
  - Pagination for large histories.

#### 5. Search for users

- Search UI for user by username (exact or partial).
- Results show:
  - Matching usernames.
  - Per-user stats such as message count and number of distinct channels.
- Clicking a user opens their profile.

#### 6. Search for channels

- Search channels by name or display label (exact or partial).
- Results show:
  - Matching channels.
  - Basic stats: message count, last active time.
- Clicking a channel opens the channel detail view.

#### 7. Search for messages

- Search messages by free-text query.
- Optional filters:
  - Channel.
  - Username.
  - Time range.
- Results:
  - Reverse chronological.
  - Show timestamp, channel, username, and message text.
  - Paginated.
  - Optional highlighting of matching terms.

## Functional Requirements

1. **Twitch IRC connectivity**
   - Connect to Twitch IRC with OAuth token and username.
   - Join multiple channels.
   - Handle PING/PONG and reconnection.
   - Respect Twitch rate limits for JOIN/PART and authentication.

2. **Message ingestion**
   - Parse incoming IRC lines into structured records.
   - For `PRIVMSG` events:
     - Extract channel, username, message text, and tags.
   - Persist each message with:
     - Unique ID.
     - Channel reference.
     - User reference.
     - Message text.
     - Timestamp.
     - Raw tags/metadata (optional but stored).
   - Ingestion must handle approximately 100–150 messages/sec sustained.
   - The ingestion pipeline should be decoupled from network I/O using in-memory queues and a dedicated database writer to avoid blocking the IRC client.

3. **Channel management**
   - Maintain channel configurations in the database.
   - CRUD via the web UI:
     - Create new channels, with enabled flag.
     - Read/list channels.
     - Update (toggle enabled, rename label).
     - Delete channel configuration, with explicit choice to delete or retain logged messages.
   - Changing `enabled`:
     - Triggers join/part operations on the IRC client.

4. **User resolution**
   - Track unique users by username (normalized).
   - Maintain metadata:
     - First seen / last seen timestamps.
     - Message count (optional but recommended).
   - Efficient lookup for profile pages and searches.

5. **Web UI**
   - Implemented using Go `net/http`, HTML templates, HTMX, and Tailwind.
   - Major pages:
     - Dashboard / overview.
     - Channels list.
     - Channel detail (live + history).
     - User search and profile.
     - Message search.
   - Must function with JavaScript disabled except for HTMX behavior (progressive enhancement).

6. **Search**
   - Provide search endpoints for:
     - Users.
     - Channels.
     - Messages.
   - Pagination is required for all result sets.
   - Full-text search may use SQLite FTS (FTS5) for message text.

7. **Configuration**
   - Support environment variables / config file for:
     - Twitch credentials.
     - SQLite path.
     - HTTP bind address/port.
     - Initial channels list (optional).
   - Validate configuration on startup and fail fast with clear errors if invalid.

8. **Resilience**
   - Automatically reconnect to Twitch on transient failures with backoff.
   - Avoid data corruption on crashes or network failures.
   - Log IRC and application errors, including reconnection attempts.

## Non-Functional Requirements

- **Throughput and latency**
  - Steady-state target: 100–150 messages/sec across all channels.
  - Under normal conditions, 95–99% of messages should:
    - Be durably written to SQLite within a few hundred milliseconds.
    - Become visible in the web UI within at most 1 second from arrival.
- **Single binary**
  - Application runs as a single Go binary.
- **Performance**
  - Use an ingestion pipeline and batching where appropriate.
  - Use SQLite WAL mode and prepared statements to reduce write latency.
- **Data integrity**
  - Messages must be durably persisted.
  - Schema migrations must be safe and forward-only.
- **Security**
  - Do not log OAuth tokens or passwords.
  - Implement basic validation and input sanitization on HTTP endpoints.
- **Portability**
  - No external DB; SQLite only.
  - Runs on Linux/macOS/Windows.

## Constraints and Assumptions

- Only public Twitch chat messages are in scope.
- Private messages/whispers and other Twitch APIs are out of scope for the initial version.
- Single-operator deployment (one set of Twitch credentials).
- Expected scale: up to low millions of messages in a single SQLite database file.

## Out of Scope (Initial Version)

- Real-time moderation actions through Twitch API.
- Rich analytics dashboards and visualizations.
- Multi-tenant deployment and multi-user authentication in the UI.
- Horizontal scaling across multiple nodes.

## Success Criteria

- The application can connect to at least 10 channels and ingest 100–150 messages/sec for several hours without manual intervention or significant message loss.
- Messages appear in the web UI within at most 1 second of being received from Twitch under normal operation.
- The user can:
  - Add/remove channels via the UI and see connections update.
  - View live and historical messages per channel.
  - View user profiles with cross-channel history.
  - Search users, channels, and messages with correct and paginated results.
