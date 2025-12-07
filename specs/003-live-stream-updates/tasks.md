---

description: "Task list for live stream updates feature"
---

# Tasks: Live Stream Updates

**Input**: Design documents from `/specs/003-live-stream-updates/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests included per story to satisfy constitutional gates (failing-first coverage). Tests are marked in each story section.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing. Quality, UX, performance, and observability tasks are captured per story where applicable.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Baseline configuration and docs alignment for SSE delivery

- [X] T001 Add SSE feature toggle default-on in `internal/config/config.go`
- [X] T002 Wire SSE route placeholder in `internal/http/server.go`
- [X] T003 [P] Document local SSE run instructions in `specs/003-live-stream-updates/quickstart.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core live infrastructure required before any user story

- [X] T004 Define shared SSE event envelopes and dispatcher in `internal/http/handlers/live_sse.go`
- [X] T005 [P] Add repository cursor/backfill helper using `messages.id` in `internal/repository/message_repository.go`
- [X] T006 [P] Add live observability counters/log fields for connect/disconnect/backpressure in `internal/observability/observability.go`
- [X] T007 Add SSE status/fallback partial for reuse in `internal/http/templates/partials/status.html`
- [X] T008 Align contracts to SSE transport in `specs/003-live-stream-updates/contracts/api.md`

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Home view stays live (Priority: P1) 🎯 MVP

**Goal**: Home shows live metrics and latest messages via SSE without reload

**Independent Test**: With home open, ingest messages and counts; observe metrics/messages update within 2s, ordered, no duplicates; status visible on disconnect.

### Tests for User Story 1

- [X] T009 [P] [US1] Add integration test for home SSE stream in `tests/integration/live_view_integration_test.go`

### Implementation for User Story 1

- [X] T010 [US1] Implement home SSE emission (metrics + latest messages) in `internal/http/handlers/live_sse.go`
- [X] T011 [P] [US1] Bind SSE to home template for metrics/messages in `internal/http/templates/home.html`
- [X] T012 [P] [US1] Add status/idle UI hook to home template in `internal/http/templates/home.html`

**Checkpoint**: User Story 1 independently testable

---

## Phase 4: User Story 2 - Messages page streams new messages (Priority: P1)

**Goal**: `/messages` streams new messages in order while retaining history

**Independent Test**: Keep `/messages` open; ingest messages; list appends in order within 2s, no duplicates; remains responsive under burst.

### Tests for User Story 2

- [X] T013 [P] [US2] Add integration test for `/messages` SSE ordering/dup avoidance in `tests/integration/live_view_integration_test.go`

### Implementation for User Story 2

- [X] T014 [US2] Implement `/messages` SSE stream with cursor/backfill in `internal/http/handlers/live_sse.go`
- [X] T015 [P] [US2] Update messages list template to append SSE events in `internal/http/templates/messages/index.html`
- [X] T016 [P] [US2] Add loading/error/status indicators for `/messages` in `internal/http/templates/messages/index.html`

**Checkpoint**: User Story 2 independently testable

---

## Phase 5: User Story 3 - Channel list reflects live counts (Priority: P2)

**Goal**: `/channels` shows live message counts per channel

**Independent Test**: With `/channels` open, ingest messages across channels; counts update within session without blocking UI.

### Tests for User Story 3

- [X] T017 [P] [US3] Add contract/integration test for channel count SSE in `tests/contract/channels_test.go`

### Implementation for User Story 3

- [X] T018 [US3] Emit `channel_count` SSE events per channel in `internal/http/handlers/live_sse.go`
- [X] T019 [P] [US3] Update channel row template to consume SSE counts in `internal/http/templates/channels/row.html`
- [X] T020 [P] [US3] Surface status/fallback indicator on channel list in `internal/http/templates/channels/index.html`

**Checkpoint**: User Story 3 independently testable

---

## Phase 6: User Story 4 - Users list shows live activity (Priority: P2)

**Goal**: `/users` shows live message counts per user

**Independent Test**: With `/users` open, ingest messages from different users; counts update within session without blocking UI.

### Tests for User Story 4

- [X] T021 [P] [US4] Add integration test for user count SSE in `tests/integration/search_integration_test.go`

### Implementation for User Story 4

- [X] T022 [US4] Emit `user_count` SSE events for users list in `internal/http/handlers/live_sse.go`
- [X] T023 [P] [US4] Update users list template to show live counts in `internal/http/templates/search/users.html`
- [X] T024 [P] [US4] Add status/fallback indicator to users list in `internal/http/templates/search/users.html`

**Checkpoint**: User Story 4 independently testable

---

## Phase 7: User Story 5 - User profile stays current (Priority: P3)

**Goal**: `/users/<user-name>` shows live counts and latest messages for that user

**Independent Test**: Keep profile open; ingest messages from that user; count and recent messages update in order without duplicates; status visible on disconnect.

### Tests for User Story 5

- [X] T025 [P] [US5] Add integration test for user profile SSE in `tests/integration/live_view_integration_test.go`

### Implementation for User Story 5

- [X] T026 [US5] Emit `user_profile` SSE events for selected user in `internal/http/handlers/live_sse.go`
- [X] T027 [P] [US5] Update user profile template to bind SSE counts/messages in `internal/http/templates/search/user_profile.html`
- [X] T028 [P] [US5] Add status/fallback indicator to user profile in `internal/http/templates/search/user_profile.html`

**Checkpoint**: User Story 5 independently testable

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Hardening, documentation, and performance/observability validation across stories

- [X] T029 [P] Refresh quickstart with final SSE + fallback steps in `specs/003-live-stream-updates/quickstart.md`
- [X] T030 Add live metrics/log fields coverage notes in `specs/003-live-stream-updates/plan.md`
- [X] T031 [P] Run gofmt and ensure lint cleanliness for changed files (`internal/http/handlers/live_sse.go`, templates)
- [X] T032 Execute full test suite `go test ./...` and record evidence in `specs/003-live-stream-updates/plan.md`

---

## Dependencies & Execution Order

- **Setup (Phase 1)** → **Foundational (Phase 2)** → User Stories (Phases 3–7) → **Polish (Phase 8)**.
- User stories depend on Phase 2 completion; afterward, US1–US5 can proceed in priority order (P1 first) or in parallel where tasks marked [P] avoid file conflicts.
- Within each story: tests (fail-first) precede implementation; event emission before template bindings; status/fallback UI before final validation.

### User Story Dependencies

- **US1 (P1)**: No story dependencies; establishes SSE path validation and can ship MVP.
- **US2 (P1)**: Depends on foundational; independent of US1 aside from shared handler.
- **US3 (P2)**: Depends on foundational; shares handler but independent of US1/US2 UI.
- **US4 (P2)**: Depends on foundational; independent of US1–US3 aside from shared handler.
- **US5 (P3)**: Depends on foundational; independent of other stories aside from shared handler.

### Parallel Opportunities

- Setup tasks T001–T003 can run concurrently where marked [P].
- Foundational tasks T005–T007–T008 can proceed in parallel after T004 is ready.
- Story template updates and tests marked [P] can run concurrently with handler work if coordination avoids file conflicts.
- Different stories (e.g., US1 vs US2) can proceed in parallel once foundational is done, staffed permitting.

### Parallel Example: User Story 1

```bash
# In parallel for US1 after foundational:
Task: T009 Add integration test for home SSE stream in tests/integration/live_view_integration_test.go
Task: T011 Bind SSE to home template for metrics/messages in internal/http/templates/home.html
Task: T012 Add status/idle UI hook to home template in internal/http/templates/home.html
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)
1. Complete Setup + Foundational.
2. Deliver US1 (home live metrics/messages) with tests and status UI.
3. Validate via integration test (T009) and manual quickstart; ship MVP.

### Incremental Delivery
1. After MVP, add US2 (messages), then US3 (channels), US4 (users), US5 (profile) in priority order.
2. Each story tested independently before integration.
3. Polish phase finalizes docs, formatting, and full-suite tests.

### Parallel Team Strategy
- Developer A: US1 → US3
- Developer B: US2 → US4
- Developer C: US5 + Polish
- Coordinate on `internal/http/handlers/live_sse.go` to avoid conflicts; templates are largely disjoint.

## Notes

- [P] tasks = different files with no hard dependency; coordinate shared handler edits.
- Each user story remains independently testable.
- Tests should fail first before implementation per constitution.
- Avoid cross-story coupling; keep SSE envelopes and status handling consistent across views.
