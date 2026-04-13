# Epic: Multi-Tenant Organizations

## Overview

GoSnag is currently single-tenant: all users share all projects, gated only by a global `admin`/`viewer` role. There is no concept of organizations, teams, or per-project access control. Any authenticated user can see every project.

This epic adds **organizations** as the tenant boundary. Each organization owns projects, has its own members with per-org roles, and isolates data so users in Org A cannot see Org B's projects. This enables GoSnag to serve multiple teams or companies on a single instance (SaaS model) or to separate concerns within a large engineering org.

### Why Now

- Cover uses GoSnag for their own services. Other teams want to use the same instance but shouldn't see each other's projects.
- The current invite-only model doesn't scale: every invited user gets access to everything.
- Domain-based auto-join (ALLOWED_DOMAIN is documented but never implemented) needs an organization to assign users to.

### Design Principles

1. **Zero disruption**: Existing instances auto-migrate to a "Default" org. Everything works identically until you create a second org.
2. **Org = tenant boundary**: Projects, alerts, rules, tokens, and all project-scoped data are isolated by org.
3. **Multi-org per user**: A user can belong to multiple orgs (like Slack workspaces). An org switcher in the UI sets the active context.
4. **Per-org roles**: Roles become org-scoped (owner, admin, member) instead of global. Global admin role remains for instance-level operations only.

---

## Architecture

### Tenant Model

```
┌──────────────────────────────────────┐
│              GoSnag Instance         │
│                                      │
│  ┌─────────────┐  ┌──────────────┐  │
│  │  Org: Cover  │  │ Org: Partner │  │
│  │             │  │              │  │
│  │ Projects:   │  │ Projects:    │  │
│  │  - Controller│  │  - App-X    │  │
│  │  - Notif Svc│  │  - App-Y    │  │
│  │             │  │              │  │
│  │ Members:    │  │ Members:     │  │
│  │  - Juan (own)│  │  - Ana (own)│  │
│  │  - Carlos   │  │  - Juan     │  │
│  └─────────────┘  └──────────────┘  │
│                                      │
│  Juan is in both orgs.              │
│  Carlos cannot see Partner projects. │
└──────────────────────────────────────┘
```

### Data Isolation

All project-scoped data is already isolated by `project_id` FK. Adding `organization_id` to `projects` creates the tenant boundary:

```
organization → projects → issues, events, alerts, rules, tokens, etc.
```

No changes needed to tables below `projects` — the existing FK chain handles isolation. The middleware validates that the authenticated user belongs to the org that owns the requested project.

### Role Model

| Role | Scope | Capabilities |
|------|-------|-------------|
| `instance_admin` | Global | Manage orgs, global settings, instance config |
| `owner` | Per-org | Everything admin can do + delete org, manage member roles, transfer projects, org settings |
| `admin` | Per-org | Create/edit/delete projects, alerts, priority rules, tag rules, Jira/GitHub rules, project settings, invite members, manage tickets, assign issues |
| `member` | Per-org | View projects and issues, create/comment on tickets, follow issues, view alerts and rules (read-only), trigger AI analysis on issues |

The current `users.role` column (`admin`/`viewer`) migrates to:
- Existing `admin` users → `instance_admin` global flag + `owner` of default org
- Existing `viewer` users → `member` of default org

---

## Database

### Migration: Organizations

```sql
-- Organizations
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    allowed_domains TEXT[] NOT NULL DEFAULT '{}',  -- auto-join on login (e.g. ['cover.com'])
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Org membership
CREATE TABLE organization_members (
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member')),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('invited', 'active')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, user_id)
);

CREATE INDEX idx_org_members_user ON organization_members(user_id);

-- Link projects to organizations
ALTER TABLE projects ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
CREATE INDEX idx_projects_org ON projects(organization_id);

-- project_groups become org-scoped
ALTER TABLE project_groups ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;

-- Add instance_admin flag (replaces global 'admin' role for instance-level ops)
ALTER TABLE users ADD COLUMN instance_admin BOOLEAN NOT NULL DEFAULT false;

-- API tokens: add organization scope
ALTER TABLE api_tokens ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
```

### Migration: Data backfill

```sql
-- Create default org for existing data
INSERT INTO organizations (id, name, slug)
VALUES ('00000000-0000-0000-0000-000000000001', 'Default', 'default');

-- Assign all existing projects to default org
UPDATE projects SET organization_id = '00000000-0000-0000-0000-000000000001'
WHERE organization_id IS NULL;

-- Make organization_id NOT NULL after backfill
ALTER TABLE projects ALTER COLUMN organization_id SET NOT NULL;

-- Migrate existing users to default org
-- admin → owner, viewer → member
INSERT INTO organization_members (organization_id, user_id, role)
SELECT '00000000-0000-0000-0000-000000000001', id,
    CASE WHEN role = 'admin' THEN 'owner' ELSE 'member' END
FROM users;

-- Set instance_admin flag for existing admins
UPDATE users SET instance_admin = true WHERE role = 'admin';

-- Migrate project_groups to default org
UPDATE project_groups SET organization_id = '00000000-0000-0000-0000-000000000001'
WHERE organization_id IS NULL;

ALTER TABLE project_groups ALTER COLUMN organization_id SET NOT NULL;

-- Migrate global API tokens to default org
UPDATE api_tokens SET organization_id = '00000000-0000-0000-0000-000000000001'
WHERE scope = 'global' AND organization_id IS NULL;

-- Drop legacy role column (replaced by organization_members.role + users.instance_admin)
ALTER TABLE users DROP COLUMN role;

-- Simplify users.status: remove 'invited' (now on membership level), keep active/disabled as global gate
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_status_check;
ALTER TABLE users ADD CONSTRAINT users_status_check CHECK (status IN ('active', 'disabled'));
UPDATE users SET status = 'active' WHERE status = 'invited';
```

### Queries (sqlc)

```sql
-- Organization CRUD
-- name: GetOrganization :one
SELECT * FROM organizations WHERE id = $1;

-- name: GetOrganizationBySlug :one
SELECT * FROM organizations WHERE slug = $1;

-- name: ListOrganizationsForUser :many
SELECT o.* FROM organizations o
JOIN organization_members om ON o.id = om.organization_id
WHERE om.user_id = $1
ORDER BY o.name;

-- name: CreateOrganization :one
INSERT INTO organizations (name, slug, allowed_domains)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateOrganization :one
UPDATE organizations SET name = $1, slug = $2, allowed_domains = $3, updated_at = now()
WHERE id = $4
RETURNING *;

-- name: DeleteOrganization :exec
DELETE FROM organizations WHERE id = $1;

-- Membership
-- name: ListOrganizationMembers :many
SELECT u.id, u.email, u.name, u.avatar_url, u.status, om.role, om.created_at
FROM organization_members om
JOIN users u ON u.id = om.user_id
WHERE om.organization_id = $1
ORDER BY om.role, u.name;

-- name: GetOrganizationMember :one
SELECT om.role FROM organization_members om
WHERE om.organization_id = $1 AND om.user_id = $2;

-- name: AddOrganizationMember :exec
INSERT INTO organization_members (organization_id, user_id, role)
VALUES ($1, $2, $3)
ON CONFLICT (organization_id, user_id) DO UPDATE SET role = $3;

-- name: UpdateMemberRole :exec
UPDATE organization_members SET role = $1
WHERE organization_id = $2 AND user_id = $3;

-- name: RemoveOrganizationMember :exec
-- Business rule: cannot remove the last owner. Check before calling.
DELETE FROM organization_members
WHERE organization_id = $1 AND user_id = $2;

-- name: CountOrgOwners :one
SELECT COUNT(*) FROM organization_members
WHERE organization_id = $1 AND role = 'owner';

-- name: ListOrganizationsByAllowedDomain :many
SELECT * FROM organizations
WHERE $1 = ANY(allowed_domains)
ORDER BY name;

-- Project queries (update existing)
-- name: ListProjectsByOrganization :many
SELECT * FROM projects WHERE organization_id = $1 ORDER BY position, name;

-- name: GetProjectWithOrgCheck :one
SELECT * FROM projects WHERE id = $1 AND organization_id = $2;

-- Project groups (update existing)
-- name: ListProjectGroupsByOrg :many
SELECT * FROM project_groups WHERE organization_id = $1 ORDER BY position, name;
```

---

## API

### Organization Endpoints

```
GET    /api/organizations                          → list user's orgs
POST   /api/organizations                          → create org (instance_admin only)
GET    /api/organizations/{org_id}                  → get org details
PUT    /api/organizations/{org_id}                  → update org (owner only)
DELETE /api/organizations/{org_id}                  → delete org (owner only)

GET    /api/organizations/{org_id}/members          → list members
POST   /api/organizations/{org_id}/members          → invite/add member (admin+)
PUT    /api/organizations/{org_id}/members/{user_id} → update member role (owner only)
DELETE /api/organizations/{org_id}/members/{user_id} → remove member (admin+)
```

### Modified Existing Endpoints

All project-scoped endpoints gain org context. Two options:

**Option A: Org prefix in URL** (breaking change)
```
GET /api/organizations/{org_id}/projects
GET /api/organizations/{org_id}/projects/{project_id}/issues
```

**Option B: Org resolved from session** (non-breaking)
```
GET /api/projects                    → filtered by active org from session/header
GET /api/projects/{project_id}       → middleware validates user has org access
```

**Recommended: Option B** with an `X-Organization-ID` header (or query param) to select the active org. Falls back to the user's first org. This keeps all existing API routes working and clients don't break.

### Auth Flow Changes

On login:
1. Authenticate user (Google OAuth / local)
2. Resolve orgs: `ListOrganizationsForUser(user.id)`
3. If user has no orgs:
   - Check `allowed_domains` for matching orgs
   - If exactly one match → auto-join that org
   - If multiple matches → prompt user to choose which org to join
   - If no match and no orgs exist at all → create "Default" org, make user owner
   - If orgs exist but user isn't in any and no domain match → reject login (not invited to any org)
4. Return org list with session; frontend stores active org

### Ingest Endpoints (No Change)

DSN keys are already scoped to projects. The ingest path (`/api/{project_id}/store/`, `/api/{project_id}/envelope/`) validates the DSN key against the project — no org check needed. The project's `organization_id` provides the tenant boundary.

---

## Middleware

### New: OrgAccess Middleware

Applied to all `/api/projects/*` routes:

```go
// On login: load user's org memberships into session cache (in-memory, keyed by user_id, 5min TTL).
// OrgAccessMiddleware reads from cache — no DB queries on the hot path.

func OrgAccessMiddleware(queries *db.Queries, memberCache *OrgMemberCache) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            user := auth.GetUserFromContext(r.Context())
            projectID := chi.URLParam(r, "project_id")
            
            // Look up project → get organization_id (cached per project_id, 5min TTL)
            orgID := memberCache.GetProjectOrg(projectID)
            // Verify user membership from cache (loaded on login, refreshed every 5min)
            role := memberCache.GetRole(user.ID, orgID)
            // If not member and not instance_admin → 403
            
            // Inject org + role into context for downstream use
            ctx = context.WithValue(ctx, orgContextKey, orgID)
            ctx = context.WithValue(ctx, orgRoleContextKey, role)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

### Modified: RequireAdmin

Currently checks `user.Role == "admin"`. Changes to:
- For org-scoped operations: check `organization_members.role IN ('owner', 'admin')`
- For instance operations: check `user.InstanceAdmin`

---

## Frontend

### Org Switcher

Top-left of the sidebar, next to the GoSnag logo:

```
┌──────────────────┐
│ 🏢 Cover     ▾  │  ← dropdown to switch orgs
├──────────────────┤
│  Cover           │
│  Partner Corp  ✓ │
│  ─────────────── │
│  + Create Org    │  ← only if instance_admin
└──────────────────┘
```

- Active org stored in `localStorage` + sent as `X-Organization-ID` header on all API calls
- Switching org reloads the project list
- If user has only one org, the switcher shows the name but isn't a dropdown

### Org Settings Page

`/settings/organization` — accessible to org owners and admins:

- **General**: Name, slug, allowed domains
- **Members**: List with role badges, invite form, role dropdown, remove button
- **Danger Zone**: Delete organization (owner only, requires confirmation)

### Instance Admin Page

`/settings/instance` — accessible to `instance_admin` users only:

- List all organizations
- Create new organization
- Global AI settings (already exists, move here)
- Instance-level stats

### Modified Pages

- **Project list**: Filtered by active org. `ListProjectsByOrganization` instead of `ListProjects`.
- **Project settings**: Create project scoped to active org. Project groups scoped to active org.
- **User management**: Move to org settings (manage org members). Instance admin can still see all users.
- **API tokens**: Global tokens become org-scoped. Project tokens unchanged.

### Auth Context Changes

```typescript
interface AuthContext {
  user: User
  organizations: Organization[]
  activeOrg: Organization | null
  switchOrg: (orgId: string) => void
}

interface Organization {
  id: string
  name: string
  slug: string
  role: 'owner' | 'admin' | 'member'  // user's role in this org
}
```

---

## Implementation Phases

### Phase 1: Database & Backend Foundation
- [ ] Create migration: `organizations` table, `organization_members` table
- [ ] Create migration: Add `organization_id` to `projects`, `project_groups`
- [ ] Create migration: Add `instance_admin` to `users`
- [ ] Create migration: Data backfill (default org, migrate users, assign projects)
- [ ] Add sqlc queries for organizations and membership
- [ ] Run `make sqlc` to regenerate
- [ ] Create `internal/organization/handler.go` — CRUD for orgs
- [ ] Create `internal/organization/member.go` — member management
- [ ] Add org access middleware
- [ ] Update `RequireAdmin` to check per-org role
- [ ] Update project handler: create project with org_id, list by org
- [ ] Update project groups handler: scope by org
- [ ] Add `X-Organization-ID` header parsing to auth middleware
- [ ] Update user invite flow: invite to org (create user if new, add membership as `invited`; on first login to that org, transition to `active`)

### Phase 2: Auth & Login Flow
- [ ] Update OAuth handler: resolve orgs on login, return org list
- [ ] Implement domain-based auto-join (`allowed_domains`)
- [ ] Update session/token to carry active org context
- [ ] Update API token validation: check org scope
- [ ] Handle edge case: user with no orgs (first user creates default)

### Phase 3: Frontend — Org Switcher & Context
- [ ] Add `Organization` type to API client
- [ ] Add org endpoints to API client
- [ ] Update `AuthContext` with org list and active org
- [ ] Add `X-Organization-ID` header to all API calls
- [ ] Build org switcher dropdown in sidebar
- [ ] Store active org in `localStorage`
- [ ] Update project list to use `ListProjectsByOrganization`
- [ ] Update project creation to use active org

### Phase 4: Frontend — Org Management UI
- [ ] Build org settings page (name, slug, allowed domains)
- [ ] Build member management UI (list, invite, role change, remove)
- [ ] Build instance admin page (list orgs, create org)
- [ ] Move user management into org settings
- [ ] Update project groups to show org-scoped groups only
- [ ] Update API token creation: org-scoped instead of global

### Phase 5: Polish & Edge Cases
- [ ] Handle org deletion (hard cascade: delete all projects/issues/events, soft-delete the org row for audit)
- [ ] Handle member removal (soft-delete membership, assignments and tickets remain as-is since user still exists globally)
- [ ] Transfer project between orgs (admin-only)
- [ ] Org-scoped AI token budgets
- [ ] Update alert email templates to include org name
- [ ] Audit log: track org-level actions (member added/removed, role changed)

---

## Backward Compatibility

### Single-Org Instances

For existing deployments that don't need multi-tenancy:
- Auto-migration creates a "Default" org with all existing data
- No env vars or config changes required
- UI works identically — org switcher shows one org, behaves like a label
- All existing API calls work (resolve to the user's only org)

### API Compatibility

- All existing endpoints keep their paths
- `X-Organization-ID` header is optional — defaults to user's first (or only) org
- API tokens created before migration continue to work (backfilled to default org)
- DSN keys and ingest endpoints are completely unchanged

### Migration Safety

- All schema changes are additive (new tables, new nullable columns)
- Backfill runs in a single transaction
- NOT NULL constraint added only after backfill
- Rolling deployment safe: old code ignores `organization_id` column

---

## Out of Scope

- **Per-project permissions** (some members see project A but not B within the same org) — can be added later as a separate feature
- **SSO/SAML** — org-level SSO is a natural follow-up but not part of this epic
- **Billing/quotas per org** — infrastructure for it exists (org is the boundary) but no billing logic
- **Org-level settings** (default cooldown, retention, etc.) — projects keep their own settings for now
- **Cross-org project sharing** — a project belongs to exactly one org
- **Nested orgs / teams within orgs** — flat org model for now
- **Shared cache (Redis)** — in-memory caches are per-process, won't scale across multiple servers. Needs a separate epic when horizontal scaling is required
- **GDPR data purge** — full PII deletion per org (events, user identifiers). Separate epic for compliance tooling

---

## Effort Estimate

- Phase 1 (DB + Backend): ~12h
- Phase 2 (Auth flow): ~6h
- Phase 3 (Frontend switcher): ~6h
- Phase 4 (Org management UI): ~8h
- Phase 5 (Polish): ~4h
- **Total: ~36h**
