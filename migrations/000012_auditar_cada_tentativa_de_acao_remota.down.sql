DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM remote_action_attempts GROUP BY action_id HAVING count(*) > 1) THEN
        RAISE EXCEPTION 'remote_action_attempts tem mais de uma tentativa para o mesmo action_id; o esquema anterior só comporta uma';
    END IF;
    IF EXISTS (SELECT 1 FROM remote_action_attempts WHERE decision = 'repetida') THEN
        RAISE EXCEPTION 'remote_action_attempts tem tentativas repetidas; o esquema anterior não as aceita';
    END IF;
END
$$;

DROP INDEX remote_action_attempts_por_acao;
DROP INDEX remote_action_attempts_uma_aceita_por_acao;

ALTER TABLE remote_action_attempts DROP CONSTRAINT remote_action_attempts_decision_check;
ALTER TABLE remote_action_attempts ADD CONSTRAINT remote_action_attempts_decision_check
    CHECK (decision IN ('aceita', 'recusada', 'falha'));

ALTER TABLE remote_action_attempts DROP CONSTRAINT remote_action_attempts_pkey;
ALTER TABLE remote_action_attempts DROP COLUMN id;
ALTER TABLE remote_action_attempts ADD PRIMARY KEY (action_id);
