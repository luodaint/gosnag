ALTER TABLE projects
    DROP COLUMN IF EXISTS max_info_issues,
    DROP COLUMN IF EXISTS info_grouping_mode;
