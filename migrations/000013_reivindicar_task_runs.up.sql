ALTER TABLE task_runs
    ADD COLUMN claimed_by         text,
    ADD COLUMN lease_expires_at   timestamptz,
    ADD COLUMN step_attempts      integer NOT NULL DEFAULT 0 CHECK (step_attempts >= 0),
    ADD COLUMN last_approval_seen bigint NOT NULL DEFAULT 0 CHECK (last_approval_seen >= 0);

CREATE INDEX task_runs_fila ON task_runs (created_at, id)
    WHERE state IN ('pendente', 'mapeando_contexto', 'testando_sandbox', 'avaliando_risco', 'aguardando_aprovacao', 'aplicando');
