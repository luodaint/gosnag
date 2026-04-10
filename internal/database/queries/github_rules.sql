-- name: ListGithubRules :many
SELECT * FROM github_rules WHERE project_id = $1 ORDER BY created_at;

-- name: GetGithubRule :one
SELECT * FROM github_rules WHERE id = $1;

-- name: CreateGithubRule :one
INSERT INTO github_rules (project_id, name, enabled, level_filter, min_events, min_users, title_pattern, conditions)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateGithubRule :one
UPDATE github_rules
SET name = $3, enabled = $4, level_filter = $5, min_events = $6, min_users = $7, title_pattern = $8, conditions = $9, updated_at = now()
WHERE id = $1 AND project_id = $2
RETURNING *;

-- name: DeleteGithubRule :exec
DELETE FROM github_rules WHERE id = $1 AND project_id = $2;

-- name: ListEnabledGithubRules :many
SELECT * FROM github_rules WHERE project_id = $1 AND enabled = true;

-- name: UpdateIssueGithubTicket :execresult
UPDATE issues SET github_issue_number = $2, github_issue_url = $3, updated_at = now()
WHERE id = $1 AND github_issue_number IS NULL;
