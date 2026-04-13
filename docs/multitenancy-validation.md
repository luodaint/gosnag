# Validation: Multi-Tenant Organizations

**Type:** Epic
**Date:** 2026-04-13
**Status:** Approved with Risks

## Summary
Solid epic with clear architecture, migration strategy, and backward compatibility. All critical entity states, business rules, and edge cases were clarified during validation. Two items deferred to future epics (shared cache, GDPR purge).

## Checklist Results

### Passed
- [x] Objective and value proposition are explicit
- [x] Target user segments specified (Cover + other teams sharing the instance)
- [x] Evidence included (Cover's need for isolation)
- [x] In-scope items explicitly listed (5 phases)
- [x] Out-of-scope items explicitly listed
- [x] Primary entities identified with properties (organizations, organization_members)
- [x] CRUD operations addressed (full API spec)
- [x] Normal use cases documented (login flow, org switching, invites)
- [x] Rolling deployment compatibility addressed
- [x] Migration safety documented (additive changes, backfill, then NOT NULL)
- [x] Backward compatibility documented (single-org instances, API compat, DSN unchanged)

### Clarified During Validation
- [x] Org deletion strategy — hard cascade on data, soft-delete the org row for audit trail
- [x] Member removal — soft-delete membership, assignments remain (user is global)
- [x] Domain conflict — domains not unique across orgs; if multiple match, prompt user to choose
- [x] Last owner protection — cannot remove or demote the last owner; enforced via `CountOrgOwners` check
- [x] `users.role` column — dropped after migration; replaced by `organization_members.role` + `users.instance_admin`
- [x] Role granularity — `owner` (org control), `admin` (project/config management), `member` (read + tickets + comments)
- [x] Invite flow — create user if new + membership as `invited`; existing user gets `invited` membership; transitions to `active` on first login
- [x] Org creation — instance_admin only
- [x] No feature flag — ship all at once
- [x] `users.status` — simplified to `active`/`disabled` (global gate); `organization_members.status` = `invited`/`active` (org onboarding)
- [x] OrgAccessMiddleware performance — cache org membership on login with 5min TTL, zero extra DB queries on hot path

### Accepted Risks
- [!] In-memory caches (velocity, stats) are per-process — won't scale to multiple servers. Deferred to future scaling epic.
- [!] GDPR full PII purge per org — not addressed. Deferred to future compliance epic.

### Skipped
- [ ] KPIs with baseline and target values — not defined (epic is infrastructure, not feature with measurable business metrics)
- [ ] Timeline / delivery window — not stated
- [ ] Observability/logging needs — not specified (audit log mentioned in Phase 5 but not detailed)

## Entity Analysis

### Organization
| Operation | Defined |
|-----------|---------|
| Create | Yes — `POST /api/organizations` (instance_admin only) |
| Read | Yes — `GET /api/organizations`, `GET /api/organizations/{org_id}` |
| Update | Yes — `PUT /api/organizations/{org_id}` (owner only) |
| Delete | Yes — hard cascade data + soft-delete org row (owner only) |

**Lifecycle:** Created → Active → Soft-deleted. No suspended/disabled state (not needed for now).

### Organization Member
| Operation | Defined |
|-----------|---------|
| Add | Yes — `POST /api/organizations/{org_id}/members` (admin+) |
| Read | Yes — `GET /api/organizations/{org_id}/members` |
| Update role | Yes — `PUT /api/organizations/{org_id}/members/{user_id}` (owner only) |
| Remove | Yes — soft-delete (admin+), last owner protected |

**Lifecycle:** Invited → Active (on first login). Soft-deleted on removal.

### Modified Entities
- **projects**: gains `organization_id` (NOT NULL after backfill)
- **project_groups**: gains `organization_id` (NOT NULL after backfill)
- **users**: gains `instance_admin`, drops `role`, `status` simplified to `active`/`disabled`
- **api_tokens**: gains `organization_id`

## Collateral Impact

- **Auth middleware**: Must check org membership from cache instead of global role
- **RequireAdmin**: Splits into per-org role check vs instance_admin check
- **All project queries**: Must be filtered by org (via middleware context)
- **Project groups**: Scoped by org
- **API tokens**: Global scope becomes org-scoped
- **User invite flow**: Now creates membership, not just user row
- **Frontend auth context**: Needs org list, active org, org switcher
- **Background workers**: No changes (already project-scoped), but caches noted as scaling risk

## Anti-Patterns Found
None.

## Recommendations
1. Consider adding an audit log for org-level actions (member add/remove/role change, org settings changes) — mentioned in Phase 5 but not detailed
2. Define observability: log org_id in structured logs for all requests to enable per-org debugging
3. Consider a future epic for org-level settings (default cooldown, retention, AI budget) once multi-tenant usage grows

## Decision
> **The user decides whether to proceed.** This validation is informational.
