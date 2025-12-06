# Research – Twitch Chat Archiver & Explorer

## Decisions and Rationale

- Decision: Use Go 1.22 single binary with server-rendered HTMX UI.
  - Rationale: Matches performance and deployment constraints; minimal dependencies; easy single-process ops.
  - Alternatives considered: Node/React SPA (more complexity, JS dependency), Python/Flask (slower throughput), Go + SPA (adds bundling/build burden without need).

- Decision: SQLite with WAL mode for durability and throughput; batching 100–200 msgs or 50–100ms flush.
  - Rationale: Local durable storage, meets 100–150 msgs/sec, simple deploy; WAL reduces writer contention.
  - Alternatives considered: Postgres (overkill/external dependency), BoltDB/buntdb (less SQL/search flexibility), flat files (harder querying/search).

- Decision: HTMX polling (500–1000ms) for live updates.
  - Rationale: Simpler than websockets/SSE; aligns with server-rendered templates; meets ≤1s latency with batched ingestion.
  - Alternatives considered: WebSockets (more infra complexity), SSE (similar but requires long-lived connections), manual refresh (worse UX).

- Decision: FTS5 optional search table alongside LIKE search.
  - Rationale: Enables efficient text search with highlighting; can fall back to LIKE; keeps schema flexible.
  - Alternatives considered: ONLY LIKE (works but slower at scale), external search (adds infra).

- Decision: Structured logging with component fields and metrics hooks for ingestion/search/IRC.
  - Rationale: Supports observability gate; provides failure insights; lightweight with stdlib/log or zerolog compatible.
  - Alternatives considered: Minimal logging (insufficient for troubleshooting), heavy APM (overhead, not needed).

- Decision: Tests: unit (services, ingestion logic), integration (SQLite repos, IRC fake), contract (HTTP handlers/HTMX fragments) with failing-first pattern.
  - Rationale: Satisfies constitution testing gate; covers key behaviors and latency targets.
  - Alternatives considered: Unit-only (misses integration issues), manual testing only (violates gate).

## Resolved Clarifications

No outstanding NEEDS CLARIFICATION items; context derived from spec and reference.
