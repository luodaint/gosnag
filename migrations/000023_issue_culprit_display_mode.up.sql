ALTER TABLE issues ADD COLUMN culprit TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN issue_display_mode TEXT NOT NULL DEFAULT 'classic';
