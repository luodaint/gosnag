ALTER TABLE issues ADD COLUMN priority INT NOT NULL DEFAULT 50;
CREATE INDEX idx_issues_priority ON issues(project_id, priority DESC);

CREATE TABLE priority_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    rule_type TEXT NOT NULL,
    pattern TEXT NOT NULL DEFAULT '',
    operator TEXT NOT NULL DEFAULT '',
    threshold INT NOT NULL DEFAULT 0,
    points INT NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    position INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_priority_rules_project ON priority_rules(project_id);
