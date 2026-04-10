package github

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/darkspock/gosnag/internal/conditions"
	"github.com/darkspock/gosnag/internal/database/db"
)

// MatchesRule checks whether an issue satisfies all conditions of a GitHub rule.
func MatchesRule(rule db.GithubRule, issue db.Issue, userCount int32, evalCtx *conditions.EvalContext) bool {
	// New engine: if conditions JSONB is set, use it
	if rule.Conditions.Valid {
		var group conditions.Group
		if err := json.Unmarshal(rule.Conditions.RawMessage, &group); err == nil {
			return conditions.Evaluate(group, evalCtx)
		}
	}

	// Legacy path: flat columns
	if rule.LevelFilter != "" {
		levels := strings.Split(rule.LevelFilter, ",")
		matched := false
		for _, l := range levels {
			if strings.TrimSpace(l) == issue.Level {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if rule.MinEvents > 0 && issue.EventCount < rule.MinEvents {
		return false
	}

	if rule.MinUsers > 0 && userCount < rule.MinUsers {
		return false
	}

	if rule.TitlePattern != "" {
		re, err := regexp.Compile(rule.TitlePattern)
		if err != nil {
			if !strings.Contains(strings.ToLower(issue.Title), strings.ToLower(rule.TitlePattern)) {
				return false
			}
		} else if !re.MatchString(issue.Title) {
			return false
		}
	}

	return true
}
