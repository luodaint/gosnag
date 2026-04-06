ALTER TABLE alert_configs DROP COLUMN IF EXISTS conditions;
ALTER TABLE priority_rules DROP COLUMN IF EXISTS conditions;
ALTER TABLE tag_rules DROP COLUMN IF EXISTS conditions;
ALTER TABLE jira_rules DROP COLUMN IF EXISTS conditions;
