CREATE TABLE issue_aliases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    fingerprint TEXT NOT NULL,
    primary_issue_id UUID NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_issue_aliases_project_fp ON issue_aliases(project_id, fingerprint);
CREATE INDEX idx_issue_aliases_primary ON issue_aliases(primary_issue_id);
