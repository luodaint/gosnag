# Epic: GoSnag Agent — Local Edge Relay

## Overview

A lightweight Go binary that runs on each application server as a local Sentry-compatible relay. The SDK points to `localhost` instead of the remote GoSnag server. The agent receives events with sub-millisecond latency, queues them, and forwards asynchronously. It does NOT filter, sample, or suppress events — every event reaches the server. Intelligence stays server-side.

## Problem Statement

1. **Latency**: The Sentry SDK POSTs to a remote server on every error, adding 50-200ms of network latency to the request lifecycle.
2. **Reliability**: If GoSnag is unreachable, the SDK drops events silently. There is no local buffer.
3. **Coupling**: The app's error path depends on an external HTTP call succeeding.

These are the only problems the agent solves in its first iteration. Analysis (N+1, spike protection) is a future concern that requires server-side contract changes first.

## Architecture

```
App (PHP/Node/Python)
  |
  |  POST localhost:9000/api/{project}/store/
  |  POST localhost:9000/api/{project}/envelope/
  |  (sub-ms, never blocks app)
  |
  v
gosnag-agent
  |-- Accept: /store/ and /envelope/ (passthrough)
  |-- Queue: in-memory ring buffer
  |-- Forward: async HTTP POST to GoSnag server
  |-- Retry: exponential backoff on failure
  |
  v
GoSnag Server (unchanged)
```

### Key Properties

- **Zero app changes**: Only change the DSN host to `localhost:9000`
- **Transparent proxy**: Every event is forwarded exactly as received. No filtering, no sampling, no suppression. GoSnag's event_count, velocity, trends, and alerts remain accurate.
- **Fail-fast for the app**: SDK POSTs to localhost — if the agent is down, connection refused in < 1ms. App is never blocked.
- **Single binary**: No runtime dependencies. Deploy via scp, Docker sidecar, or package manager.

### Compatibility Matrix

The agent proxies these envelope item types (matching what GoSnag server handles today):

| Item type | Agent behavior | Server behavior |
|-----------|---------------|-----------------|
| `event` | Forward | Process (upsert issue, create event) |
| `transaction` | Forward | Ignore silently |
| `session` / `sessions` | Forward | Ignore silently |
| `client_report` | Forward | Ignore silently |
| Unknown types | Forward | Ignore silently |

The agent does NOT parse, validate, or transform envelope contents. It forwards the raw HTTP request body with original headers (including `X-Sentry-Auth`, `Content-Type`, `Content-Encoding`).

## Phases

### Phase 1 — Transparent Relay (MVP)

The smallest useful thing: a local proxy that queues and forwards.

| Feature | Description |
|---------|-------------|
| Ingest endpoints | Accept `POST /api/{project}/store/` and `POST /api/{project}/envelope/` |
| Passthrough | Forward raw request body + headers. No parsing. |
| In-memory queue | Ring buffer (default 5000 events). Oldest dropped on overflow. |
| Async forwarding | Goroutine reads from queue, POSTs to GoSnag server. |
| Retry | On HTTP error or timeout: exponential backoff (1s, 2s, 4s, ... max 60s). On 4xx: drop (bad request, no point retrying). |
| Health | `GET /health` returns `{"status":"ok","queue_depth":N,"server_reachable":bool}` |
| Config | CLI flags or env vars: `GOSNAG_SERVER_URL`, `GOSNAG_LISTEN_ADDR` (default `127.0.0.1:9000`), `GOSNAG_BUFFER_SIZE` (default 5000) |
| Logging | Structured JSON logs to stderr. Levels: info, warn, error. |
| Graceful shutdown | On SIGTERM: stop accepting, drain queue (up to 10s timeout), exit. |

**What Phase 1 does NOT include**: disk persistence, spike protection, heartbeat, metrics endpoint, systemd unit, config file. These are explicitly deferred.

**Exit criteria**: Agent proxies events reliably under normal load. If GoSnag goes down for < 5 minutes, no events are lost (within buffer size). App latency is not affected.

### Phase 2 — Disk Persistence & Idempotency

**Prerequisite (server-side)**: Add a UNIQUE constraint on `events(issue_id, event_id)` so duplicate inserts from retry are safely ignored. Migration required.

| Feature | Description |
|---------|-------------|
| Disk WAL | When memory buffer is full, append events to a WAL file on disk (append-only, sequential writes). |
| Max disk usage | Configurable limit (default 100MB). Oldest segments dropped when full. |
| Startup replay | On agent restart, replay disk WAL before accepting new events. |
| Idempotent forwarding | Each forwarded event carries its original `event_id`. Server's UNIQUE constraint on `events(project_id, event_id)` deduplicates retries. Using `project_id` instead of `issue_id` because the same event could be assigned to a different issue if fingerprint grouping or aliases change between retries. |
| Config file | YAML config at `/etc/gosnag-agent/config.yaml` as alternative to env vars. |
| Systemd unit | Service file, runs as `gosnag-agent` user, `Restart=always`. |

**Exit criteria**: Agent survives restarts and GoSnag outages up to disk limit. No duplicate events in the database after retries.

### Phase 3 — Heartbeat & Control Plane

**Prerequisite (server-side)**: New endpoint `POST /api/v1/agents/heartbeat` authenticated by a dedicated agent token (new token type, not DSN key, not user session). New `agent_heartbeats` table. New `GET /api/v1/agents` endpoint for monitoring tools.

| Feature | Description |
|---------|-------------|
| Agent token | New token type in GoSnag: `agent` scope. Created per-project in Settings. Used only for heartbeat, not for ingest (ingest continues using DSN). |
| Heartbeat POST | Every 30s, POST to `/api/v1/agents/heartbeat` with: `agent_id`, `hostname`, `group`, `instance_tag`, `version`, `uptime_seconds`, `stats` (received/forwarded/dropped/queue_depth). |
| Agent status endpoints | Server exposes `GET /api/v1/agents` (list all, filter by group/status), `GET /api/v1/agents/{id}`. Status: online (< 90s), degraded (< 5min), offline (> 5min). |
| Auto-cleanup | Server removes agents offline > 7 days. |
| Auto-replace | New agent with same hostname + different instance_tag marks the old one as replaced (not offline). |

**Auth boundary**: The DSN key authenticates ingest (existing). The agent token authenticates heartbeat (new). These are separate credentials with separate scopes. The heartbeat endpoint is on the management API (`/api/v1/`) behind the agent token, not on the ingest path.

**Exit criteria**: External monitoring tools can poll `/api/v1/agents` to verify all expected agents are online.

### Phase 4 — Spike Protection

**Prerequisites (server-side)**:
1. New ingest endpoint or envelope item type: `aggregate_count`. Payload: `{"fingerprint":"abc","suppressed_count":847,"window_start":"...","window_end":"...","sample_event_id":"..."}`.
2. Server-side handler: on receiving aggregate_count, increment `issues.event_count` by `suppressed_count` WITHOUT creating individual event rows. This preserves accurate counters and alert thresholds while avoiding row explosion.
3. Velocity queries updated to include aggregate counts (or a separate `issue_event_aggregates` table for windowed counts).

Only after these server changes can the agent safely suppress events without breaking counters.

| Feature | Description |
|---------|-------------|
| Fingerprint tracking | Agent computes fingerprint per event (same algorithm as server). Tracks count per fingerprint per sliding window (default 60s). |
| Throttle | When count exceeds threshold (default 50/min per fingerprint): forward first N events normally as full events (these trigger priority, auto-tags, and alerts on the server as usual). Then stop forwarding individual events for the remainder of the window. |
| Last-event forwarding | Forward the LAST event in each window as a full event too (in addition to the first N). This ensures the server's postEventFn (priority evaluator, auto-tagger, condition engine) runs on a recent payload with current event_data, environment, and release. Without this, suppressed windows would have stale automation data. |
| Aggregate flush | At end of window, send one `aggregate_count` message with the suppressed count (excluding the first N + last 1 already forwarded). Server increments `event_count` by that amount WITHOUT creating event rows. |
| Per-project config | Threshold and window configurable per project (pulled from server config or local YAML). |
| Rate limiting | Global safety: max events/second across all fingerprints. Excess dropped with counter included in next aggregate. |

**What this does NOT include**: N+1 detection, breadcrumb analysis, smart sampling. Those are separate features that can layer on top later.

**Exit criteria**: During a spike of 10,000 identical events/min, the agent forwards ~6 full events (first 5 + last 1) + 1 aggregate message. Server counters show 10,000. Priority rules and auto-tags evaluate against the last real event payload. Alerts fire based on accurate velocity. Trends reflect the real volume.

### Phase 5 — Observability & Analysis (Future)

Not designed yet. Potential features, each requiring its own analysis:

- Prometheus `/metrics` endpoint
- N+1 query detection from breadcrumbs
- Smart sampling (keep % of low-priority events)
- Remote config pull from GoSnag server
- Breadcrumb trimming before forwarding

## Configuration

### Phase 1 (env vars only)

```bash
GOSNAG_SERVER_URL=https://sentry.cover-aws.com
GOSNAG_LISTEN_ADDR=127.0.0.1:9000
GOSNAG_BUFFER_SIZE=5000
GOSNAG_LOG_LEVEL=info
GOSNAG_FORWARD_TIMEOUT=10s
```

### Phase 2+ (YAML config)

```yaml
# /etc/gosnag-agent/config.yaml
server_url: https://sentry.cover-aws.com
listen_addr: 127.0.0.1:9000

buffer:
  memory_size: 5000
  disk_path: /var/lib/gosnag-agent/buffer
  disk_max_mb: 100

forwarding:
  timeout: 10s
  retry_max_interval: 60s

# Phase 3
heartbeat:
  agent_token: gsna_xxxxx
  interval: 30s
  group: monolith-production

# Phase 4
spike_protection:
  enabled: true
  window: 60s
  threshold: 50
  keep_first: 5

logging:
  level: info
```

## Server-Side Changes Required

Each phase has explicit server-side prerequisites. The agent cannot safely implement a feature until its server dependency is in place.

| Phase | Server change needed | Why |
|-------|---------------------|-----|
| 1 | None | Pure proxy, server is unchanged |
| 2 | UNIQUE constraint on `events(project_id, event_id)` | Idempotent retry without duplicates (project-scoped because issue assignment can change between retries) |
| 3 | Agent token type + `agent_heartbeats` table + `/api/v1/agents` endpoints | Heartbeat auth and status queries |
| 4 | `aggregate_count` ingest + counter increment without row creation | Spike protection without breaking counters |

## Deployment

### Phase 1 (simplest)

```bash
# Download
wget https://github.com/darkspock/gosnag-agent/releases/latest/download/gosnag-agent-linux-amd64
chmod +x gosnag-agent-linux-amd64
sudo mv gosnag-agent-linux-amd64 /usr/local/bin/gosnag-agent

# Run
GOSNAG_SERVER_URL=https://sentry.cover-aws.com gosnag-agent

# Update PHP DSN
# Before: https://key@sentry.cover-aws.com/project-id
# After:  http://key@127.0.0.1:9000/project-id
```

### Docker sidecar (EB / compose)

```yaml
services:
  app:
    image: my-php-app
    environment:
      SENTRY_DSN: http://key@gosnag-agent:9000/project-id
    depends_on: [gosnag-agent]

  gosnag-agent:
    image: darkspock/gosnag-agent:latest
    environment:
      GOSNAG_SERVER_URL: https://sentry.cover-aws.com
      GOSNAG_LISTEN_ADDR: 0.0.0.0:9000
```

## Technical Decisions

| Decision | Rationale |
|----------|-----------|
| Transparent proxy (no parsing in Phase 1) | Minimizes bugs, maximizes compatibility, simplifies testing |
| Memory-only buffer in Phase 1 | Disk adds complexity (WAL, replay, cleanup). Memory-only is correct for "server down < 5 min" |
| No spike protection until Phase 4 | Suppressing events breaks server counters. Requires aggregate ingest API first. |
| Separate auth for heartbeat vs ingest | DSN keys are embedded in client apps. Agent tokens are server-managed. Different trust levels. |
| Agent does NOT compute fingerprints until Phase 4 | Fingerprint algorithm must match server exactly. Duplicating it is a maintenance risk. Deferred until needed. |

## Risks

| Risk | Mitigation |
|------|------------|
| Agent down = events lost | Fail-fast (< 1ms). SDK's built-in retry may catch some. Phase 2 adds disk persistence. |
| Memory buffer overflow during long outage | Known limitation in Phase 1. Oldest events dropped. Phase 2 adds disk. |
| Duplicate events on retry | Phase 1: rare, only on ambiguous network errors. Phase 2: UNIQUE constraint eliminates duplicates. |
| Agent adds latency | Localhost POST is < 1ms. Agent forwards async. Net effect: app is faster, not slower. |
| Fingerprint mismatch (Phase 4) | Must import GoSnag's fingerprint code as a Go package, not reimplement. |

## Success Metrics

| Phase | Metric |
|-------|--------|
| 1 | App error-path latency reduced from 50-200ms to < 1ms. Zero events lost under normal operation. |
| 2 | Zero events lost during GoSnag outage up to disk limit. Zero duplicates after retry. |
| 3 | External monitoring can verify all agents are online. |
| 4 | During 10k events/min spike, server counters are accurate. Priority/tags evaluate on real payload. Network traffic reduced by ~95%. |

## Repository

Separate repo: `gosnag-agent` (same org as GoSnag).
