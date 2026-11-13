-- name: CreateApproval :execrows
INSERT INTO approvals (tenant_id, task_run_id, delivery_id, decision, reviewer, origin, feedback_reason, justification, decided_at, commit_sha)
SELECT tr.tenant_id, tr.id, sqlc.arg(delivery_id), sqlc.arg(decision), sqlc.arg(reviewer), sqlc.arg(origin),
       sqlc.narg(feedback_reason), sqlc.narg(justification), sqlc.arg(decided_at), sqlc.narg(commit_sha)
FROM task_runs tr
WHERE tr.id = sqlc.arg(task_run_id)
ON CONFLICT (delivery_id) DO NOTHING;

-- name: NextApprovalAfter :one
SELECT * FROM approvals
WHERE task_run_id = sqlc.arg(task_run_id) AND id > sqlc.arg(after_id)
ORDER BY id
LIMIT 1;
