---

description: "Task list for Statistics-Centric Startpage"
---

# Tasks: Statistics-Centric Startpage

**Input**: Design documents from `specs/005-startpage-stats/`
**Prerequisites**: `plan.md` (required), `spec.md` (required), `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: Included because `specs/005-startpage-stats/spec.md` requires failing-first coverage via **NFR-TESTING**.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story. Quality, UX, performance, and observability tasks are captured where relevant.

## Checklist Format (strict)

Every task MUST use:

`- [ ] T### [P?] [US?] Action with file path`

- `[P]` only when parallelizable (different files / no dependency).
- `[US#]` only inside user story phases.
- Every task description includes at least one concrete file path.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish baseline behavior and document current start page implementation.

- [x] T001 Run baseline tests `go test ./...` and paste output into `specs/005-startpage-stats/research.md`
- [x] T002 Document current home page behavior (KPI tiles + shortcuts + latest messages + SSE) in `specs/005-startpage-stats/research.md`
- [x] T003 [P] Confirm template parsing supports `internal/http/templates/dashboard/*.html` (validate `ParseFS(..., "*.html", "*/*.html")`) and record result in `specs/005-startpage-stats/research.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add dashboard scaffolding required by all user stories.

⚠️ **CRITICAL**: No user story work should begin until these routes compile and render placeholder HTML.

- [x] T004 Add Prometheus config fields (base URL + timeout) in `internal/config/config.go`
- [x] T005 Create `internal/http/handlers/home_dashboard.go` with `HomeDashboardHandler` skeleton and `RegisterRoutes(*http.ServeMux)`
- [x] T006 [P] Create dashboard fragment templates `internal/http/templates/dashboard/home_summary.html` and `internal/http/templates/dashboard/home_diagrams.html` (placeholder HTML)
- [x] T007 Wire dashboard routes in `internal/http/server.go` (register `GET /dashboard/home/summary` and `GET /dashboard/home/diagrams`)
- [x] T008 Ensure dashboard templates are parsed/registered in `internal/http/server.go` (or template loader used by server)

**Checkpoint**: `GET /dashboard/home/summary` and `GET /dashboard/home/diagrams` return 200 with basic HTML.

---

## Phase 3: User Story 1 - Monitor system activity at a glance (Priority: P1) 🎯 MVP

A user opens the start page to quickly understand whether the system is ingesting messages and generally “healthy”, without reading a live message feed.

**Goal**: Add a statistics-first dashboard: KPI tiles + “Last updated” + two lightweight time-series diagrams.

**Independent Test**:
- Load `/` and confirm KPI + diagram widgets render.
- Confirm both widgets auto-refresh at least once per minute (HTMX polling).
- Force Prometheus timeout/unavailability and confirm diagrams show degraded state (HTTP 200 fragment).

### Tests for User Story 1 (write first; ensure they fail)

- [x] T009 [P] [US1] Add integration test for `GET /dashboard/home/summary` HTML fragment in `tests/integration/home_dashboard_summary_test.go`
- [x] T010 [P] [US1] Add integration test for `GET /dashboard/home/diagrams` success path (fake Prometheus) in `tests/integration/home_dashboard_diagrams_test.go`
- [x] T011 [P] [US1] Add integration test for `GET /dashboard/home/diagrams` timeout/unavailable degraded HTML (fake Prometheus) in `tests/integration/home_dashboard_diagrams_test.go`
- [x] T012 [P] [US1] Add Prometheus HTTP fake server helper in `tests/integration/fakes/prometheus_fake.go`
- [x] T013 [P] [US1] Add unit test asserting HTMX polling containers exist in `internal/http/templates/home.html` via `tests/unit/home_template_test.go`

### Implementation for User Story 1

- [ ] T014 [US1] Implement DB-backed KPI snapshot builder (messages/channels/enabled/users + last updated + partial error capture) in `internal/http/handlers/home_dashboard.go`
- [ ] T015 [US1] Implement minimal Prometheus range-query client (HTTP + JSON decode) in `internal/http/handlers/home_dashboard.go`
- [ ] T016 [US1] Implement `GET /dashboard/home/summary` handler returning HTML fragment in `internal/http/handlers/home_dashboard.go`
- [ ] T017 [US1] Implement `GET /dashboard/home/diagrams` handler using PromQL from `specs/005-startpage-stats/spec.md` (15m window, 30s step) in `internal/http/handlers/home_dashboard.go`
- [ ] T018 [P] [US1] Implement KPI fragment markup (labels, values, last-updated, partial-error placeholders) in `internal/http/templates/dashboard/home_summary.html`
- [ ] T019 [P] [US1] Implement diagrams fragment markup (two labeled SVG charts, time window labels, empty + degraded states) in `internal/http/templates/dashboard/home_diagrams.html`
- [ ] T020 [US1] Refactor `/` page to include HTMX-polled containers for summary + diagrams fragments in `internal/http/templates/home.html`

**Checkpoint**: Visiting `/` shows dashboard blocks; “Last updated” changes on refresh; diagrams degrade within timeout budget when Prometheus fails.

---

## Phase 4: User Story 2 - Navigate quickly to primary areas (Priority: P2)

A user uses shortcut links on the start page to jump to commonly used areas (channels, users, message search, etc.) while still using the start page as an overview dashboard.

**Goal**: Ensure shortcut links remain visible and usable regardless of dashboard loading/failure.

**Independent Test**:
- Load `/` and verify links to `/channels`, `/users`, `/messages` exist.
- Simulate dashboard fragment failures (e.g., stop Prometheus) and confirm shortcuts are still visible.

### Tests for User Story 2 (write first; ensure they fail)

- [ ] T021 [P] [US2] Add unit test asserting shortcuts to `/channels`, `/users`, `/messages` exist in `internal/http/templates/home.html` via `tests/unit/home_template_test.go`

### Implementation for User Story 2

- [ ] T022 [US2] Preserve/adjust shortcut cards section while refactoring dashboard containers in `internal/http/templates/home.html`

**Checkpoint**: Shortcuts are always visible and clickable.

---

## Phase 5: User Story 3 - No more live message feed on start page (Priority: P3)

A user no longer sees “latest messages” streaming into the start page; instead, the space is used for diagrams and statistics.

**Goal**: Remove the latest messages section and remove the start page’s reliance on `/live?view=home` SSE.

**Independent Test**:
- Verify `internal/http/templates/home.html` contains no “Latest Messages” section and no `/live?view=home` client logic.
- Load `/` and confirm it does not query recent messages and does not rely on SSE to populate KPI tiles.

### Tests for User Story 3 (write first; ensure they fail)

- [ ] T023 [P] [US3] Add unit test asserting `internal/http/templates/home.html` does not contain "Latest Messages" or `/live?view=home` via `tests/unit/home_template_test.go`
- [ ] T024 [P] [US3] Add integration test asserting `GET /` HTML has no "Latest Messages" and no `/live?view=home` in `tests/integration/home_view_integration_test.go`

### Implementation for User Story 3

- [ ] T025 [US3] Remove “Latest Messages” markup block from `internal/http/templates/home.html`
- [ ] T026 [US3] Remove home SSE client script block (and related DOM ids) from `internal/http/templates/home.html`
- [ ] T027 [US3] Stop querying recent messages for `/` render in `internal/http/server.go`
- [ ] T028 [US3] Remove `view=home` support from SSE handler (valid views and any `sendHomeData`) in `internal/http/handlers/live_sse.go`

**Checkpoint**: `/` has no live feed UI and no SSE home dependency; other SSE views keep working.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories.

- [ ] T029 [P] Add structured logs for dashboard refresh failures (rate-limited / low-noise) in `internal/http/handlers/home_dashboard.go`
- [ ] T030 [P] Add unit test coverage for SVG empty/degraded rendering helpers in `tests/unit/home_dashboard_diagrams_test.go`
- [ ] T031 [P] Update verification steps in `specs/005-startpage-stats/quickstart.md` to include Prometheus required/soft dependency behavior
- [ ] T032 Run full test suite via `make test` (see `Makefile`)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies
- **Foundational (Phase 2)**: Depends on Setup; BLOCKS all user stories
- **User Stories (Phases 3–5)**: Depend on Foundational
- **Polish (Phase 6)**: Depends on completing desired user stories

### User Story Dependencies

- **US1 (P1)**: No dependencies on other stories
- **US2 (P2)**: No functional dependency on US1 (shortcuts must exist regardless)
- **US3 (P3)**: Depends on US1 insofar as the home page must still show useful content after removing the live feed

Suggested completion order (MVP-first): `US1 → US2 → US3`

---

## Parallel Execution Examples

### US1

```text
Task: "Add integration test for GET /dashboard/home/summary in tests/integration/home_dashboard_summary_test.go"
Task: "Add Prometheus HTTP fake server helper in tests/integration/fakes/prometheus_fake.go"
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
Task: "Remove latest messages markup in internal/http/templates/home.html"
Task: "Remove view=home support in internal/http/handlers/live_sse.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 → Phase 2
2. Implement and validate US1 end-to-end
3. STOP and validate `/` dashboard UX (loading/empty/error) + polling cadence

### Incremental Delivery

- Land US1 dashboard polling and fragments first
- Ensure shortcuts remain intact (US2)
- Remove old live feed / SSE home behavior (US3)
- Finish with polish tasks and final test run
