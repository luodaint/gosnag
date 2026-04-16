ALTER TABLE projects
    ADD COLUMN info_grouping_mode TEXT NOT NULL DEFAULT 'normal',
    ADD COLUMN max_info_issues INT NOT NULL DEFAULT 0;
