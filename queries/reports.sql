-- name: GetWeeklyReport :one
SELECT
    count(*) AS total,
    count(*) FILTER (WHERE tr.state = 'concluido') AS succeeded,
    count(*) FILTER (WHERE tr.state = 'rejeitado') AS rejected,
    count(*) FILTER (WHERE tr.state = 'falhou') AS failed,
    count(*) FILTER (WHERE EXISTS (
        SELECT 1 FROM task_steps ts
        WHERE ts.task_run_id = tr.id AND ts.to_state = 'aguardando_aprovacao'
    )) AS needed_approval,
    count(*) FILTER (WHERE EXISTS (
        SELECT 1 FROM risk_gates rg
        WHERE rg.task_run_id = tr.id AND rg.level = 'medio'
    )) AS medium_risk,
    (
        SELECT count(*) FROM code_memory cm
        WHERE cm.tenant_id = sqlc.arg(tenant_id)
          AND cm.outcome = 'revertida_apos_merge'
          AND cm.created_at >= sqlc.arg(starts_at)
          AND cm.created_at < sqlc.arg(ends_at)
    ) AS reverted
FROM task_runs tr
WHERE tr.tenant_id = sqlc.arg(tenant_id)
  AND tr.received_at >= sqlc.arg(starts_at)
  AND tr.received_at < sqlc.arg(ends_at);
