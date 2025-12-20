# Implementation Plan: Profiles, Organizations & Collaborations

**Branch**: `004-profiles-orgs-collabs` | **Date**: 2025-12-20 | **Spec**: `specs/004-profiles-orgs-collabs/spec.md`
**Input**: Feature specification from `specs/004-profiles-orgs-collabs/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Add curated, operator-managed entities (Profiles, Organizations, Events, Collaborations) that group channels into real-world identities and relationships. Preserve the existing high-throughput message ingestion path by keeping `messages` storage unchanged and representing new relationships via additive, indexed tables.

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: Go 1.22 (single binary)  
**Primary Dependencies**: Go stdlib (`net/http`, `html/template`), HTMX (server-rendered); SQLite (`modernc.org/sqlite`) and optional Postgres (`lib/pq`)  
**Storage**: SQLite (WAL) for local/dev; Postgres optional (existing migrations)  
**Testing**: `go test ./...` with unit, integration, and contract suites under `tests/`  
**Target Platform**: Linux/macOS server (single HTTP binary)  
**Project Type**: Web backend with server-rendered templates + HTMX progressive enhancement  
**Performance Goals**: Preserve existing message ingest throughput; profile/org/event/collab CRUD feels responsive (typical actions complete without noticeable delay)  
**Constraints**: No additional per-message writes or triggers; additive tables only; indexing required on new relation keys; pages must handle loading/empty/error states  
**Scale/Scope**: High-throughput message writes; low-frequency editorial writes; moderate read traffic for browsing profiles/organizations

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Quality**: Plan keeps scope cohesive; docs updated alongside behavior changes. **Status: PASS**
- **Testing**: Add failing-first unit/integration/contract coverage for new CRUD and linking rules. **Status: PASS**
- **UX**: Pages follow existing patterns and document loading/empty/error states and accessibility. **Status: PASS**
- **Performance**: Explicitly protect ingestion: no new per-message writes; ensure new relation queries are index-backed. **Status: PASS**
- **Observability**: Add logs + counters for CRUD/link failures; no telemetry changes required for ingest path. **Status: PASS**

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)
<!--
  ACTION REQUIRED: Replace the placeholder tree below with the concrete layout
  for this feature. Delete unused options and expand the chosen structure with
  real paths (e.g., apps/admin, packages/something). The delivered plan must
  not include Option labels.
-->

```text
cmd/server/
internal/
  config/
  http/
    handlers/
    templates/
  ingestion/
  observability/
  repository/
  search/
  services/
tests/
  contract/
  integration/
  unit/
specs/004-profiles-orgs-collabs/
```

**Structure Decision**: Single Go backend binary with server-rendered templates. This feature adds repository/service/handler/template code under `internal/` and adds tests under existing `tests/` suites.

## Complexity Tracking

> No constitutional violations requiring justification at this time.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | n/a | n/a |
