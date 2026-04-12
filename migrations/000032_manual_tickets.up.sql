-- Allow tickets without a linked issue (manual tickets)
ALTER TABLE tickets ALTER COLUMN issue_id DROP NOT NULL;
