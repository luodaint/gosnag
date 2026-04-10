package github

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/darkspock/gosnag/internal/config"
	"github.com/darkspock/gosnag/internal/database/db"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
)

type Handler struct {
	queries *db.Queries
	cfg     *config.Config
}

func NewHandler(queries *db.Queries, cfg *config.Config) *Handler {
	return &Handler{queries: queries, cfg: cfg}
}

// TestConnection tests the GitHub connection for a project.
func (h *Handler) TestConnection(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "project_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	project, err := h.queries.GetProject(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	cfg := ConfigFromProject(project)
	if !cfg.IsConfigured() {
		writeError(w, http.StatusBadRequest, "GitHub is not configured for this project")
		return
	}

	if err := TestConnection(cfg); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// CreateIssueHandler manually creates a GitHub issue from a GoSnag issue.
func (h *Handler) CreateIssueHandler(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "project_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	issueID, err := uuid.Parse(chi.URLParam(r, "issue_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid issue id")
		return
	}

	project, err := h.queries.GetProject(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	cfg := ConfigFromProject(project)
	if !cfg.IsConfigured() {
		writeError(w, http.StatusBadRequest, "GitHub is not configured for this project")
		return
	}

	issue, err := h.queries.GetIssue(r.Context(), issueID)
	if err != nil {
		writeError(w, http.StatusNotFound, "issue not found")
		return
	}

	if issue.ProjectID != projectID {
		writeError(w, http.StatusBadRequest, "issue does not belong to this project")
		return
	}

	if issue.GithubIssueNumber.Valid {
		writeError(w, http.StatusConflict, fmt.Sprintf("issue already has a GitHub issue: #%d", issue.GithubIssueNumber.Int32))
		return
	}

	stacktrace := h.getLatestStacktrace(r, issueID)

	baseURL := h.cfg.BaseURL
	title := "[GoSnag] " + truncate(issue.Title, 200)
	body := BuildDescription(issue, baseURL, projectID.String(), stacktrace)

	result, err := CreateIssue(cfg, title, body)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to create GitHub issue: "+err.Error())
		return
	}

	res, err := h.queries.UpdateIssueGithubTicket(r.Context(), db.UpdateIssueGithubTicketParams{
		ID:                issueID,
		GithubIssueNumber: sql.NullInt32{Int32: int32(result.Number), Valid: true},
		GithubIssueUrl:    sql.NullString{String: result.URL, Valid: true},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("issue created in GitHub (#%d) but failed to save reference", result.Number))
		return
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		writeError(w, http.StatusConflict, "issue was linked to a GitHub issue concurrently")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"number": result.Number,
		"url":    result.URL,
	})
}

// ListRules lists all GitHub rules for a project.
func (h *Handler) ListRules(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "project_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	rules, err := h.queries.ListGithubRules(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list rules")
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

type RuleRequest struct {
	Name         string          `json:"name"`
	Enabled      bool            `json:"enabled"`
	LevelFilter  string          `json:"level_filter"`
	MinEvents    int32           `json:"min_events"`
	MinUsers     int32           `json:"min_users"`
	TitlePattern string          `json:"title_pattern"`
	Conditions   json.RawMessage `json:"conditions,omitempty"`
}

func toNullJSON(raw json.RawMessage) pqtype.NullRawMessage {
	if len(raw) == 0 || string(raw) == "null" {
		return pqtype.NullRawMessage{}
	}
	return pqtype.NullRawMessage{RawMessage: raw, Valid: true}
}

// CreateRule creates a new GitHub auto-creation rule.
func (h *Handler) CreateRule(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "project_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	var req RuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	rule, err := h.queries.CreateGithubRule(r.Context(), db.CreateGithubRuleParams{
		ProjectID:    projectID,
		Name:         req.Name,
		Enabled:      req.Enabled,
		LevelFilter:  req.LevelFilter,
		MinEvents:    req.MinEvents,
		MinUsers:     req.MinUsers,
		TitlePattern: req.TitlePattern,
		Conditions:   toNullJSON(req.Conditions),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create rule")
		return
	}

	writeJSON(w, http.StatusCreated, rule)
}

// UpdateRule updates a GitHub rule.
func (h *Handler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "project_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	ruleID, err := uuid.Parse(chi.URLParam(r, "rule_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid rule id")
		return
	}

	var req RuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rule, err := h.queries.UpdateGithubRule(r.Context(), db.UpdateGithubRuleParams{
		ID:           ruleID,
		ProjectID:    projectID,
		Name:         req.Name,
		Enabled:      req.Enabled,
		LevelFilter:  req.LevelFilter,
		MinEvents:    req.MinEvents,
		MinUsers:     req.MinUsers,
		TitlePattern: req.TitlePattern,
		Conditions:   toNullJSON(req.Conditions),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update rule")
		return
	}

	writeJSON(w, http.StatusOK, rule)
}

// DeleteRule deletes a GitHub rule.
func (h *Handler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "project_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	ruleID, err := uuid.Parse(chi.URLParam(r, "rule_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid rule id")
		return
	}

	err = h.queries.DeleteGithubRule(r.Context(), db.DeleteGithubRuleParams{
		ID:        ruleID,
		ProjectID: projectID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete rule")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) getLatestStacktrace(r *http.Request, issueID uuid.UUID) string {
	events, err := h.queries.ListEventsByIssue(r.Context(), db.ListEventsByIssueParams{
		IssueID: issueID,
		Limit:   1,
		Offset:  0,
	})
	if err != nil || len(events) == 0 {
		return ""
	}

	var data map[string]any
	if err := json.Unmarshal(events[0].Data, &data); err != nil {
		return ""
	}

	if exc, ok := data["exception"].(map[string]any); ok {
		if values, ok := exc["values"].([]any); ok && len(values) > 0 {
			if v, ok := values[0].(map[string]any); ok {
				if st, ok := v["stacktrace"].(map[string]any); ok {
					if frames, ok := st["frames"].([]any); ok {
						return formatFrames(frames)
					}
				}
			}
		}
	}

	return ""
}

func formatFrames(frames []any) string {
	start := 0
	if len(frames) > 10 {
		start = len(frames) - 10
	}

	var result string
	for i := start; i < len(frames); i++ {
		f, ok := frames[i].(map[string]any)
		if !ok {
			continue
		}
		file, _ := f["filename"].(string)
		lineno, _ := f["lineno"].(float64)
		fn, _ := f["function"].(string)
		if file != "" {
			result += fmt.Sprintf("  %s:%.0f in %s\n", file, lineno, fn)
		}
	}
	return result
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
