# Quickstart: Profiles, Organizations & Collaborations

## Goal

Allow an operator to curate and maintain:
- Profiles that represent a person/company and link multiple Twitch channels
- Organizations that contain profiles
- Events that represent curated time-bounded collaborations
- Collaborations that represent ongoing/ad-hoc relationships

## Assumptions

- The system already has Channels ingested/known (from existing channel management).
- Operator/admin screens exist for creating and editing entities.

## Manual Verification Checklist

### Profiles

1. Create a profile with a name and description.
2. Link two existing channels to that profile.
3. Verify the profile detail page shows both channels.
4. Try linking one of those channels to a different profile.
   - Expected: blocked with a clear message.

### Organizations

1. Create an organization with a name and description.
2. Add the profile as a member.
3. Verify:
   - Organization shows the profile in its members list.
   - Profile shows the organization in its affiliations list.

### Events

1. Create an event with title, description, start date/time, and a participant profile.
2. Verify the event appears on the participant profile page.
3. Try saving an event where end date/time is before start date/time.
   - Expected: blocked with a clear message.

### Collaborations

1. Create a collaboration with a name/description.
2. Add at least two participant profiles.
3. Verify the collaboration appears on each participant profile page.

## Throughput Safety Check

- Confirm chat ingestion and message browsing behave unchanged while using the new profile/org/event/collaboration UI.
- Confirm no new per-message writes were introduced as part of this feature.
