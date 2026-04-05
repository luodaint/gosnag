package priority

import (
	"encoding/json"
	"net/http"

	"github.com/darkspock/gosnag/internal/database/db"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	queries *db.Queries
}

func NewHandler(queries *db.Queries) *Handler {
	return &Handler{queries: queries}
}

type RuleRequest struct {
	Name      string `json:"name"`
	RuleType  string `json:"rule_type"`
	Pattern   string `json:"pattern"`
	Operator  string `json:"operator"`
	Threshold int32  `json:"threshold"`
	Points    int32  `json:"points"`
	Enabled   bool   `json:"enabled"`
}

func (h *Handler) ListRules(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "project_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	rules, err := h.queries.ListPriorityRules(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list rules")
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

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
	if req.RuleType == "" {
		writeError(w, http.StatusBadRequest, "rule_type is required")
		return
	}

	rule, err := h.queries.CreatePriorityRule(r.Context(), db.CreatePriorityRuleParams{
		ProjectID: projectID,
		Name:      req.Name,
		RuleType:  req.RuleType,
		Pattern:   req.Pattern,
		Operator:  req.Operator,
		Threshold: req.Threshold,
		Points:    req.Points,
		Enabled:   req.Enabled,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create rule")
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

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

	rule, err := h.queries.UpdatePriorityRule(r.Context(), db.UpdatePriorityRuleParams{
		ID:        ruleID,
		ProjectID: projectID,
		Name:      req.Name,
		RuleType:  req.RuleType,
		Pattern:   req.Pattern,
		Operator:  req.Operator,
		Threshold: req.Threshold,
		Points:    req.Points,
		Enabled:   req.Enabled,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update rule")
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

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

	err = h.queries.DeletePriorityRule(r.Context(), db.DeletePriorityRuleParams{
		ID:        ruleID,
		ProjectID: projectID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete rule")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// RecalcAll recalculates priority for all issues in a project.
func (h *Handler) RecalcAll(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "project_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	count, err := EvaluateAll(r.Context(), h.queries, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to recalculate")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"recalculated": count})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
