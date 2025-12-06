# Implementation Plan: Twitch Chat Archiver & Explorer

**Branch**: `001-spec-reference-spec` | **Date**: 2025-12-06 | **Spec**: specs/001-spec-reference-spec/spec.md
**Input**: Feature specification from `/specs/001-spec-reference-spec/spec.md`

## Summary

Archive Twitch chat into SQLite with a Go single-binary service that ingests 100–150 msgs/sec, surfaces live channel views with ≤1s latency using HTMX polling, and supports searchable histories across channels, users, and messages.

## Technical Context

**Language/Version**: Go 1.22 (single binary)
**Primary Dependencies**: Go stdlib (`net`, `database/sql`, `net/http`), `modernc.org/sqlite` driver, HTMX for progressive enhancement, Tailwind CSS for styling templates
**Storage**: SQLite (WAL, durable local file)
**Testing**: `go test` (unit, integration, contract with HTTP handlers)
**Target Platform**: Linux/macOS single-process server (CLI entry)
**Project Type**: Single backend with server-rendered web UI
**Performance Goals**: Ingest 100–150 msgs/sec; live UI shows new messages ≤1s; HTTP p95 ≤250ms/p99 ≤500ms; minimal batching latency (<100ms flush)
**Constraints**: No external DB; single-operator OAuth; WCAG 2.1 AA UI states; avoid +100MB sustained memory growth; single process manages channels
**Scale/Scope**: ~10+ channels, low millions of messages, thousands of users/messages per query

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Quality**: Scope cohesive to chat archiver; plan documents docs updates for new behavior. **Status: PASS**
- **Testing**: Unit + integration + contract tests planned for ingestion, repositories, HTTP routes. **Status: PASS**
- **UX**: Server-rendered HTMX views with documented loading/empty/error states and WCAG 2.1 AA considerations. **Status: PASS**
- **Performance**: Budgets declared (ingestion throughput, UI latency, HTTP p95/p99); batching + WAL/FTS choices documented for validation. **Status: PASS**
- **Observability**: Structured logs for IRC/connectivity/ingestion/search and metrics hooks planned; failure modes documented. **Status: PASS**

## Project Structure

### Documentation (this feature)

```text
specs/001-spec-reference-spec/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
└── tasks.md (Phase 2 via /speckit.tasks)
```

### Source Code (repository root)

```text
cmd/
└── server/             # main entrypoint

internal/
├── config/             # load & validate config
├── irc/                # Twitch IRC client
├── ingestion/          # pipelines, batching, caches
├── repository/         # SQLite repositories and migrations
├── services/           # channel/user/search domain services
├── http/
│   ├── handlers/       # HTMX endpoints & routing
│   └── templates/      # html/template views + Tailwind artifacts
└── search/             # search adapters (LIKE/FTS5)

tests/
├── unit/
├── integration/        # DB + ingestion + IRC fakes
└── contract/           # HTTP/api contract tests
```

**Structure Decision**: Single Go service with `cmd/server` entry and `internal/*` domain packages; server-rendered HTMX UI and SQLite persistence with dedicated test roots for unit/integration/contract coverage.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|---------------------------------------|
| None | N/A | N/A |

## Constitution Check (Post-Design)

- Re-evaluated after Phase 1: design artifacts align with gates; no violations introduced. **Status: PASS**
