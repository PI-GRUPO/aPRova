-- name: ListWebAnalyses :many
SELECT tr.*, rg.level AS risk_level
FROM task_runs tr
LEFT JOIN risk_gates rg ON rg.task_run_id = tr.id AND rg.tenant_id = tr.tenant_id
WHERE tr.tenant_id = sqlc.arg(tenant_id)
 AND tr.repository = ANY(sqlc.arg(repositories)::text[])
 AND tr.created_at <= sqlc.arg(snapshot_at)
 AND (sqlc.arg(pr_number)::integer = 0 OR tr.pr_number = sqlc.arg(pr_number))
 AND (sqlc.arg(state_filter)::text = 'todos' OR tr.state = sqlc.arg(state_filter))
 AND (sqlc.arg(level_filter)::text = 'todos' OR rg.level = sqlc.arg(level_filter)
      OR (sqlc.arg(level_filter) = 'nao_avaliado' AND rg.id IS NULL))
 AND tr.received_at >= sqlc.arg(starts_at) AND tr.received_at < sqlc.arg(ends_at)
ORDER BY
 CASE WHEN sqlc.arg(sort_order)::text = 'antigas' THEN tr.received_at END ASC,
 CASE WHEN sqlc.arg(sort_order)::text = 'recentes' THEN tr.received_at END DESC,
 tr.id DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountWebAnalyses :one
SELECT count(*) FROM task_runs tr
LEFT JOIN risk_gates rg ON rg.task_run_id = tr.id AND rg.tenant_id = tr.tenant_id
WHERE tr.tenant_id = sqlc.arg(tenant_id)
 AND tr.repository = ANY(sqlc.arg(repositories)::text[])
 AND tr.created_at <= sqlc.arg(snapshot_at)
 AND (sqlc.arg(pr_number)::integer = 0 OR tr.pr_number = sqlc.arg(pr_number))
 AND (sqlc.arg(state_filter)::text = 'todos' OR tr.state = sqlc.arg(state_filter))
 AND (sqlc.arg(level_filter)::text = 'todos' OR rg.level = sqlc.arg(level_filter)
      OR (sqlc.arg(level_filter) = 'nao_avaliado' AND rg.id IS NULL))
 AND tr.received_at >= sqlc.arg(starts_at) AND tr.received_at < sqlc.arg(ends_at);

-- name: GetWebAnalysis :one
SELECT tr.*, rg.level AS risk_level,
 NOT EXISTS(SELECT 1 FROM task_runs newer
   WHERE newer.tenant_id=tr.tenant_id AND newer.repository=tr.repository AND newer.pr_number=tr.pr_number
    AND (newer.created_at,newer.id)>(tr.created_at,tr.id))::boolean AS is_current
FROM task_runs tr
LEFT JOIN risk_gates rg ON rg.task_run_id=tr.id AND rg.tenant_id=tr.tenant_id
WHERE tr.id=sqlc.arg(id) AND tr.tenant_id=sqlc.arg(tenant_id)
 AND tr.repository=ANY(sqlc.arg(repositories)::text[]);

-- name: ListWebSteps :many
SELECT * FROM task_steps WHERE tenant_id=$1 AND task_run_id=$2 ORDER BY created_at,id;

-- name: ListWebSandboxResults :many
SELECT id, tenant_id, task_run_id, attempt, compiled, tests_passed, tests_failed,
 left(raw_output,65536)::text AS raw_output, (length(raw_output)>65536)::boolean AS output_truncated,
 duration_ms,failure_type,created_at
FROM sandbox_results WHERE tenant_id=$1 AND task_run_id=$2 ORDER BY attempt,id;

-- name: GetWebRiskGate :one
SELECT * FROM risk_gates WHERE tenant_id=$1 AND task_run_id=$2;

-- name: ListWebApprovals :many
SELECT * FROM approvals WHERE tenant_id=$1 AND task_run_id=$2 ORDER BY decided_at,id;

-- name: GetWebReceiptID :one
SELECT id FROM receipts WHERE tenant_id=$1 AND task_run_id=$2;
