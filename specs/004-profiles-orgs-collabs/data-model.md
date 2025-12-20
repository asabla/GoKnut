# Data Model: Profiles, Organizations & Collaborations

**Feature**: `specs/004-profiles-orgs-collabs/spec.md`

## Existing Entities (unchanged)

- **Channel**: Represents a Twitch channel, already persisted and referenced by messages.
- **User**: Represents a chat user, already persisted and referenced by messages.
- **Message**: High-throughput message stream; unchanged for this feature.

## New Entities

### Profile

Represents a real-world person or company.

Recommended attributes:
- `id`
- `name` (display name)
- `description` (optional)
- `created_at`, `updated_at`

Relationships:
- Has 0..N linked channels (via ProfileChannel)
- Has 0..N organization memberships (via OrganizationMember)
- Has 0..N event participations (via EventParticipant)
- Has 0..N collaboration participations (via CollaborationParticipant)

### ProfileChannel (channel ↔ profile link)

Links a channel to exactly one profile (enforces “a channel belongs to at most one profile”).

Recommended attributes:
- `profile_id`
- `channel_id`
- `created_at`

Constraints:
- `channel_id` must be unique across this table.

Indexes:
- Lookup by `profile_id`
- Lookup by `channel_id` (unique)

### Organization

Represents an affiliation grouping.

Recommended attributes:
- `id`
- `name`
- `description` (optional)
- `created_at`, `updated_at`

Relationships:
- Has 0..N members (profiles) via OrganizationMember

### OrganizationMember (profile ↔ organization link)

Recommended attributes:
- `organization_id`
- `profile_id`
- `created_at`

Constraints:
- Unique per `(organization_id, profile_id)`.

Indexes:
- Lookup by `organization_id`
- Lookup by `profile_id`

### Event

Manually curated collaboration with time bounds.

Recommended attributes:
- `id`
- `title`
- `description` (optional)
- `start_at`
- `end_at` (optional)
- `created_at`, `updated_at`

Validation:
- If `end_at` is set, it must be greater than or equal to `start_at`.

Relationships:
- Has 1..N participants (profiles) via EventParticipant

### EventParticipant (profile ↔ event link)

Recommended attributes:
- `event_id`
- `profile_id`
- `created_at`

Constraints:
- Unique per `(event_id, profile_id)`.

Indexes:
- Lookup by `event_id`
- Lookup by `profile_id`

### Collaboration

Represents an ongoing or ad-hoc collaboration between profiles.

Recommended attributes:
- `id`
- `name`
- `description` (optional)
- `shared_chat` (optional flag)
- `created_at`, `updated_at`

Relationships:
- Has 2..N participants (profiles) via CollaborationParticipant

### CollaborationParticipant (profile ↔ collaboration link)

Recommended attributes:
- `collaboration_id`
- `profile_id`
- `created_at`

Constraints:
- Unique per `(collaboration_id, profile_id)`.

Indexes:
- Lookup by `collaboration_id`
- Lookup by `profile_id`

## Deletion Rules (Business-Level)

- Deleting a Profile should be blocked if it would orphan bindings unintentionally, or it should remove its link rows (membership/participants) when explicitly confirmed.
- Deleting an Organization should remove memberships.
- Deleting an Event should remove participants.
- Deleting a Collaboration should remove participants.
- Deleting a Channel should remove its ProfileChannel link.

## High-Throughput Constraint

These new tables must not add any writes or triggers to the message ingestion hot path. All ingestion continues to only write `messages` (and its existing supporting updates).