ALTER TABLE task_runs ADD COLUMN started_at timestamptz;

CREATE INDEX task_runs_inicio_por_tenant_idx ON task_runs (tenant_id, started_at) WHERE started_at IS NOT NULL;
