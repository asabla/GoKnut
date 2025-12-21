# Feature Specification: Statistics-Centric Startpage

**Feature Branch**: `[005-startpage-stats]`  
**Created**: 2025-12-21  
**Status**: Draft  
**Input**: User description: "Refactor startpage to be statistic centric, but still have shortcut links to other parts of the application. The goal is to remove the live view of messages and instead transition over to automatically polling and displaying statistics. It should rely on using diagrams. Statistics values should come from were we already collect them (i think grafana). If some values makes more sense coming from the database, then use that. Keep in mind that we're not on spec-001, so make sure you're actually incrementing the spec number properly"

## Clarifications

### Session 2025-12-21

- Q: Where do time-series diagram values come from? → A: Query Prometheus HTTP API (PromQL) for time-series values
- Q: What is the source of truth split for dashboards? → A: Prometheus for diagrams; DB for totals
- Q: How should diagrams behave when Prometheus is slow/unavailable? → A: Soft dependency with strict timeout; return degraded UI (200)
- Q: What time window and sampling step should diagrams use by default? → A: 15m window, 30s step (configurable later)
- Q: Should the spec define exact PromQL for diagrams? → A: Yes; specify metric names + PromQL

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Monitor system activity at a glance (Priority: P1)

A user opens the start page to quickly understand whether the system is ingesting messages, serving queries, and generally “healthy”, without reading a live message feed.

**Why this priority**: This is the core purpose of making the start page statistic-centric.

**Independent Test**: Can be fully tested by loading the start page and verifying that key statistics and diagrams are displayed, refreshed automatically, and have clear loading/empty/error behavior.

**Acceptance Scenarios**:

1. **Given** the user navigates to the start page, **When** the page loads, **Then** the user sees a summary of key statistics and at least one diagram that visualizes system activity over time.
2. **Given** the user keeps the start page open, **When** the refresh interval elapses, **Then** the displayed statistics and diagrams update automatically without requiring a full page reload.
3. **Given** the statistics source is temporarily unavailable, **When** the user opens the start page, **Then** the page clearly indicates that statistics are unavailable and still allows the user to use navigation shortcuts.

---

### User Story 2 - Navigate quickly to primary areas (Priority: P2)

A user uses shortcut links on the start page to jump to commonly used areas (channels, users, message search, etc.) while still using the start page as an “overview dashboard”.

**Why this priority**: Shortcuts preserve fast navigation and prevent the start page from becoming an informational dead-end.

**Independent Test**: Can be tested by verifying the presence of shortcut links and that each link routes to the intended destination.

**Acceptance Scenarios**:

1. **Given** the user is on the start page, **When** the user selects a shortcut link, **Then** they are taken to the corresponding page.
2. **Given** the start page is showing statistics (or is in an error state), **When** the user looks for navigation shortcuts, **Then** shortcuts are still visible and usable.

---

### User Story 3 - No more live message feed on start page (Priority: P3)

A user no longer sees “latest messages” streaming into the start page; instead, the space is used for diagrams and statistics.

**Why this priority**: Removes the previous real-time message browsing workflow from the start page and makes room for dashboards.

**Independent Test**: Can be tested by verifying the absence of a latest-messages list on the start page and confirming the page does not present a live message stream UI.

**Acceptance Scenarios**:

1. **Given** the user opens the start page, **When** the page renders, **Then** no “latest messages” feed is shown.
2. **Given** the user wants to browse messages, **When** they use navigation/shortcuts, **Then** they can still reach message browsing/search features elsewhere in the application.

---

### Edge Cases

- Statistics are partially available (some values present, others missing).
- Statistics are delayed or stale (the page indicates “last updated” and does not mislead users).
- Extremely large values (remain readable; no layout breakage).
- No data yet (fresh environment) renders empty states without errors.
- Background refresh is unavailable (users still see an initial snapshot).
- Statistics refresh returns out-of-order timestamps (the page continues showing the newest snapshot).

## Requirements *(mandatory)*

### Assumptions

- The application already collects operational statistics in a monitoring system and/or can derive basic totals from the database.
- The start page is public to the same audience as the rest of the web UI (no new access control is introduced by this feature).
- “Diagrams” are simple time-series visualizations suitable for quick comprehension (line/bar style), not complex interactive analytics.

### Dependencies

- Availability of a Prometheus HTTP API endpoint for querying time-series statistics via PromQL.
- Availability of database aggregates for totals when the monitoring system is not the source of truth.
- Access to the same monitoring data that operators use today for ingestion/search/HTTP/SSE metrics.

### Out of Scope

- Building new operational dashboards outside of the start page.
- Adding new message browsing functionality to replace the removed live feed.
- Introducing new authentication/authorization behavior.

### Functional Requirements

- **FR-001**: Start page MUST present a compact summary of key system statistics, including at minimum:
  - Total messages archived (DB aggregate)
  - Total channels (DB aggregate)
  - Active/enabled channels (DB aggregate)
  - Total unique users (DB aggregate)

  **Acceptance Criteria**:
  - On initial load, each statistic is visible with a label and a value.
  - In an empty environment, each statistic shows 0 (or equivalent empty representation) rather than erroring.

- **FR-002**: Start page MUST display at least 2 diagrams that visualize trends over time, using a clearly stated time window (default: last 15 minutes, 30s step).

  **Acceptance Criteria**:
  - Diagram A communicates system activity (e.g., message ingestion volume over time).
  - Diagram B communicates reliability or load (e.g., failures/dropped work over time, or request/latency volume over time).
  - Diagram series values are retrieved by querying Prometheus via PromQL (historical window, not just current counters).
  - Each diagram clearly shows its time window (default: last 15 minutes).
  - Optional enhancement: allow the window/step to be configured via app config later.

- **FR-003**: While the start page is open, it MUST refresh its statistics and diagrams automatically at least once per minute.

  **Acceptance Criteria**:
  - Without user interaction, values change (or "last updated" changes) within 60 seconds.

- **FR-004**: Start page MUST show when the dashboard was last updated so users can judge data freshness.

  **Acceptance Criteria**:
  - A "Last updated" timestamp is visible and updates when new data is loaded.

- **FR-005**: Start page MUST provide shortcut links to primary parts of the application (at minimum: channels, users, and messages/search).

  **Acceptance Criteria**:
  - Each shortcut is visible on the start page and navigates to the correct destination.

- **FR-006**: Start page MUST NOT display a live message feed.

  **Acceptance Criteria**:
  - No "latest messages" list, stream, or continuously appended message content is present on the start page.

- **FR-007**: Each statistic displayed on the start page MUST come from the established source of truth for that value.

  **Acceptance Criteria**:
  - KPI totals (FR-001) are derived from database aggregates.
  - Diagram series (FR-002) are derived from Prometheus PromQL queries over the specified time window.

- **FR-008**: When statistics cannot be retrieved (fully or partially), the start page MUST show a clear error/empty state and MUST still render shortcut links.

  **Acceptance Criteria**:
  - If all statistics fail to load, the user sees a clear message indicating the dashboard is unavailable.
  - If some statistics fail to load, the page shows partial data without breaking layout.
  - If Prometheus queries for diagrams fail or exceed the per-request timeout, return a degraded diagrams widget (HTTP 200) that preserves layout and shows an explicit unavailable/timeout state.
  - Shortcuts remain visible and clickable in either case.

### Non-Functional Requirements (Constitutional)

- **NFR-QUALITY**: Code remains small, cohesive, lint/format clean; docs updated with behavior changes.
- **NFR-TESTING**: Automated failing-first coverage for new/changed behavior (unit/integration/contract/regression as applicable).
- **NFR-UX**: Uses design system; validates accessibility (WCAG 2.1 AA) and loading/empty/error states with evidence.
- **NFR-PERF**: Declares and validates budgets; defaults backend p95≤250ms/p99≤500ms, frontend critical render/interaction ≤2s unless otherwise specified.
  - Dashboard polling endpoints MUST enforce a strict timeout for Prometheus queries and return degraded UI rather than blocking beyond the budget.
- **NFR-OBS**: Defines structured logs/metrics/traces for new paths and failure modes; observability plan recorded.

### Key Entities *(include if feature involves data)*

- **Statistics Snapshot**: A point-in-time set of values shown on the start page (includes a "last updated" timestamp).
- **Diagram Series**: A set of time-ordered values used to render a diagram (includes a time window and sampling interval).
- **Dashboard Widget**: A visual container on the start page (KPI tile or diagram) with a title, value(s), and a known empty/error presentation.

### Prometheus Metrics & PromQL (for diagrams)

- **Diagram A (Activity)**
  - Metric: `goknut.ingestion.messages_ingested` (counter)
  - PromQL (rate-ish per step): `increase(goknut_ingestion_messages_ingested_total[30s])`

- **Diagram B (Reliability)**
  - Metric: `goknut.ingestion.dropped_messages` (counter)
  - PromQL (rate-ish per step): `increase(goknut_ingestion_dropped_messages_total[30s])`

**Notes**:
- Prometheus counter names use the `_total` suffix when scraped from OTel/Prometheus.
- Step/window defaults are `30s` / `15m` respectively; step can be adjusted by implementation if Prometheus resolution requires it.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can answer “is the system active/healthy?” from the start page within 10 seconds of loading it, using only the statistics and diagrams shown.
- **SC-002**: While the start page remains open, statistics refresh automatically at least once per minute, and the refresh does not block navigation.
- **SC-003**: In a new/empty environment, the start page still renders and shows meaningful empty states, and diagrams render an empty/no-data state without errors.
- **SC-004**: In a failure scenario where statistics cannot be retrieved, the start page still renders shortcuts and communicates the failure within 2 seconds of page load.
