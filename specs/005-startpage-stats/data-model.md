# Phase 1 Data Model: Statistics-Centric Startpage

## Overview

This feature introduces a dashboard-style start page composed of:

- A KPI snapshot (totals + last updated)
- Two small time-series diagrams rendered as SVG

No new database tables are introduced.

## Entities

### HomeDashboardSnapshot

Point-in-time values suitable for display in KPI tiles.

- `TotalMessages` (int64) – DB aggregate
- `TotalChannels` (int64) – DB aggregate
- `EnabledChannels` (int64) – DB aggregate
- `TotalUsers` (int64) – DB aggregate
- `LastUpdated` (time) – time the snapshot was produced
- `PartialErrors` (optional) – per-field errors, rendered as degraded UI

### TimeSeries

A bounded series used to render diagrams. Values should be normalized to a numeric sequence.

- `Window` (duration) – e.g. 15m
- `Step` (duration) – e.g. 30s
- `Points` ([]Point)
- `Source` (string) – e.g. "prometheus"

### Point

- `T` (time) – sample timestamp
- `V` (float64) – value at that time

## Diagram Series Definitions

### Diagram A: Activity

- Title: "Ingestion activity"
- Values: messages ingested per step (Prometheus range query results; increase per step)

### Diagram B: Reliability

- Title: "Dropped messages"
- Values: dropped messages per step (Prometheus range query results; increase per step)

## Storage / Persistence

- Dashboard snapshot is derived per request (no persistence).
- Time series values are fetched per request from Prometheus (no local persistence).
  - If Prometheus is unavailable/slow, the diagrams widget renders a degraded/unavailable state while keeping the page layout intact.
