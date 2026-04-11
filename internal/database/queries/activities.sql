-- name: InsertActivity :one
INSERT INTO issue_activities (issue_id, ticket_id, user_id, action, old_value, new_value, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListActivitiesByIssue :many
SELECT a.*, u.name AS user_name, u.email AS user_email, u.avatar_url AS user_avatar
FROM issue_activities a
LEFT JOIN users u ON u.id = a.user_id
WHERE a.issue_id = $1
ORDER BY a.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountActivitiesByIssue :one
SELECT count(*) FROM issue_activities WHERE issue_id = $1;
