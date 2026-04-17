package routegroup

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/darkspock/gosnag/internal/database/db"
	"github.com/darkspock/gosnag/internal/sourcecode"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	queries *db.Queries
}

func NewHandler(queries *db.Queries) *Handler {
	return &Handler{queries: queries}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "project_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	rules, err := ListRules(r.Context(), h.queries, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list route rules")
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
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

	var req struct {
		Method        string  `json:"method"`
		MatchPattern  string  `json:"match_pattern"`
		CanonicalPath string  `json:"canonical_path"`
		Target        string  `json:"target"`
		Source        string  `json:"source"`
		Confidence    float64 `json:"confidence"`
		Enabled       bool    `json:"enabled"`
		Framework     string  `json:"framework"`
		SourceFile    string  `json:"source_file"`
		Notes         string  `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.MatchPattern) == "" || strings.TrimSpace(req.CanonicalPath) == "" {
		writeError(w, http.StatusBadRequest, "match_pattern and canonical_path are required")
		return
	}

	rule, err := UpdateRuleFields(r.Context(), h.queries, projectID, ruleID, UpsertRuleInput{
		Method:        req.Method,
		MatchPattern:  req.MatchPattern,
		CanonicalPath: req.CanonicalPath,
		Target:        req.Target,
		Source:        req.Source,
		Confidence:    req.Confidence,
		Enabled:       req.Enabled,
		Framework:     req.Framework,
		SourceFile:    req.SourceFile,
		Notes:         req.Notes,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update route rule")
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "project_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	var req struct {
		Method        string  `json:"method"`
		MatchPattern  string  `json:"match_pattern"`
		CanonicalPath string  `json:"canonical_path"`
		Target        string  `json:"target"`
		Source        string  `json:"source"`
		Confidence    float64 `json:"confidence"`
		Enabled       bool    `json:"enabled"`
		Framework     string  `json:"framework"`
		SourceFile    string  `json:"source_file"`
		Notes         string  `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.MatchPattern) == "" || strings.TrimSpace(req.CanonicalPath) == "" {
		writeError(w, http.StatusBadRequest, "match_pattern and canonical_path are required")
		return
	}

	rule, err := CreateRule(r.Context(), h.queries, projectID, UpsertRuleInput{
		Method:        req.Method,
		MatchPattern:  req.MatchPattern,
		CanonicalPath: req.CanonicalPath,
		Target:        req.Target,
		Source:        req.Source,
		Confidence:    req.Confidence,
		Enabled:       req.Enabled,
		Framework:     req.Framework,
		SourceFile:    req.SourceFile,
		Notes:         req.Notes,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create route rule")
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
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

	if err := DeleteRule(r.Context(), h.queries, projectID, ruleID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete route rule")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
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
	framework, err := h.getFramework(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load route settings")
		return
	}
	if framework != "codeigniter" {
		writeError(w, http.StatusBadRequest, "route import currently supports only CodeIgniter projects")
		return
	}

	var req struct {
		Source    string `json:"source"`
		Framework string `json:"framework"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if selected := normalizeFramework(req.Framework); selected != "" && selected != "generic" {
		framework = selected
	}
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "source_code"
	}

	var rules []ImportRule
	switch source {
	case "source_code":
		cfg := sourcecode.ConfigFromProject(project)
		provider := sourcecode.NewImportProvider(cfg)
		if provider == nil {
			writeError(w, http.StatusBadRequest, "source code is not available from a connected repository or local checkout")
			return
		}
		if err := provider.TestConnection(r.Context()); err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		ref := cfg.DefaultBranch
		if ref == "" {
			ref = "main"
		}
		rules, err = ImportCodeIgniterRules(r.Context(), provider, ref)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
	case "framework_convention":
		rules = []ImportRule{
			{
				Method:        "*",
				MatchPattern:  "(:any)/(:any)",
				CanonicalPath: "/:any/:any",
				Target:        "controller::method",
				Source:        "framework_convention",
				Confidence:    0.9,
				Enabled:       true,
				Framework:     "codeigniter",
				Notes:         "Generic CodeIgniter controller/method fallback",
			},
			{
				Method:        "*",
				MatchPattern:  "(:any)/(:any)/(:any)",
				CanonicalPath: "/:any/:any/:any",
				Target:        "controller::method",
				Source:        "framework_convention",
				Confidence:    0.8,
				Enabled:       true,
				Framework:     "codeigniter",
				Notes:         "Generic CodeIgniter route with one parameter",
			},
			{
				Method:        "*",
				MatchPattern:  "(:any)/(:any)/(:any)/(:any)",
				CanonicalPath: "/:any/:any/:any/:any",
				Target:        "controller::method",
				Source:        "framework_convention",
				Confidence:    0.7,
				Enabled:       true,
				Framework:     "codeigniter",
				Notes:         "Generic CodeIgniter route with two parameters",
			},
			{
				Method:        "*",
				MatchPattern:  "(:any)/(:any)/(:any)/(:any)/(:any)",
				CanonicalPath: "/:any/:any/:any/:any/:any",
				Target:        "controller::method",
				Source:        "framework_convention",
				Confidence:    0.6,
				Enabled:       true,
				Framework:     "codeigniter",
				Notes:         "Generic CodeIgniter route with three parameters",
			},
		}
	default:
		writeError(w, http.StatusBadRequest, "unsupported import source")
		return
	}
	if err := ReplaceImportedRules(r.Context(), h.queries, projectID, framework, source, rules); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save imported route rules")
		return
	}

	saved, err := ListRules(r.Context(), h.queries, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list route rules")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"source":   source,
		"imported": len(rules),
		"rules":    saved,
	})
}

func (h *Handler) getFramework(ctx context.Context, projectID uuid.UUID) (string, error) {
	var framework string
	err := h.queries.RawDB().QueryRowContext(ctx, `
		SELECT framework
		FROM project_route_settings
		WHERE project_id = $1
	`, projectID).Scan(&framework)
	if err == sql.ErrNoRows {
		return "generic", nil
	}
	if err != nil {
		return "", err
	}
	if framework == "" {
		return "generic", nil
	}
	return framework, nil
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
