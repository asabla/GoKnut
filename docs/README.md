# GoKnut - Twitch Chat Archiver & Explorer

A Go single-binary service for archiving Twitch chat messages and providing searchable history.

## Quick Links

- **Specification**: [specs/001-spec-reference-spec/spec.md](../specs/001-spec-reference-spec/spec.md)
- **Quickstart Guide**: [specs/001-spec-reference-spec/quickstart.md](../specs/001-spec-reference-spec/quickstart.md)
- **Technical Plan**: [specs/001-spec-reference-spec/plan.md](../specs/001-spec-reference-spec/plan.md)
- **API Contracts**: [specs/001-spec-reference-spec/contracts/api.md](../specs/001-spec-reference-spec/contracts/api.md)

## Features

- Archive Twitch chat messages to SQLite
- Live channel view with real-time updates (<1s latency)
- Search users and messages with filters
- Server-rendered HTMX UI with Tailwind CSS

## Prerequisites

- Go 1.22+
- Twitch credentials (username and OAuth token)

## Running

```bash
export TWITCH_USERNAME=your_username
export TWITCH_OAUTH_TOKEN=oauth:your_token

go run ./cmd/server --db-path=./twitch.db --http-addr=:8080
```

Access the UI at `http://localhost:8080`.

## Testing

```bash
go test ./...
```
