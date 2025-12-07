# Research: Message Search View

## Technical Context Unknowns → Resolutions

- None. All required stack choices are established (Go 1.22 single binary, sqlite+fts5, html/template with HTMX/Tailwind). No new dependencies allowed or needed.

## Best Practices & Patterns (Existing Stack)

- **SQLite FTS5 for messages**: Keep triggers in sync (already present) and scope searches with pagination to control payload sizes; guard query length (≥2 chars) to avoid table scans.
- **Filtering by IDs**: Prefer channel/user identifiers server-side; validate parse errors and ignore invalid IDs rather than failing searches.
- **Time filters**: Parse using `2006-01-02`; expand end-date to end-of-day for inclusive ranges; block end < start with clear error.
- **HTMX + templates**: Use partial templates for results; ensure empty and error states render with guidance; preserve form values in responses.
- **Performance**: Default budgets HTTP p95 ≤250ms/p99 ≤500ms; keep result pages small (e.g., 20–50 items) to meet render ≤2s.
- **Observability**: Log query, filters, counts, latency; emit metrics per search type to monitor relevance and performance.

## Decisions

- **Decision**: Use existing `messages_fts` table and search repository for text search with pagination.
  - **Rationale**: Already provisioned FTS5 + triggers; aligns with no-new-dependency rule.
  - **Alternatives considered**: New index or external search (rejected: violates no-new-deps and unnecessary complexity).

- **Decision**: Validate query length (≥2 chars) and time range (end ≥ start) at handler layer; ignore invalid numeric filters.
  - **Rationale**: Prevents expensive scans and poor UX; ensures predictable behavior.
  - **Alternatives considered**: Hard failing on any invalid filter (rejected: degrades UX for non-critical parse issues).

- **Decision**: Preserve user inputs and render explicit empty/invalid states in HTMX responses.
  - **Rationale**: UX clarity and accessibility; keeps interaction within existing template system.
  - **Alternatives considered**: Generic error pages or silent resets (rejected: obscures cause, breaks flow).

## Open Questions

- None; feature scope is clear per spec and existing architecture.
