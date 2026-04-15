# Issue Merge — Manual Issue Grouping

## Overview
Allow manually merging duplicate issues into one. Events from merged issues are reassigned to the primary issue. Future events matching merged fingerprints automatically go to the primary issue via an alias table.

## Design

### Database
- New table `issue_aliases`: maps alternative fingerprints → primary issue ID
- When merging: move events, sum counts, keep earliest first_seen/latest last_seen, delete secondary issues, register their fingerprints as aliases

### Ingest Flow Change
- Before upserting a new issue, check `issue_aliases` for the fingerprint
- If alias exists, redirect to the primary issue (use its fingerprint for the upsert)
- This ensures future events with merged fingerprints go to the correct issue

### API
- POST `/api/v1/projects/{id}/issues/merge` — body: `{ primary_id, issue_ids: [...] }`
- Validates all issues belong to same project
- Moves events, creates aliases, deletes secondary issues
- Returns updated primary issue

### Frontend
- Checkbox selection on issue list
- "Merge" button appears when 2+ issues selected
- Dialog: choose which issue is the primary (default: most events)
- Confirm and merge

## Tasks

### Backend — Database
- [ ] Migration `000009_issue_aliases` — create `issue_aliases` table
- [ ] SQL queries: create alias, lookup alias by fingerprint+project, move events, delete issues
- [ ] Run sqlc

### Backend — Merge Handler
- [ ] POST endpoint for merge operation
- [ ] Transaction: move events → update primary counts/timestamps → create aliases → delete secondaries
- [ ] Register in router

### Backend — Ingest Change
- [ ] Before UpsertIssue, check issue_aliases for fingerprint redirect
- [ ] If alias found, use primary issue's fingerprint instead

### Frontend
- [ ] Checkbox selection mode on IssueList
- [ ] Merge button + confirmation dialog
- [ ] Primary issue selector in dialog
