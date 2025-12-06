---

description: "Task list for Twitch Chat Archiver & Explorer"
---

# Tasks: Twitch Chat Archiver & Explorer

**Input**: Design documents from `/specs/001-spec-reference-spec/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests are required by the specification (unit, integration, contract with failing-first pattern).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story. Capture quality, UX, performance, and observability tasks per story where applicable.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and baseline tooling

- [X] T001 Confirm Go 1.22 toolchain and sqlite driver in `go.mod`
- [X] T002 [P] Establish feature configuration flags/env parsing in `internal/config/config.go`
- [X] T003 [P] Add feature README pointer in `docs/` to spec and quickstart
- [X] T004 Set baseline HTMX/Tailwind template layout shell in `internal/http/templates/layout.html`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T005 Initialize SQLite schema migrations for channels/users/messages in `internal/repository/migrations/001_init.sql`
- [X] T006 [P] Implement database bootstrap with WAL and pragmas in `internal/repository/sqlite.go`
- [X] T007 [P] Add IRC client scaffolding with connect/reconnect hooks in `internal/irc/client.go`
- [X] T008 [P] Define ingestion pipeline interfaces and batcher skeleton in `internal/ingestion/pipeline.go`
- [X] T009 Add structured logging and metrics hooks for IRC/ingestion/search in `internal/observability/observability.go`
- [X] T010 Configure HTTP router, middleware, and health endpoint in `internal/http/server.go`
- [X] T011 Seed shared template partials (loading/empty/error states) in `internal/http/templates/partials/`
- [X] T012 Define shared DTOs and validation helpers in `internal/http/dto/dto.go`
- [X] T013 Establish test fakes for IRC and repositories in `tests/integration/fakes/`
- [X] T014 Add make/Go script for contract test harness in `tests/contract/README.md`

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Manage connected channels (Priority: P1) 🎯 MVP

**Goal**: Moderators manage tracked channels (list, add, enable/disable, delete with retention choice) and system joins/leaves accordingly

**Independent Test**: Configure channels list, enable a new channel, observe join and message recording without disrupting existing channels

### Tests for User Story 1

- [X] T015 [P] [US1] Contract tests for channel CRUD endpoints in `tests/contract/channels_test.go`
- [X] T016 [P] [US1] Integration tests for channel config persistence and IRC join/part in `tests/integration/channels_integration_test.go`
- [X] T017 [P] [US1] Unit tests for channel validation and service logic in `tests/unit/channel_service_test.go`

### Implementation for User Story 1

- [X] T018 [P] [US1] Implement Channel repository (CRUD, stats) in `internal/repository/channel_repository.go`
- [X] T019 [P] [US1] Implement Channel service with enable/disable/delete semantics in `internal/services/channel_service.go`
- [X] T020 [US1] Wire channel service to IRC join/part callbacks in `internal/irc/client.go`
- [X] T021 [US1] Add HTTP handlers for list/create/update/delete in `internal/http/handlers/channels.go`
- [X] T022 [US1] Add HTMX templates for channel list and forms in `internal/http/templates/channels/`
- [X] T023 [US1] Add input validation and error/empty states for channels in `internal/http/handlers/channels.go`
- [X] T024 [US1] Add logging and metrics for channel lifecycle events in `internal/services/channel_service.go`

**Checkpoint**: User Story 1 fully functional and independently testable

---

## Phase 4: User Story 2 - View live stream per channel (Priority: P2)

**Goal**: Provide live channel view with recent backlog and streaming updates within 1s latency

**Independent Test**: Open a channel view, observe new messages within target latency, confirm historical backlog loads on entry

### Tests for User Story 2

- [X] T025 [P] [US2] Contract tests for channel view/stream endpoints in `tests/contract/channel_view_test.go`
- [X] T026 [P] [US2] Integration tests for ingestion → storage → HTMX fragments in `tests/integration/live_view_integration_test.go`
- [X] T027 [P] [US2] Unit tests for message formatting/pagination helpers in `tests/unit/message_format_test.go`

### Implementation for User Story 2

- [X] T028 [P] [US2] Implement Message repository (recent, paginated, stream) in `internal/repository/message_repository.go`
- [X] T029 [P] [US2] Implement ingestion processor to normalize/store messages in `internal/ingestion/processor.go`
- [X] T030 [US2] Implement live view handlers (page, messages, stream) in `internal/http/handlers/channel_view.go`
- [X] T031 [US2] Add HTMX templates for live feed and pagination in `internal/http/templates/live/`
- [X] T032 [US2] Ensure batching/WAL settings meet latency budgets in `internal/ingestion/processor.go`
- [X] T033 [US2] Add logging/metrics for ingestion latency and stream delivery in `internal/ingestion/processor.go`

**Checkpoint**: User Stories 1 AND 2 independently functional

---

## Phase 5: User Story 3 - Search users and messages (Priority: P3)

**Goal**: Search across users/messages with filters and user profile views

**Independent Test**: Execute username and text searches with filters; confirm paginated results and user profile summaries

### Tests for User Story 3

- [X] T034 [P] [US3] Contract tests for search endpoints in `tests/contract/search_test.go`
- [X] T035 [P] [US3] Integration tests for FTS/LIKE search paths in `tests/integration/search_integration_test.go`
- [X] T036 [P] [US3] Unit tests for search query builders and highlighting in `tests/unit/search_service_test.go`

### Implementation for User Story 3

- [X] T037 [P] [US3] Implement Search/FTS repository helpers in `internal/search/search_repository.go`
- [X] T038 [P] [US3] Implement Search service for users/messages with filters in `internal/services/search_service.go`
- [X] T039 [US3] Implement HTTP handlers for user/message search in `internal/http/handlers/search.go`
- [X] T040 [US3] Add HTMX templates for search results and user profiles in `internal/http/templates/search/`
- [X] T041 [US3] Add pagination and highlighting utilities in `internal/http/templates/partials/highlight.html`
- [X] T042 [US3] Add logging/metrics for search latency and empty states in `internal/services/search_service.go`

**Checkpoint**: All user stories independently functional and testable

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [X] T043 [P] Documentation updates in `docs/` for setup, latency budgets, observability
- [X] T044 Code cleanup and dependency audit in `internal/`
- [X] T045 Performance validation runbook for ingestion/search in `docs/perf.md`
- [X] T046 [P] Additional regression/unit tests for shared helpers in `tests/unit/`
- [X] T047 Security and resilience review for IRC reconnect/backoff in `internal/irc/client.go`
- [X] T048 Run quickstart validation steps in `quickstart.md`

---

## Dependencies & Execution Order

- Setup (Phase 1): No dependencies - start immediately
- Foundational (Phase 2): Depends on Setup completion - BLOCKS all user stories
- User Story 1 (Phase 3, P1): Depends on Foundational; no other story dependencies
- User Story 2 (Phase 4, P2): Depends on Foundational; may reuse US1 channel data but test independently
- User Story 3 (Phase 5, P3): Depends on Foundational; may reuse US1/US2 data but test independently
- Polish (Phase 6): Depends on all desired user stories being complete

### User Story Dependency Graph

- US1 → US2 (shared channel/message data)
- US1 → US3 (channel/user data)
- US2 ↔ US3 (independent, can develop in parallel after US1 data model exists)

### Parallel Opportunities

- Setup: T002, T003, T004 in parallel
- Foundational: T006–T008, T011, T012, T013 in parallel
- US1: Tests T015–T017 parallel; repos/services/templates T018, T019, T022 parallel
- US2: Tests T025–T027 parallel; repos/ingestion templates T028, T029, T031 parallel
- US3: Tests T034–T036 parallel; search repo/service/templates T037, T038, T040 parallel

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Run US1 contract/integration/unit tests
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP)
3. Add User Story 2 → Test independently → Deploy/Demo
4. Add User Story 3 → Test independently → Deploy/Demo
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1
   - Developer B: User Story 2
   - Developer C: User Story 3
3. Stories complete and integrate independently
