DROP INDEX task_runs_inicio_por_tenant_idx;

ALTER TABLE task_runs DROP COLUMN started_at;
