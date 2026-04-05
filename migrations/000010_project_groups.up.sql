CREATE TABLE project_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    position INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE projects ADD COLUMN group_id UUID REFERENCES project_groups(id) ON DELETE SET NULL;
CREATE INDEX idx_projects_group ON projects(group_id);
