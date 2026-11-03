DROP INDEX task_runs_fila;

ALTER TABLE task_runs
    DROP COLUMN last_approval_seen,
    DROP COLUMN step_attempts,
    DROP COLUMN lease_expires_at,
    DROP COLUMN claimed_by;
