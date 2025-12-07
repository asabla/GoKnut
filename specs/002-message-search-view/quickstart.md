# Quickstart: Message Search View

## Getting Started

1) **Start the server**
```bash
go run ./cmd/server
```
- Ensure SQLite database is present/migrated (existing setup).

2) **Access search UI**
- Open `http://localhost:8080/messages` in your browser.
- Enter a keyword (min 2 chars) in the search box and press Enter or click Search.

## Features

### Text Search
- Enter any search term (minimum 2 characters).
- Results are highlighted with matching terms.
- Results are ordered by newest first.

### Filters
Expand "Advanced Filters" to use:
- **Channel ID**: Limit results to a specific channel.
- **User ID**: Limit results to messages from a specific user.
- **From Date / To Date**: Limit results to a time range (inclusive).

Example URL with all filters:
```
/messages?q=hello&channel_id=1&user_id=2&start=2025-12-01&end=2025-12-07
```

### Navigation
- Click on a username to view their profile.
- Click on a channel name to view that channel's live view.
- Use pagination controls to navigate through results.

## Validation

- **Query too short**: Shows an error if query is less than 2 characters.
- **Invalid time range**: Shows an error if end date is before start date.
- All filter values are preserved when errors occur.

## API Usage

Send GET requests with `Accept: application/json` header:
```bash
curl -H "Accept: application/json" \
  "http://localhost:8080/messages?q=hello&page=1&page_size=20"
```

Response includes:
- `Messages`: Array of message results with highlighted text
- `TotalCount`, `Page`, `TotalPages`: Pagination metadata
- `HasNext`, `HasPrev`: Navigation flags

## Performance

- Pagination keeps responses small (default 20 items per page, max 100).
- Target latency: HTTP p95 ≤250ms, p99 ≤500ms.
- FTS5 full-text search is used when available for fast queries.

## Troubleshooting

- **No results**: Try broader search terms or remove filters.
- **Slow queries**: Ensure FTS5 is enabled; check database size.
- **Time range errors**: Ensure end date is on or after start date.
