-- name: ListProjectGroups :many
SELECT * FROM project_groups ORDER BY position, name;

-- name: CreateProjectGroup :one
INSERT INTO project_groups (name, position)
VALUES ($1, COALESCE((SELECT max(position) + 1 FROM project_groups), 0))
RETURNING *;

-- name: UpdateProjectGroup :one
UPDATE project_groups SET name = $2 WHERE id = $1 RETURNING *;

-- name: DeleteProjectGroup :exec
DELETE FROM project_groups WHERE id = $1;

-- name: SetProjectGroup :exec
UPDATE projects SET group_id = $2, updated_at = now() WHERE id = $1;
