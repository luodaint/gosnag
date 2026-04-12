-- Release version → Git commit mapping
CREATE TABLE release_commits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    release_version TEXT NOT NULL,
    commit_sha TEXT NOT NULL,
    commit_url TEXT,
    committed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, release_version)
);

CREATE INDEX idx_release_commits_project ON release_commits(project_id, created_at DESC);

-- Deploy events
CREATE TABLE deploys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    release_version TEXT NOT NULL,
    commit_sha TEXT,
    environment TEXT NOT NULL DEFAULT 'production',
    url TEXT,
    deployed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_deploys_project ON deploys(project_id, deployed_at DESC);
