-- name: CreateCodeMemory :one
INSERT INTO code_memory (tenant_id, task_run_id, repository, summary, embedding)
SELECT tr.tenant_id, tr.id, tr.repository, sqlc.arg(summary), sqlc.arg(embedding)
FROM task_runs tr
WHERE tr.id = sqlc.arg(task_run_id) AND tr.tenant_id = sqlc.arg(tenant_id)
RETURNING id;

-- name: FindSimilarAnalyses :many
SELECT tr.id AS task_run_id, tr.tenant_id, tr.repository, tr.pr_number,
       (1 - proximos.distance)::float8 AS similarity,
       (CASE
           WHEN revertido.pr_number IS NOT NULL THEN 'revertido'
           WHEN desfecho.pr_number IS NULL THEN 'em_aberto'
           WHEN NOT desfecho.merged THEN 'fechado_sem_merge'
           WHEN desfecho.head_sha <> tr.head_sha THEN 'substituido'
           WHEN EXISTS (
               SELECT 1 FROM approvals a
               JOIN task_runs revisado ON revisado.id = a.task_run_id
               WHERE revisado.tenant_id = tr.tenant_id AND revisado.repository = tr.repository
                 AND revisado.pr_number = tr.pr_number AND a.decision = 'aprovada' AND a.commit_sha = tr.head_sha
           ) THEN 'aprovado_e_mergeado'
           ELSE 'mergeado_sem_aprovacao'
       END)::text AS outcome
FROM (
    SELECT cm.task_run_id, cm.embedding <=> sqlc.arg(embedding)::vector AS distance
    FROM code_memory cm
    WHERE cm.tenant_id = sqlc.arg(tenant_id)
      AND NOT EXISTS (
          SELECT 1 FROM task_runs proprio
          WHERE proprio.id = cm.task_run_id
            AND proprio.repository = sqlc.arg(repository) AND proprio.pr_number = sqlc.arg(pr_number))
    ORDER BY cm.embedding <=> sqlc.arg(embedding)::vector
    LIMIT sqlc.arg(max_results)
) proximos
JOIN task_runs tr ON tr.id = proximos.task_run_id
LEFT JOIN pull_request_outcomes desfecho
    ON desfecho.tenant_id = tr.tenant_id AND desfecho.repository = tr.repository AND desfecho.pr_number = tr.pr_number
LEFT JOIN pull_request_reverts revertido
    ON revertido.tenant_id = tr.tenant_id AND revertido.repository = tr.repository AND revertido.pr_number = tr.pr_number
ORDER BY proximos.distance, tr.id;
