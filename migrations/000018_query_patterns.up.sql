-- Phase C: store aggregated SQL query patterns for N+1 detection.
-- Populated from gosnag_query_summary injected by gosnag-agent.
CREATE TABLE query_patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    transaction TEXT NOT NULL DEFAULT '',
    query_hash TEXT NOT NULL,
    normalized_query TEXT NOT NULL,
    table_name TEXT NOT NULL DEFAULT '',
    event_count INT NOT NULL DEFAULT 0,
    distinct_events INT NOT NULL DEFAULT 0,
    total_exec_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    first_seen TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, transaction, query_hash)
);

CREATE INDEX idx_query_patterns_project ON query_patterns(project_id);
CREATE INDEX idx_query_patterns_last_seen ON query_patterns(last_seen);
