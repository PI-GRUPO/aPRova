-- name: CreateTaskRun :one
INSERT INTO task_runs (tenant_id, delivery_id, repository, pr_number, head_sha, author, event_action, received_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (delivery_id) DO NOTHING
RETURNING *;

-- name: GetTaskRun :one
SELECT * FROM task_runs WHERE id = $1;

-- name: GetTaskRunByDeliveryID :one
SELECT * FROM task_runs WHERE delivery_id = $1;

-- name: GetLatestTaskRunForPullRequest :one
SELECT * FROM task_runs
WHERE tenant_id = $1 AND repository = $2 AND pr_number = $3
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: TaskRunExists :one
SELECT EXISTS (SELECT 1 FROM task_runs WHERE id = $1);

-- name: TransitionTaskRun :one
UPDATE task_runs
SET state = sqlc.arg(to_state), updated_at = now()
WHERE id = sqlc.arg(id) AND state = sqlc.arg(from_state)
RETURNING *;

-- name: ClaimNextTaskRun :one
UPDATE task_runs
SET claimed_by = sqlc.arg(worker), lease_expires_at = now() + make_interval(secs => sqlc.arg(lease_seconds)::double precision)
WHERE id = (
    SELECT tr.id FROM task_runs tr
    WHERE tr.state IN ('pendente', 'mapeando_contexto', 'testando_sandbox', 'avaliando_risco', 'aguardando_aprovacao', 'aplicando')
      AND (tr.lease_expires_at IS NULL OR tr.lease_expires_at <= now())
      AND (
          tr.state <> 'aguardando_aprovacao'
          OR EXISTS (SELECT 1 FROM approvals a WHERE a.task_run_id = tr.id AND a.id > tr.last_approval_seen)
          OR EXISTS (
              SELECT 1 FROM task_runs novo
              WHERE novo.tenant_id = tr.tenant_id AND novo.repository = tr.repository AND novo.pr_number = tr.pr_number
                AND (novo.created_at, novo.id) > (tr.created_at, tr.id))
      )
    ORDER BY tr.created_at, tr.id
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING *;

-- name: RenewTaskRunLease :execrows
UPDATE task_runs
SET lease_expires_at = now() + make_interval(secs => sqlc.arg(lease_seconds)::double precision)
WHERE id = sqlc.arg(id) AND claimed_by = sqlc.arg(worker);

-- name: ReleaseTaskRun :execrows
UPDATE task_runs
SET claimed_by = NULL, lease_expires_at = now() + make_interval(secs => sqlc.arg(wait_seconds)::double precision)
WHERE id = sqlc.arg(id) AND claimed_by = sqlc.arg(worker);

-- name: TransitionClaimedTaskRun :one
UPDATE task_runs
SET state = sqlc.arg(to_state),
    updated_at = now(),
    step_attempts = 0,
    last_approval_seen = CASE
        WHEN sqlc.arg(to_state) = 'aguardando_aprovacao'
            THEN (SELECT coalesce(max(a.id), 0) FROM approvals a WHERE a.task_run_id = task_runs.id)
        ELSE last_approval_seen
    END
WHERE task_runs.id = sqlc.arg(id) AND task_runs.state = sqlc.arg(from_state) AND task_runs.claimed_by = sqlc.arg(worker)
RETURNING *;

-- name: IncrementStepAttempts :one
UPDATE task_runs
SET step_attempts = step_attempts + 1
WHERE id = sqlc.arg(id) AND claimed_by = sqlc.arg(worker)
RETURNING step_attempts;

-- name: AdvanceApprovalCursor :execrows
UPDATE task_runs
SET last_approval_seen = greatest(last_approval_seen, sqlc.arg(seen))
WHERE id = sqlc.arg(id) AND claimed_by = sqlc.arg(worker);

-- name: NewerTaskRunExists :one
SELECT EXISTS (
    SELECT 1 FROM task_runs novo
    WHERE novo.tenant_id = sqlc.arg(tenant_id) AND novo.repository = sqlc.arg(repository) AND novo.pr_number = sqlc.arg(pr_number)
      AND (novo.created_at, novo.id) > (sqlc.arg(created_at)::timestamptz, sqlc.arg(id)::uuid)
);

-- name: GetCheckRunID :one
SELECT check_run_id FROM task_runs WHERE id = $1;

-- name: SetCheckRunID :execrows
UPDATE task_runs SET check_run_id = sqlc.arg(check_run_id) WHERE id = sqlc.arg(id);
