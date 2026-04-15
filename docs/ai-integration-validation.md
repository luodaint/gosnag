# Validation: AI Integration Requirements

**Type:** Epic
**Date:** 2026-04-12
**Status:** Approved

## Summary

The AI integration requirements document is comprehensive and well-structured with 70+ requirements covering 6 feature blocks. Validation found 13 gaps — all clarified during the interactive session. A second review identified 7 inconsistencies — all corrected in the requirements document. The document is now internally consistent and ready for design.

## Checklist Results

### Passed
- [x] Objective and value proposition are explicit (Section 1.1)
- [x] In-scope items are explicitly listed (Sections 2-11)
- [x] Out-of-scope items are explicitly listed (Section 1.2)
- [x] Data protection requirements called out (Section 10)
- [x] Security constraints stated (API key handling, _set pattern)
- [x] Primary entities are identified with properties
- [x] CRUD operations are addressed per entity
- [x] Statuses and transitions defined (merge suggestions: pending/accepted/dismissed)
- [x] Normal use cases documented
- [x] Error scenarios documented (Section 9)
- [x] Performance expectations stated (acceptance criteria: 10s, 30s, 15min)
- [x] Feature toggles defined per project (REQ-AI-030)
- [x] Implementation phases defined (Section 12)
- [x] Acceptance criteria defined (Section 13)

### Clarified During Validation

- [x] **Missing data models** — Token usage, RCA storage, and triage suggestions tables will be defined during design phase
- [x] **Admin Settings vs Env Vars** — Env vars only for now (REQ-UI-010 admin UI is out of scope)
- [x] **Auto-merge concurrency** — Auto-merge runs as a batch job every 5 minutes (only if new issues exist since last check), not inline with ingestion. Eliminates race conditions.
- [x] **Description generation mechanism** — On-demand only. User clicks "Generate description" button, synchronous response fills the editor. REQ-DESC-002 (async on ticket creation) is incorrect and should be updated.
- [x] **Overlapping deploy analysis windows** — New deploy cancels any pending analysis for the previous deploy
- [x] **Token tracking persistence** — Single `ai_usage_log` table serves both audit (REQ-AI-050) and token tracking (REQ-AI-040). One row per AI call.
- [x] **Rate limiting** — Query `ai_usage_log` count in last 60 seconds per project. DB-based, survives restarts.
- [x] **RCA regeneration** — Versioned: keep history of analyses per issue, not replace
- [x] **Prompt caching** — Hash-based cache in `ai_usage_log`. Store prompt hash + response, check before calling provider. 5-minute window.
- [x] **Merge suggestion chains** — Skip candidates that already have pending merge suggestions (as source or target)
- [x] **Triage assignee source** — Deferred to Phase 3. No decision needed now.
- [x] **AI system observability** — No dashboard for now. `ai_usage_log` has the raw data. Project settings shows today's token usage.
- [x] **GDPR/PII stripping** — Deferred to Phase 3. Documented as known risk for external providers. Ollama users are safe.

### Corrected After Review (2nd pass)

- [x] **Auto-merge flag gap** — Added `ai_merge_suggestions` flag separate from `ai_auto_merge`. MVP uses suggestions only (Phase 1), auto-merge in Phase 2.
- [x] **Auto-merge SLA contradiction** — Acceptance criteria updated from "30s" to "within 5 minutes" to match batch job design.
- [x] **Ticket description inconsistency** — REQ-DESC-022 corrected: description returned to frontend, NOT auto-saved. On-demand, not on-creation. HTML format (not Markdown).
- [x] **RCA contract incomplete** — Canonical format defined: JSON with `summary/evidence/suggested_fix` fields → stored in dedicated columns → rendered as labeled Markdown sections. API is issue-only; ticket page links to issue analysis.
- [x] **HTML sanitization missing** — REQ-DESC-020 now requires backend HTML sanitization with tag allowlist before returning AI-generated content.
- [x] **Multi-provider ambiguity** — Clarified: single global provider for all features. Per-feature provider selection added to Non-Goals.
- [x] **ai_usage_log operational rules** — Added REQ-DATA-002: cache hits don't count against token budget, failures count against rate limit, cached_response cleaned by background goroutine.

### Accepted Risks

- [!] **PII in prompts to external providers** — Stack traces, breadcrumbs, and request context may contain personal data. PII stripping deferred to Phase 3. Mitigation: Ollama keeps data local; external providers are opt-in per project.
- [!] **XSS via AI-generated HTML** — Mitigated by backend HTML sanitization with tag allowlist (REQ-DESC-020).
- [!] **No KPIs with baseline/target values** — Goals mention "reduce mean time to triage" but no specific numbers. Acceptable for internal tooling.

### Skipped

- [ ] **Triage assignee data model** — Deferred to Phase 3 design
- [ ] **AI admin settings UI** — Out of scope; env vars only for provider config

## Entity Analysis

| Entity | Create | Read | Update | Delete | States |
|--------|--------|------|--------|--------|--------|
| ai_merge_suggestions | Auto (batch) | API | Accept/Dismiss | CASCADE | pending → accepted/dismissed |
| deploy_analyses | Auto (worker) | API | — | CASCADE | — (stored result) |
| ai_usage_log | Auto (every call) | Aggregate queries | — | — | — (append-only) |
| ai_analyses (RCA) | On-demand | API | — (versioned, new row) | CASCADE | — (versioned history) |
| ai_triage_suggestions | Auto (Phase 3) | API (Phase 3) | Apply | CASCADE | pending → applied/dismissed |

## Collateral Impact

- **Ingestion flow**: No direct impact — auto-merge is decoupled as a batch job
- **Existing merge**: Auto-merge uses existing merge functionality; must verify compatibility
- **Activity log**: Auto-merge creates activity entries (ai_auto_merged action)
- **Alert system**: Deploy anomaly detection sends alerts through existing email + Slack channels
- **Ticket system**: AI description generation hooks into existing ticket detail page (new button)
- **Project settings**: New AI section in project settings UI

## Anti-Patterns Found

- ~~**REQ-DESC-002 contradicts intended behavior**~~ — Corrected
- ~~**REQ-UI-010 (Admin Settings UI) conflicts with env-var-only approach**~~ — Marked out of scope

## Recommendations

All recommendations from the first pass have been addressed:
1. ~~Correct REQ-DESC-002~~ — Done
2. ~~Remove REQ-UI-010 from scope~~ — Done
3. ~~Define the three missing tables~~ — Done (Section 11)
4. ~~Add a formal risk section~~ — Done (Section 12)

## Decision
> **The user decides whether to proceed.** This validation is informational.
