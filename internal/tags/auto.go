package tags

import (
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"strings"

	"github.com/darkspock/gosnag/internal/database/db"
	"github.com/google/uuid"
)

// AutoTag evaluates tag rules for an issue and applies matching tags.
// Searches both the issue title and the full event data (stacktrace, message, etc.).
// Should be called asynchronously after event ingestion.
func AutoTag(ctx context.Context, queries *db.Queries, projectID uuid.UUID, issue db.Issue, eventData json.RawMessage) {
	rules, err := queries.ListEnabledTagRules(ctx, projectID)
	if err != nil || len(rules) == 0 {
		return
	}

	searchText := issue.Title + "\n" + string(eventData)

	for _, rule := range rules {
		if matchesPattern(rule.Pattern, searchText) {
			err := queries.AddIssueTag(ctx, db.AddIssueTagParams{
				IssueID: issue.ID,
				Key:     rule.TagKey,
				Value:   rule.TagValue,
			})
			if err != nil {
				slog.Error("failed to auto-tag issue", "error", err, "issue_id", issue.ID, "tag", rule.TagKey+":"+rule.TagValue)
			}
		}
	}
}

func matchesPattern(pattern, text string) bool {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return strings.Contains(strings.ToLower(text), strings.ToLower(pattern))
	}
	return re.MatchString(text)
}
