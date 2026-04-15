# AI Integration — Requirements Document

Reference: [ai-integration-epic.md](ai-integration-epic.md)

---

## 1. Overview

Add AI capabilities to GoSnag for automated issue analysis, duplicate detection, deploy monitoring, and ticket assistance. Support multiple AI providers to allow teams to choose based on cost, latency, privacy, and existing infrastructure.

### 1.1 Goals

- Reduce mean time to triage by automating duplicate detection and priority suggestions
- Reduce time writing ticket descriptions by auto-generating context-rich first drafts
- Detect deploy-related regressions automatically and alert before manual detection
- Provide root cause analysis that correlates stack traces, commits, and deploy history
- Allow self-hosted AI (Ollama) for teams with data privacy requirements

### 1.2 Non-Goals

- Training custom models on project data
- AI-powered alert rule authoring
- Natural language search
- AI code fix suggestions or auto-generated PRs
- Multi-provider fallback chains
- Per-feature provider selection (e.g., Groq for merge, OpenAI for RCA). One global provider for all features. The epic's recommendation about mixing providers is aspirational — may be revisited in a future iteration.

---

## 2. AI Provider Infrastructure

### 2.1 Provider Interface

**REQ-AI-001**: The system MUST define a Go interface `AIProvider` with methods:
- `Chat(ctx, request) → response, error`
- `Name() → string`

**REQ-AI-002**: The `ChatRequest` struct MUST support:
- System prompt (string)
- Message list (role + content)
- Max tokens (int)
- Temperature (float)
- JSON mode flag (bool)

**REQ-AI-003**: The `ChatResponse` struct MUST return:
- Content (string)
- Tokens used (int)

### 2.2 Provider Implementations

**REQ-AI-010**: The system MUST implement an OpenAI provider.
- Endpoint: `POST https://api.openai.com/v1/chat/completions`
- Auth: Bearer token from `AI_API_KEY`
- JSON mode: `response_format: { type: "json_object" }`
- Models: gpt-4o, gpt-4o-mini, gpt-4-turbo, o1-mini

**REQ-AI-011**: The system MUST implement a Groq provider.
- Endpoint: `POST https://api.groq.com/openai/v1/chat/completions`
- Auth: Bearer token from `AI_API_KEY`
- API is OpenAI-compatible; implementation MAY share code with OpenAI provider via a base URL parameter
- Models: llama-3.3-70b-versatile, llama-3.1-8b-instant, mixtral-8x7b-32768

**REQ-AI-012**: The system MUST implement an Amazon Bedrock provider.
- API: AWS SDK v2 `Converse` API
- Auth: AWS credential chain (env vars `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY`, instance role, SSO profile)
- Region: from `AI_BEDROCK_REGION` (default: `eu-west-1`)
- Model ID: from `AI_BEDROCK_MODEL_ID`
- Models: anthropic.claude-3-haiku, anthropic.claude-3-sonnet, meta.llama3-70b-instruct, amazon.titan-text-express

**REQ-AI-013**: The system SHOULD implement an Anthropic Claude provider (direct API, not through Bedrock).
- Endpoint: `POST https://api.anthropic.com/v1/messages`
- Auth: `x-api-key` header
- JSON mode: via system prompt instruction

**REQ-AI-014**: The system SHOULD implement an Ollama provider.
- Endpoint: configurable via `AI_BASE_URL` (default: `http://localhost:11434`)
- API: `POST /api/chat`
- Auth: none (local)
- No token cost. Self-hosted.

**REQ-AI-015**: The system MAY implement a Google Gemini provider in a future iteration.

### 2.3 Configuration

**REQ-AI-020**: Global AI configuration via environment variables only (no admin UI for provider config):

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `AI_PROVIDER` | Yes (if AI used) | `""` | Provider: `openai`, `groq`, `bedrock`, `claude`, `ollama` |
| `AI_API_KEY` | Provider-dependent | `""` | API key (not needed for Bedrock/Ollama) |
| `AI_MODEL` | No | Provider default | Model name/ID |
| `AI_BASE_URL` | No | `""` | Custom endpoint URL |
| `AI_BEDROCK_REGION` | Bedrock only | `eu-west-1` | AWS region |
| `AI_BEDROCK_MODEL_ID` | Bedrock only | `""` | Bedrock model ID |
| `AI_MAX_TOKENS_PER_DAY` | No | `100000` | Token budget per project per day |
| `AI_MAX_CALLS_PER_MINUTE` | No | `10` | Rate limit |

**REQ-AI-021**: If `AI_PROVIDER` is empty or not set, all AI features MUST be disabled without error. The system operates normally without AI.

**REQ-AI-022**: If `AI_PROVIDER` is set but the API key is invalid, the system MUST log a warning at startup and disable AI features gracefully.

### 2.4 Per-Project Settings

**REQ-AI-030**: Each project MUST have the following AI-related settings:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `ai_enabled` | boolean | false | Master toggle |
| `ai_model` | string | "" | Override global model |
| `ai_merge_suggestions` | boolean | false | Evaluate new issues for duplicates and show merge suggestion banners |
| `ai_auto_merge` | boolean | false | Auto-merge duplicates without user confirmation (requires `ai_merge_suggestions = true`) |
| `ai_anomaly_detection` | boolean | false | Run post-deploy anomaly analysis |
| `ai_ticket_description` | boolean | true | Show "Generate description" button on tickets |
| `ai_root_cause` | boolean | false | Enable root cause analysis button |
| `ai_triage` | boolean | false | Show triage suggestions |

**REQ-AI-031**: All AI features MUST check both the global provider config AND the project's `ai_enabled` flag before executing. If either is disabled, the feature MUST NOT run.

**REQ-AI-032**: The project settings UI MUST show the AI section only when a global AI provider is configured.

### 2.5 Cost Control

**REQ-AI-040**: The system MUST track token usage per project per day.

**REQ-AI-041**: If a project exceeds its daily token budget (`AI_MAX_TOKENS_PER_DAY`), all AI features for that project MUST be paused until the next calendar day (UTC).

**REQ-AI-042**: The system MUST rate-limit AI calls to `AI_MAX_CALLS_PER_MINUTE` per project. Rate limiting is DB-based: query `ai_usage_log` count in the last 60 seconds for the project. Calls exceeding the limit MUST be dropped silently (no error to the user for background features) or return a user-facing message for on-demand features.

**REQ-AI-043**: The system SHOULD cache identical prompts within a 5-minute window and return the cached response. Caching is hash-based: store a prompt hash + response in `ai_usage_log`, check before calling the provider.

### 2.6 Audit

**REQ-AI-050**: Each AI call MUST be logged in the `ai_usage_log` table with: project ID, feature name, model used, input token count, output token count, timestamp, latency (ms), prompt hash, cached response (for prompt caching).

> This single table serves three purposes: audit logging, token budget tracking (REQ-AI-040/041), and rate limiting (REQ-AI-042 — query count in last 60s per project).

**REQ-AI-051**: The system MUST NOT log the full prompt or response content. Only metadata. Exception: cached responses are stored for the 5-minute cache window (REQ-AI-043).

**REQ-AI-052**: Token usage MUST be visible in Project Settings → AI section (today's usage, this week's usage). No separate AI dashboard for now — the `ai_usage_log` table has raw data for future dashboards.

---

## 3. Smart Auto-Merge

### 3.1 Trigger

**REQ-MERGE-001**: The system MUST run a periodic batch job (every 5 minutes) that checks for new issues since the last run. For each project with `ai_enabled = true` AND `ai_merge_suggestions = true`, it MUST evaluate new issues for potential duplicates. The job MUST NOT run if there are no new issues since the last check.

**REQ-MERGE-002**: The batch job runs as a background goroutine. It MUST NOT block any request or ingestion flow.

### 3.2 Input

**REQ-MERGE-010**: The system MUST fetch up to 10 open issues in the same project, ordered by last_seen DESC, including their latest event's stack trace (top 5 frames). Issues that already have a pending merge suggestion (as source or target) MUST be excluded from candidates.

**REQ-MERGE-011**: The system MUST send the new issue's title, stack trace (top 5 frames), level, and platform, alongside the candidate issues, to the AI provider.

### 3.3 Output

**REQ-MERGE-020**: The AI response MUST be parsed as JSON with fields:
- `merge_with`: issue ID (string) or null
- `confidence`: float 0.0–1.0
- `reason`: string explanation

**REQ-MERGE-021**: If `confidence >= 0.8` AND `ai_auto_merge = true`, the system MUST automatically merge the new issue into the target issue using the existing merge functionality. An activity entry MUST be recorded with action `ai_auto_merged`.

**REQ-MERGE-022**: If `confidence >= 0.8` AND `ai_auto_merge = false`, the system MUST create a merge suggestion record (pending status) and NOT merge automatically.

**REQ-MERGE-023**: If `confidence < 0.8`, no action. The suggestion MUST NOT be stored.

### 3.4 Data Model

**REQ-MERGE-030**: New table `ai_merge_suggestions`:
- id (UUID, PK)
- issue_id (UUID, FK → issues, ON DELETE CASCADE)
- target_issue_id (UUID, FK → issues, ON DELETE CASCADE)
- confidence (REAL)
- reason (TEXT)
- status (TEXT: 'pending', 'accepted', 'dismissed')
- created_at (TIMESTAMPTZ)

### 3.5 UI

**REQ-MERGE-040**: If an issue has a pending merge suggestion, the issue detail page MUST show a banner with:
- The target issue title
- The confidence percentage
- The AI-generated reason
- "Merge" button (executes merge, sets status = 'accepted')
- "Dismiss" button (sets status = 'dismissed')

**REQ-MERGE-041**: Dismissed suggestions MUST NOT reappear.

### 3.6 API

**REQ-MERGE-050**: `GET /projects/{id}/issues/{id}/merge-suggestion` — returns the pending suggestion or null.
**REQ-MERGE-051**: `POST /projects/{id}/issues/{id}/merge-suggestion/accept` — executes merge.
**REQ-MERGE-052**: `POST /projects/{id}/issues/{id}/merge-suggestion/dismiss` — dismisses.

---

## 4. Deploy Anomaly Detection

### 4.1 Trigger

**REQ-DEPLOY-001**: When a deploy is recorded (`POST /projects/{id}/deploys`), IF the project has `ai_enabled = true` AND `ai_anomaly_detection = true`, the system MUST schedule an analysis to run 15 minutes after the deploy timestamp. If a new deploy is recorded while a previous analysis is still pending, the previous analysis MUST be cancelled.

**REQ-DEPLOY-002**: The analysis MUST run as a background worker, not blocking any request.

### 4.2 Data Collection

**REQ-DEPLOY-010**: The system MUST query:
- Events in the 15 minutes after deploy vs. the 15 minutes before deploy
- New issues (fingerprints that didn't exist before the deploy)
- Issues with event velocity increase > 3x post-deploy
- Issues that transitioned from resolved/ignored to reopened within the post-deploy window

**REQ-DEPLOY-011**: If no anomalies are detected (no new issues, no spikes, no reopens), the system MUST store an analysis with severity = 'none' and NOT call the AI provider.

### 4.3 AI Analysis

**REQ-DEPLOY-020**: If anomalies are detected, the system MUST send them to the AI provider with:
- Deploy info (version, commit, environment, timestamp)
- List of new issues (title + top 3 stack frames)
- List of spiked issues (title + pre/post event rate)
- List of reopened issues
- Commit diff summary (if source code integration is configured)

**REQ-DEPLOY-021**: The AI response MUST be parsed as JSON:
- `severity`: "critical", "warning", "info", "none"
- `summary`: one-line summary
- `details`: multi-line explanation
- `likely_caused_by_deploy`: boolean
- `recommended_action`: "rollback", "investigate", "monitor", "ignore"

### 4.4 Actions

**REQ-DEPLOY-030**: On severity "critical": send alert via all configured channels (email + Slack).
**REQ-DEPLOY-031**: On severity "warning": send alert via all configured channels.
**REQ-DEPLOY-032**: On severity "info" or "none": no alert. Store the analysis.

### 4.5 Data Model

**REQ-DEPLOY-040**: New table `deploy_analyses`:
- id (UUID, PK)
- deploy_id (UUID, FK → deploys, ON DELETE CASCADE)
- project_id (UUID, FK → projects, ON DELETE CASCADE)
- severity (TEXT)
- summary (TEXT)
- details (TEXT)
- likely_deploy_caused (BOOLEAN)
- recommended_action (TEXT)
- new_issues_count (INT)
- spiked_issues_count (INT)
- reopened_issues_count (INT)
- created_at (TIMESTAMPTZ)

### 4.6 UI

**REQ-DEPLOY-050**: The project page MUST show a deploy health banner when the latest deploy has a critical or warning analysis.

**REQ-DEPLOY-051**: A deploys page (`/projects/{id}/deploys`) MUST list recent deploys with their AI analysis summary and severity badge.

---

## 5. AI Ticket Description

### 5.1 Trigger

**REQ-DESC-001**: AI description generation is on-demand. The user clicks a "Generate description" button on the ticket detail page. The system checks `ai_enabled = true` AND `ai_ticket_description = true` before executing.

**REQ-DESC-002**: The generation is synchronous: the API call blocks until the AI responds (up to 30s timeout), then returns the generated HTML directly. The frontend fills the WYSIWYG editor with the response.

**REQ-DESC-003**: The "Generate description" button MUST be available on tickets linked to an issue (the issue provides context for generation). For manually created tickets with no linked issue, the button SHOULD still be available but will use only the ticket title and any existing description as context.

### 5.2 Input

**REQ-DESC-010**: The system MUST gather:
- Issue title, level, platform, culprit
- Latest event's stack trace (top 10 frames)
- Event count and user count
- First seen and last seen timestamps
- Breadcrumbs from latest event (last 10)
- Request context (method, URL, headers)
- Tags
- Suspect commits (if source code integration is configured)

### 5.3 Output

**REQ-DESC-020**: The AI response MUST be formatted as HTML (to be inserted into the WYSIWYG editor). The backend MUST sanitize the HTML before returning it — strip `<script>`, `<iframe>`, `on*` event attributes, and any other XSS vectors. Use an allowlist of safe tags (`p`, `h1`-`h3`, `ul`, `ol`, `li`, `strong`, `em`, `code`, `pre`, `a`, `br`, `blockquote`, `table`, `thead`, `tbody`, `tr`, `th`, `td`).

**REQ-DESC-021**: The description MUST include:
- A summary of the error
- Likely root cause based on the stack trace
- Impact assessment (event count, user count, frequency)
- Suggested investigation steps

**REQ-DESC-022**: The generated description is returned to the frontend, NOT auto-saved. The user reviews and edits it in the WYSIWYG editor, then saves via the normal ticket update flow. The API endpoint returns the HTML content directly; it does NOT write to the database.

### 5.4 UI

**REQ-DESC-030**: The ticket detail page MUST show a "Generate description" button when AI is enabled for the project. Clicking the button shows a loading state while the AI generates the description.

**REQ-DESC-031**: Once generated, the description MUST appear in the WYSIWYG editor, editable by the user.

**REQ-DESC-032**: If the ticket already has a description, the button label SHOULD change to "Regenerate description" and the new content replaces the existing description in the editor (user can undo via editor).

### 5.5 API

**REQ-DESC-040**: `POST /projects/{id}/tickets/{id}/generate-description` — triggers AI description generation. Returns the generated HTML directly: `{ description: "<html content>" }`. Synchronous call (up to 30s).
**REQ-DESC-041**: The frontend receives the description in the response and fills the WYSIWYG editor. The description is NOT auto-saved — the user reviews and saves via the normal ticket update flow.

---

## 6. AI Root Cause Analysis

### 6.1 Trigger

**REQ-RCA-001**: Root cause analysis is on-demand only. The user clicks "Analyze" on an issue detail page. On the ticket detail page, the "Analyze" button is available only if the ticket is linked to an issue — it triggers analysis on the linked issue and displays the result.

**REQ-RCA-002**: The system MUST check `ai_enabled = true` AND `ai_root_cause = true` before executing.

### 6.2 Input

**REQ-RCA-010**: The system MUST gather:
- Full stack trace of the latest event
- Event timeline (last 24h: are events increasing, stable, bursty?)
- Top 5 similar issues in the project (by stack trace similarity)
- Suspect commits (if source code integration is configured)
- Recent deploys (last 3)
- Tags, environment, release

### 6.3 Output

**REQ-RCA-020**: The AI MUST return a JSON response with three fields:
- `summary` (string): 1–2 sentence conclusion
- `evidence` (array of strings): what data supports the conclusion
- `suggested_fix` (string): actionable steps in Markdown format

This is the **canonical format**. Storage uses dedicated columns matching these fields (`ai_analyses` table). The UI renders each field as a labeled section using Markdown rendering (same `react-markdown` + `remark-gfm` already used in comments).

**REQ-RCA-021**: The analysis MUST be stored and displayed. It MUST NOT be regenerated on every page load. Analyses are versioned — regenerating creates a new entry (incrementing `version`), preserving history.

### 6.4 UI

**REQ-RCA-030**: The issue detail page MUST show a collapsible "AI Analysis" section with the analysis content rendered as Markdown.

**REQ-RCA-031**: A "Regenerate" button MUST allow the user to request a fresh analysis.

**REQ-RCA-032**: A "Copy to ticket" button MUST copy the analysis content to the linked ticket's description (appended, not replaced).

### 6.5 API

**REQ-RCA-040**: `POST /projects/{id}/issues/{id}/analyze` — triggers analysis. Returns the new analysis.
**REQ-RCA-041**: `GET /projects/{id}/issues/{id}/analysis` — returns the latest stored analysis or null.
**REQ-RCA-042**: `GET /projects/{id}/issues/{id}/analyses` — returns all stored analyses for the issue, ordered by created_at DESC (version history).

---

## 7. AI Triage Suggestions

### 7.1 Trigger

**REQ-TRIAGE-001**: When a new issue is created, IF the project has `ai_enabled = true` AND `ai_triage = true`, the system MUST generate triage suggestions.

**REQ-TRIAGE-002**: The suggestions MUST be generated asynchronously after event ingestion.

### 7.2 Input

**REQ-TRIAGE-010**: The system MUST gather:
- Stack trace files/modules
- Suspect commits (who last touched those files)
- Historical assignment patterns (who was assigned similar issues)
- Event velocity, user count, error level

### 7.3 Output

**REQ-TRIAGE-020**: The AI response MUST include:
- `suggested_assignee`: user ID or null
- `assignee_reason`: one-line explanation
- `suggested_priority`: integer (90/70/50/25)
- `priority_reason`: one-line explanation

**REQ-TRIAGE-021**: Suggestions MUST be stored per issue.

### 7.4 UI

**REQ-TRIAGE-030**: The issue detail page MUST show suggestions inline next to the assignee and priority dropdowns. Example: a lightbulb icon with tooltip showing the suggestion and reason.

**REQ-TRIAGE-031**: Clicking the suggestion MUST apply it (assign the user or set the priority). One-click accept.

### 7.5 API

**REQ-TRIAGE-040**: `GET /projects/{id}/issues/{id}/triage-suggestion` — returns the suggestion or null.

---

## 8. Settings UI

### 8.1 Project Settings — AI Section

**REQ-UI-001**: The Project Settings page MUST show an "AI" section when a global AI provider is configured.

**REQ-UI-002**: The section MUST contain:
- Master toggle: "Enable AI features" (controls `ai_enabled`)
- Feature toggles for each capability (auto-merge, anomaly detection, ticket descriptions, root cause, triage)
- Model override input (optional, text field)
- Token usage display: "Today: X tokens / Y budget" with progress bar

**REQ-UI-003**: Feature toggles MUST be disabled (grayed out) when the master toggle is off.

### 8.2 Admin Settings — AI Provider

> **OUT OF SCOPE** — Provider configuration is via environment variables only (REQ-AI-020). An admin UI for provider settings may be added in a future iteration.

---

## 9. Error Handling

**REQ-ERR-001**: If the AI provider returns an error (rate limit, timeout, invalid response), the system MUST:
- Log the error with provider name, HTTP status, and latency
- NOT crash, hang, or affect non-AI functionality
- Show a user-friendly message for on-demand features ("AI analysis temporarily unavailable")
- Silently skip for background features (auto-merge, triage, anomaly detection)

**REQ-ERR-002**: AI responses that fail JSON parsing MUST be logged and discarded. The feature MUST fall back to no-AI behavior.

**REQ-ERR-003**: All AI calls MUST have a timeout of 30 seconds. Exceeded timeouts MUST be treated as errors.

---

## 10. Privacy

**REQ-PRIV-001**: AI features MUST be opt-in per project (`ai_enabled = false` by default).

**REQ-PRIV-002**: With Ollama provider, no data leaves the GoSnag instance. The system MUST document this clearly in the settings UI.

**REQ-PRIV-003**: The system SHOULD offer a PII stripping option that removes email addresses and IP addresses from prompts before sending to external providers. **Deferred to Phase 3.** This is an accepted risk — stack traces, breadcrumbs, and request context may contain personal data when sent to external providers.

**REQ-PRIV-004**: API keys MUST never be returned in API responses. Use the `_set` boolean pattern.

---

## 11. Additional Data Models

> These tables were identified as missing during validation and added here.

### 11.1 AI Usage Log (Audit + Token Tracking + Rate Limiting + Cache)

**REQ-DATA-001**: New table `ai_usage_log`:
- id (UUID, PK)
- project_id (UUID, FK → projects, ON DELETE CASCADE)
- feature (TEXT) — 'auto_merge', 'description', 'rca', 'anomaly', 'triage'
- model (TEXT) — model name used
- input_tokens (INT)
- output_tokens (INT)
- latency_ms (INT)
- prompt_hash (TEXT, nullable) — SHA-256 hash for caching
- cached_response (TEXT, nullable) — cached AI response (cleared after 5 min)
- created_at (TIMESTAMPTZ)

Indexes:
- `idx_ai_usage_log_project_day` ON (project_id, created_at) — for daily token budget queries
- `idx_ai_usage_log_prompt_hash` ON (project_id, prompt_hash, created_at) — for cache lookups

**REQ-DATA-002**: Operational rules for `ai_usage_log`:
- **Cache hits**: When a cached response is returned, a new row is logged with `input_tokens = 0`, `output_tokens = 0`, `latency_ms = 0`. Cache hits do NOT count against the daily token budget but DO count against the rate limit (to prevent abuse).
- **Failed calls**: Logged with `input_tokens = 0`, `output_tokens = 0` and the actual `latency_ms`. Failures count against the rate limit (to prevent retry storms) but NOT against the token budget.
- **Cache cleanup**: A background goroutine MUST clear `cached_response` values older than 5 minutes (set to NULL). This runs periodically (e.g., every minute) to prevent unbounded storage growth.
- **Log retention**: `ai_usage_log` rows are kept indefinitely for audit purposes. The `cached_response` column is the only one that gets cleaned.

### 11.2 AI Analyses (Root Cause Analysis — Versioned)

**REQ-DATA-010**: New table `ai_analyses`:
- id (UUID, PK)
- issue_id (UUID, FK → issues, ON DELETE CASCADE)
- project_id (UUID, FK → projects, ON DELETE CASCADE)
- summary (TEXT)
- evidence (TEXT) — JSON array of evidence items
- suggested_fix (TEXT)
- model (TEXT) — model used
- version (INT) — incremented on each regeneration
- created_at (TIMESTAMPTZ)

Index:
- `idx_ai_analyses_issue` ON (issue_id, created_at DESC) — for fetching latest/all

### 11.3 AI Triage Suggestions (Phase 3)

**REQ-DATA-020**: New table `ai_triage_suggestions` (Phase 3):
- id (UUID, PK)
- issue_id (UUID, FK → issues, ON DELETE CASCADE, UNIQUE)
- project_id (UUID, FK → projects, ON DELETE CASCADE)
- suggested_assignee_id (UUID, nullable, FK → users)
- assignee_reason (TEXT)
- suggested_priority (INT)
- priority_reason (TEXT)
- status (TEXT: 'pending', 'applied', 'dismissed')
- created_at (TIMESTAMPTZ)

---

## 12. Risks

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| PII sent to external AI providers | High (privacy) | Medium | PII stripping planned for Phase 3; Ollama keeps data local; AI is opt-in per project |
| XSS via AI-generated HTML | High (security) | Low | Backend HTML sanitization with tag allowlist (REQ-DESC-020) |
| AI provider outage blocks on-demand features | Medium (UX) | Low | 30s timeout, graceful fallback to no-AI behavior (REQ-ERR-001) |
| Token budget exhaustion on high-volume projects | Medium (features paused) | Medium | Configurable budget, batch job pattern limits calls, project settings show usage |
| Auto-merge false positive | High (data integrity) | Low | Confidence threshold 0.8, manual-accept mode available, merge can be undone |

---

## 13. Testing

> Section renumbered from 11 to 13 after adding sections 11 (Data Models) and 12 (Risks).

**REQ-TEST-001**: Each provider implementation MUST have unit tests that verify request construction and response parsing using a mock HTTP server.

**REQ-TEST-002**: The auto-merge feature MUST have integration tests verifying: merge on high confidence, suggestion on medium confidence, no action on low confidence.

**REQ-TEST-003**: The deploy anomaly detection MUST have tests for: no anomalies = no AI call, anomalies detected = AI called, critical severity = alert sent.

---

## 14. Implementation Phases

### Phase 1 (MVP)
- AI provider interface + OpenAI + Groq + Bedrock implementations
- Global config (env vars) + per-project settings (DB + UI, including `ai_merge_suggestions` flag)
- Token tracking, rate limiting, and `ai_usage_log` with cache + cleanup
- AI ticket description generation (on-demand with HTML sanitization)
- Auto-merge suggestions via batch job (`ai_merge_suggestions = true`, manual accept only, `ai_auto_merge` deferred to Phase 2)

### Phase 2
- Auto-merge (automatic execution when `ai_auto_merge = true`)
- Deploy anomaly detection + alerts
- Root cause analysis (on-demand)

### Phase 3
- Triage suggestions
- Claude + Ollama providers
- PII stripping
- Token usage dashboard

---

## 15. Acceptance Criteria

| Feature | Acceptance Criteria |
|---------|-------------------|
| Provider infra | Can switch between OpenAI, Groq, and Bedrock by changing env var; all features work with each |
| Ticket description | Clicking "Generate description" on a ticket produces a description within 10s that includes summary, root cause, and impact |
| Auto-merge suggestion | New issue that is a clear duplicate shows a merge suggestion banner within 5 minutes (next batch cycle) |
| Auto-merge execution | With `ai_auto_merge = true`, the duplicate is merged automatically within 5 minutes with activity log entry |
| Deploy anomaly | 15 min after a deploy that introduces errors, an alert is sent with severity and recommendation |
| Root cause analysis | Clicking "Analyze" generates a structured analysis within 15s |
| Triage suggestion | New issue shows assignee and priority suggestions within 30s |
| Cost control | Exceeding daily token budget pauses AI features; resumes next day |
| Privacy | Ollama provider keeps all data local; no external API calls |
