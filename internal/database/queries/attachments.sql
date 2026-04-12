-- name: CreateAttachment :one
INSERT INTO ticket_attachments (ticket_id, filename, url, content_type, size_bytes, uploaded_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListAttachmentsByTicket :many
SELECT a.*, u.name AS uploader_name, u.email AS uploader_email
FROM ticket_attachments a
JOIN users u ON u.id = a.uploaded_by
WHERE a.ticket_id = $1
ORDER BY a.created_at;

-- name: GetAttachment :one
SELECT * FROM ticket_attachments WHERE id = $1 AND ticket_id = $2;

-- name: DeleteAttachment :exec
DELETE FROM ticket_attachments WHERE id = $1 AND ticket_id = $2;
