# GoKnut Development Guidelines

Auto-generated from all feature plans. Last updated: 2025-12-06

## Active Technologies
- Go 1.22 (single binary) + Go stdlib (`net`, `database/sql`, `net/http`), `modernc.org/sqlite`, `html/template`, HTMX, Tailwind CSS (already present) (002-message-search-view)
- SQLite (WAL) with `messages` table + FTS5 virtual table for message conten (002-message-search-view)
- Go 1.22 (single binary) + Go stdlib (`net/http`, `html/template`), HTMX (server-rendered), SSE event streams; `modernc.org/sqlite` (003-live-stream-updates)
- SQLite (WAL) using existing tables and triggers; no new migrations (003-live-stream-updates)
- Go 1.22 (single binary) + Go stdlib (`net/http`, `html/template`), HTMX (server-rendered); SQLite (`modernc.org/sqlite`) and optional Postgres (`lib/pq`) (004-profiles-orgs-collabs)
- SQLite (WAL) for local/dev; Postgres optional (existing migrations) (004-profiles-orgs-collabs)

- Go 1.22 (single binary) + Go stdlib (`net`, `database/sql`, `net/http`), `modernc.org/sqlite` driver, HTMX for progressive enhancement, Tailwind CSS for styling templates (001-spec-reference-spec)

## Project Structure

```text
backend/
frontend/
tests/
```

## Commands

# Add commands for Go 1.22 (single binary)

## Code Style

Go 1.22 (single binary): Follow standard conventions

## Recent Changes
- 004-profiles-orgs-collabs: Added Go 1.22 (single binary) + Go stdlib (`net/http`, `html/template`), HTMX (server-rendered); SQLite (`modernc.org/sqlite`) and optional Postgres (`lib/pq`)
- 003-live-stream-updates: Added Go 1.22 (single binary) + Go stdlib (`net/http`, `html/template`), HTMX (server-rendered), SSE event streams; `modernc.org/sqlite`
- 002-message-search-view: Added Go 1.22 (single binary) + Go stdlib (`net`, `database/sql`, `net/http`), `modernc.org/sqlite`, `html/template`, HTMX, Tailwind CSS (already present)


<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
