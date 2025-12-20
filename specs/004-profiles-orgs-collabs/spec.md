# Feature Specification: Profiles, Organizations & Collaborations

**Feature Branch**: `004-profiles-orgs-collabs`  
**Created**: 2025-12-20  
**Status**: Draft  
**Input**: User description: "We need support for profiles and groups/organizations. Where multiple twitch accounts belong to the same person/company. But then there is also colaborations, where mutliple twitch accounts can have a continous colaboration or for that matter an event.

Meaning. We need to support the following scenarions:
- Profile were we can connect one or more twitch accounts, this profile will in the future support more metadata, descriptions and other sources (like youtube etc).
- Events, were one or more profiles are having a colaboration about something. It can have a start and end date. It needs to support metadata, description all that. This will not be created automatically by manually.
- Colaborations, were one or more profiles are having some sort of colaboration. Like a spontanous stream together where the chat could be shared or not.
- Organizations, were one or more channels can be part of an organization. It also needs to support metadata, description and what not."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Maintain Profiles (Priority: P1)

As an operator, I can create a Profile that represents a person or company and connect one or more Twitch accounts (channels) to it, so the system can treat multiple channels as the same real-world identity.

**Why this priority**: Profiles are the foundation for organizations, events, and collaborations.

**Independent Test**: Can be fully tested by creating a profile, linking channels, and verifying that profile views show all linked channels.

**Acceptance Scenarios**:

1. **Given** an operator is on “Create Profile”, **When** they provide a profile name and save, **Then** the profile is created and is viewable.
2. **Given** an existing profile and an existing channel, **When** the operator links the channel to the profile, **Then** the channel appears as part of that profile in the UI.
3. **Given** a channel is already linked to a different profile, **When** an operator attempts to link it to a new profile, **Then** the system prevents ambiguous linking and provides a clear resolution path (e.g., unlink first).

---

### User Story 2 - Maintain Organizations (Priority: P2)

As an operator, I can create an Organization and associate profiles/channels with it, so viewers can understand affiliations (teams, companies, networks).

**Why this priority**: Organizations are a core grouping concept distinct from a single person/company profile.

**Independent Test**: Can be tested by creating an organization, adding members, and verifying membership is visible on both the organization and profile/channel views.

**Acceptance Scenarios**:

1. **Given** an operator is on “Create Organization”, **When** they enter a name and save, **Then** the organization is created and viewable.
2. **Given** an organization and a profile, **When** the operator adds the profile as a member, **Then** the organization shows the profile as a member and the profile shows the organization as an affiliation.

---

### User Story 3 - Curate Events (Priority: P3)

As an operator, I can manually create an Event that involves one or more profiles, includes descriptive metadata, and has a start date and optional end date, so collaborations can be curated and discovered.

**Why this priority**: Events are intentionally curated and time-bounded (or time-scoped) and must not rely on automatic detection.

**Independent Test**: Can be tested by creating an event with participants and dates, then verifying it appears on each participant’s page.

**Acceptance Scenarios**:

1. **Given** an operator is creating an event, **When** they specify a title, at least one participating profile, and a start date, **Then** the event is saved and visible on the event page.
2. **Given** an event with an end date, **When** the operator saves with an end date earlier than the start date, **Then** the system rejects the change with a clear validation message.
3. **Given** an event with multiple participating profiles, **When** a viewer visits any participant profile page, **Then** the event is shown as an associated event.

---

### User Story 4 - Record Collaborations (Priority: P4)

As an operator, I can create a Collaboration that links one or more profiles (e.g., an on-going co-stream relationship) and optionally annotate it (e.g., “shared chat”), so recurring or ad-hoc collaborations can be represented separately from scheduled Events.

**Why this priority**: Collaborations capture an ongoing or ad-hoc relationship that may not be a single curated event.

**Independent Test**: Can be tested by creating a collaboration, adding participants, and verifying it displays on participant profile pages.

**Acceptance Scenarios**:

1. **Given** an operator is creating a collaboration, **When** they specify a name and at least two participating profiles, **Then** the collaboration is saved and viewable.
2. **Given** a collaboration, **When** an operator removes a participant, **Then** the participant no longer shows that collaboration.

---

### Edge Cases

- A profile has zero linked channels after unlinking: profile remains but shows “no linked channels”.
- The same channel is proposed to be linked to multiple profiles: system prevents ambiguity.
- An organization has a single member or no members: organization remains viewable with empty state.
- Event end date omitted: event is treated as ongoing/open-ended.
- Duplicate participant added to an event/collaboration: the system prevents duplicates.
- Deleting something that is referenced elsewhere (profile/organization/event/collaboration): the system prevents accidental data loss or requires explicit confirmation.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST support creating, editing, viewing, and deleting Profiles.
- **FR-002**: A Profile MUST support linking one or more Twitch channels.
- **FR-003**: A channel MUST be linkable to at most one Profile at a time.
- **FR-004**: The system MUST support adding and editing Profile metadata including at least: display name and description.
- **FR-005**: The system MUST support creating, editing, viewing, and deleting Organizations.
- **FR-006**: An Organization MUST support metadata including at least: name and description.
- **FR-007**: The system MUST support associating Profiles with Organizations (organization membership).
- **FR-008**: The system MUST show organization affiliations on Profile views and show members on Organization views.
- **FR-009**: The system MUST support creating, editing, viewing, and deleting Events.
- **FR-010**: An Event MUST have: title, description, participants (one or more Profiles), start date/time, and optional end date/time.
- **FR-011**: The system MUST validate Event dates such that if an end date/time is provided it is not earlier than the start date/time.
- **FR-012**: Events MUST be created and maintained manually by operators (not automatically created by ingestion or detection).
- **FR-013**: The system MUST support creating, editing, viewing, and deleting Collaborations.
- **FR-014**: A Collaboration MUST include: name, description, and participants (two or more Profiles).
- **FR-015**: A Collaboration MUST support optional metadata flags (e.g., whether chat is shared).
- **FR-016**: Profile pages MUST surface linked channels, organization affiliations, associated events, and collaborations.
- **FR-017**: The system MUST provide clear empty, loading, and error states for all pages and forms introduced by this feature.

#### Acceptance Criteria (System-Level)

- **AC-001**: Creating a profile with a name succeeds and is viewable immediately.
- **AC-002**: Linking a channel to a profile shows that channel on the profile page.
- **AC-003**: Linking a channel that is already linked to a different profile is blocked with a clear message.
- **AC-004**: Creating an organization with a name succeeds and is viewable immediately.
- **AC-005**: Adding a profile to an organization shows the membership on both entities.
- **AC-006**: Creating an event requires at least one participating profile and a start date/time.
- **AC-007**: Saving an event with an end date/time before the start date/time is blocked with a clear message.
- **AC-008**: Creating a collaboration requires at least two participating profiles.

#### Assumptions

- “Channel” refers to a Twitch account/channel already known to the system.
- Operator access to administrative create/edit/delete screens already exists.
- Profiles are the canonical “real-world identity”; organizations represent affiliations across multiple profiles.
- Collaboration and Event are distinct concepts: Events are curated with dates; Collaborations represent an ongoing/ad-hoc relationship and may have optional metadata such as “shared chat”.

#### Dependencies

- Each channel can be uniquely identified so it can be linked consistently over time.
- Existing profile/channel pages can be extended to surface affiliations (organizations, events, collaborations).

#### Out of Scope

- Automatic creation of events/collaborations based on stream detection.
- Non-Twitch account linking (e.g., YouTube) beyond reserving metadata fields.
- Permission/role redesign (assumes an existing operator/admin capability).

### Non-Functional Requirements (Constitutional)

- **NFR-QUALITY**: Changes remain small and cohesive; documentation reflects behavior changes.
- **NFR-TESTING**: Automated coverage exists for new and changed behavior.
- **NFR-UX**: Pages and forms are accessible and include clear loading/empty/error states.
- **NFR-PERF**: Typical user actions (create/edit/link) feel responsive, with no noticeable delays under expected usage.
- **NFR-OBS**: Failures and key actions can be diagnosed using recorded operational data.

### Key Entities *(include if feature involves data)*

- **Profile**: Represents a person or company; has metadata (name, description) and links to one or more channels.
- **Channel**: Represents a Twitch account/channel; may be linked to at most one profile.
- **Organization**: Represents a group/affiliation (team/company/network); has metadata and has members.
- **Organization Membership**: Links a profile to an organization.
- **Event**: Curated collaboration with metadata and a start date/time and (optional) end date/time; includes one or more participating profiles.
- **Collaboration**: Represents an ongoing or ad-hoc relationship between two or more profiles; includes metadata (e.g., shared chat flag).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An operator can create a Profile and link two channels in under 2 minutes.
- **SC-002**: An operator can create an Organization and attach three profiles in under 2 minutes.
- **SC-003**: An operator can create an Event with two profiles and valid dates in under 2 minutes.
- **SC-004**: 100% of attempts to link a channel to more than one profile are blocked with a clear error message.
- **SC-005**: In usability review, at least 90% of test users can find a profile’s linked channels, affiliations, and collaborations without assistance.
