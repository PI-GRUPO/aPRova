-- Preserva a identidade das ações já auditadas. O rollback é recusado se
-- houver registros do painel que o esquema anterior não consegue representar.
ALTER TABLE approvals DROP CONSTRAINT approvals_origin_check;
ALTER TABLE approvals ADD CONSTRAINT approvals_origin_check CHECK (origin IN ('github', 'slack'));

ALTER TABLE remote_action_attempts
    ALTER COLUMN slack_user_id SET NOT NULL,
    DROP CONSTRAINT remote_action_attempts_origem_check,
    DROP COLUMN origin;
