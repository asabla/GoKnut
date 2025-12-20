# Research: Profiles, Organizations & Collaborations

**Feature**: `specs/004-profiles-orgs-collabs/spec.md`
**Date**: 2025-12-20

## Goal

Add Profiles, Organizations, Collaborations, and curated Events while preserving the existing high-throughput message ingestion pipeline.

## Key Decisions

### Decision 1: Keep ingestion hot path unchanged

**Decision**: Do not change how messages are ingested or stored in the `messages` table for this feature.

**Rationale**:
- Message ingestion is the system’s highest-throughput write path.
- Adding joins/triggers/extra writes per message would risk throughput and latency.
- Profiles/organizations/events are low-frequency editorial data and should not impact message inserts.

**Alternatives considered**:
- Storing profile/org IDs directly on each message (rejected: adds per-message write complexity and migration/backfill cost).
- Adding triggers on `messages` to infer collaborations/events (rejected: violates “manual only” and risks throughput).

### Decision 2: Use normalized “link tables” with strict uniqueness

**Decision**: Represent relationships using dedicated link entities:
- Channel ↔ Profile via `profile_channels`, with a uniqueness rule enforcing “channel belongs to at most one profile”.
- Profile ↔ Organization via `organization_members`.
- Profile ↔ Event via `event_participants`.
- Profile ↔ Collaboration via `collaboration_participants`.

**Rationale**:
- Relationships are many-to-many except channel→profile (many channels per profile, but one profile per channel).
- Link tables allow indexed lookups without touching the message write path.

**Alternatives considered**:
- Storing arrays on the parent entity (rejected: harder to query and update; weaker integrity).

### Decision 3: Extend existing schema via additive tables only

**Decision**: Add new tables for profiles/orgs/events/collaborations and link tables; avoid changing existing core tables (`messages`, `channels`, `users`).

**Rationale**:
- Minimizes risk to ingestion performance and existing screens.
- Additive tables are easy to roll out and can coexist with existing data.

**Alternatives considered**:
- Renaming/remodeling channels/users (rejected: broad refactor with no direct user value for this feature).

### Decision 4: Read paths may join; keep joins indexed and bounded

**Decision**: Profile/organization/event/collaboration pages may query across link tables, but these queries must be index-backed and sized for interactive use.

**Rationale**:
- Editorial and browse flows are read-heavy but do not have the same sustained write throughput as ingestion.
- Index-backed joins on small link tables are cheap compared to modifying message insert flow.

**Alternatives considered**:
- Precomputing denormalized “member list” strings (rejected: introduces data duplication and update complexity).

## Performance Notes (High Throughput)

- No additional writes are added to message ingestion beyond what already exists.
- Any new foreign key / trigger behavior must be limited to editorial tables; no triggers on `messages` are introduced.
- New queries must prefer lookups by IDs and use indexes on all join keys.

## Observability Notes

- New create/edit/link actions should emit structured logs with entity IDs and outcomes.
- Add lightweight counters for CRUD actions and error counts.
- No new telemetry is required for the ingestion path.
