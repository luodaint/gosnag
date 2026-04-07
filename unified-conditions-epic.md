# Epic: Unified Condition Engine

## Overview

GoSnag currently has 4 independent filtering systems (alerts, priority rules, tag rules, Jira rules), each with its own condition logic, schema, and UI. They share the same concepts (level filter, title pattern, event threshold, velocity) but implement them differently with no way to combine conditions with AND/OR logic. This epic unifies them into a single, composable condition engine shared across all features.

## Problem Statement

1. **No AND/OR composition**: Alert filters are hardcoded AND. Priority rules are independent scoring. There's no way to say "level=error AND (velocity_1h > 10 OR total_events > 100)".
2. **Duplicated logic**: `matchesPattern()` is implemented 3 times. Level filtering is duplicated in alerts and Jira. `min_events` threshold exists in both alerts and Jira with slightly different implementations.
3. **Inconsistent capabilities**: Priority rules support 8 condition types with operators. Alerts support 5 conditions with no operators. Jira supports 4. Tags support only 1.
4. **Rigid schemas**: Adding a new condition type (e.g., "environment_is") requires modifying each feature's migration, queries, handler, and UI independently.

## Current State

| Feature | Conditions | Combination | Implementation |
|---------|-----------|-------------|----------------|
| **Alerts** | level, title_pattern, exclude_pattern, min_events, min_velocity_1h | Hardcoded AND | `matchesAlert()` + inline velocity check |
| **Priority** | velocity_1h, velocity_24h, total_events, user_count, title_contains, title_not_contains, level_is, platform_is | Independent scoring (each adds points) | `Evaluate()` with rule_type switch |
| **Tags** | pattern (title + event data) | Independent (each rule applies separately) | `AutoTag()` loop |
| **Jira** | level, min_events, min_users, title_pattern | Hardcoded AND | `MatchesRule()` |

**Duplicated code:**
- `matchesPattern()` — 3 identical copies (priority, tags, alert service inline)
- Level CSV parsing — 2 copies (alerts, jira)
- Min events check — 2 copies (alerts, jira)
- Lazy velocity loading — 2 copies (alerts, priority)

## Proposed Solution

A shared `conditions` package that all features use. Conditions are stored as JSON arrays, evaluated by a common engine, and rendered by a shared UI component.

### Architecture

```
┌─────────────────────────────────────────────────────┐
│                  Condition Engine                     │
│                                                       │
│  Evaluate(conditions []Condition, ctx EvalContext)     │
│                                                       │
│  Supports:                                            │
│  - AND / OR groups (nestable)                         │
│  - Condition types: level, platform, title,           │
│    exclude_title, total_events, velocity_1h,          │
│    velocity_24h, user_count, environment, release,    │
│    tag_exists, pattern (full event data)              │
│  - Operators: eq, neq, gt, gte, lt, lte, contains,   │
│    not_contains, matches (regex), in, not_in          │
│                                                       │
│  EvalContext (lazy-loaded):                            │
│  - issue: level, platform, title, event_count, ...    │
│  - velocity_1h, velocity_24h (on demand)              │
│  - user_count (on demand)                             │
│  - event_data: full JSON (on demand)                  │
│  - tags: current issue tags                           │
└──────────────┬───────────────┬───────────────┬───────┘
               │               │               │
        ┌──────┴──────┐ ┌─────┴─────┐ ┌──────┴──────┐
        │   Alerts    │ │ Priority  │ │  Tags/Jira  │
        │             │ │           │ │             │
        │ conditions  │ │ conditions│ │ conditions  │
        │ + action:   │ │ + action: │ │ + action:   │
        │   notify    │ │   score   │ │   tag/ticket│
        └─────────────┘ └───────────┘ └─────────────┘
```

### Condition Data Model

```json
{
  "operator": "and",
  "conditions": [
    {
      "type": "level",
      "op": "in",
      "value": ["error", "fatal"]
    },
    {
      "operator": "or",
      "conditions": [
        {
          "type": "velocity_1h",
          "op": "gte",
          "value": 10
        },
        {
          "type": "total_events",
          "op": "gte",
          "value": 100
        }
      ]
    },
    {
      "type": "title",
      "op": "not_contains",
      "value": "HealthCheck"
    }
  ]
}
```

This reads as: **level IN (error, fatal) AND (velocity_1h >= 10 OR total_events >= 100) AND title NOT CONTAINS "HealthCheck"**

### Condition Types

| Type | Description | Operators | Value Type |
|------|-------------|-----------|------------|
| `level` | Issue log level | `eq`, `neq`, `in`, `not_in` | string or string[] |
| `platform` | Issue platform | `eq`, `neq`, `in`, `not_in` | string or string[] |
| `title` | Issue title | `contains`, `not_contains`, `matches` (regex) | string |
| `event_data` | Full event JSON (stacktrace, breadcrumbs) | `contains`, `not_contains`, `matches` | string |
| `total_events` | Issue total event count | `eq`, `gt`, `gte`, `lt`, `lte` | number |
| `velocity_1h` | Events in last hour | `eq`, `gt`, `gte`, `lt`, `lte` | number |
| `velocity_24h` | Events in last 24 hours | `eq`, `gt`, `gte`, `lt`, `lte` | number |
| `user_count` | Distinct affected users | `eq`, `gt`, `gte`, `lt`, `lte` | number |
| `environment` | Event environment | `eq`, `neq`, `in`, `not_in` | string or string[] |
| `release` | Event release version | `eq`, `neq`, `contains` | string |
| `tag_exists` | Issue has a specific tag | `eq` | string (key:value) |
| `first_seen` | Time since first seen | `gt`, `lt` | duration string ("24h", "7d") |
| `last_seen` | Time since last seen | `gt`, `lt` | duration string |

### Operators

| Operator | Description | Applies to |
|----------|-------------|------------|
| `eq` | Equals | all |
| `neq` | Not equals | all |
| `gt` | Greater than | numbers, durations |
| `gte` | Greater than or equal | numbers, durations |
| `lt` | Less than | numbers, durations |
| `lte` | Less than or equal | numbers, durations |
| `in` | Value is in list | strings |
| `not_in` | Value is not in list | strings |
| `contains` | Substring match (case-insensitive) | strings |
| `not_contains` | Negated substring match | strings |
| `matches` | Regex match | strings |

## Database Changes

### Option A: JSON column (recommended)

Add a `conditions` JSONB column to each feature table. The condition tree is stored as JSON. The existing flat columns (`level_filter`, `title_pattern`, `min_events`, etc.) are kept for backward compatibility and migrated over time.

```sql
-- Migration: add conditions column
ALTER TABLE alert_configs ADD COLUMN conditions JSONB;
ALTER TABLE priority_rules ADD COLUMN conditions JSONB;
ALTER TABLE tag_rules ADD COLUMN conditions JSONB;
ALTER TABLE jira_rules ADD COLUMN conditions JSONB;
```

**Evaluation logic**: If `conditions` is not null, use the engine. If null, fall back to the legacy flat columns. This allows gradual migration.

### Option B: Separate conditions table (normalized)

```sql
CREATE TABLE condition_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id UUID REFERENCES condition_groups(id) ON DELETE CASCADE,
    owner_type TEXT NOT NULL,   -- 'alert', 'priority_rule', 'tag_rule', 'jira_rule'
    owner_id UUID NOT NULL,
    operator TEXT NOT NULL DEFAULT 'and',  -- 'and' or 'or'
    position INT NOT NULL DEFAULT 0
);

CREATE TABLE conditions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES condition_groups(id) ON DELETE CASCADE,
    condition_type TEXT NOT NULL,
    operator TEXT NOT NULL,
    value JSONB NOT NULL,
    position INT NOT NULL DEFAULT 0
);
```

**Recommendation**: Option A (JSON column). Simpler queries, no joins, the condition tree is always loaded as a whole unit. The engine deserializes and evaluates in Go. PostgreSQL JSONB supports indexing if needed later.

## Go Implementation

### Package: `internal/conditions`

```go
package conditions

// ConditionGroup represents an AND/OR group of conditions
type ConditionGroup struct {
    Operator   string           `json:"operator"` // "and" or "or"
    Conditions []ConditionOrGroup `json:"conditions"`
}

// ConditionOrGroup is either a leaf condition or a nested group
type ConditionOrGroup struct {
    // Leaf condition fields
    Type  string `json:"type,omitempty"`
    Op    string `json:"op,omitempty"`
    Value any    `json:"value,omitempty"`

    // Nested group fields
    Operator   string           `json:"operator,omitempty"`
    Conditions []ConditionOrGroup `json:"conditions,omitempty"`
}

// EvalContext provides lazy-loaded data for condition evaluation
type EvalContext struct {
    Issue     IssueData
    EventData string  // raw JSON, loaded on first access

    // Lazy-loaded (only queried if a condition needs them)
    velocity1h  *int32
    velocity24h *int32
    userCount   *int32
    loader      DataLoader
}

// DataLoader interface for lazy-loading expensive data
type DataLoader interface {
    GetVelocity1h(issueID uuid.UUID) (int32, error)
    GetVelocity24h(issueID uuid.UUID) (int32, error)
    GetUserCount(issueID uuid.UUID) (int32, error)
}

// Evaluate checks if a condition group matches the given context
func Evaluate(group ConditionGroup, ctx *EvalContext) bool {
    if group.Operator == "or" {
        for _, c := range group.Conditions {
            if evaluateOne(c, ctx) {
                return true  // short-circuit OR
            }
        }
        return false
    }
    // Default: AND
    for _, c := range group.Conditions {
        if !evaluateOne(c, ctx) {
            return false  // short-circuit AND
        }
    }
    return true
}
```

### Integration with existing features

Each feature calls the engine instead of its own filtering logic:

```go
// Alert service (before)
if !matchesAlert(ac, issue) { continue }

// Alert service (after)
if ac.Conditions != nil {
    ctx := conditions.NewEvalContext(issue, eventData, loader)
    if !conditions.Evaluate(*ac.Conditions, ctx) { continue }
} else {
    // Legacy fallback
    if !matchesAlert(ac, issue) { continue }
}
```

## Frontend: Shared Condition Builder Component

A reusable React component that renders the condition tree and allows editing:

```
┌─ AND ──────────────────────────────────────────────┐
│                                                     │
│  [Level] [is one of ▾] [error, fatal]    [× Delete]│
│                                                     │
│  ┌─ OR ──────────────────────────────────────────┐  │
│  │  [Velocity 1h] [>=] [10]             [× Del]  │  │
│  │  [Total events] [>=] [100]           [× Del]  │  │
│  │                          [+ Add condition]     │  │
│  └────────────────────────────────────────────────┘  │
│                                                      │
│  [Title] [does not contain] [HealthCheck]  [× Del]   │
│                                                      │
│                    [+ Add condition] [+ Add group]   │
└──────────────────────────────────────────────────────┘
```

The component is used in:
- Alert settings form
- Priority rule form
- Tag rule form
- Jira rule form

### Component API

```tsx
<ConditionBuilder
  value={conditions}
  onChange={setConditions}
  availableTypes={['level', 'platform', 'title', 'total_events', 'velocity_1h', ...]}
  maxDepth={2}  // prevent deeply nested groups
/>
```

## Migration Strategy

### Phase 1 — Engine + backward compatibility

1. Create `internal/conditions` package with `Evaluate()`, types, and helpers
2. Extract shared `matchesPattern()` into the package
3. Add `conditions JSONB` column to all 4 tables
4. Each feature checks `conditions` first, falls back to legacy columns
5. Frontend: add `ConditionBuilder` component, use in alert form only (pilot)
6. Existing data continues to work unchanged

### Phase 2 — Roll out to all features

1. Use `ConditionBuilder` in priority rules, tag rules, Jira rules
2. Each feature's UI shows either the new builder or the legacy form (based on whether `conditions` is set)
3. "Upgrade" button in UI to convert legacy flat fields to a conditions JSON

### Phase 3 — Simplify and clean up

1. Write a migration to convert all legacy flat-column configs to `conditions` JSON
2. Remove legacy evaluation code paths
3. Remove legacy flat columns (or keep as read-only for API backward compatibility)
4. All features now use the unified engine

## Examples

### Alert: "Notify on error spikes, but not health checks"

```json
{
  "operator": "and",
  "conditions": [
    { "type": "level", "op": "in", "value": ["error", "fatal"] },
    { "type": "velocity_1h", "op": "gte", "value": 10 },
    { "type": "title", "op": "not_contains", "value": "HealthCheck" }
  ]
}
```

### Alert: "Notify on any fatal, OR error with > 100 events"

```json
{
  "operator": "or",
  "conditions": [
    { "type": "level", "op": "eq", "value": "fatal" },
    {
      "operator": "and",
      "conditions": [
        { "type": "level", "op": "eq", "value": "error" },
        { "type": "total_events", "op": "gte", "value": 100 }
      ]
    }
  ]
}
```

### Priority rule: "High priority if payment-related AND high velocity"

```json
{
  "operator": "and",
  "conditions": [
    { "type": "event_data", "op": "contains", "value": "payment" },
    { "type": "velocity_1h", "op": "gte", "value": 20 }
  ]
}
```

Result: `points: +30` (configured on the rule, applied when conditions match)

### Tag rule: "Tag as team:infrastructure if it's a DB or Redis error"

```json
{
  "operator": "or",
  "conditions": [
    { "type": "event_data", "op": "matches", "value": "(?i)(deadlock|connection refused|timeout.*postgres)" },
    { "type": "event_data", "op": "contains", "value": "redis" }
  ]
}
```

Result: tag `team:infrastructure` applied

### Jira rule: "Create ticket for production fatals affecting > 5 users"

```json
{
  "operator": "and",
  "conditions": [
    { "type": "level", "op": "eq", "value": "fatal" },
    { "type": "environment", "op": "eq", "value": "production" },
    { "type": "user_count", "op": "gte", "value": 5 }
  ]
}
```

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Complex nested conditions hurt performance | Max depth of 2 levels. Lazy loading for expensive data. Short-circuit evaluation. |
| Users create broken conditions | Validate JSON schema on save. Frontend builder prevents invalid structures. |
| Migration breaks existing rules | Dual-path evaluation (new engine + legacy fallback). Existing data untouched until explicitly migrated. |
| JSON conditions harder to query in SQL | Rarely needed — conditions are loaded and evaluated in Go. JSONB supports `@>` queries if needed. |
| Frontend builder too complex | Start with flat AND-only for simple cases. "Advanced mode" toggle for nested groups. |

## Success Metrics

- All 4 features use the same condition engine
- Zero duplicated matching/comparison code
- Users can create AND/OR conditions in the UI
- Adding a new condition type requires changes in 1 place (the engine), not 4
- Existing alerts/rules continue to work without modification during migration
