# Performance Validation Runbook

This document provides procedures for validating that GoKnut meets its performance targets.

## Performance Targets

| Component | Target | Acceptable Range |
|-----------|--------|------------------|
| Ingestion throughput | 100-150 msgs/sec | 80-200 msgs/sec |
| Live UI latency | < 1s | < 1.5s |
| HTTP p95 latency | < 250ms | < 300ms |
| HTTP p99 latency | < 500ms | < 600ms |
| Batch flush latency | < 100ms | < 150ms |
| Memory growth | < 100MB sustained | < 150MB |

## Pre-Validation Checklist

- [ ] Clean database state (or known baseline)
- [ ] Application running with production-like configuration
- [ ] SQLite WAL mode enabled (verify with `PRAGMA journal_mode;`)
- [ ] Monitoring/logging enabled for metrics collection
- [ ] No other resource-intensive processes running

## Validation Procedures

### 1. Ingestion Throughput Test

**Objective**: Verify the system can ingest 100-150 messages/second.

**Setup**:
```bash
# Start the server with default batch settings
export TWITCH_USERNAME=test_user
export TWITCH_OAUTH_TOKEN=oauth:test_token
go run ./cmd/server --db-path=./perf_test.db --batch-size=100 --flush-timeout=100
```

**Test Procedure**:
1. Join 2-3 active Twitch channels with high chat volume
2. Monitor logs for batch processing metrics:
   ```json
   {"msg":"stored message batch","count":100,"latency_ms":12}
   ```
3. Calculate messages/second from batch counts over a 5-minute window

**Metrics to Collect**:
- Total messages ingested
- Number of batches processed
- Average batch latency
- Dropped message count

**Pass Criteria**:
- Sustained throughput >= 100 msgs/sec
- Average batch latency < 100ms
- Dropped message rate < 1%

### 2. Live UI Latency Test

**Objective**: Verify new messages appear in the UI within 1 second.

**Setup**:
1. Open a channel's live view in the browser
2. Have the channel receiving active messages

**Test Procedure**:
1. Note the timestamp when a message is sent in Twitch chat
2. Note when the message appears in the GoKnut UI
3. Repeat 20 times across different message volumes

**Manual Measurement**:
```
Message send time: T0
UI appearance time: T1
Latency = T1 - T0
```

**Pass Criteria**:
- p95 of latencies < 1s
- p99 of latencies < 1.5s

**Factors Affecting Latency**:
- HTMX polling interval (500-1000ms)
- Batch flush timeout (100ms default)
- Network round-trip time

### 3. HTTP Endpoint Latency Test

**Objective**: Verify HTTP endpoints meet latency SLOs.

**Test Endpoints**:
| Endpoint | Method | Expected Latency |
|----------|--------|------------------|
| `/healthz` | GET | < 10ms |
| `/channels` | GET | < 100ms |
| `/channels/{id}` | GET | < 150ms |
| `/channels/{id}/messages/stream` | GET | < 100ms |
| `/search/messages?q=test` | GET | < 250ms |
| `/users?q=username` | GET | < 200ms |

**Using curl for measurement**:
```bash
# Time a single request
curl -w "@-" -o /dev/null -s "http://localhost:8080/healthz" <<'EOF'
    time_total:  %{time_total}s\n
EOF

# Repeat 100 times and calculate percentiles
for i in {1..100}; do
  curl -w "%{time_total}\n" -o /dev/null -s "http://localhost:8080/channels"
done | sort -n | awk '
  {a[NR]=$1}
  END {
    print "p50:", a[int(NR*0.5)]
    print "p95:", a[int(NR*0.95)]
    print "p99:", a[int(NR*0.99)]
  }
'
```

**Pass Criteria**:
- p95 < 250ms for all endpoints
- p99 < 500ms for all endpoints

### 4. Search Performance Test

**Objective**: Verify search queries complete within latency budget.

**Test Queries** (run after database has significant data):
```bash
# Message search with term
curl "http://localhost:8080/search/messages?q=hello&page=1&page_size=50"

# Message search with filters
curl "http://localhost:8080/search/messages?q=hello&channel_id=1&user_id=2"

# Message search with time range
curl "http://localhost:8080/search/messages?q=hello&start=2025-01-01&end=2025-12-31"

# User search
curl "http://localhost:8080/users?q=user&page=1&page_size=50"
```

**Metrics**:
- Query execution time
- Result count
- FTS vs LIKE path used
- Filter combinations applied

**Pass Criteria**:
- FTS queries: < 100ms for 1M messages
- LIKE queries: < 250ms for 100K messages
- Filtered queries: < 150ms (additional filter processing)
- Empty result queries: < 50ms

**Observability**:
Search operations emit structured logs with the following fields:
- `query`: Search term
- `channel_id`, `user_id`: Applied filters
- `results`: Number of results returned
- `total`: Total matching results
- `latency_ms`: Query execution time

Example log entry:
```json
{"level":"info","msg":"message search completed","query":"hello","channel_id":1,"results":20,"total":156,"latency_ms":45}
```

### 5. Memory Stability Test

**Objective**: Verify no sustained memory growth beyond budget.

**Procedure**:
1. Start server with memory profiling:
   ```bash
   go run ./cmd/server --db-path=./perf_test.db 2>&1 | tee server.log &
   ```
2. Record initial memory usage
3. Run ingestion for 30 minutes with active channels
4. Record memory every 5 minutes

**Memory Measurement**:
```bash
# Get memory of running process
ps -o rss= -p $(pgrep -f "cmd/server") | awk '{print $1/1024 "MB"}'
```

**Pass Criteria**:
- Memory growth < 100MB over 30 minutes of active ingestion
- No monotonic increase indicating leaks

### 6. Database Size and Query Performance

**Objective**: Verify SQLite performance at scale.

**Check database stats**:
```bash
sqlite3 ./twitch.db <<EOF
SELECT 'Messages', COUNT(*) FROM messages;
SELECT 'Users', COUNT(*) FROM users;
SELECT 'Channels', COUNT(*) FROM channels;
SELECT 'DB Size', page_count * page_size / 1024 / 1024 || ' MB' FROM pragma_page_count(), pragma_page_size();
EOF
```

**Query plan analysis**:
```bash
sqlite3 ./twitch.db "EXPLAIN QUERY PLAN SELECT * FROM messages WHERE channel_id = 1 ORDER BY sent_at DESC LIMIT 100;"
```

**Pass Criteria**:
- Queries use indexes (no full table scans for common operations)
- Recent messages query < 50ms
- FTS search query < 100ms

## Troubleshooting

### High Batch Latency
- Check SQLite WAL mode is enabled
- Verify no disk I/O contention
- Consider reducing batch size

### Memory Growth
- Check for unclosed database connections
- Review channel/user cache sizes
- Profile with `go tool pprof`

### Slow Search Queries
- Verify FTS5 table is populated
- Check query uses FTS vs LIKE
- Consider adding indexes

## Reporting Template

```markdown
# Performance Validation Report

**Date**: YYYY-MM-DD
**Version**: v0.x.x
**Tester**: Name

## Environment
- OS: 
- Go version:
- CPU:
- RAM:
- Disk type:

## Results

| Test | Target | Result | Status |
|------|--------|--------|--------|
| Ingestion throughput | 100-150 msg/s | X msg/s | PASS/FAIL |
| Live UI latency p95 | < 1s | Xs | PASS/FAIL |
| HTTP p95 latency | < 250ms | Xms | PASS/FAIL |
| HTTP p99 latency | < 500ms | Xms | PASS/FAIL |
| Memory growth (30min) | < 100MB | XMB | PASS/FAIL |

## Notes
- 
```
