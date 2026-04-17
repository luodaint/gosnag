ALTER TABLE projects
    ADD COLUMN warning_as_error BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN jira_base_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN jira_email TEXT NOT NULL DEFAULT '',
    ADD COLUMN jira_api_token TEXT NOT NULL DEFAULT '',
    ADD COLUMN jira_project_key TEXT NOT NULL DEFAULT '',
    ADD COLUMN jira_issue_type TEXT NOT NULL DEFAULT 'Bug',
    ADD COLUMN max_events_per_issue INT NOT NULL DEFAULT 1000,
    ADD COLUMN issue_display_mode TEXT NOT NULL DEFAULT 'classic',
    ADD COLUMN github_token TEXT NOT NULL DEFAULT '',
    ADD COLUMN github_owner TEXT NOT NULL DEFAULT '',
    ADD COLUMN github_repo TEXT NOT NULL DEFAULT '',
    ADD COLUMN github_labels TEXT NOT NULL DEFAULT 'bug',
    ADD COLUMN repo_provider TEXT NOT NULL DEFAULT '',
    ADD COLUMN repo_owner TEXT NOT NULL DEFAULT '',
    ADD COLUMN repo_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN repo_default_branch TEXT NOT NULL DEFAULT 'main',
    ADD COLUMN repo_token TEXT NOT NULL DEFAULT '',
    ADD COLUMN repo_path_strip TEXT NOT NULL DEFAULT '',
    ADD COLUMN ai_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN ai_model TEXT NOT NULL DEFAULT '',
    ADD COLUMN ai_merge_suggestions BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN ai_auto_merge BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN ai_anomaly_detection BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN ai_ticket_description BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN ai_root_cause BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN ai_triage BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN stacktrace_rules JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN info_grouping_mode TEXT NOT NULL DEFAULT 'normal',
    ADD COLUMN max_info_issues INT NOT NULL DEFAULT 0,
    ADD COLUMN analysis_db_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN analysis_db_driver TEXT NOT NULL DEFAULT '',
    ADD COLUMN analysis_db_dsn TEXT NOT NULL DEFAULT '',
    ADD COLUMN analysis_db_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN analysis_db_schema TEXT NOT NULL DEFAULT '',
    ADD COLUMN analysis_db_notes TEXT NOT NULL DEFAULT '';

UPDATE projects p
SET
    warning_as_error = s.warning_as_error,
    max_events_per_issue = s.max_events_per_issue,
    issue_display_mode = s.issue_display_mode,
    info_grouping_mode = s.info_grouping_mode,
    max_info_issues = s.max_info_issues
FROM project_issue_settings s
WHERE s.project_id = p.id;

UPDATE projects p
SET
    jira_base_url = s.base_url,
    jira_email = s.email,
    jira_api_token = s.api_token,
    jira_project_key = s.project_key,
    jira_issue_type = s.issue_type
FROM project_jira_settings s
WHERE s.project_id = p.id;

UPDATE projects p
SET
    github_token = s.token,
    github_owner = s.owner,
    github_repo = s.repo,
    github_labels = s.labels
FROM project_github_settings s
WHERE s.project_id = p.id;

UPDATE projects p
SET
    repo_provider = s.provider,
    repo_owner = s.owner,
    repo_name = s.name,
    repo_default_branch = s.default_branch,
    repo_token = s.token,
    repo_path_strip = s.path_strip
FROM project_repo_settings s
WHERE s.project_id = p.id;

UPDATE projects p
SET
    ai_enabled = s.enabled,
    ai_model = s.model,
    ai_merge_suggestions = s.merge_suggestions,
    ai_auto_merge = s.auto_merge,
    ai_anomaly_detection = s.anomaly_detection,
    ai_ticket_description = s.ticket_description,
    ai_root_cause = s.root_cause,
    ai_triage = s.triage
FROM project_ai_settings s
WHERE s.project_id = p.id;

UPDATE projects p
SET
    stacktrace_rules = s.rules
FROM project_stacktrace_settings s
WHERE s.project_id = p.id;

UPDATE projects p
SET
    analysis_db_enabled = s.enabled,
    analysis_db_driver = s.driver,
    analysis_db_dsn = s.dsn,
    analysis_db_name = s.name,
    analysis_db_schema = s.schema,
    analysis_db_notes = s.notes
FROM project_db_analysis_settings s
WHERE s.project_id = p.id;
