# GoKnut

![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?style=flat&logo=go)
![Build](https://img.shields.io/badge/build-passing-brightgreen?style=flat)

A self-hosted Twitch Chat Archiver & Explorer — archive chat messages in real-time and explore them through a web interface.

## Features

- **Live Chat Streaming** — Real-time chat view with <1s latency via Server-Sent Events
- **Full-Text Search** — Search messages and users with filters (channel, user, time range) using SQLite FTS5
- **Channel Management** — Add, enable/disable, and manage tracked Twitch channels
- **User Activity Tracking** — View user profiles with message history and activity stats
- **Server-Rendered UI** — Fast, lightweight interface built with HTMX and Tailwind CSS

## Quick Start

**Prerequisites:** Go 1.22+

```bash
# Build
make build

# Run (anonymous mode - no credentials required)
export TWITCH_AUTH_MODE=anonymous
./bin/goknut --db-path=./twitch.db --http-addr=:8080
```

Open [http://localhost:8080](http://localhost:8080) in your browser.

## Documentation

See [docs/README.md](docs/README.md) for detailed documentation including:

- Configuration options (environment variables and flags)
- Architecture overview
- Performance targets
- Running with authentication
- Docker deployment with observability stack

## Development

```bash
make build    # Build binary to bin/goknut
make run      # Build and run
make test     # Run all tests
make clean    # Clean build artifacts
```
