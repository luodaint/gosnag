CREATE TABLE project_route_settings (
    project_id UUID PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    framework TEXT NOT NULL DEFAULT 'generic',
    enabled BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO project_route_settings (project_id)
SELECT id FROM projects
ON CONFLICT (project_id) DO NOTHING;

CREATE TABLE route_grouping_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    method TEXT NOT NULL DEFAULT '*',
    match_pattern TEXT NOT NULL,
    canonical_path TEXT NOT NULL,
    target TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT 'manual',
    confidence REAL NOT NULL DEFAULT 1,
    enabled BOOLEAN NOT NULL DEFAULT true,
    framework TEXT NOT NULL DEFAULT '',
    source_file TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, method, match_pattern, target)
);

CREATE INDEX idx_route_grouping_rules_project
    ON route_grouping_rules(project_id, enabled, source);
