-- Partial index: only error/fatal events, covering project dashboard queries
-- (trend, weekly errors, latest release). Excludes info/warning/debug (~90% of rows).
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_events_errors
ON events (project_id, timestamp DESC)
WHERE level IN ('error', 'fatal');
