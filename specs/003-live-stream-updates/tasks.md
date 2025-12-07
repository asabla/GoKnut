---

description: "Task list for live stream updates feature"
---

# Tasks: Live Stream Updates

**Input**: Design documents from `/specs/003-live-stream-updates/`
**Prerequisites**: plan.md (required), spec.md (required for user stories)

**Tests**: Tests are mandated by spec non-functional gate but not explicitly requested as tasks; include per-story test coverage only where valuable and story-specific.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story. Quality, UX, performance, and observability are captured per story where applicable.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Shared initialization and live-transport wiring prerequisites

- [ ] T001 Add WebSocket dependency `nhooyr.io/websocket` to `go.mod`
- [ ] T002 Configure baseline WebSocket server wiring in `internal/http/server.go`
- [ ] T003 [P] Add live status/idle/error partial placeholders in `internal/http/templates/partials/` (new partial files or augment existing)
- [ ] T004 Document live transport bootstrap steps in `specs/003-live-stream-updates/quickstart.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that must exist before story work

- [ ] T005 Implement live event fan-out skeleton and connection registry in `internal/http/handlers/live/` (new package)
- [ ] T006 Define live message envelope structs (metrics, messages, counts) in `internal/http/dto/dto.go`
- [ ] T007 [P] Add observability hooks for live transport (connect/disconnect/errors) in `internal/observability/observability.go`
- [ ] T008 [P] Add graceful degradation flag/handshake for when WebSocket unavailable in `internal/http/server.go`
- [ ] T009 Add backpressure/queue bounds configuration in `internal/config/config.go`
- [ ] T010 Create shared HTMX/WebSocket client initializer script reference in `internal/http/templates/partials/` (referenced from views)

**Checkpoint**: Foundation ready—user story implementation can begin; live transport skeleton, DTOs, config, and observability in place.

---

## Phase 3: User Story 1 - Home view stays live (Priority: P1) 🎯 MVP

**Goal**: Home page streams metrics and latest messages without reload.
**Independent Test**: Keep home open; ingest messages; metrics and latest messages update in order without duplicates or reload.

### Implementation for User Story 1

- [ ] T011 [P] [US1] Wire home WebSocket subscription endpoint/route in `internal/http/handlers/live/channel_view.go`
- [ ] T012 [US1] Stream metric summaries to home template `internal/http/templates/home.html` (inject partial/hooks)
- [ ] T013 [P] [US1] Stream latest messages list partial in `internal/http/templates/live/message.html`
- [ ] T014 [US1] Add idle/disconnected status UX on home in `internal/http/templates/home.html`
- [ ] T015 [US1] Ensure ordering/deduplication in live home handler `internal/http/handlers/live/channel_view.go`
- [ ] T016 [US1] Add observability logs/metrics for home stream in `internal/observability/observability.go`

**Checkpoint**: Home view independently delivers live metrics and messages.

---

## Phase 4: User Story 2 - Messages page streams new messages (Priority: P1)

**Goal**: `/messages` appends new messages live, preserving history.
**Independent Test**: Open `/messages`; ingest messages; new entries append in order without reload or loss.

### Implementation for User Story 2

- [ ] T017 [P] [US2] Add WebSocket subscription for `/messages` in `internal/http/handlers/live/messages.go`
- [ ] T018 [P] [US2] Stream message list partial updates in `internal/http/templates/live/messages.html`
- [ ] T019 [US2] Preserve chronological ordering/deduplication in `internal/http/handlers/live/messages.go`
- [ ] T020 [US2] Add connection status UX for `/messages` in `internal/http/templates/messages/index.html`
- [ ] T021 [US2] Add observability logs/metrics for messages stream in `internal/observability/observability.go`

**Checkpoint**: `/messages` independently streams new messages in order.

---

## Phase 5: User Story 3 - Channel list reflects live counts (Priority: P2)

**Goal**: `/channels` updates each channel’s count live.
**Independent Test**: Open `/channels`; ingest across channels; counts update without reload.

### Implementation for User Story 3

- [ ] T022 [P] [US3] Add channel counts stream handler in `internal/http/handlers/live/channels.go`
- [ ] T023 [P] [US3] Update channel row partial to consume live counts in `internal/http/templates/channels/row.html`
- [ ] T024 [US3] Handle burst updates/backpressure for channel counts in `internal/http/handlers/live/channels.go`
- [ ] T025 [US3] Add status UX for `/channels` live state in `internal/http/templates/channels/index.html`
- [ ] T026 [US3] Add observability logs/metrics for channels stream in `internal/observability/observability.go`

**Checkpoint**: `/channels` independently streams live counts.

---

## Phase 6: User Story 4 - Users list shows live activity (Priority: P2)

**Goal**: `/users` shows per-user message counts live.
**Independent Test**: Open `/users`; ingest messages from users; counts update without reload.

### Implementation for User Story 4

- [ ] T027 [P] [US4] Add user counts stream handler in `internal/http/handlers/live/users.go`
- [ ] T028 [P] [US4] Update users row partial to consume live counts in `internal/http/templates/search/users_results.html`
- [ ] T029 [US4] Handle burst updates/backpressure for user counts in `internal/http/handlers/live/users.go`
- [ ] T030 [US4] Add status UX for `/users` live state in `internal/http/templates/search/users.html`
- [ ] T031 [US4] Add observability logs/metrics for users stream in `internal/observability/observability.go`

**Checkpoint**: `/users` independently streams live user counts.

---

## Phase 7: User Story 5 - User profile stays current (Priority: P3)

**Goal**: `/users/<user-name>` streams that user’s counts and latest messages.
**Independent Test**: Open profile; ingest messages for user; counts and latest messages update in order without reload.

### Implementation for User Story 5

- [ ] T032 [P] [US5] Add per-user stream handler in `internal/http/handlers/live/user_profile.go`
- [ ] T033 [P] [US5] Update profile template for live counts/messages in `internal/http/templates/search/user_profile.html`
- [ ] T034 [US5] Ensure ordering/deduplication for profile stream in `internal/http/handlers/live/user_profile.go`
- [ ] T035 [US5] Add status UX for profile live state in `internal/http/templates/search/user_profile.html`
- [ ] T036 [US5] Add observability logs/metrics for user profile stream in `internal/observability/observability.go`

**Checkpoint**: `/users/<user-name>` independently streams live counts and messages.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Hardening and consistency across stories

- [ ] T037 [P] Add shared reconnect/backoff client script reference in `internal/http/templates/partials/`
- [ ] T038 [P] Add accessibility review notes for live statuses in `specs/003-live-stream-updates/research.md`
- [ ] T039 [P] Add performance budget validation notes for live paths in `docs/perf.md`
- [ ] T040 Run quickstart validation for live updates in `specs/003-live-stream-updates/quickstart.md`
- [ ] T041 Consolidate observability dashboards/alerts entries for live streams in `internal/observability/observability.go`
- [ ] T042 Code cleanup and dead path removal in `internal/http/handlers/live/`

---

## Dependencies & Execution Order

### Phase Dependencies

- Setup (Phase 1): No dependencies
- Foundational (Phase 2): Depends on Setup completion; blocks all user stories
- User Stories (Phase 3-7): Depend on Foundational completion; can run in priority order (P1 → P2 → P3) or in parallel per story if capacity
- Polish (Phase 8): Depends on desired user stories completion

### User Story Dependencies

- US1 (P1): No story dependencies once foundation ready
- US2 (P1): Independent after foundation; shares transport patterns with US1
- US3 (P2): Independent after foundation
- US4 (P2): Independent after foundation
- US5 (P3): Independent after foundation; reuses transport structures

### Within Each User Story

- Ordering/dedup logic precedes UX polish where applicable
- Observability added with each handler to keep scope local

### Parallel Opportunities

- Setup: T003 can run in parallel
- Foundational: T007, T008 can run in parallel; T006 after T005; T010 after T006
- US1: T011 & T013 parallel; status/ordering tasks follow wiring
- US2: T017 & T018 parallel; status and ordering follow
- US3: T022 & T023 parallel
- US4: T027 & T028 parallel
- US5: T032 & T033 parallel
- Polish: T037-T039 can run in parallel

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1
4. Validate US1 independently; demo as MVP

### Incremental Delivery

1. Foundation ready → US1 (P1) → validate
2. US2 (P1) → validate
3. US3 (P2) and US4 (P2) → validate independently
4. US5 (P3) → validate

### Parallel Team Strategy

- After foundation, each user story can be staffed independently following parallel hints above.
