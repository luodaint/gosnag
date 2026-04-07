-- name: UpsertQueryPattern :exec
INSERT INTO query_patterns (project_id, transaction, query_hash, normalized_query, table_name, event_count, distinct_events, total_exec_ms, first_seen, last_seen)
VALUES ($1, $2, $3, $4, $5, $6, 1, $7, now(), now())
ON CONFLICT (project_id, transaction, query_hash) DO UPDATE
SET event_count = query_patterns.event_count + EXCLUDED.event_count,
    distinct_events = query_patterns.distinct_events + 1,
    total_exec_ms = query_patterns.total_exec_ms + EXCLUDED.total_exec_ms,
    last_seen = now();

-- name: ListQueryPatterns :many
SELECT * FROM query_patterns
WHERE project_id = $1
ORDER BY event_count DESC
LIMIT $2;

-- name: ListN1Candidates :many
SELECT *,
       (event_count::float / GREATEST(distinct_events, 1)) AS avg_per_event
FROM query_patterns
WHERE project_id = $1
  AND distinct_events >= $2
  AND (event_count::float / GREATEST(distinct_events, 1)) >= $3
ORDER BY (event_count::float / GREATEST(distinct_events, 1)) * distinct_events DESC;

-- name: CleanupOldQueryPatterns :exec
DELETE FROM query_patterns WHERE last_seen < now() - interval '30 days';
