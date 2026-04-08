CREATE TABLE issue_follows (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    issue_id UUID NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, issue_id)
);

CREATE INDEX idx_issue_follows_user ON issue_follows(user_id);
