---

description: "Task list: Profiles, Organizations & Collaborations"
---

# Tasks: Profiles, Organizations & Collaborations

**Input**: Design documents from `specs/004-profiles-orgs-collabs/`
**Prerequisites**: `specs/004-profiles-orgs-collabs/plan.md`, `specs/004-profiles-orgs-collabs/spec.md`, `specs/004-profiles-orgs-collabs/research.md`, `specs/004-profiles-orgs-collabs/data-model.md`, `specs/004-profiles-orgs-collabs/contracts/api.md`

**Organization**: Tasks are grouped by user story (US1-US4) so each story can be implemented and validated independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[US#]**: Which user story this task belongs to. Setup/foundation/polish tasks omit story tag.
- Every task includes exact file paths.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add minimal scaffolding and navigation without touching ingestion hot-path.

- [ ] T001 Confirm no ingestion changes planned (no edits required under `internal/ingestion/` and `internal/repository/message_repository.go`)
- [ ] T002 Add navigation entry points for new sections in `internal/http/templates/partials/base.html`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Schema + shared repository/service primitives required by all user stories.

**CRITICAL**: No user story work should start until schema and core repos compile.

- [ ] T003 Add SQLite tables + indexes for profiles/orgs/events/collabs in `internal/repository/migrations/001_init.sql`
- [ ] T004 Add Postgres tables + indexes for profiles/orgs/events/collabs in `internal/repository/migrations/postgres/001_init.sql`
- [ ] T005 [P] Create profile repository (types + CRUD + channel linking) in `internal/repository/profile_repository.go`
- [ ] T006 [P] Create organization repository (types + CRUD + memberships) in `internal/repository/organization_repository.go`
- [ ] T007 [P] Create event repository (types + CRUD + participants) in `internal/repository/event_repository.go`
- [ ] T008 [P] Create collaboration repository (types + CRUD + participants) in `internal/repository/collaboration_repository.go`
- [ ] T009 Add shared repository errors (e.g., conflict/not-found mapping) in `internal/repository/db.go`
- [ ] T010 [P] Create profile service (validations + orchestration) in `internal/services/profile_service.go`
- [ ] T011 [P] Create organization service (validations + orchestration) in `internal/services/organization_service.go`
- [ ] T012 [P] Create event service (date validation + orchestration) in `internal/services/event_service.go`
- [ ] T013 [P] Create collaboration service (min participants + orchestration) in `internal/services/collaboration_service.go`
- [ ] T014 Wire new services into server config and struct in `internal/http/server.go`

**Checkpoint**: DB schema + repositories compile; server can be wired with new handlers.

---

## Phase 3: User Story 1 - Maintain Profiles (Priority: P1) MVP

**Goal**: Operators can create profiles and link one or more channels, enforcing the rule that a channel may be linked to at most one profile.

**Independent Test**: Create profile -> link channels -> profile detail lists linked channels; attempting conflicting link is blocked with a clear error.

### Tests for User Story 1

- [ ] T015 [P] [US1] Unit tests for linking validation + uniqueness errors in `tests/unit/profile_service_test.go`
- [ ] T016 [P] [US1] Integration test for profile creation + channel linking in `tests/integration/profiles_integration_test.go`

### Implementation for User Story 1

- [ ] T017 [P] [US1] Add profile DTOs (create/update/link) in `internal/http/dto/dto.go`
- [ ] T018 [P] [US1] Add profile templates (index/list/detail/new) in `internal/http/templates/profiles/index.html`
- [ ] T019 [P] [US1] Add profile templates (index/list/detail/new) in `internal/http/templates/profiles/list.html`
- [ ] T020 [P] [US1] Add profile templates (index/list/detail/new) in `internal/http/templates/profiles/detail.html`
- [ ] T021 [P] [US1] Add profile templates (index/list/detail/new) in `internal/http/templates/profiles/new.html`
- [ ] T022 [US1] Implement profile routes + handlers per contract in `internal/http/handlers/profiles.go`
- [ ] T023 [US1] Register profile routes in `internal/http/server.go`

**Checkpoint**: US1 fully functional and independently demoable.

---

## Phase 4: User Story 2 - Maintain Organizations (Priority: P2)

**Goal**: Operators can create organizations and manage memberships; profile pages show affiliations.

**Independent Test**: Create organization -> add profile member -> organization detail lists members and profile detail lists affiliations.

### Tests for User Story 2

- [ ] T024 [P] [US2] Unit tests for membership uniqueness in `tests/unit/organization_service_test.go`
- [ ] T025 [P] [US2] Integration test for organization membership round-trip in `tests/integration/organizations_integration_test.go`

### Implementation for User Story 2

- [ ] T026 [P] [US2] Add organization templates (index/list/detail/new) in `internal/http/templates/organizations/index.html`
- [ ] T027 [P] [US2] Add organization templates (index/list/detail/new) in `internal/http/templates/organizations/list.html`
- [ ] T028 [P] [US2] Add organization templates (index/list/detail/new) in `internal/http/templates/organizations/detail.html`
- [ ] T029 [P] [US2] Add organization templates (index/list/detail/new) in `internal/http/templates/organizations/new.html`
- [ ] T030 [US2] Implement organization routes + handlers per contract in `internal/http/handlers/organizations.go`
- [ ] T031 [US2] Register organization routes in `internal/http/server.go`
- [ ] T032 [US2] Extend profile detail to show org affiliations in `internal/http/handlers/profiles.go` and `internal/http/templates/profiles/detail.html`

**Checkpoint**: US2 functional; profile page shows org affiliations.

---

## Phase 5: User Story 3 - Curate Events (Priority: P3)

**Goal**: Operators can create curated events with start/end date validation and profile participants.

**Independent Test**: Create event with start and participants; saving end < start is blocked; participant profiles show event.

### Tests for User Story 3

- [ ] T033 [P] [US3] Unit tests for event date validation in `tests/unit/event_service_test.go`
- [ ] T034 [P] [US3] Integration test for event creation + participants in `tests/integration/events_integration_test.go`

### Implementation for User Story 3

- [ ] T035 [P] [US3] Add event templates (index/list/detail/new) in `internal/http/templates/events/index.html`
- [ ] T036 [P] [US3] Add event templates (index/list/detail/new) in `internal/http/templates/events/list.html`
- [ ] T037 [P] [US3] Add event templates (index/list/detail/new) in `internal/http/templates/events/detail.html`
- [ ] T038 [P] [US3] Add event templates (index/list/detail/new) in `internal/http/templates/events/new.html`
- [ ] T039 [US3] Implement event routes + handlers per contract in `internal/http/handlers/events.go`
- [ ] T040 [US3] Register event routes in `internal/http/server.go`
- [ ] T041 [US3] Extend profile detail to show associated events in `internal/http/handlers/profiles.go` and `internal/http/templates/profiles/detail.html`

**Checkpoint**: US3 functional; events appear on participant profiles.

---

## Phase 6: User Story 4 - Record Collaborations (Priority: P4)

**Goal**: Operators can create collaborations with 2+ participants and optional flags (e.g., shared chat).

**Independent Test**: Create collaboration -> add 2 participant profiles -> remove one participant -> profile page updates.

### Tests for User Story 4

- [ ] T042 [P] [US4] Unit tests for minimum participant validation in `tests/unit/collaboration_service_test.go`
- [ ] T043 [P] [US4] Integration test for collaboration participants add/remove in `tests/integration/collaborations_integration_test.go`

### Implementation for User Story 4

- [ ] T044 [P] [US4] Add collaboration templates (index/list/detail/new) in `internal/http/templates/collaborations/index.html`
- [ ] T045 [P] [US4] Add collaboration templates (index/list/detail/new) in `internal/http/templates/collaborations/list.html`
- [ ] T046 [P] [US4] Add collaboration templates (index/list/detail/new) in `internal/http/templates/collaborations/detail.html`
- [ ] T047 [P] [US4] Add collaboration templates (index/list/detail/new) in `internal/http/templates/collaborations/new.html`
- [ ] T048 [US4] Implement collaboration routes + handlers per contract in `internal/http/handlers/collaborations.go`
- [ ] T049 [US4] Register collaboration routes in `internal/http/server.go`
- [ ] T050 [US4] Extend profile detail to show collaborations in `internal/http/handlers/profiles.go` and `internal/http/templates/profiles/detail.html`

**Checkpoint**: US4 functional; collaborations appear on participant profiles.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: UX/observability/performance hardening and cross-story validation.

- [ ] T051 Add consistent empty/error states via shared template partials in `internal/http/templates/partials/empty.html` and `internal/http/templates/partials/error.html`
- [ ] T052 Add structured logs for create/update/link actions in `internal/http/handlers/profiles.go`, `internal/http/handlers/organizations.go`, `internal/http/handlers/events.go`, `internal/http/handlers/collaborations.go`
- [ ] T053 Add lightweight metrics counters for CRUD/link outcomes in `internal/observability/observability.go`
- [ ] T054 Validate all new queries are indexed (verify indexes in `internal/repository/migrations/001_init.sql` and `internal/repository/migrations/postgres/001_init.sql`)
- [ ] T055 Run quickstart manual verification from `specs/004-profiles-orgs-collabs/quickstart.md` and update it if behavior differs
- [ ] T056 Run full test suite: `go test ./...`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: can start immediately.
- **Foundational (Phase 2)**: depends on Phase 1; blocks all user stories.
- **US1 (Phase 3)**: depends on Phase 2.
- **US2 (Phase 4)**: depends on US1 (organizations reference profiles).
- **US3 (Phase 5)**: depends on US1 (events reference profiles).
- **US4 (Phase 6)**: depends on US1 (collaborations reference profiles).
- **Polish (Phase 7)**: depends on whichever user stories are shipped.

### Parallel Opportunities

- Phase 2 repositories/services can be done in parallel: T005-T008 and T010-T013.
- Within each story, templates can be done in parallel (e.g., US1: T018-T021).

---

## Parallel Example: US1

```bash
Task: "[P] [US1] Add profile DTOs in internal/http/dto/dto.go"
Task: "[P] [US1] Add profile templates in internal/http/templates/profiles/*.html"
Task: "[US1] Implement profile routes in internal/http/handlers/profiles.go"
```

---

## Implementation Strategy

### MVP First (US1 only)

1. Complete Phase 1 and Phase 2.
2. Complete Phase 3 (US1).
3. Validate with the Profiles section in `specs/004-profiles-orgs-collabs/quickstart.md`.

### Incremental Delivery

- After US1, add US2/US3/US4 in priority order.
- Each story must remain independently testable and must not modify the ingestion hot path.
