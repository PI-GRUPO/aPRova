ALTER TABLE pull_request_outcomes
    ADD COLUMN base_ref text CHECK (char_length(base_ref) BETWEEN 1 AND 255);
