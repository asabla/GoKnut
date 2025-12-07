# Implementation Plan: Live Stream Updates

**Branch**: `[003-live-stream-updates]` | **Date**: 2025-12-07 | **Spec**: `specs/003-live-stream-updates/spec.md`
**Input**: Feature specification from `/specs/003-live-stream-updates/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Deliver live streaming for home metrics, latest messages, `/messages`, `/channels`, `/users`, and `/users/<user-name>` using the existing Go 1.22 + HTMX + SQLite stack, adding a WebSocket transport for near-real-time updates with graceful degradation and clear status handling.

## Technical Context

**Language/Version**: Go 1.22 (single binary)
**Primary Dependencies**: `net/http`, `html/template`, HTMX (templates), `modernc.org/sqlite`, `nhooyr.io/websocket` for WebSocket transport
**Storage**: SQLite (WAL) with messages table + FTS5 virtual table
**Testing**: `go test` with unit, integration, and contract suites
**Target Platform**: Server-rendered web UI on Linux/macOS; HTTP + WebSocket endpoints
**Project Type**: Single backend binary with server-side templates and HTMX partials
**Performance Goals**: Backend p95≤250ms/p99≤500ms; live delivery target 95% of updates visible <2s (SC-001); avoid duplicate rendering
**Constraints**: Graceful degradation when WebSocket unavailable; reconnection with backoff; fan-out must not block ingestion; bounded in-memory queues; accessibility for status/idle states
**Scale/Scope**: Single instance serving home/messages/channels/users views; multiple concurrent WebSocket listeners per view

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Quality**: Lint/format clean, cohesive scope, documentation updated with behavior changes.
- **Testing**: Failing-first unit/integration/contract/regression coverage for new or changed behavior.
- **UX**: Design system usage, accessibility (WCAG 2.1 AA), and loading/empty/error states documented and validated.
- **Performance**: Budgets declared; defaults backend p95≤250ms/p99≤500ms, frontend critical render/interaction ≤2s; validation or planned measurement recorded.
- **Observability**: Structured logs/metrics/traces defined for new paths and failure modes, with review evidence noted.

Status: PASS (no planned violations).

## Project Structure

### Documentation (this feature)

```text
specs/003-live-stream-updates/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
cmd/
└── server/main.go
internal/
├── http/                # handlers, templates, server wiring
│   ├── handlers/
│   ├── templates/
│   └── server.go
├── ingestion/           # ingestion pipeline, processor
├── services/            # channel/search services
├── repository/          # SQLite repositories, migrations
└── observability/       # logging, metrics

tests/
├── contract/
├── integration/
└── unit/
```

**Structure Decision**: Single backend binary serving server-rendered HTML + HTMX partials; WebSocket endpoint added under `internal/http` with fan-out to existing views.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |

## Phase 0 – Outline & Research

- Extracted clarifications: WebSocket transport choice; fan-out model per view; ordering/dedup + backpressure; reconnect/idle UX; HTMX integration approach.
- Resolved in `research.md` with decisions, rationale, and alternatives.

## Phase 1 – Design & Contracts

- Data model updates captured in `data-model.md` (LiveEvent, MetricSummary, ChannelSummary, UserSummary, Message stream envelope).
- API contracts and WebSocket message schema captured in `contracts/`.
- Quickstart instructions for running server and exercising WebSocket stream in `quickstart.md`.
- Agent context updated via `.specify/scripts/bash/update-agent-context.sh opencode` to record WebSocket addition.

## Post-Design Constitution Check

- Reaffirmed gates: no deviations planned; performance budget and observability hooks documented for streaming path.
