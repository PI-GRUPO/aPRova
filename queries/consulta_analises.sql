-- name: ListAnalyses :many
SELECT * FROM task_runs
WHERE tenant_id = sqlc.arg(tenant_id)
  AND (sqlc.arg(repository)::text = '' OR lower(repository) = lower(sqlc.arg(repository)))
  AND (sqlc.arg(pr_number)::integer = 0 OR pr_number = sqlc.arg(pr_number))
  AND (sqlc.arg(state_filter)::text = 'todos'
    OR (sqlc.arg(state_filter) = 'em_andamento' AND state NOT IN ('concluido', 'rejeitado', 'falhou'))
    OR state = sqlc.arg(state_filter))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: GetScopedAnalysis :one
SELECT * FROM task_runs
WHERE id = sqlc.arg(id) AND tenant_id = sqlc.arg(tenant_id)
  AND (sqlc.arg(repository)::text = '' OR lower(repository) = lower(sqlc.arg(repository)));
