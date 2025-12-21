---

description: "Task list for Statistics-Centric Startpage"
---

# Tasks: Statistics-Centric Startpage

**Input**: Design documents from `specs/005-startpage-stats/`

**Prerequisites** (available docs):
- Required: `specs/005-startpage-stats/plan.md`, `specs/005-startpage-stats/spec.md`
- Optional (present): `specs/005-startpage-stats/research.md`, `specs/005-startpage-stats/data-model.md`, `specs/005-startpage-stats/contracts/api.md`, `specs/005-startpage-stats/quickstart.md`

**Tests**: Included (spec requires failing-first coverage via **NFR-TESTING**).

**Organization**: Tasks are grouped by user story so each story can be implemented and tested independently.

## Checklist Format (strict)

Every task MUST use:

`- [ ] T### [P?] [US?] Action with file path`

- `[P]` only when parallelizable (different files / no dependency).
- `[US#]` only inside user story phases.
- Every task description includes at least one concrete file path.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm baseline behavior and capture current start page behavior before refactor.

- [ ] T001 Run baseline tests `go test ./...` and paste output into `specs/005-startpage-stats/research.md`
- [ ] T002 Document current home page behavior (stats tiles + shortcuts + latest messages + SSE) in `specs/005-startpage-stats/research.md`
- [ ] T003 [P] Confirm template discovery supports `internal/http/templates/dashboard/*.html` (validate `ParseFS(..., "*.html", "*/*.html")`) and note result in `specs/005-startpage-stats/research.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add dashboard scaffolding required by all user stories.

⚠️ **CRITICAL**: No user story work should begin until these routes compile and render placeholder HTML.

- [ ] T004 Create `internal/http/handlers/home_dashboard.go` with `HomeDashboardHandler` skeleton and `RegisterRoutes(*http.ServeMux)`
- [ ] T005 [P] Create dashboard fragment templates `internal/http/templates/dashboard/home_summary.html` and `internal/http/templates/dashboard/home_diagrams.html`
- [ ] T006 Wire dashboard routes in `internal/http/server.go` (register `GET /dashboard/home/summary` and `GET /dashboard/home/diagrams`)

**Checkpoint**: `GET /dashboard/home/summary` and `GET /dashboard/home/diagrams` return 200 with basic HTML.

---

## Phase 3: User Story 1 - Monitor system activity at a glance (Priority: P1) 🎯 MVP

A user opens the start page to quickly understand whether the system is ingesting messages, serving queries, and generally “healthy”, without reading a live message feed.

**Goal**: Replace the start page’s live feed focus with a statistics-first dashboard: KPI tiles + “Last updated” + two lightweight time-series diagrams.

**Independent Test**: Can be fully tested by loading `/` and verifying:
- KPI fragment and diagrams fragment are visible.
- Both fragments automatically refresh at least every 60 seconds (HTMX polling).
- Clear loading/empty/error behavior when data is missing.

### Tests for User Story 1 (write first; ensure they fail)

- [ ] T007 [P] [US1] Add integration test for `GET /dashboard/home/summary` HTML fragment in `tests/integration/home_dashboard_summary_test.go`
- [ ] T008 [P] [US1] Add integration test for `GET /dashboard/home/diagrams` HTML fragment in `tests/integration/home_dashboard_diagrams_test.go`
- [ ] T009 [P] [US1] Add unit test asserting HTMX polling containers exist in `internal/http/templates/home.html` via `tests/unit/home_template_test.go`

### Implementation for User Story 1

- [ ] T010 [US1] Implement DB-backed KPI snapshot builder (messages/channels/enabled/users + last updated + partial error capture) in `internal/http/handlers/home_dashboard.go`
- [ ] T011 [US1] Add minimal Prometheus HTTP client for range queries (base URL + timeout) in `internal/http/handlers/home_dashboard.go`
- [ ] T012 [US1] Implement `GET /dashboard/home/summary` handler returning HTML fragment in `internal/http/handlers/home_dashboard.go`
- [ ] T013 [US1] Implement `GET /dashboard/home/diagrams` handler querying Prometheus via PromQL (15m window, 30s step) and returning HTML fragment with SVG + empty/error states in `internal/http/handlers/home_dashboard.go`
- [ ] T014 [P] [US1] Implement KPI fragment markup (labels, placeholders, last-updated) in `internal/http/templates/dashboard/home_summary.html`
- [ ] T015 [P] [US1] Implement diagrams fragment markup (two labeled SVG charts + time window labels + degraded-unavailable UI) in `internal/http/templates/dashboard/home_diagrams.html`
- [ ] T016 [US1] Refactor `/` page to include HTMX-polled containers for the two fragments (30–60s cadence) in `internal/http/templates/home.html`

**Checkpoint**: Visiting `/` shows dashboard blocks; values refresh without full page reload; “Last updated” changes on refresh.

---

## Phase 4: User Story 2 - Navigate quickly to primary areas (Priority: P2)

A user uses shortcut links on the start page to jump to commonly used areas (channels, users, message search, etc.).

**Goal**: Shortcut links remain visible and usable regardless of dashboard loading/failure.

**Independent Test**: Load `/` and verify links to `/channels`, `/users`, and `/messages` exist and remain visible even if dashboard fragments fail.

### Tests for User Story 2 (write first; ensure they fail)

- [ ] T017 [P] [US2] Add unit test asserting shortcuts to `/channels`, `/users`, `/messages` exist in `internal/http/templates/home.html` via `tests/unit/home_template_test.go`

### Implementation for User Story 2

- [ ] T018 [US2] Preserve shortcut cards section while refactoring dashboard containers in `internal/http/templates/home.html`

**Checkpoint**: Shortcuts are always visible and clickable.

---

## Phase 5: User Story 3 - No more live message feed on start page (Priority: P3)

A user no longer sees “latest messages” streaming into the start page.

**Goal**: Remove the latest messages section and remove the start page’s reliance on `/live?view=home` SSE.

**Independent Test**: Verify that `internal/http/templates/home.html` contains no “Latest Messages” section and no home SSE client script.

### Tests for User Story 3 (write first; ensure they fail)

- [ ] T019 [P] [US3] Add unit test asserting `internal/http/templates/home.html` does not contain "Latest Messages" or `/live?view=home` via `tests/unit/home_template_test.go`
- [ ] T020 [P] [US3] Replace SSE-home integration coverage with dashboard polling coverage by updating `tests/integration/live_view_integration_test.go`

### Implementation for User Story 3

- [ ] T021 [US3] Remove “Latest Messages” markup block from `internal/http/templates/home.html`
- [ ] T022 [US3] Remove home SSE client script block (and related DOM ids) from `internal/http/templates/home.html`
- [ ] T023 [US3] Stop querying recent messages for `/` render in `internal/http/server.go`
- [ ] T024 [US3] Remove `view=home` support from SSE handler (valid views and `sendHomeData`) in `internal/http/handlers/live_sse.go`

**Checkpoint**: `/` has no live feed UI and no SSE dependency; other SSE views keep working.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Ensure quality, UX, performance, and observability requirements are met.

- [ ] T025 [P] Add structured logs for dashboard refresh failures (rate-limited / low-noise) in `internal/http/handlers/home_dashboard.go`
- [ ] T026 [P] Add unit test coverage for empty-series SVG rendering in `internal/http/templates/dashboard/home_diagrams.html` via `tests/unit/home_dashboard_diagrams_test.go`
- [ ] T027 [P] Add unit test coverage for Prometheus-timeout degraded UI state in `internal/http/templates/dashboard/home_diagrams.html` via `tests/unit/home_dashboard_diagrams_test.go`
- [ ] T028 [P] Update verification steps in `specs/005-startpage-stats/quickstart.md` if any runtime behavior changed
- [ ] T029 Run full test suite via `make test` (see `Makefile`)

---

## Dependencies & Execution Order

### Phase Dependencies

- Phase 1 (Setup) → Phase 2 (Foundational) → User Story phases
- Phase 2 blocks all user stories
- Phase 6 (Polish) depends on completing the desired user stories

### User Story Dependency Graph

- **US1 (P1)** depends on Phase 2 (`/dashboard/home/*` routes + templates)
- **US2 (P2)** depends on Phase 2 (it modifies `internal/http/templates/home.html`)
- **US3 (P3)** depends on Phase 2 (it modifies `internal/http/templates/home.html` and SSE internals)

Suggested completion order (MVP-first): `US1 → US2 → US3`

---

## Parallel Opportunities

- Phase 2: T004 and T005 can run in parallel (new handler vs new templates)
- US1: T007–T009 (separate test files) can run in parallel; T014 and T015 can run in parallel
- Polish: T025–T027 can run in parallel (different files)

---

## Parallel Execution Examples (per user story)

### US1

```text
Task: "Add integration test for GET /dashboard/home/summary in tests/integration/home_dashboard_summary_test.go"
Task: "Add integration test for GET /dashboard/home/diagrams in tests/integration/home_dashboard_diagrams_test.go"
Task: "Implement KPI fragment markup in internal/http/templates/dashboard/home_summary.html"
Task: "Implement diagrams fragment markup in internal/http/templates/dashboard/home_diagrams.html"
```

### US2

```text
Task: "Add shortcuts presence test in tests/unit/home_template_test.go"
Task: "Preserve shortcut cards section in internal/http/templates/home.html"
```

### US3

```text
Task: "Add no-live-feed assertions in tests/unit/home_template_test.go"
Task: "Remove latest messages + SSE script from internal/http/templates/home.html"
Task: "Remove view=home support from internal/http/handlers/live_sse.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 → Phase 2
2. Implement and validate US1 end-to-end
3. Stop and validate `/` dashboard UX (loading/empty/error) + polling cadence

### Incremental Delivery

- Land US1 dashboard polling and fragments first
- Keep/verify shortcuts (US2)
- Remove old live feed/SSE home behavior (US3)
- Finish with polish tasks and final test run
