-- name: GetSarifUpload :one
SELECT * FROM sarif_uploads WHERE task_run_id = $1;

-- name: UpsertSarifUpload :execrows
INSERT INTO sarif_uploads (tenant_id, task_run_id, sarif_id, status, detail)
SELECT tr.tenant_id, tr.id, sqlc.narg(sarif_id), sqlc.arg(status), sqlc.arg(detail)
FROM task_runs tr
WHERE tr.id = sqlc.arg(task_run_id)
ON CONFLICT (task_run_id) DO UPDATE
SET sarif_id = EXCLUDED.sarif_id, status = EXCLUDED.status, detail = EXCLUDED.detail, updated_at = now()
WHERE sarif_uploads.sarif_id IS NULL OR sarif_uploads.sarif_id = EXCLUDED.sarif_id;

-- name: IsSarifDisabled :one
SELECT EXISTS (
    SELECT 1 FROM sarif_disabled_repositories WHERE tenant_id = $1 AND repository = $2
);

-- name: DisableSarif :exec
INSERT INTO sarif_disabled_repositories (tenant_id, repository, reason)
VALUES ($1, $2, $3)
ON CONFLICT (tenant_id, repository) DO NOTHING;
