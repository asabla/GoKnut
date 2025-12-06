# Quickstart – Twitch Chat Archiver & Explorer

## Prerequisites
- Go 1.22+
- `modernc.org/sqlite` driver vendored or go mod download
- Twitch credentials: `TWITCH_USERNAME`, `TWITCH_OAUTH_TOKEN`

## Setup
1. `go run ./cmd/server --db-path=./twitch.db --http-addr=:8080`
2. Provide env vars for Twitch credentials; optional `TWITCH_CHANNELS` comma-separated.
3. Access UI at `http://localhost:8080`.

## Features to Validate
- Add/enable/disable/delete channels; verify join/part behavior.
- Live channel view updates within 1s; load earlier messages via pagination.
- Search users/messages with filters; user profile shows cross-channel history.

## Testing
- `go test ./...` (unit/integration/contract)
- Validate ingestion throughput ~100–150 msgs/sec with WAL and batching.

## Observability
- Structured logs for IRC/connectivity/ingestion/search.
- Metrics hooks for throughput/latency/error rates (add to server instrumentation).
