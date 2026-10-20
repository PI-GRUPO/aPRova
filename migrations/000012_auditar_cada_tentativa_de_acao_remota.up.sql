ALTER TABLE remote_action_attempts DROP CONSTRAINT remote_action_attempts_pkey;
ALTER TABLE remote_action_attempts ADD COLUMN id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY;

ALTER TABLE remote_action_attempts DROP CONSTRAINT remote_action_attempts_decision_check;
ALTER TABLE remote_action_attempts ADD CONSTRAINT remote_action_attempts_decision_check
    CHECK (decision IN ('aceita', 'recusada', 'falha', 'repetida'));

CREATE UNIQUE INDEX remote_action_attempts_uma_aceita_por_acao ON remote_action_attempts (action_id)
    WHERE decision = 'aceita';
CREATE INDEX remote_action_attempts_por_acao ON remote_action_attempts (action_id, id);
