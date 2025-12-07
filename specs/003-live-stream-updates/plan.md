# Implementation Plan: Live Stream Updates

**Branch**: `[003-live-stream-updates]` | **Date**: 2025-12-07 | **Spec**: `specs/003-live-stream-updates/spec.md`
**Input**: Feature specification from `specs/003-live-stream-updates/spec.md`

## Summary

Deliver live updates for home, messages, channels, users, and user profile views using Server-Sent Events (SSE) for read-only streams with graceful degradation to existing manual refresh/polling. Preserve ordering/deduplication via `messages.id` cursor, keep pages responsive during bursts, and surface status on disconnect/reconnect.

## Technical Context

**Language/Version**: Go 1.22 (single binary)  
**Primary Dependencies**: Go stdlib (`net/http`, `html/template`), HTMX (server-rendered), SSE event streams; `modernc.org/sqlite`  
**Storage**: SQLite (WAL) using existing tables and triggers; no new migrations  
**Testing**: `go test ./...` with unit, integration, and contract suites; failing-first coverage for live handlers and templates  
**Target Platform**: Server-rendered web app on Linux/macOS (single HTTP binary)  
**Project Type**: Web backend with server-rendered templates + HTMX progressive enhancement  
**Performance Goals**: Backend p95 ≤250ms/p99 ≤500ms; live delivery shows 95% of events in ≤2s (normal load); frontend critical render/interaction ≤2s  
**Constraints**: SSE-only for read-only live updates; bounded per-connection buffers and backfill; graceful fallback to polling/manual refresh on failure  
**Scale/Scope**: Moderate traffic with burst tolerance; thousands of messages/minute per session with bounded replay (≤500 events or ~5s backlog)  

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Quality**: Plan keeps scope cohesive; docs updated alongside behavior changes. **Status: PASS**
- **Testing**: Failing-first coverage required for new live handlers, reconnect/backfill, and template behaviors. **Status: PASS**
- **UX**: Uses existing design system; documents loading/empty/error/disconnected states and accessibility. **Status: PASS**
- **Performance**: Budgets declared (backend p95≤250ms/p99≤500ms; live ≤2s delivery); backlog capped. **Status: PASS**
- **Observability**: Logs/metrics for connect/disconnect/errors/backpressure; traces optional for handler path. **Status: PASS**

## Project Structure

### Documentation (this feature)

```text
specs/003-live-stream-updates/
├── plan.md              # This file (/speckit.plan output)
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output (not created here)
```

### Source Code (repository root)

```text
cmd/server/
internal/
  config/
  http/
    handlers/
    templates/
  ingestion/
  observability/
  repository/
  search/
  services/
tests/
  contract/
  integration/
  unit/
specs/003-live-stream-updates/
```

**Structure Decision**: Single Go backend binary with server-rendered templates and existing test suites; live SSE handlers and template hooks land under `internal/http`, with supporting services in `internal/services` and repository reads in `internal/repository`.

## Complexity Tracking

> No constitutional violations requiring justification at this time.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | n/a | n/a |

## Observability Coverage

The SSE implementation includes the following observability features:

### Log Fields
- `client_id`: Unique identifier for each SSE client connection
- `view`: The view type (home, messages, channels, users, user_profile)
- `after_id`: Cursor position for backfill tracking
- `reason`: Disconnect reason (context_done, client_gone, etc.)

### Metrics (counters/gauges in observability package)
- `sse_connections_total`: Total SSE connections
- `sse_connections_active`: Current active connections
- `sse_events_sent_total`: Total events emitted
- `sse_backpressure_drops`: Events dropped due to client backpressure
- `sse_reconnects_total`: Client reconnection attempts

### Log Events
- `SSE client connected` (INFO): New client connection with view/cursor
- `SSE client disconnected` (INFO): Client disconnection with reason
- `SSE backpressure` (WARN): Client buffer full, events dropped
- `SSE heartbeat sent` (DEBUG): Periodic keepalive events

## Test Evidence

Full test suite executed successfully on 2025-12-07:
```
ok  	github.com/asabla/goknut/tests/contract	0.563s
ok  	github.com/asabla/goknut/tests/integration	4.032s
ok  	github.com/asabla/goknut/tests/unit	0.854s
```

All SSE-related tests pass:
- `TestHomeSSEStream` - Home view SSE with metrics and messages
- `TestMessagesSSEStream` - Messages view SSE with deduplication
- `TestMessagesSSEStreamNoDuplicates` - No duplicate messages on backfill
- `TestChannelsSSEStream` - Channel list SSE with counts
- `TestUsersSSEStream` - Users list SSE with counts
- `TestUserProfileSSEStream` - User profile SSE with stats
