# Implementation Plan: Message Search View

**Branch**: `002-message-search-view` | **Date**: 2025-12-07 | **Spec**: specs/002-message-search-view/spec.md
**Input**: Feature specification from `/specs/002-message-search-view/spec.md`

## Summary

Refactor the search experience into a message-focused view that lets users search message text and apply author, channel, and ingestion-time filters, returning navigable results with context (channel, sender, timestamp) using the existing Go 1.22 single-binary stack with SQLite + FTS5, HTMX-rendered templates, and no new dependencies.

## Technical Context

**Language/Version**: Go 1.22 (single binary)  
**Primary Dependencies**: Go stdlib (`net`, `database/sql`, `net/http`), `modernc.org/sqlite`, `html/template`, HTMX, Tailwind CSS (already present)  
**Storage**: SQLite (WAL) with `messages` table + FTS5 virtual table for message content  
**Testing**: `go test` (unit, integration, contract HTTP handlers)  
**Target Platform**: macOS/Linux single-process server  
**Project Type**: Single backend with server-rendered web UI  
**Performance Goals**: Default: HTTP p95 ≤250ms / p99 ≤500ms for search; render critical path ≤2s with HTMX partials; keep pagination to avoid large payloads  
**Constraints**: No new dependencies; reuse existing schema/FTS; validate queries server-side (min length, safe params); adhere to WCAG 2.1 AA for the view states  
**Scale/Scope**: Low millions of messages, thousands of results per query with pagination; multi-channel, multi-user usage

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Quality**: Lint/format clean, cohesive search-view scope, docs updated with behavior changes. **Status: PASS**
- **Testing**: Failing-first unit + integration + contract coverage for search query/filters, empty/error states. **Status: PASS**
- **UX**: HTMX templates using design system; document loading/empty/error states; WCAG 2.1 AA considerations. **Status: PASS**
- **Performance**: Budgets declared (HTTP p95 ≤250ms/p99 ≤500ms; render ≤2s); pagination avoids large payloads; plan to validate. **Status: PASS**
- **Observability**: Structured logs/metrics around search queries, filters, validation failures; ensure error paths observable. **Status: PASS**

## Project Structure

### Documentation (this feature)

```text
specs/002-message-search-view/
├── plan.md              # This file (/speckit.plan output)
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit.tasks, not in this run)
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
├── search/             # search adapters (LIKE/FTS5)
└── http/
    ├── handlers/       # HTMX endpoints & routing
    └── templates/      # html/template views + Tailwind artifacts

tests/
├── unit/
├── integration/        # DB + ingestion + IRC fakes
└── contract/           # HTTP/api contract tests
```

**Structure Decision**: Single Go service with `cmd/server` entry and `internal/*` domain packages; server-rendered HTMX UI and SQLite persistence with dedicated test roots for unit/integration/contract coverage. No new projects or dependencies introduced.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|---------------------------------------|
| None | N/A | N/A |

## Constitution Check (Post-Design)

- Re-evaluated after Phase 1 design artifacts. **Status: PENDING**
