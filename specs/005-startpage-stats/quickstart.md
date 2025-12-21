# Quickstart: Statistics-Centric Startpage

## Prerequisites

- Go 1.22
- Docker (optional, for Prometheus/Grafana stack)

## Run the app

1. Start the server (existing workflow):

- `go run ./cmd/server`

2. Open the start page:

- `http://localhost:8080/`

## Run observability stack (optional)

The repo includes a Prometheus/Grafana stack via `docker-compose.yml`.

- `docker compose up -d`

Prometheus should scrape the app `/metrics` endpoint when OTel metrics are enabled.

## Verify behavior

- The start page shows KPI tiles and shortcut links.
- The start page no longer shows a "Latest Messages" list.
- KPI tiles and diagrams update automatically (at least once per minute) without a full page reload.
- If stats/diagram data is unavailable, the start page still renders shortcuts and shows clear empty/error states.
