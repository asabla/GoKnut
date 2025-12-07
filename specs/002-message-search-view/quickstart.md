# Quickstart: Message Search View

1) **Start the server**
- `go run ./cmd/server`
- Ensure SQLite database is present/migrated (existing setup).

2) **Access search UI**
- Open `http://localhost:8080/search/messages`.
- Enter a keyword (min 2 chars) and optional filters (channel ID, user ID, start/end dates).

3) **Interpret results**
- Results show message text with channel, sender, and timestamp.
- Use pagination controls to navigate pages.
- Click channel/message links to jump to source context.

4) **Empty/invalid states**
- Empty queries show the form; no results show an empty state with guidance.
- Invalid time ranges return a clear error; fix inputs and resubmit.

5) **API (JSON)**
- Send GET requests to `/search/messages?q=hello&channel_id=1&user_id=2&start=2025-12-01&end=2025-12-07&page=1&page_size=20` with `Accept: application/json` to receive structured results and pagination metadata.

6) **Performance & observability**
- Pagination keeps responses small; target HTTP p95 ≤250ms/p99 ≤500ms.
- Logs capture query, filters, counts, and latency; metrics record per-search-type timings.
