-- Per-project workflow mode
ALTER TABLE projects ADD COLUMN workflow_mode TEXT NOT NULL DEFAULT 'simple';

-- Tickets: management layer for issues
CREATE TABLE tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    issue_id UUID NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'acknowledged'
        CHECK (status IN ('acknowledged', 'in_progress', 'in_review', 'done', 'wontfix', 'escalated')),
    assigned_to UUID REFERENCES users(id) ON DELETE SET NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    priority INT NOT NULL DEFAULT 50,
    due_date TIMESTAMPTZ,
    resolution_type TEXT,
    resolution_notes TEXT,
    fix_reference TEXT,
    title TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    escalated_system TEXT,
    escalated_key TEXT,
    escalated_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tickets_issue ON tickets(issue_id);
CREATE INDEX idx_tickets_project_status ON tickets(project_id, status);
CREATE INDEX idx_tickets_assigned ON tickets(assigned_to) WHERE assigned_to IS NOT NULL;

-- Prevent race condition: only one active ticket per issue
CREATE UNIQUE INDEX idx_tickets_one_active_per_issue ON tickets(issue_id) WHERE status != 'done';
