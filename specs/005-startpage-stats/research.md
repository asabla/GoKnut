# Phase 0 Research: Statistics-Centric Startpage

## Existing Behavior

### Template parsing for dashboard templates

- Server template parsing uses `ParseFS(tmplFS, "*.html", "*/*.html")` from `internal/http/server.go`.
- This already includes subdirectories under `internal/http/templates/` (one level deep), so `internal/http/templates/dashboard/*.html` will be parsed automatically.


### Baseline test run

Command: `go test ./...`

Output:

```text
?   	github.com/asabla/goknut/cmd/server	[no test files]
?   	github.com/asabla/goknut/internal/config	[no test files]
?   	github.com/asabla/goknut/internal/http	[no test files]
?   	github.com/asabla/goknut/internal/http/dto	[no test files]
?   	github.com/asabla/goknut/internal/http/handlers	[no test files]
?   	github.com/asabla/goknut/internal/ingestion	[no test files]
?   	github.com/asabla/goknut/internal/irc	[no test files]
?   	github.com/asabla/goknut/internal/observability	[no test files]
?   	github.com/asabla/goknut/internal/repository	[no test files]
?   	github.com/asabla/goknut/internal/search	[no test files]
?   	github.com/asabla/goknut/internal/services	[no test files]
ok  	github.com/asabla/goknut/tests/contract	(cached)
ok  	github.com/asabla/goknut/tests/integration	(cached)
?   	github.com/asabla/goknut/tests/integration/fakes	[no test files]
ok  	github.com/asabla/goknut/tests/unit	(cached)
```

### Start page behavior (current)

- The start page template (`internal/http/templates/home.html`) currently includes:
  - KPI tiles for totals + ids: `#stat-messages`, `#stat-channels`, `#stat-enabled`, `#stat-users`
  - Shortcut links (`/channels`, `/users`, `/messages`)
  - A "Latest Messages" list
  - A home-scoped SSE client (`/live?view=home`) that updates KPI values and prepends messages
- The home handler (`internal/http/server.go`):
  - Queries DB repositories for totals
  - Queries recent messages via `MessageRepository.GetRecentGlobal(ctx, 20)`
  - Renders `home` template

## Constraints (from spec)

- Must use the current stack (server-rendered templates + HTMX); no new infrastructure.
- Must remove the live message feed from the start page.
- Must refresh stats/diagrams automatically (at least every 60 seconds).
- Prefer existing monitoring/telemetry as the source of truth when available; use DB aggregates where that is the source of truth.

## Candidate Statistics Sources

### Database (source-of-truth totals)

- Message total: `MessageRepository.GetTotalCount(ctx)`
- Channel totals/active: `ChannelRepository.GetCount(ctx)`, `ChannelRepository.GetEnabledCount(ctx)`
- User total: `UserRepository.GetCount(ctx)`

These are already used for the home page. They are stable, easy to query, and aligned with the spec’s “use DB when it makes sense”.

### OTel / Prometheus metrics (operational trends)

The repository already defines OTel counters/histograms and exposes them via `/metrics` when OTel is enabled (`internal/observability/otel.go`). Relevant candidates:

- Activity diagram: `goknut.ingestion.messages_ingested` (counter, visualize as rate over time)
- Reliability diagram: `goknut.ingestion.dropped_messages` (counter, visualize as rate over time)
- Optional diagram alternatives (if we later decide):
  - `goknut.http.requests` (requests per interval)
  - `goknut.search.queries`

## Diagram Rendering Approach

### Decision: server-rendered SVG (no canvas, minimal JS)

- Render diagrams as small SVG sparklines/line charts server-side.
- Reasoning:
  - Works with server-rendered templates.
  - No new client-side charting library.
  - Easy to return as HTML partial via HTMX.
  - Easy to provide empty/error states.

### Time window + cadence

- Window: last 15 minutes (initial)
- Poll cadence: 30 seconds for diagram + snapshot blocks (still satisfies "at least once per minute")

## Data Acquisition Strategy

### Decision: query Prometheus HTTP API with PromQL

Use Prometheus as the source of truth for time-series diagrams by querying the Prometheus HTTP API using PromQL over a historical range window.

- Query strategy:
  - Use the PromQL expressions defined in `specs/005-startpage-stats/spec.md`.
  - Request a fixed range window (default: last 15 minutes) and a fixed step size (default: 30 seconds).
- Failure behavior (soft dependency):
  - Enforce a strict per-request timeout for Prometheus calls.
  - If Prometheus is unavailable/slow/returns errors, return a degraded diagrams widget with HTTP 200 that preserves layout and clearly indicates diagrams are unavailable.

Reasoning:
- Produces real historical time-series values (not just in-process snapshots).
- Aligns with operators’ existing monitoring source.
- Keeps KPI totals sourced from the DB (separate concern).

## Polling / Refresh Mechanism

### Decision: HTMX polling of partials

- Add a small HTML partial endpoint (or endpoints) that returns dashboard blocks:
  - KPI tiles + "last updated" label
  - Diagrams (SVG)
- The main home page output embeds containers with:
  - `hx-get="/dashboard/home/...”` (exact routes to be defined in contracts)
  - `hx-trigger="load, every 30s"`

Reasoning:
- Matches existing stack and patterns.
- Keeps rendering server-side.
- Avoids the bespoke SSE client script.

## Alternatives Considered

- Internal in-process time-series ring buffer: not chosen because it would not reflect historical Prometheus-scraped data and would be reset on process restart.
- Keep SSE for stats only: rejected because the requirement explicitly pivots away from the live view behavior; HTMX polling is simpler and more failure-tolerant.
