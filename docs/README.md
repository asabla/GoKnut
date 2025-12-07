# GoKnut - Twitch Chat Archiver & Explorer

A Go single-binary service for archiving Twitch chat messages and providing searchable history.

## Quick Links

- **Specification**: [specs/001-spec-reference-spec/spec.md](../specs/001-spec-reference-spec/spec.md)
- **Quickstart Guide**: [specs/001-spec-reference-spec/quickstart.md](../specs/001-spec-reference-spec/quickstart.md)
- **Technical Plan**: [specs/001-spec-reference-spec/plan.md](../specs/001-spec-reference-spec/plan.md)
- **API Contracts**: [specs/001-spec-reference-spec/contracts/api.md](../specs/001-spec-reference-spec/contracts/api.md)
- **Performance Validation**: [perf.md](./perf.md)

## Features

- Archive Twitch chat messages to SQLite
- Live channel view with real-time updates (<1s latency)
- Search users and messages with filters
- Server-rendered HTMX UI with Tailwind CSS

## Prerequisites

- Go 1.22+
- Twitch credentials (username and OAuth token) OR anonymous mode for read-only access

## Setup

### Authentication Modes

GoKnut supports two authentication modes for connecting to Twitch IRC:

#### Authenticated Mode (Full Access)
Uses your Twitch username and OAuth token. Provides full access including:
- Reading chat from all public channels
- Joining subscriber-only or verified-only channels (if your account has access)
- More stable connection with lower disconnect rates

```bash
export TWITCH_AUTH_MODE=authenticated  # Optional, this is the default
export TWITCH_USERNAME=your_username
export TWITCH_OAUTH_TOKEN=oauth:your_token
```

#### Anonymous Mode (Read-Only)
Uses a "justinfan" anonymous login. Limitations:
- **Read-only**: Cannot send messages
- **Public channels only**: No access to subscriber-only or verified-only channels
- **Higher disconnect risk**: Anonymous connections may be rate-limited more aggressively
- **Best for**: Testing, read-only monitoring of public channels

```bash
export TWITCH_AUTH_MODE=anonymous
# No username or token required - a justinfan nick is auto-generated
```

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `TWITCH_AUTH_MODE` | No | `authenticated` | Auth mode: `authenticated` or `anonymous` |
| `TWITCH_USERNAME` | Auth mode | - | Your Twitch username (required for authenticated) |
| `TWITCH_OAUTH_TOKEN` | Auth mode | - | OAuth token (format: `oauth:xxxxx`, required for authenticated) |
| `TWITCH_CHANNELS` | No | - | Comma-separated list of channels to auto-join |
| `DB_PATH` | No | `./twitch.db` | Path to SQLite database file |
| `HTTP_ADDR` | No | `:8080` | HTTP server listen address |
| `BATCH_SIZE` | No | `100` | Message batch size for ingestion |
| `FLUSH_TIMEOUT` | No | `100` | Batch flush timeout in milliseconds |
| `ENABLE_FTS` | No | `true` | Enable FTS5 full-text search |

### Command Line Flags

```bash
--db-path       Path to SQLite database file (default: ./twitch.db)
--http-addr     HTTP server listen address (default: :8080)
--batch-size    Message batch size for ingestion (default: 100)
--flush-timeout Batch flush timeout in milliseconds (default: 100)
--enable-fts    Enable FTS5 full-text search (default: true)
```

## Running

### Using Makefile (recommended)

```bash
# Build binary to bin/goknut
make build

# Run directly without building
make run

# Run tests
make test

# Cross-compile artifact to dist/goknut-<os>-<arch>
make publish
```

Access the UI at `http://localhost:8080`.

### Authenticated Mode (Recommended)

```bash
export TWITCH_USERNAME=your_username
export TWITCH_OAUTH_TOKEN=oauth:your_token

go run ./cmd/server --db-path=./twitch.db --http-addr=:8080
```

### Anonymous Mode (Read-Only, No Credentials)

```bash
export TWITCH_AUTH_MODE=anonymous

go run ./cmd/server --db-path=./twitch.db --http-addr=:8080
```

Access the UI at `http://localhost:8080`.

## Latency Budgets

The application is designed to meet the following performance targets:

| Component | Target | Notes |
|-----------|--------|-------|
| Ingestion throughput | 100-150 msgs/sec | Batched writes with WAL mode |
| Live UI updates | < 1s | HTMX polling at 500-1000ms intervals |
| HTTP p95 latency | < 250ms | All HTTP endpoints |
| HTTP p99 latency | < 500ms | All HTTP endpoints |
| Batch flush | < 100ms | Message batching timeout |

### Ingestion Pipeline

- Messages are batched (default: 100 messages or 100ms timeout)
- SQLite WAL mode reduces writer contention
- In-memory caches for channel/user ID lookups

### Live View Streaming

- HTMX polling at configurable intervals (500-1000ms)
- Messages fetched by `after_id` for efficient delta updates
- Server-rendered HTML fragments minimize client processing

## Observability

### Structured Logging

All logs are JSON-formatted with component and subsystem tags:

```json
{
  "time": "2025-12-06T10:00:00Z",
  "level": "INFO",
  "msg": "stored message batch",
  "component": "ingestion",
  "subsystem": "ingestion",
  "count": 50,
  "latency_ms": 12,
  "dropped": 0
}
```

### Subsystems

| Subsystem | Description |
|-----------|-------------|
| `irc` | IRC connection events, channel joins/parts |
| `ingestion` | Message batching, storage, cache operations |
| `search` | Search queries, FTS operations |
| `http` | HTTP request handling |

### Metrics

The application tracks internal metrics accessible via `Metrics.Stats()`:

| Metric | Description |
|--------|-------------|
| `IRCConnections` | Total IRC connection attempts |
| `IRCDisconnections` | Total IRC disconnections |
| `IRCMessagesRecv` | Total messages received from IRC |
| `BatchesProcessed` | Total batches stored to database |
| `MessagesIngested` | Total messages stored |
| `DroppedMessages` | Messages dropped (validation failures) |
| `AvgBatchLatency` | Average batch processing time |
| `SearchQueries` | Total search queries executed |
| `AvgSearchLatency` | Average search query time |
| `HTTPRequests` | Total HTTP requests |
| `AvgHTTPLatency` | Average HTTP request time |
| `StreamPollRequests` | Total stream poll requests |
| `AvgStreamLatency` | Average stream poll time |

## Testing

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific test suites
go test ./tests/unit/...
go test ./tests/integration/...
go test ./tests/contract/...

# Run with coverage
go test -cover ./...
```

## Architecture

```
internal/
├── config/         Configuration loading and validation
├── http/           HTTP server, handlers, and templates
├── ingestion/      Message batching and storage pipeline
├── irc/            Twitch IRC client with reconnection
├── observability/  Structured logging and metrics
├── repository/     SQLite repositories and migrations
├── search/         Full-text search (FTS5/LIKE)
└── services/       Business logic (channel, search)
```
