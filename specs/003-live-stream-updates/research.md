# Research: Live Stream Updates

## Technical Unknowns → Resolutions

- **Transport choice**: Prefer WebSocket (bidirectional, flexible envelopes) with graceful fallback to existing HTMX polling when unavailable. SSE considered simpler but less flexible for future bidirectional needs; keep fallback ready regardless.
- **HTMX integration**: Use HTMX `ws:` capability for live fragments; when absent or disconnected, retain current polling endpoints as backup.
- **Ordering/dedup**: Use `messages.id` as the stream cursor (monotonic per insertion) to ensure chronological order and deduplicate on reconnect.
- **Counts authority**: Use repository counts already maintained by triggers; stream count deltas or refreshed totals per view.
- **Backpressure**: Bound in-memory queues per-connection; drop/close with status when limits exceeded; rely on catch-up via `after_id` cursor on reconnect.
- **Reconnect UX**: Client shows status (connected/idle/reconnecting/fallback); retry with backoff; on reconnect, request from last seen cursor to avoid gaps.

## Decisions

- **Primary transport**: WebSocket using `nhooyr.io/websocket` (matches plan). SSE kept as a future alternative if complexity grows, but not primary.
- **Envelope**: Typed messages per view (`metrics`, `message`, `channel_count`, `user_count`, `user_profile`, `status/error`) with `cursor` and `sent_at` where applicable.
- **Catch-up strategy**: On reconnect, server accepts `after_id`/`cursor` to send backlog in bounded batches; if backlog too large, instruct client to hard refresh or fall back to polling.
- **Degradation**: If live not available or closed due to limits, surface status and revert to existing polling endpoints without blocking navigation.
- **Observability**: Emit connect/disconnect/error metrics and structured logs (reason, view, counts delivered, queue drops) for each handler.

## Open Questions (keep visible)

- Max backlog/batch size on reconnect before forcing fallback? (proposal: cap to protect memory/latency)
- Preferred reconnect backoff timings (e.g., start at 1s, cap at ~30s)?
- Any auth/ACL expected for live endpoints? Current app is open.

## Risks & Mitigations

- **High-throughput bursts**: Risk of queue growth → bound queues, drop with explicit status, rely on replay via cursor.
- **Client compatibility**: HTMX `ws:` must be available; ensure polling fallback documented and tested.
- **Schema reliance on `id` ordering**: Assumes insert order matches desired stream order; if ingestion reorders, consider `sent_at` secondary sort on catch-up.
- **Resource limits**: Too many concurrent connections → enforce max connections per view or global; surface graceful errors.
