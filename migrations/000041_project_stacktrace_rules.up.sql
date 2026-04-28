ALTER TABLE projects
    ADD COLUMN stacktrace_rules JSONB NOT NULL DEFAULT '{}'::jsonb;
