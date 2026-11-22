WITH filtradas AS (
    SELECT id, attempted_at AS data, github_login AS pessoa, action_type AS acao,
           repository AS repo, pr_number AS pr, decision AS decisao, left(reason, 500) AS motivo
    FROM remote_action_attempts a
    WHERE tenant_id = @tenant AND repository = ANY(@repos::text[])
      AND (a.task_run_id IS NULL OR EXISTS (
          SELECT 1 FROM task_runs t
          WHERE (t.id, t.tenant_id, t.repository) = (a.task_run_id, a.tenant_id, a.repository)
      ))
      AND attempted_at >= @inicio AND attempted_at < @fim
      AND attempted_at <= @snapshot AND created_at <= @snapshot
      AND (@decisao::text = '' OR decision = @decisao) AND @origem::text <> 'Painel'
), pagina AS (
    SELECT * FROM filtradas
    ORDER BY data DESC, id DESC
    LIMIT @limite OFFSET @offset
)
SELECT jsonb_build_object(
    'total', (SELECT count(*) FROM filtradas),
    'itens', coalesce(jsonb_agg(to_jsonb(p) || jsonb_build_object('id', id::text) ORDER BY data DESC, id DESC), '[]')
)
FROM pagina p;
