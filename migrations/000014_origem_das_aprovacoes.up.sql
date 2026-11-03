ALTER TABLE approvals
    ADD COLUMN origin text NOT NULL DEFAULT 'github' CONSTRAINT approvals_origin_check CHECK (origin IN ('github', 'slack'));
