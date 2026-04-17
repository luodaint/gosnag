package github

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/darkspock/gosnag/internal/conditions"
	"github.com/darkspock/gosnag/internal/database/db"
	projectcfg "github.com/darkspock/gosnag/internal/project"
	"github.com/google/uuid"
)

// CheckAndCreateIssue evaluates GitHub rules for an issue and creates a GitHub issue if matched.
func CheckAndCreateIssue(ctx context.Context, queries *db.Queries, baseURL string, projectID uuid.UUID, issue db.Issue) {
	// Skip if already has a GitHub issue
	if issue.GithubIssueNumber.Valid {
		return
	}

	_, settings, err := projectcfg.LoadSettingsByProjectID(ctx, queries, projectID)
	if err != nil {
		return
	}

	cfg := ConfigFromSettings(settings)
	if !cfg.IsConfigured() {
		return
	}

	rules, err := queries.ListEnabledGithubRules(ctx, projectID)
	if err != nil || len(rules) == 0 {
		return
	}

	userCount := int32(0)
	if uc, err := queries.GetIssueUserCount(ctx, issue.ID); err == nil {
		userCount = int32(uc)
	}
	eventData := conditions.LoadLatestEventData(ctx, queries, issue.ID)

	evalCtx := conditions.NewEvalContext(conditions.IssueData{
		ID:          issue.ID,
		Title:       issue.Title,
		Level:       issue.Level,
		Platform:    issue.Platform,
		EventCount:  issue.EventCount,
		HasAppFrame: conditions.HasAppFrame(eventData, settings.StacktraceRules),
	}, string(eventData), &githubLoader{queries: queries, ctx: ctx})

	for _, rule := range rules {
		if MatchesRule(rule, issue, userCount, evalCtx) {
			// Re-check right before creating (race condition guard)
			fresh, err := queries.GetIssue(ctx, issue.ID)
			if err != nil || fresh.GithubIssueNumber.Valid {
				return
			}

			title := "[GoSnag] " + truncate(issue.Title, 200)
			body := BuildDescription(issue, baseURL, projectID.String(), "")

			result, err := CreateIssue(cfg, title, body)
			if err != nil {
				slog.Error("failed to auto-create GitHub issue", "error", err, "issue_id", issue.ID, "rule", rule.Name)
				return
			}

			res, err := queries.UpdateIssueGithubTicket(ctx, db.UpdateIssueGithubTicketParams{
				ID:                issue.ID,
				GithubIssueNumber: sql.NullInt32{Int32: int32(result.Number), Valid: true},
				GithubIssueUrl:    sql.NullString{String: result.URL, Valid: true},
			})
			if err != nil {
				slog.Error("failed to save GitHub issue reference", "error", err, "number", result.Number, "issue_id", issue.ID)
				return
			}
			if rows, _ := res.RowsAffected(); rows == 0 {
				slog.Warn("GitHub issue created but issue was linked concurrently", "number", result.Number, "issue_id", issue.ID)
				return
			}

			slog.Info("auto-created GitHub issue", "number", result.Number, "issue_id", issue.ID, "rule", rule.Name)
			return
		}
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

type githubLoader struct {
	queries *db.Queries
	ctx     context.Context
}

func (l *githubLoader) GetVelocity1h(issueID uuid.UUID) (int32, error) {
	return l.queries.GetIssueVelocity1h(l.ctx, issueID)
}

func (l *githubLoader) GetVelocity24h(issueID uuid.UUID) (int32, error) {
	return l.queries.GetIssueVelocity24h(l.ctx, issueID)
}

func (l *githubLoader) GetUserCount(issueID uuid.UUID) (int32, error) {
	count, err := l.queries.GetIssueUserCount(l.ctx, issueID)
	return int32(count), err
}
