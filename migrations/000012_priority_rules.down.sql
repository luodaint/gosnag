DROP TABLE IF EXISTS priority_rules;
DROP INDEX IF EXISTS idx_issues_priority;
ALTER TABLE issues DROP COLUMN IF EXISTS priority;
