# Epic: Smart Agent + N+1 Detection

## Status: IMPLEMENTED

All 4 phases shipped. Agent repo: `gosnag-agent`. Server: `GoSnag`.

| Phase | Where | Status | Commit |
|-------|-------|--------|--------|
| **A** | Agent | Done | `f114db1` — Event parsing + spike protection |
| **B** | Agent | Done | `d6800b7` — SQL breadcrumb aggregation + stripping |
| **C** | Server | Done | `56f8ba2` — Query pattern storage + ingest extraction |
| **D** | Server | Done | `56f8ba2` — N+1 detector background job |

## Overview

The gosnag-agent evolved from a transparent proxy into an event-aware relay with spike protection and SQL query compression. The server analyzes aggregated query data to detect N+1 patterns and surfaces them as regular GoSnag issues.

### What was built

1. **Spike protection** (agent): Per-fingerprint rate limiting — drops identical errors beyond 50/min to protect the server during error storms
2. **Query aggregation** (agent): Compresses 200 individual SQL breadcrumbs into a compact summary (94% payload reduction) before forwarding
3. **Query pattern storage** (server): Extracts `gosnag_query_summary` from events at ingest time and upserts into `query_patterns` table
4. **N+1 detection** (server): Background job every 10 minutes that flags repeated query patterns as warning issues with auto-tagging and auto-resolve

## Architecture (as implemented)

```
App (PHP, sql_queries=true in Sentry SDK)
  |
  POST localhost:9000  ← full payload (200+ breadcrumbs), sub-ms
  |
  v
gosnag-agent
  |-- Parse envelope → extract event JSON
  |-- Fingerprint: extract from event (SDK fingerprint, exception type+value, or message)
  |-- Spike gate: if fingerprint > 50/min → drop, increment counter
  |-- SQL aggregation:
  |     Extract db.sql.query breadcrumbs
  |     Normalize (lowercase, collapse whitespace)
  |     Group by normalized form → {query, count, total_ms, avg_ms, table, hash}
  |     Inject into event.extra.gosnag_query_summary
  |     Strip SQL breadcrumbs from payload (keep http, log, etc.)
  |-- Forward: trimmed event (~94% smaller) to GoSnag server
  |
  v
GoSnag Server
  |-- Ingest: process event normally (issue upsert, alerts, tags, priority)
  |-- n1.ExtractAndStore: read gosnag_query_summary → upsert query_patterns table
  |-- N+1 Detector (background, every 10 min):
  |     ListN1Candidates: queries with avg >= 5/event across >= 3 distinct events
  |     Upsert [N+1] issue (fingerprint: n1:{hash})
  |     Auto-tag: n1:detected, table:{name}, transaction:{path}
  |     Create synthetic event with detection details
  |     Auto-resolve if pattern not seen in 24h
  |
  v
Dashboard: [N+1] issues appear alongside normal errors
  → Works with alerts, Jira, priority rules, assignment
```

## Agent Implementation

### Packages

| Package | File | Purpose |
|---------|------|---------|
| `envelope` | `parse.go` | Parse Sentry store/envelope payloads, extract fingerprints. Handles gzip/deflate decompression. Light parsing (only fingerprint fields, not full event). |
| `spike` | `gate.go` | Per-fingerprint sliding window rate limiter. Thread-safe. Configurable threshold + window. Periodic cleanup of expired entries. |
| `transform` | `queries.go` | Extract `db.sql.query` breadcrumbs, normalize queries, aggregate into `QuerySummary`, inject into `event.extra.gosnag_query_summary`, strip SQL breadcrumbs. Handles both store and envelope payloads. |
| `server` | `server.go` | HTTP handlers — now route /store/ and /envelope/ through spike gate + transform before queuing. |

### Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `GOSNAG_SPIKE_THRESHOLD` | `50` | Max events per fingerprint per window (0 = disabled) |
| `GOSNAG_SPIKE_WINDOW` | `60s` | Sliding window duration |
| `GOSNAG_STRIP_SQL` | `true` | Aggregate SQL breadcrumbs into summary and strip |

YAML:
```yaml
spike:
  threshold: 50
  window: 60s

transform:
  strip_sql: true
```

### Tests: 35 total

- `envelope`: 8 tests — store/envelope parsing, gzip, SDK fingerprints, edge cases
- `spike`: 7 tests — threshold, suppression, independent fingerprints, window reset, cleanup
- `transform`: 8 tests — SQL aggregation, stripping, envelope transform, payload size reduction, table extraction, query normalization
- `server`: 6 tests — health, ingest store/envelope, spike integration, SQL stripping integration, 404
- `buffer`: 3 tests, `forwarder`: 3 tests, `queue`: 3 tests, `wal`: 3 tests

### Payload reduction measured

```
50 identical SQL queries: 7,844 bytes → 454 bytes (94% reduction)
```

## Server Implementation

### Migration 000018: `query_patterns`

```sql
CREATE TABLE query_patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    transaction TEXT NOT NULL DEFAULT '',
    query_hash TEXT NOT NULL,
    normalized_query TEXT NOT NULL,
    table_name TEXT NOT NULL DEFAULT '',
    event_count INT NOT NULL DEFAULT 0,
    distinct_events INT NOT NULL DEFAULT 0,
    total_exec_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    first_seen TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, transaction, query_hash)
);
```

### Packages

| Package | File | Purpose |
|---------|------|---------|
| `n1` | `extract.go` | `ExtractAndStore()` — reads `gosnag_query_summary` from event extra, upserts each entry into `query_patterns`. Called async from ingest pipeline. |
| `n1` | `detector.go` | `Detector.Run()` — background job every 10 min. Lists N+1 candidates per project, upserts warning issues with `n1:` fingerprint prefix, auto-tags, creates synthetic detection events, auto-resolves stale patterns. |

### SQL Queries

| Query | Purpose |
|-------|---------|
| `UpsertQueryPattern` | Insert or increment counters on each event ingest |
| `ListN1Candidates` | Find patterns with avg_per_event >= threshold across enough events |
| `ListQueryPatterns` | Top repeated queries per project (for future API/UI) |
| `CleanupOldQueryPatterns` | Delete entries not seen in 30 days |

### N+1 Detection Rules

A query Q on transaction T is flagged when:
- `distinct_events >= 3` (not a fluke)
- `event_count / distinct_events >= 5` (avg 5+ executions per event)

Output:
- Issue title: `[N+1] select * from "users" where "id" = ? — avg 47/req on POST /api/bookings`
- Level: `warning`
- Platform: `sql`
- Tags: `n1:detected`, `table:users`, `transaction:POST /api/bookings`
- Auto-resolve: if `last_seen > 24h` ago

### Background Workers

| Worker | Interval | Purpose |
|--------|----------|---------|
| `n1Detector.Run` | 10 min | Detect N+1 patterns, create/update issues |
| `queryPatternCleanup` | 24 hours | Delete patterns not seen in 30 days |

## SDK Configuration Required

For PHP/Laravel:
```php
// config/sentry.php
'breadcrumbs' => [
    'sql_queries' => true,
    'sql_bindings' => false,
],
'max_breadcrumbs' => 200,
```

Key insight: enabling `sql_queries` with the agent means zero bandwidth increase to the remote server. The POST goes to localhost, and the agent strips breadcrumbs before forwarding.

## Technical Decisions

| Decision | Rationale |
|----------|-----------|
| Agent compresses, server analyzes | Agent is fast at per-event work (Go, local). Server is good at cross-event aggregation (has all data, PostgreSQL). |
| Drop spikes instead of aggregate_count | Simpler. No server protocol changes. event_count during spikes is approximate — acceptable trade-off. |
| Summary in `event.extra` | No server protocol changes. Standard Sentry servers ignore extra fields. GoSnag checks for `gosnag_query_summary`. |
| N+1 issues as regular issues | Zero new UI. Works with existing alerts, Jira auto-create, priority rules, assignment. |
| Auto-resolve after 24h | Prevents stale detections. If pattern returns, issue reopens via upsert. |
| Fingerprint prefix `n1:` | Easy to identify and filter N+1 issues in auto-resolve logic without separate table. |

## Known Limitations

1. **No alert cooldown**: Alerts fire on every matching event, not just the first. High-velocity issues can flood Slack. Needs a per-issue alert cooldown mechanism.
2. **Agent fingerprint is approximate**: Uses exception type + truncated value, not full stacktrace frames like the server. Sufficient for spike grouping but not identical to server fingerprints.
3. **No per-project spike config**: Threshold is global to the agent. Per-project overrides would need server-side config pull.
4. **N+1 detector is basic**: No stacktrace correlation, no trend tracking, no dedicated UI. Issues appear in normal issue list only.

## Future Work (not designed)

- Alert cooldown per issue (prevents Slack flooding)
- Per-project spike thresholds pulled from server config
- Performance tab UI with transaction ranking by N+1 severity
- Stacktrace correlation for N+1 queries (link to code location)
- Slow query detection (individual queries above time threshold)
- Trend tracking (did the fix reduce query count?)
