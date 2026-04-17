package routegroup

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/darkspock/gosnag/internal/database/db"
	"github.com/google/uuid"
)

type Rule struct {
	ID            uuid.UUID `json:"id"`
	ProjectID     uuid.UUID `json:"project_id"`
	Method        string    `json:"method"`
	MatchPattern  string    `json:"match_pattern"`
	CanonicalPath string    `json:"canonical_path"`
	Target        string    `json:"target"`
	Source        string    `json:"source"`
	Confidence    float64   `json:"confidence"`
	Enabled       bool      `json:"enabled"`
	Framework     string    `json:"framework"`
	SourceFile    string    `json:"source_file"`
	Notes         string    `json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ImportRule struct {
	Method        string
	MatchPattern  string
	CanonicalPath string
	Target        string
	Source        string
	Confidence    float64
	Enabled       bool
	Framework     string
	SourceFile    string
	Notes         string
}

type UpsertRuleInput struct {
	Method        string
	MatchPattern  string
	CanonicalPath string
	Target        string
	Source        string
	Confidence    float64
	Enabled       bool
	Framework     string
	SourceFile    string
	Notes         string
}

func normalizeMethod(method string) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		return "*"
	}
	return method
}

func ListRules(ctx context.Context, queries *db.Queries, projectID uuid.UUID) ([]Rule, error) {
	rows, err := queries.RawDB().QueryContext(ctx, `
		SELECT id, project_id, method, match_pattern, canonical_path, target, source, confidence, enabled, framework, source_file, notes, created_at, updated_at
		FROM route_grouping_rules
		WHERE project_id = $1
		ORDER BY enabled DESC, confidence DESC, method, canonical_path, created_at
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Rule, 0)
	for rows.Next() {
		var r Rule
		if err := rows.Scan(
			&r.ID,
			&r.ProjectID,
			&r.Method,
			&r.MatchPattern,
			&r.CanonicalPath,
			&r.Target,
			&r.Source,
			&r.Confidence,
			&r.Enabled,
			&r.Framework,
			&r.SourceFile,
			&r.Notes,
			&r.CreatedAt,
			&r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func ListEnabledRules(ctx context.Context, queries *db.Queries, projectID uuid.UUID) ([]Rule, error) {
	rows, err := queries.RawDB().QueryContext(ctx, `
		SELECT id, project_id, method, match_pattern, canonical_path, target, source, confidence, enabled, framework, source_file, notes, created_at, updated_at
		FROM route_grouping_rules
		WHERE project_id = $1 AND enabled = true
		ORDER BY confidence DESC, method, canonical_path, created_at
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Rule, 0)
	for rows.Next() {
		var r Rule
		if err := rows.Scan(
			&r.ID,
			&r.ProjectID,
			&r.Method,
			&r.MatchPattern,
			&r.CanonicalPath,
			&r.Target,
			&r.Source,
			&r.Confidence,
			&r.Enabled,
			&r.Framework,
			&r.SourceFile,
			&r.Notes,
			&r.CreatedAt,
			&r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func UpdateRule(ctx context.Context, queries *db.Queries, projectID, ruleID uuid.UUID, enabled bool) (Rule, error) {
	row := queries.RawDB().QueryRowContext(ctx, `
		UPDATE route_grouping_rules
		SET enabled = $3, updated_at = now()
		WHERE id = $1 AND project_id = $2
		RETURNING id, project_id, method, match_pattern, canonical_path, target, source, confidence, enabled, framework, source_file, notes, created_at, updated_at
	`, ruleID, projectID, enabled)

	var r Rule
	err := row.Scan(
		&r.ID,
		&r.ProjectID,
		&r.Method,
		&r.MatchPattern,
		&r.CanonicalPath,
		&r.Target,
		&r.Source,
		&r.Confidence,
		&r.Enabled,
		&r.Framework,
		&r.SourceFile,
		&r.Notes,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	return r, err
}

func CreateRule(ctx context.Context, queries *db.Queries, projectID uuid.UUID, input UpsertRuleInput) (Rule, error) {
	row := queries.RawDB().QueryRowContext(ctx, `
		INSERT INTO route_grouping_rules (
			project_id, method, match_pattern, canonical_path, target, source, confidence, enabled, framework, source_file, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, project_id, method, match_pattern, canonical_path, target, source, confidence, enabled, framework, source_file, notes, created_at, updated_at
	`,
		projectID,
		normalizeMethod(input.Method),
		strings.Trim(strings.TrimSpace(input.MatchPattern), "/"),
		normalizeCanonicalPath(input.CanonicalPath),
		strings.TrimSpace(input.Target),
		normalizeSource(input.Source),
		normalizeConfidence(input.Confidence),
		input.Enabled,
		normalizeFramework(input.Framework),
		strings.TrimSpace(input.SourceFile),
		strings.TrimSpace(input.Notes),
	)

	var r Rule
	err := row.Scan(
		&r.ID,
		&r.ProjectID,
		&r.Method,
		&r.MatchPattern,
		&r.CanonicalPath,
		&r.Target,
		&r.Source,
		&r.Confidence,
		&r.Enabled,
		&r.Framework,
		&r.SourceFile,
		&r.Notes,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	return r, err
}

func UpdateRuleFields(ctx context.Context, queries *db.Queries, projectID, ruleID uuid.UUID, input UpsertRuleInput) (Rule, error) {
	row := queries.RawDB().QueryRowContext(ctx, `
		UPDATE route_grouping_rules
		SET
			method = $3,
			match_pattern = $4,
			canonical_path = $5,
			target = $6,
			source = $7,
			confidence = $8,
			enabled = $9,
			framework = $10,
			source_file = $11,
			notes = $12,
			updated_at = now()
		WHERE id = $1 AND project_id = $2
		RETURNING id, project_id, method, match_pattern, canonical_path, target, source, confidence, enabled, framework, source_file, notes, created_at, updated_at
	`,
		ruleID,
		projectID,
		normalizeMethod(input.Method),
		strings.Trim(strings.TrimSpace(input.MatchPattern), "/"),
		normalizeCanonicalPath(input.CanonicalPath),
		strings.TrimSpace(input.Target),
		normalizeSource(input.Source),
		normalizeConfidence(input.Confidence),
		input.Enabled,
		normalizeFramework(input.Framework),
		strings.TrimSpace(input.SourceFile),
		strings.TrimSpace(input.Notes),
	)

	var r Rule
	err := row.Scan(
		&r.ID,
		&r.ProjectID,
		&r.Method,
		&r.MatchPattern,
		&r.CanonicalPath,
		&r.Target,
		&r.Source,
		&r.Confidence,
		&r.Enabled,
		&r.Framework,
		&r.SourceFile,
		&r.Notes,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	return r, err
}

func DeleteRule(ctx context.Context, queries *db.Queries, projectID, ruleID uuid.UUID) error {
	_, err := queries.RawDB().ExecContext(ctx, `
		DELETE FROM route_grouping_rules
		WHERE id = $1 AND project_id = $2
	`, ruleID, projectID)
	return err
}

func ReplaceImportedRules(ctx context.Context, queries *db.Queries, projectID uuid.UUID, framework, source string, rules []ImportRule) error {
	raw := queries.RawDB()
	sqlDB, ok := raw.(*sql.DB)
	if !ok {
		return fmt.Errorf("route grouping import requires *sql.DB handle")
	}

	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM route_grouping_rules
		WHERE project_id = $1 AND framework = $2 AND source = $3
	`, projectID, framework, source); err != nil {
		return err
	}

	for _, rule := range rules {
		if strings.TrimSpace(rule.MatchPattern) == "" || strings.TrimSpace(rule.CanonicalPath) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO route_grouping_rules (
				project_id, method, match_pattern, canonical_path, target, source,
				confidence, enabled, framework, source_file, notes
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`,
			projectID,
			normalizeMethod(rule.Method),
			rule.MatchPattern,
			rule.CanonicalPath,
			rule.Target,
			rule.Source,
			rule.Confidence,
			rule.Enabled,
			rule.Framework,
			rule.SourceFile,
			rule.Notes,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func MatchRule(method, requestPath string, rule Rule) bool {
	if normalizeMethod(rule.Method) != "*" && normalizeMethod(rule.Method) != normalizeMethod(method) {
		return false
	}
	pattern := routePatternToRegex(rule.MatchPattern)
	if pattern == "" {
		return false
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(strings.TrimPrefix(requestPath, "/"))
}

func FindCanonicalRoute(ctx context.Context, queries *db.Queries, projectID uuid.UUID, method, requestPath string) (Rule, bool, error) {
	enabled, err := isEnabled(ctx, queries, projectID)
	if err != nil {
		return Rule{}, false, err
	}
	if !enabled {
		return Rule{}, false, nil
	}

	rules, err := ListEnabledRules(ctx, queries, projectID)
	if err != nil {
		return Rule{}, false, err
	}
	sort.SliceStable(rules, func(i, j int) bool {
		return compareRules(rules[i], rules[j])
	})
	for _, rule := range rules {
		if MatchRule(method, requestPath, rule) {
			return rule, true, nil
		}
	}

	framework, err := projectFramework(ctx, queries, projectID)
	if err != nil {
		return Rule{}, false, err
	}
	if framework == "codeigniter" {
		if rule, ok := codeIgniterConventionRule(method, requestPath); ok {
			return rule, true, nil
		}
	}
	return Rule{}, false, nil
}

func isEnabled(ctx context.Context, queries *db.Queries, projectID uuid.UUID) (bool, error) {
	var enabled bool
	err := queries.RawDB().QueryRowContext(ctx, `
		SELECT enabled
		FROM project_route_settings
		WHERE project_id = $1
	`, projectID).Scan(&enabled)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return enabled, nil
}

func projectFramework(ctx context.Context, queries *db.Queries, projectID uuid.UUID) (string, error) {
	var framework string
	err := queries.RawDB().QueryRowContext(ctx, `
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
	return normalizeFramework(framework), nil
}

func compareRules(a, b Rule) bool {
	if sourceRank(a.Source) != sourceRank(b.Source) {
		return sourceRank(a.Source) > sourceRank(b.Source)
	}
	if a.Confidence != b.Confidence {
		return a.Confidence > b.Confidence
	}
	if ruleSpecificity(a) != ruleSpecificity(b) {
		return ruleSpecificity(a) > ruleSpecificity(b)
	}
	return a.CreatedAt.Before(b.CreatedAt)
}

func sourceRank(source string) int {
	switch normalizeSource(source) {
	case "manual":
		return 120
	case "source_code":
		return 110
	case "framework_convention":
		return 90
	case "observed_issue":
		return 70
	case "ai_suggestion":
		return 50
	default:
		return 10
	}
}

func ruleSpecificity(rule Rule) int {
	score := 0
	for _, segment := range strings.Split(strings.Trim(rule.MatchPattern, "/"), "/") {
		switch segment {
		case "", "(:any)":
			score += 1
		case "(:num)":
			score += 2
		default:
			if strings.HasPrefix(segment, "(") && strings.HasSuffix(segment, ")") {
				score += 1
			} else {
				score += 5
			}
		}
	}
	if normalizeMethod(rule.Method) != "*" {
		score += 2
	}
	return score
}

func normalizeCanonicalPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func normalizeSource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "source_code", "framework_convention", "observed_issue", "ai_suggestion", "manual":
		return strings.ToLower(strings.TrimSpace(source))
	default:
		return "manual"
	}
}

func normalizeFramework(framework string) string {
	switch strings.ToLower(strings.TrimSpace(framework)) {
	case "codeigniter":
		return "codeigniter"
	default:
		return "generic"
	}
}

func normalizeConfidence(confidence float64) float64 {
	if confidence <= 0 {
		return 0.5
	}
	if confidence > 1 {
		return 1
	}
	return confidence
}

func routePatternToRegex(pattern string) string {
	pattern = strings.Trim(strings.TrimSpace(pattern), "/")
	if pattern == "" {
		return "^$"
	}

	segments := strings.Split(pattern, "/")
	parts := make([]string, 0, len(segments))
	for _, segment := range segments {
		switch {
		case segment == "(:any)":
			parts = append(parts, `[^/]+`)
		case segment == "(:num)":
			parts = append(parts, `\d+`)
		case strings.HasPrefix(segment, "(") && strings.HasSuffix(segment, ")"):
			parts = append(parts, `[^/]+`)
		default:
			parts = append(parts, regexp.QuoteMeta(segment))
		}
	}

	return "^" + strings.Join(parts, `/`) + "$"
}
