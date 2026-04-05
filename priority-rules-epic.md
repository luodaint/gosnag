# Epic: Rule-Based Issue Priority Scoring

## Overview
Each project defines priority rules that produce a numeric score (0–100) for every issue. The score is recalculated asynchronously on each incoming event. Issues are sortable by priority in the UI.

## How It Works

### Score Model
- Each rule contributes points (positive or negative) to the issue's score
- Final score is clamped to 0–100
- Default score for new issues with no rules: 50
- Score stored on the issue row for fast sorting

### Rule Types

| Rule Type | Description | Example |
|-----------|-------------|---------|
| `velocity_1h` | Events in the last hour exceed threshold | >10 events/hour → +20 points |
| `velocity_24h` | Events in the last 24h exceed threshold | >100 events/day → +15 points |
| `total_events` | Total event count exceeds threshold | >500 total → +10 points |
| `user_count` | Unique affected users exceed threshold | >5 users → +25 points |
| `title_contains` | Issue title matches a pattern | contains "database" → +30 points |
| `title_not_contains` | Issue title does NOT match a pattern | not contains "deprecation" → -20 points |
| `level_is` | Issue level matches | level = "fatal" → +40 points |
| `platform_is` | Issue platform matches | platform = "php" → +10 points |

### Rule Definition
```json
{
  "name": "Critical database errors",
  "rule_type": "title_contains",
  "pattern": "database|connection refused",
  "operator": "gte",       // for numeric rules: gte, lte, eq
  "threshold": 0,          // for numeric rules
  "points": 30,            // points to add (negative to subtract)
  "enabled": true
}
```

### Evaluation Flow
1. Event ingested → issue upserted → HTTP 200 returned
2. Goroutine fires: load enabled rules for the project
3. For each rule, evaluate against current issue state
4. Sum matched rule points, clamp 0–100
5. UPDATE issue SET priority = score
6. Quick path: most rules are simple field checks (title, level, total_events)
7. Slow path: velocity rules need a COUNT query with timestamp filter — batch these

### Performance Considerations
- Velocity queries (events in last 1h/24h) are the expensive ones
- Cache velocity counts briefly (per-issue, 60s TTL in memory) to avoid hammering the DB on burst events
- Simple field rules (title, level, platform, total_events) are evaluated in-memory from the issue struct — no DB queries needed
- The goroutine runs after ingest, so it doesn't block the SDK response

## Database

### Migration
```sql
-- Priority score on issues
ALTER TABLE issues ADD COLUMN priority INT NOT NULL DEFAULT 50;
CREATE INDEX idx_issues_priority ON issues(project_id, priority DESC);

-- Priority rules per project
CREATE TABLE priority_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    rule_type TEXT NOT NULL,
    pattern TEXT NOT NULL DEFAULT '',
    operator TEXT NOT NULL DEFAULT '',
    threshold INT NOT NULL DEFAULT 0,
    points INT NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    position INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_priority_rules_project ON priority_rules(project_id);
```

### Queries
- `ListEnabledPriorityRules(project_id)` — for evaluation
- `CRUD` for priority_rules — for settings UI
- `GetIssueVelocity(issue_id, interval)` — COUNT events in last N hours
- `UpdateIssuePriority(issue_id, priority)` — set score

## API

### Rules CRUD (admin)
```
GET    /api/v1/projects/{id}/priority-rules
POST   /api/v1/projects/{id}/priority-rules
PUT    /api/v1/projects/{id}/priority-rules/{rule_id}
DELETE /api/v1/projects/{id}/priority-rules/{rule_id}
```

### Issues
- `GET /projects/{id}/issues` — add `sort=priority` option
- Issue response includes `priority` field (0–100)

## Frontend

### Settings
- New section "Priority Rules" in Project Settings
- Add/edit/delete rules with form:
  - Name
  - Type (dropdown)
  - Pattern/threshold (dynamic based on type)
  - Points (+/-)
  - Enabled toggle
- Preview: show current issues with their calculated scores

### Issue List
- Priority score badge on each issue (color-coded: red >75, amber >50, blue >25, gray ≤25)
- New sort option: "Priority" (default: highest first)
- Priority column in the list

## Tasks

### Phase 1: Backend
- [ ] Migration `000012_priority_rules`
- [ ] SQL queries: rules CRUD, velocity count, update priority
- [ ] sqlc regenerate
- [ ] Priority evaluator: `internal/priority/evaluator.go`
- [ ] Priority handler: CRUD routes
- [ ] Hook into ingest pipeline (goroutine after event processed)
- [ ] Register routes

### Phase 2: Frontend
- [ ] Priority rules section in ProjectSettings
- [ ] Rule form with dynamic fields based on type
- [ ] Priority badge on issue list
- [ ] Sort by priority option
- [ ] Priority display on issue detail

### Phase 3: Polish
- [ ] In-memory velocity cache (60s TTL) to avoid DB spam on bursts
- [ ] Bulk re-evaluate button in settings (recalc all issues)
- [ ] Default rule templates (e.g. "Fatal errors = high priority")

## Effort Estimate
- Backend (evaluator + handler + migration): ~3h
- Frontend (settings + list + detail): ~3h
- Velocity cache + polish: ~2h
- Total: ~8h
