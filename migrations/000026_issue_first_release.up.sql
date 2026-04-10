ALTER TABLE issues ADD COLUMN first_release TEXT NOT NULL DEFAULT '';

-- Covers GetIssueCountsByStatus, CountIssuesToday, CountIssuesAssigned, CountIssuesAssignedToUser
-- which all filter by (project_id, level) and were doing a filter scan on idx_issues_project_status
CREATE INDEX idx_issues_project_level ON issues(project_id, level);
