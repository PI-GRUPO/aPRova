ALTER TABLE remote_action_attempts
    ADD COLUMN justification text NOT NULL DEFAULT ''
    CHECK (char_length(justification) <= 4000);
