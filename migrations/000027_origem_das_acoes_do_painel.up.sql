ALTER TABLE remote_action_attempts
    ALTER COLUMN slack_user_id DROP NOT NULL,
    ADD COLUMN origin text NOT NULL DEFAULT 'slack',
    ADD CONSTRAINT remote_action_attempts_origem_check CHECK (
        (origin = 'slack' AND slack_user_id IS NOT NULL) OR
        (origin = 'painel' AND slack_user_id IS NULL AND github_login IS NOT NULL)
    );

ALTER TABLE approvals DROP CONSTRAINT approvals_origin_check;
ALTER TABLE approvals ADD CONSTRAINT approvals_origin_check CHECK (origin IN ('github', 'slack', 'painel'));
