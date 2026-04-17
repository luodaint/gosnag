package sourcecode

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/darkspock/gosnag/internal/database/db"
	projectcfg "github.com/darkspock/gosnag/internal/project"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ResolveRelease resolves a release version string to a commit SHA using the Git provider.
// Results are cached in the release_commits table.
func ResolveRelease(ctx context.Context, queries *db.Queries, projectID uuid.UUID, releaseVersion string) (*db.ReleaseCommit, error) {
	if releaseVersion == "" || releaseVersion == "unknown" {
		return nil, nil
	}

	// Check cache
	if rc, err := queries.GetReleaseCommit(ctx, db.GetReleaseCommitParams{
		ProjectID:      projectID,
		ReleaseVersion: releaseVersion,
	}); err == nil {
		return &rc, nil
	}

	// Resolve via Git provider
	_, settings, err := projectcfg.LoadSettingsByProjectID(ctx, queries, projectID)
	if err != nil {
		return nil, err
	}

	cfg := ConfigFromSettings(settings)
	if !cfg.IsConfigured() {
		return nil, nil
	}

	provider := NewProvider(cfg)
	if provider == nil {
		return nil, nil
	}

	// Try common release tag patterns
	patterns := []string{
		releaseVersion,
		"v" + releaseVersion,
		"release-" + releaseVersion,
		"release/" + releaseVersion,
	}

	for _, ref := range patterns {
		sha, err := provider.ResolveRef(ctx, ref)
		if err == nil && sha != "" {
			commitURL := ""
			if cfg.Provider == "github" {
				commitURL = fmt.Sprintf("https://github.com/%s/%s/commit/%s", cfg.Owner, cfg.Name, sha)
			} else if cfg.Provider == "bitbucket" {
				commitURL = fmt.Sprintf("https://bitbucket.org/%s/%s/commits/%s", cfg.Owner, cfg.Name, sha)
			}

			rc, err := queries.UpsertReleaseCommit(ctx, db.UpsertReleaseCommitParams{
				ProjectID:      projectID,
				ReleaseVersion: releaseVersion,
				CommitSha:      sha,
				CommitUrl:      sql.NullString{String: commitURL, Valid: commitURL != ""},
			})
			if err != nil {
				slog.Error("failed to cache release commit", "error", err)
			}
			return &rc, nil
		}
	}

	return nil, nil
}

// DiffURL returns a URL to the diff between two releases.
func DiffURL(cfg Config, fromSHA, toSHA string) string {
	if cfg.Provider == "github" {
		return fmt.Sprintf("https://github.com/%s/%s/compare/%s...%s", cfg.Owner, cfg.Name, fromSHA[:min(7, len(fromSHA))], toSHA[:min(7, len(toSHA))])
	}
	if cfg.Provider == "bitbucket" {
		return fmt.Sprintf("https://bitbucket.org/%s/%s/branches/compare/%s..%s", cfg.Owner, cfg.Name, toSHA, fromSHA)
	}
	return ""
}

// GetReleaseInfo returns release info for an issue.
func (h *Handler) GetReleaseInfo(w http.ResponseWriter, r *http.Request) {
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

	issue, err := h.queries.GetIssue(r.Context(), issueID)
	if err != nil || issue.ProjectID != projectID {
		writeError(w, http.StatusNotFound, "issue not found")
		return
	}

	_, settings, err := projectcfg.LoadSettingsByProjectID(r.Context(), h.queries, projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	result := map[string]any{
		"first_release": issue.FirstRelease,
	}

	cfg := ConfigFromSettings(settings)

	// Resolve first release to commit
	if issue.FirstRelease != "" && issue.FirstRelease != "unknown" {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		rc, err := ResolveRelease(ctx, h.queries, projectID, issue.FirstRelease)
		if err == nil && rc != nil {
			result["commit_sha"] = rc.CommitSha
			if rc.CommitUrl.Valid {
				result["commit_url"] = rc.CommitUrl.String
			}

			// Get previous release for diff
			prev, err := h.queries.GetPreviousRelease(ctx, db.GetPreviousReleaseParams{
				ProjectID:      projectID,
				ReleaseVersion: issue.FirstRelease,
			})
			if err == nil && cfg.IsConfigured() {
				result["previous_release"] = prev.ReleaseVersion
				result["diff_url"] = DiffURL(cfg, prev.CommitSha, rc.CommitSha)
			}
		}

		// Get deploy info
		deploy, err := h.queries.GetLatestDeployForRelease(r.Context(), db.GetLatestDeployForReleaseParams{
			ProjectID:      projectID,
			ReleaseVersion: issue.FirstRelease,
		})
		if err == nil {
			result["deployed_at"] = deploy.DeployedAt
			result["deploy_environment"] = deploy.Environment
		}
	}

	writeJSON(w, http.StatusOK, result)
}

// Deploy records a deployment event (webhook called by CI/CD).
func (h *Handler) Deploy(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "project_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	var req struct {
		Version     string `json:"version"`
		Commit      string `json:"commit"`
		Environment string `json:"environment"`
		URL         string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Version == "" {
		writeError(w, http.StatusBadRequest, "version is required")
		return
	}

	if req.Environment == "" {
		req.Environment = "production"
	}

	deploy, err := h.queries.CreateDeploy(r.Context(), db.CreateDeployParams{
		ProjectID:      projectID,
		ReleaseVersion: req.Version,
		CommitSha:      sql.NullString{String: req.Commit, Valid: req.Commit != ""},
		Environment:    req.Environment,
		Url:            sql.NullString{String: req.URL, Valid: req.URL != ""},
		DeployedAt:     time.Now(),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record deploy")
		return
	}

	// Also cache the release→commit mapping if commit is provided
	if req.Commit != "" {
		cfg := Config{}
		if _, settings, err := projectcfg.LoadSettingsByProjectID(r.Context(), h.queries, projectID); err == nil {
			cfg = ConfigFromSettings(settings)
		}
		commitURL := ""
		if cfg.Provider == "github" {
			commitURL = fmt.Sprintf("https://github.com/%s/%s/commit/%s", cfg.Owner, cfg.Name, req.Commit)
		} else if cfg.Provider == "bitbucket" {
			commitURL = fmt.Sprintf("https://bitbucket.org/%s/%s/commits/%s", cfg.Owner, cfg.Name, req.Commit)
		}
		h.queries.UpsertReleaseCommit(r.Context(), db.UpsertReleaseCommitParams{
			ProjectID:      projectID,
			ReleaseVersion: req.Version,
			CommitSha:      req.Commit,
			CommitUrl:      sql.NullString{String: commitURL, Valid: commitURL != ""},
		})
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":          deploy.ID,
		"version":     deploy.ReleaseVersion,
		"environment": deploy.Environment,
		"deployed_at": deploy.DeployedAt,
	})
}

// ListDeploys returns recent deploys for a project.
func (h *Handler) ListDeploys(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "project_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	deploys, err := h.queries.ListDeploys(r.Context(), db.ListDeploysParams{
		ProjectID: projectID,
		Limit:     20,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list deploys")
		return
	}

	writeJSON(w, http.StatusOK, deploys)
}
