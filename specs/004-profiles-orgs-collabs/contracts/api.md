# API Contracts: Profiles, Organizations & Collaborations

## Scope

Define user-facing endpoints for creating and viewing:
- Profiles (and linking channels)
- Organizations (and memberships)
- Events (and participants)
- Collaborations (and participants)

Endpoints follow existing conventions in this repository (server-rendered HTML with form posts, compatible with progressive enhancement).

## Conventions

- **Read endpoints**: `GET` return HTML pages.
- **Mutations**: `POST` via forms; on success redirect back to the relevant detail page.
- **Errors**: return the same page with an error banner/message and preserve user input.

## Profiles

- `GET /profiles`
  - Lists profiles.

- `GET /profiles/new`
  - Create profile form.

- `POST /profiles`
  - Creates a profile.

- `GET /profiles/{id}`
  - Profile detail (shows linked channels, organizations, events, collaborations).

- `POST /profiles/{id}`
  - Updates profile metadata (name/description).

- `POST /profiles/{id}/delete`
  - Deletes profile (must be protected against accidental deletion).

### Channel Linking

- `POST /profiles/{id}/channels`
  - Adds a channel link: `{ channel_name }` or `{ channel_id }`.
  - Must enforce: a channel can only be linked to one profile.

- `POST /profiles/{id}/channels/{channel_id}/remove`
  - Removes a channel link.

## Organizations

- `GET /organizations`
- `GET /organizations/new`
- `POST /organizations`
- `GET /organizations/{id}`
- `POST /organizations/{id}`
- `POST /organizations/{id}/delete`

### Membership

- `POST /organizations/{id}/members`
  - Adds profile membership: `{ profile_id }`.

- `POST /organizations/{id}/members/{profile_id}/remove`
  - Removes membership.

## Events

- `GET /events`
- `GET /events/new`
- `POST /events`
- `GET /events/{id}`
- `POST /events/{id}`
- `POST /events/{id}/delete`

### Participants

- `POST /events/{id}/participants`
  - Adds participant: `{ profile_id }`.

- `POST /events/{id}/participants/{profile_id}/remove`
  - Removes participant.

Validation:
- `start_at` required.
- `end_at` optional but must be after or equal to `start_at`.

## Collaborations

- `GET /collaborations`
- `GET /collaborations/new`
- `POST /collaborations`
- `GET /collaborations/{id}`
- `POST /collaborations/{id}`
- `POST /collaborations/{id}/delete`

### Participants

- `POST /collaborations/{id}/participants`
  - Adds participant: `{ profile_id }`.

- `POST /collaborations/{id}/participants/{profile_id}/remove`
  - Removes participant.

Validation:
- Must have at least 2 participants.

## Non-Goals

- No write endpoints for chat ingestion.
- No automatic event/collaboration detection.
