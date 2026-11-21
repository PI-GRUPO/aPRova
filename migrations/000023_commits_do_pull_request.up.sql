ALTER TABLE pull_request_outcomes
    ADD COLUMN commits integer NOT NULL DEFAULT 0 CHECK (commits >= 0);
