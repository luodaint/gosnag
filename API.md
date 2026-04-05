# GoSnag API Reference

Base URL: `https://your-instance/api/v1`

## Authentication

| Method | Header | Used for |
|--------|--------|----------|
| Session cookie | `Cookie: session=...` | Web UI (set by Google OAuth login) |
| API token | `Authorization: Bearer gsn_...` | External integrations, CI/CD |
| DSN key | Sentry SDK auth header | Event ingestion only |

API tokens are project-scoped. They can only access endpoints under the project they belong to.

**Permissions:**
- `read` — GET endpoints only (list issues, events, etc.)
- `readwrite` — also PUT, POST, DELETE (resolve, assign, merge, etc.)

---

## Projects

### List projects
```
GET /projects
```
Response: array of projects with stats (`total_issues`, `open_issues`, `trend`, `errors_this_week`, etc.)

### Get project
```
GET /projects/{project_id}
```
Response includes `dsn` for Sentry SDK configuration.

### Create project
```
POST /projects
```
Requires: **admin**
```json
{
  "name": "My Service",
  "slug": "my-service",
  "default_cooldown_minutes": 60
}
```

### Update project
```
PUT /projects/{project_id}
```
Requires: **admin**
```json
{
  "name": "My Service",
  "slug": "my-service",
  "default_cooldown_minutes": 60,
  "warning_as_error": false,
  "jira_base_url": "https://company.atlassian.net",
  "jira_email": "user@company.com",
  "jira_api_token": "...",
  "jira_project_key": "DEV",
  "jira_issue_type": "Bug"
}
```
Fields not sent are preserved from the existing project. The `jira_api_token` is never returned in responses.

### Delete project
```
DELETE /projects/{project_id}
```
Requires: **admin**. Deletes all issues, events, alerts, and tokens.

---

## Issues

### List issues
```
GET /projects/{project_id}/issues?status=open&level=error&limit=25&offset=0
```

| Param | Type | Description |
|-------|------|-------------|
| `status` | string | Filter: `open`, `resolved`, `reopened`, `ignored`, `snoozed` |
| `level` | string | Filter: `errors` (error+fatal), `warnings`, `info`, or specific level |
| `search` | string | Search in issue title |
| `today` | bool | Only issues first seen today |
| `assigned_to` | string | `me` for current user |
| `assigned_any` | bool | Only assigned issues |
| `limit` | int | Max 100, default 50 |
| `offset` | int | Pagination offset |

Response:
```json
{
  "issues": [{
    "id": "uuid",
    "title": "TypeError: Cannot read property...",
    "status": "open",
    "level": "error",
    "platform": "javascript",
    "event_count": 42,
    "user_count": 8,
    "first_seen": "2026-04-01T10:00:00Z",
    "last_seen": "2026-04-03T15:30:00Z",
    "jira_ticket_key": "DEV-123",
    "jira_ticket_url": "https://company.atlassian.net/browse/DEV-123",
    "trend": [0, 3, 5, 2, ...]
  }],
  "total": 150,
  "limit": 25,
  "offset": 0
}
```

### Get issue
```
GET /projects/{project_id}/issues/{issue_id}
```

### Get issue counts
```
GET /projects/{project_id}/issues/counts?level=errors
```
Response:
```json
{
  "total": 42,
  "by_status": { "open": 30, "resolved": 10, "ignored": 2 },
  "today": 5,
  "assigned_to_me": 3,
  "assigned_any": 8
}
```

### Update issue status
```
PUT /projects/{project_id}/issues/{issue_id}
```
Requires: **readwrite**
```json
{
  "status": "resolved",
  "cooldown_minutes": 120,
  "resolved_in_release": "v1.2.3"
}
```

| Field | When |
|-------|------|
| `cooldown_minutes` | With `resolved` — time before issue can auto-reopen. Omit to use project default. |
| `resolved_in_release` | With `resolved` — reopen if event arrives from different release |
| `snooze_minutes` | With `snoozed` — auto-reopen after N minutes |
| `snooze_event_threshold` | With `snoozed` — auto-reopen after N new events |

### Assign issue
```
PUT /projects/{project_id}/issues/{issue_id}/assign
```
Requires: **readwrite**
```json
{ "assigned_to": "user-uuid" }
```
Send `null` to unassign.

### Bulk delete issues
```
DELETE /projects/{project_id}/issues
```
Requires: **readwrite**
```json
{ "ids": ["uuid1", "uuid2"] }
```

### Merge issues
```
POST /projects/{project_id}/issues/merge
```
Requires: **readwrite**
```json
{
  "primary_id": "uuid-of-surviving-issue",
  "issue_ids": ["uuid-to-merge-1", "uuid-to-merge-2"]
}
```
Moves all events to the primary issue, creates fingerprint aliases so future events with the merged fingerprints go to the primary, then deletes the secondary issues. Returns the updated primary issue.

### List events
```
GET /projects/{project_id}/issues/{issue_id}/events?limit=50&offset=0
```
Response:
```json
{
  "events": [{
    "id": "uuid",
    "event_id": "sentry-event-id",
    "timestamp": "2026-04-03T15:30:00Z",
    "level": "error",
    "message": "...",
    "release": "v1.2.3",
    "environment": "production",
    "data": { /* full Sentry event payload */ }
  }],
  "total": 42,
  "limit": 50,
  "offset": 0
}
```

---

## Jira Integration

### Test connection
```
POST /projects/{project_id}/jira/test
```
Response: `{ "ok": true }` or `{ "ok": false, "error": "..." }`

### Create Jira ticket from issue
```
POST /projects/{project_id}/issues/{issue_id}/jira
```
Requires: **readwrite**. Fails with 409 if issue already has a ticket.

Response:
```json
{ "key": "DEV-123", "url": "https://company.atlassian.net/browse/DEV-123" }
```

### Jira auto-creation rules

```
GET    /projects/{project_id}/jira/rules
POST   /projects/{project_id}/jira/rules          (admin)
PUT    /projects/{project_id}/jira/rules/{rule_id} (admin)
DELETE /projects/{project_id}/jira/rules/{rule_id} (admin)
```

Rule body:
```json
{
  "name": "Critical errors",
  "enabled": true,
  "level_filter": "fatal,error",
  "min_events": 5,
  "min_users": 2,
  "title_pattern": "database|timeout"
}
```
All conditions are AND. Empty/zero means "match all". `title_pattern` supports regex.

---

## API Tokens

```
GET    /projects/{project_id}/tokens              — list tokens
POST   /projects/{project_id}/tokens              — create (admin)
DELETE /projects/{project_id}/tokens/{token_id}    — revoke (admin)
```

Create:
```json
{
  "name": "CI Pipeline",
  "permission": "read",
  "expires_in": 90
}
```
`expires_in` is in days. Omit for no expiration.

Response includes `token` field (plain text, shown only once):
```json
{
  "id": "uuid",
  "name": "CI Pipeline",
  "permission": "read",
  "token": "gsn_a1b2c3d4e5f6...",
  "expires_at": "2026-07-03T00:00:00Z",
  "created_at": "2026-04-03T00:00:00Z"
}
```

---

## Alerts

```
GET    /projects/{project_id}/alerts              — list
POST   /projects/{project_id}/alerts              — create (admin)
PUT    /projects/{project_id}/alerts/{alert_id}    — update (admin)
DELETE /projects/{project_id}/alerts/{alert_id}    — delete (admin)
```

Email alert:
```json
{
  "alert_type": "email",
  "config": { "recipients": ["dev@example.com", "ops@example.com"] },
  "enabled": true,
  "level_filter": "error,fatal",
  "title_pattern": ""
}
```

Slack alert:
```json
{
  "alert_type": "slack",
  "config": { "webhook_url": "https://hooks.slack.com/services/..." },
  "enabled": true
}
```

---

## Users

```
GET  /users                          — list all users
POST /users/invite                   — invite user (admin)
PUT  /users/{user_id}                — update role (admin)
PUT  /users/{user_id}/status         — enable/disable (admin)
```

Invite:
```json
{ "email": "dev@example.com", "role": "viewer" }
```

---

## Event Ingestion

These endpoints accept Sentry SDK events. Configure your SDK with the DSN from project settings.

```
POST /api/{project_id}/store/       — legacy Sentry format
POST /api/{project_id}/envelope/    — modern Sentry envelope format
```

Authenticated by DSN key (not session or API token). Rate limited.

---

## Examples

### Create a token and list issues
```bash
# Create token (requires session auth or admin token)
curl -X POST https://sentry.example.com/api/v1/projects/PROJECT_ID/tokens \
  -H "Authorization: Bearer gsn_admin_token" \
  -H "Content-Type: application/json" \
  -d '{"name": "monitoring", "permission": "read"}'

# Use the returned token to list issues
curl https://sentry.example.com/api/v1/projects/PROJECT_ID/issues?status=open&level=errors \
  -H "Authorization: Bearer gsn_returned_token"
```

### Resolve an issue
```bash
curl -X PUT https://sentry.example.com/api/v1/projects/PROJECT_ID/issues/ISSUE_ID \
  -H "Authorization: Bearer gsn_readwrite_token" \
  -H "Content-Type: application/json" \
  -d '{"status": "resolved", "cooldown_minutes": 120}'
```

### Merge duplicate issues
```bash
curl -X POST https://sentry.example.com/api/v1/projects/PROJECT_ID/issues/merge \
  -H "Authorization: Bearer gsn_readwrite_token" \
  -H "Content-Type: application/json" \
  -d '{"primary_id": "ISSUE_UUID", "issue_ids": ["DUPE_1", "DUPE_2"]}'
```

### Create a Jira ticket
```bash
curl -X POST https://sentry.example.com/api/v1/projects/PROJECT_ID/issues/ISSUE_ID/jira \
  -H "Authorization: Bearer gsn_readwrite_token"
```
