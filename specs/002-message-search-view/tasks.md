---
description: "Task list for Message Search View feature"
status: Complete
---

# Tasks: Message Search View

**Input**: Design documents from `/specs/002-message-search-view/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Testing is required per specification (failing-first unit/integration/contract).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story. Quality, UX, performance, and observability are captured per story where applicable.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Ensure routing and templates baseline for search view.

- [X] T001 Confirm `/search/messages` route and HTMX handlers are registered in `internal/http/server.go` and `internal/http/handlers/search.go`
- [X] T002 [P] Ensure search templates directory exists and is referenced in `internal/http/templates/search/messages.html`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core search plumbing and validation that all user stories depend on.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T003 Add base query validation (min length, pagination defaults) in `internal/http/handlers/search.go`
- [X] T004 [P] Confirm search repository supports FTS query with pagination defaults in `internal/search/search_repository.go`

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel.

---

## Phase 3: User Story 1 - Find messages by text (Priority: P1) 🎯 MVP

**Goal**: Users can search message text and see results with channel, sender, and timestamp context ordered by newest first.

**Independent Test**: Submit text query and verify relevant message results with channel/sender/timestamp are returned and navigable.

### Tests for User Story 1 (required per spec)

- [X] T005 [P] [US1] Add contract test for GET `/search/messages` text query in `tests/contract/search_test.go`
- [X] T006 [P] [US1] Add integration test for text search results in `tests/integration/search_integration_test.go`

### Implementation for User Story 1

- [X] T007 [P] [US1] Ensure search service performs text search and returns context fields in `internal/services/search_service.go`
- [X] T008 [P] [US1] Render message search results (list and empty state) in `internal/http/templates/search/messages.html`
- [X] T009 [US1] Handle HTMX request flow and preserve inputs in `internal/http/handlers/search.go`
- [X] T010 [US1] Include navigation links from results to channel/message view in `internal/http/templates/search/messages.html`

**Checkpoint**: User Story 1 fully functional and independently testable.

---

## Phase 4: User Story 2 - Filter by author or channel (Priority: P2)

**Goal**: Users can narrow results by author or channel while searching text.

**Independent Test**: Apply author or channel filter with a query and verify only matching messages appear.

### Tests for User Story 2 (required per spec)

- [X] T011 [P] [US2] Add contract test for author/channel filters on `/search/messages` in `tests/contract/search_test.go`
- [X] T012 [P] [US2] Add integration test covering author/channel filters in `tests/integration/search_integration_test.go`

### Implementation for User Story 2

- [X] T013 [P] [US2] Parse author/channel filters and validate IDs in `internal/http/handlers/search.go`
- [X] T014 [P] [US2] Apply author/channel filters in search service and repository in `internal/services/search_service.go` and `internal/search/search_repository.go`
- [X] T015 [US2] Update search form UI to display and persist author/channel filters in `internal/http/templates/search/messages.html`

**Checkpoint**: User Stories 1 and 2 independently functional.

---

## Phase 5: User Story 3 - Filter by time ingested (Priority: P3)

**Goal**: Users can constrain results to a specified ingestion time range.

**Independent Test**: Set start/end dates with a query and verify results fall within the window; invalid ranges are rejected with clear feedback.

### Tests for User Story 3 (required per spec)

- [X] T016 [P] [US3] Add contract test for time-range filter on `/search/messages` in `tests/contract/search_test.go`
- [X] T017 [P] [US3] Add integration test for time-range filtering in `tests/integration/search_integration_test.go`

### Implementation for User Story 3

- [X] T018 [P] [US3] Parse and validate start/end date inputs in `internal/http/handlers/search.go`
- [X] T019 [P] [US3] Apply time-range filtering and end-of-day expansion in `internal/services/search_service.go` and `internal/search/search_repository.go`
- [X] T020 [US3] Show time filter fields and validation errors in `internal/http/templates/search/messages.html`

**Checkpoint**: All user stories independently functional.

---

## Phase 6: UX Refactor - Align Messages with Users/Channels

**Goal**: Move message search to `/messages`, align templates/layout with users/channels, and add expandable rows.

- [X] T024 Create `internal/http/templates/messages/index.html` (page shell) and wire to `/messages`
- [X] T025 Create `internal/http/templates/messages/list.html` (HTMX partial) with table layout + expandable rows for full text
- [X] T026 Update `internal/http/handlers/search.go` to serve `/messages` and new templates
- [X] T027 Update navigation links across templates to point to `/messages` (rename "Search" to "Messages")
- [X] T028 Remove legacy `internal/http/templates/search/messages*.html`
- [X] T029 Update contract test URLs in `tests/contract/search_test.go` to `/messages`
- [X] T030 Build/verify route + HTMX flow (no new dependencies)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories.
- **User Stories (Phase 3+)**: Depend on Foundational completion. Stories can proceed in priority order (P1 → P2 → P3) or in parallel if staffing allows.
- **Polish (Final Phase)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational; no dependency on other stories.
- **User Story 2 (P2)**: Depends on Foundational; integrates with US1 data but remains independently testable.
- **User Story 3 (P3)**: Depends on Foundational; may reuse US1/US2 components but remains independently testable.

### Within Each User Story

- Tests MUST be written and FAIL before implementation tasks.
- Implement services/repository changes before handler/template wiring where dependencies exist.
- Ensure navigation contexts remain intact when adding filters.

### Parallel Opportunities

- [P] tasks within Setup and Foundational can run concurrently.
- Within each story, [P] tasks in different files (tests, service, repository, templates) can proceed in parallel after prerequisites.
- Different stories can be developed in parallel after Foundational if teams are separate.

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Setup and Foundational phases.
2. Deliver User Story 1 (text search with context) and validate via contract and integration tests.
3. Demo/ship MVP before additional filters.

### Incremental Delivery

1. Complete Setup + Foundational → foundation ready.
2. Add User Story 1 → test independently → deliver.
3. Add User Story 2 → test independently → deliver.
4. Add User Story 3 → test independently → deliver.
5. Apply Polish tasks across all stories.

### Parallel Team Strategy

- After Foundational: split User Stories across contributors (US1, US2, US3) with clear file ownership to avoid conflicts.
- Coordinate handler/template merges for filters to minimize overlap.
