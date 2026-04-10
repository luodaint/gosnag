-- GitHub connection config per project
ALTER TABLE projects
    ADD COLUMN github_token TEXT NOT NULL DEFAULT '',
    ADD COLUMN github_owner TEXT NOT NULL DEFAULT '',
    ADD COLUMN github_repo TEXT NOT NULL DEFAULT '',
    ADD COLUMN github_labels TEXT NOT NULL DEFAULT 'bug';

-- Track GitHub issue on issues
ALTER TABLE issues
    ADD COLUMN github_issue_number INT,
    ADD COLUMN github_issue_url TEXT;

-- Auto-creation rules
CREATE TABLE github_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    level_filter TEXT NOT NULL DEFAULT '',
    min_events INT NOT NULL DEFAULT 0,
    min_users INT NOT NULL DEFAULT 0,
    title_pattern TEXT NOT NULL DEFAULT '',
    conditions JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_github_rules_project ON github_rules(project_id);
