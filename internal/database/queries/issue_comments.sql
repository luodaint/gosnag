-- name: ListIssueComments :many
SELECT c.*, u.name AS user_name, u.email AS user_email, u.avatar_url AS user_avatar
FROM issue_comments c
JOIN users u ON u.id = c.user_id
WHERE c.issue_id = $1
ORDER BY c.created_at ASC;

-- name: CreateIssueComment :one
INSERT INTO issue_comments (issue_id, user_id, body)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateIssueComment :one
UPDATE issue_comments SET body = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteIssueComment :exec
DELETE FROM issue_comments WHERE id = $1;

-- name: GetIssueComment :one
SELECT * FROM issue_comments WHERE id = $1;

-- name: CountIssueComments :one
SELECT count(*) FROM issue_comments WHERE issue_id = $1;
