# API Contracts: Message Search View

## Endpoints

### GET `/messages`
- **Purpose**: Search messages by text with optional filters.
- **Query Params**:
  - `q` (string, required, min 2 chars): text to search within message content.
  - `channel_id` (int, optional): filter to a specific channel.
  - `user_id` (int, optional): filter to a specific sender.
  - `start` (date `YYYY-MM-DD`, optional): inclusive start of ingestion window.
  - `end` (date `YYYY-MM-DD`, optional): inclusive end of ingestion window; end-of-day applied.
  - `page` (int, optional, default 1): page number.
  - `page_size` (int, optional, default 20, max 100): results per page.
- **Responses**:
  - `200 OK` (HTML/JSON): list of messages with `id`, `channel_id`, `channel_name`, `user_id`, `username`, `display_name`, `text`, `sent_at`, `highlighted_text`, pagination metadata. Empty state when no matches.
  - `400 Bad Request`: invalid query (e.g., missing or too-short `q`, end before start).
  - `500 Internal Server Error`: unexpected errors.
- **Notes**: HTMX partials used for progressive enhancement; preserves form values in responses.

### GET `/search/messages` (Legacy - Redirects to `/messages`)
- **Purpose**: Legacy route for backwards compatibility.
- **Behavior**: Permanently redirects (301) to `/messages` with query params preserved.

### GET `/users/{username}/messages`
- **Purpose**: View messages by user with optional channel filter.
- **Query Params**:
  - `channel` (string, optional): channel name filter.
  - `page`, `page_size` as above.
- **Responses**:
  - `200 OK` (HTML/JSON): messages for user with pagination metadata.
  - `400/404/500` as appropriate.
- **Notes**: Reuses existing handler; included for navigation context from search results.

### Navigation from Results
- Each message result should link to its channel or message anchor in the live view to satisfy FR-009; if anchor not yet implemented, link to channel view with timestamp context.

## Error States
- **Empty results**: return 200 with empty state guidance.
- **Invalid time range**: 400 with clear message; avoid server errors.

## Performance & Observability
- Pagination required; default page_size tuned to meet p95 ≤250ms/p99 ≤500ms.
- Log query, filters, counts, and latency; emit metrics per search type.
