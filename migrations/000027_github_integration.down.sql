DROP TABLE IF EXISTS github_rules;
ALTER TABLE issues DROP COLUMN IF EXISTS github_issue_number;
ALTER TABLE issues DROP COLUMN IF EXISTS github_issue_url;
ALTER TABLE projects DROP COLUMN IF EXISTS github_token;
ALTER TABLE projects DROP COLUMN IF EXISTS github_owner;
ALTER TABLE projects DROP COLUMN IF EXISTS github_repo;
ALTER TABLE projects DROP COLUMN IF EXISTS github_labels;
