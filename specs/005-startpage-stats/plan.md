# Implementation Plan: Statistics-Centric Startpage

**Branch**: `005-startpage-stats` | **Date**: 2025-12-21 | **Spec**: `specs/005-startpage-stats/spec.md`
**Input**: Feature specification from `specs/005-startpage-stats/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Refactor the start page to remove the live "Latest Messages" feed and replace it with a statistics-first dashboard. The dashboard keeps shortcut links to primary sections and automatically refreshes KPI values and simple time-series diagrams via lightweight polling, using existing OTel/Prometheus metrics where possible and DB aggregates for totals.

## Technical Context

**Language/Version**: Go 1.22 (single binary)
**Primary Dependencies**: Go stdlib (`net/http`, `html/template`) + HTMX for progressive enhancement; OpenTelemetry + Prometheus scrape via `/metrics`
**Storage**: SQLite (WAL) for local/dev; Postgres optional (existing migrations)
**Testing**: `go test ./...` with unit/integration/contract suites under `tests/`
**Target Platform**: Linux/macOS server (single HTTP binary)
**Project Type**: Server-rendered web UI with HTMX partial updates
**Performance Goals**: Home dashboard refresh endpoints respond fast enough for periodic polling
**Constraints**: No new infrastructure/services; diagrams must be lightweight (SSR SVG), graceful no-data/failure behavior
**Scale/Scope**: High-throughput ingestion metrics; low-frequency dashboard polling (every 60s per open page)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Quality**: Plan keeps scope cohesive (home page only); follow existing handler/template patterns. **Status: PASS**
- **Testing**: Add tests for start page rendering and absence of latest-messages feed; verify new polling endpoints. **Status: PASS**
- **UX**: Keep shortcuts visible; add loading/empty/error states for stats/diagrams; diagrams include labels/time window. **Status: PASS**
- **Performance**: Declare budgets for polling endpoints (p95≤250ms/p99≤500ms); avoid full-page reloads and heavy JS. **Status: PASS**
- **Observability**: Reuse existing OTel metrics; add structured logs for dashboard refresh failures (rate-limited/minimal noise). **Status: PASS**

## Project Structure

### Documentation (this feature)

```text
specs/005-startpage-stats/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
└── tasks.md
```

### Source Code (repository root)

```text
cmd/server/
internal/
  config/
  http/
    handlers/
    templates/
  observability/
  repository/
tests/
  contract/
  integration/
  unit/
```

**Structure Decision**: Single Go backend binary with server-rendered templates. This feature adds/adjusts HTTP handlers and templates under `internal/http/` and adds/updates tests under `tests/`.

## Complexity Tracking

> No constitutional violations requiring justification at this time.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | n/a | n/a |
