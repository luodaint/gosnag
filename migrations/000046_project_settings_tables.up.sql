CREATE TABLE project_issue_settings (
    project_id UUID PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    warning_as_error BOOLEAN NOT NULL DEFAULT false,
    max_events_per_issue INT NOT NULL DEFAULT 1000,
    issue_display_mode TEXT NOT NULL DEFAULT 'classic',
    info_grouping_mode TEXT NOT NULL DEFAULT 'normal',
    max_info_issues INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE project_jira_settings (
    project_id UUID PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    base_url TEXT NOT NULL DEFAULT '',
    email TEXT NOT NULL DEFAULT '',
    api_token TEXT NOT NULL DEFAULT '',
    project_key TEXT NOT NULL DEFAULT '',
    issue_type TEXT NOT NULL DEFAULT 'Bug',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE project_github_settings (
    project_id UUID PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    token TEXT NOT NULL DEFAULT '',
    owner TEXT NOT NULL DEFAULT '',
    repo TEXT NOT NULL DEFAULT '',
    labels TEXT NOT NULL DEFAULT 'bug',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE project_repo_settings (
    project_id UUID PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    provider TEXT NOT NULL DEFAULT '',
    owner TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL DEFAULT '',
    default_branch TEXT NOT NULL DEFAULT 'main',
    token TEXT NOT NULL DEFAULT '',
    path_strip TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE project_ai_settings (
    project_id UUID PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT false,
    model TEXT NOT NULL DEFAULT '',
    merge_suggestions BOOLEAN NOT NULL DEFAULT false,
    auto_merge BOOLEAN NOT NULL DEFAULT false,
    anomaly_detection BOOLEAN NOT NULL DEFAULT false,
    ticket_description BOOLEAN NOT NULL DEFAULT true,
    root_cause BOOLEAN NOT NULL DEFAULT false,
    triage BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE project_stacktrace_settings (
    project_id UUID PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    rules JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE project_db_analysis_settings (
    project_id UUID PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT false,
    driver TEXT NOT NULL DEFAULT '',
    dsn TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL DEFAULT '',
    schema TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO project_issue_settings (
    project_id,
    warning_as_error,
    max_events_per_issue,
    issue_display_mode,
    info_grouping_mode,
    max_info_issues
)
SELECT
    id,
    warning_as_error,
    max_events_per_issue,
    issue_display_mode,
    info_grouping_mode,
    max_info_issues
FROM projects
ON CONFLICT (project_id) DO NOTHING;

INSERT INTO project_jira_settings (
    project_id,
    base_url,
    email,
    api_token,
    project_key,
    issue_type
)
SELECT
    id,
    jira_base_url,
    jira_email,
    jira_api_token,
    jira_project_key,
    jira_issue_type
FROM projects
ON CONFLICT (project_id) DO NOTHING;

INSERT INTO project_github_settings (
    project_id,
    token,
    owner,
    repo,
    labels
)
SELECT
    id,
    github_token,
    github_owner,
    github_repo,
    github_labels
FROM projects
ON CONFLICT (project_id) DO NOTHING;

INSERT INTO project_repo_settings (
    project_id,
    provider,
    owner,
    name,
    default_branch,
    token,
    path_strip
)
SELECT
    id,
    repo_provider,
    repo_owner,
    repo_name,
    repo_default_branch,
    repo_token,
    repo_path_strip
FROM projects
ON CONFLICT (project_id) DO NOTHING;

INSERT INTO project_ai_settings (
    project_id,
    enabled,
    model,
    merge_suggestions,
    auto_merge,
    anomaly_detection,
    ticket_description,
    root_cause,
    triage
)
SELECT
    id,
    ai_enabled,
    ai_model,
    ai_merge_suggestions,
    ai_auto_merge,
    ai_anomaly_detection,
    ai_ticket_description,
    ai_root_cause,
    ai_triage
FROM projects
ON CONFLICT (project_id) DO NOTHING;

INSERT INTO project_stacktrace_settings (
    project_id,
    rules
)
SELECT
    id,
    stacktrace_rules
FROM projects
ON CONFLICT (project_id) DO NOTHING;

INSERT INTO project_db_analysis_settings (
    project_id,
    enabled,
    driver,
    dsn,
    name,
    schema,
    notes
)
SELECT
    id,
    analysis_db_enabled,
    analysis_db_driver,
    analysis_db_dsn,
    analysis_db_name,
    analysis_db_schema,
    analysis_db_notes
FROM projects
ON CONFLICT (project_id) DO NOTHING;
