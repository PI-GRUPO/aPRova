ALTER TABLE approvals
    ADD COLUMN commit_sha text CONSTRAINT approvals_commit_sha_formato CHECK (commit_sha ~ '^[0-9a-f]{40}$');
