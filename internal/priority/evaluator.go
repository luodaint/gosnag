package priority

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/darkspock/gosnag/internal/database/db"
	"github.com/google/uuid"
)

// velocityCache caches event counts per issue to avoid repeated queries during bursts.
type velocityCache struct {
	mu    sync.RWMutex
	items map[string]cachedCount
}

type cachedCount struct {
	count     int32
	expiresAt time.Time
}

var cache = &velocityCache{items: make(map[string]cachedCount)}

func (c *velocityCache) get(key string) (int32, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, ok := c.items[key]
	if !ok || time.Now().After(item.expiresAt) {
		return 0, false
	}
	return item.count, true
}

func (c *velocityCache) set(key string, count int32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = cachedCount{count: count, expiresAt: time.Now().Add(60 * time.Second)}
	// Lazy cleanup: remove expired entries when cache grows
	if len(c.items) > 1000 {
		now := time.Now()
		for k, v := range c.items {
			if now.After(v.expiresAt) {
				delete(c.items, k)
			}
		}
	}
}

// Evaluate calculates the priority score for an issue based on project rules.
// Should be called asynchronously after event ingestion.
func Evaluate(ctx context.Context, queries *db.Queries, projectID uuid.UUID, issue db.Issue) {
	rules, err := queries.ListEnabledPriorityRules(ctx, projectID)
	if err != nil || len(rules) == 0 {
		return
	}

	// Lazy-load expensive data only if needed
	var velocity1h, velocity24h *int32
	var userCount *int32

	score := int32(50) // base score

	for _, rule := range rules {
		matched := false

		switch rule.RuleType {
		case "velocity_1h":
			if velocity1h == nil {
				v := getVelocity1h(ctx, queries, issue.ID)
				velocity1h = &v
			}
			matched = compareInt(*velocity1h, rule.Operator, rule.Threshold)

		case "velocity_24h":
			if velocity24h == nil {
				v := getVelocity24h(ctx, queries, issue.ID)
				velocity24h = &v
			}
			matched = compareInt(*velocity24h, rule.Operator, rule.Threshold)

		case "total_events":
			matched = compareInt(issue.EventCount, rule.Operator, rule.Threshold)

		case "user_count":
			if userCount == nil {
				uc := getUserCount(ctx, queries, issue.ID)
				userCount = &uc
			}
			matched = compareInt(*userCount, rule.Operator, rule.Threshold)

		case "title_contains":
			if rule.Pattern != "" {
				re, err := regexp.Compile(rule.Pattern)
				if err != nil {
					matched = strings.Contains(strings.ToLower(issue.Title), strings.ToLower(rule.Pattern))
				} else {
					matched = re.MatchString(issue.Title)
				}
			}

		case "title_not_contains":
			if rule.Pattern != "" {
				re, err := regexp.Compile(rule.Pattern)
				if err != nil {
					matched = !strings.Contains(strings.ToLower(issue.Title), strings.ToLower(rule.Pattern))
				} else {
					matched = !re.MatchString(issue.Title)
				}
			}

		case "level_is":
			matched = strings.EqualFold(issue.Level, rule.Pattern)

		case "platform_is":
			matched = strings.EqualFold(issue.Platform, rule.Pattern)
		}

		if matched {
			score += rule.Points
		}
	}

	// Clamp 0–100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	// Only update if changed
	if score != issue.Priority {
		if err := queries.UpdateIssuePriority(ctx, db.UpdateIssuePriorityParams{
			ID:       issue.ID,
			Priority: score,
		}); err != nil {
			slog.Error("failed to update issue priority", "error", err, "issue_id", issue.ID)
		}
	}
}

// EvaluateAll recalculates priority for all issues in a project.
func EvaluateAll(ctx context.Context, queries *db.Queries, projectID uuid.UUID) (int, error) {
	issueIDs, err := queries.ListIssueIDsByProject(ctx, projectID)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, id := range issueIDs {
		issue, err := queries.GetIssue(ctx, id)
		if err != nil {
			continue
		}
		Evaluate(ctx, queries, projectID, issue)
		count++
	}
	return count, nil
}

func getVelocity1h(ctx context.Context, queries *db.Queries, issueID uuid.UUID) int32 {
	key := "v1h:" + issueID.String()
	if v, ok := cache.get(key); ok {
		return v
	}
	count, err := queries.GetIssueVelocity1h(ctx, issueID)
	if err != nil {
		return 0
	}
	cache.set(key, count)
	return count
}

func getVelocity24h(ctx context.Context, queries *db.Queries, issueID uuid.UUID) int32 {
	key := "v24h:" + issueID.String()
	if v, ok := cache.get(key); ok {
		return v
	}
	count, err := queries.GetIssueVelocity24h(ctx, issueID)
	if err != nil {
		return 0
	}
	cache.set(key, count)
	return count
}

func getUserCount(ctx context.Context, queries *db.Queries, issueID uuid.UUID) int32 {
	count, err := queries.GetIssueUserCount(ctx, issueID)
	if err != nil {
		return 0
	}
	return int32(count)
}

func compareInt(value int32, operator string, threshold int32) bool {
	switch operator {
	case "gte", ">=", "":
		return value >= threshold
	case "lte", "<=":
		return value <= threshold
	case "eq", "==":
		return value == threshold
	case "gt", ">":
		return value > threshold
	case "lt", "<":
		return value < threshold
	default:
		return value >= threshold
	}
}
