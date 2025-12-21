# Phase 1 Contracts: Statistics-Centric Startpage

## Overview

The start page remains `GET /` (server-rendered HTML). It includes HTMX-polled containers that update KPI values and diagrams without a full page reload.

All endpoints return HTML fragments (not JSON) to match the current server-rendered + HTMX approach.

## Routes

### `GET /` (home)

- Returns the full start page.
- Contains:
  - Shortcut links to `/channels`, `/users`, `/messages`
  - A dashboard section that includes HTMX containers for live refreshing.

### `GET /dashboard/home/summary`

- Returns an HTML fragment containing:
  - KPI tiles (messages/channels/enabled/users)
  - A "Last updated" timestamp
- Refresh cadence: requested by client at 30–60 seconds.

**Success (200)**: HTML fragment

**Degraded (200)**: HTML fragment with one or more values replaced by placeholders (e.g. `—`) and a small "stats unavailable" indicator.

### `GET /dashboard/home/diagrams`

- Returns an HTML fragment containing:
  - Diagram A: Ingestion activity sparkline (SVG)
  - Diagram B: Dropped messages sparkline (SVG)
  - Each diagram includes its time window label (e.g. "Last 15m")

**Success (200)**: HTML fragment

**No data yet (200)**: HTML fragment with empty-state diagram sections.

**Error (200)**: HTML fragment with error states (keeps page layout stable).

## Notes

- Polling uses HTMX attributes (`hx-get`, `hx-trigger`, `hx-swap`) on the home page.
- Do not require SSE for the home page after this feature.
