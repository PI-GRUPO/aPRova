WITH gates AS (
    SELECT g.id, g.task_run_id, g.tenant_id, t.repository, t.pr_number, t.head_sha,
           g.level, g.deterministic_level, g.rules_hit, g.calibration_status, g.created_at
    FROM risk_gates g
    JOIN task_runs t ON (t.tenant_id, t.id) = (g.tenant_id, g.task_run_id)
    WHERE g.tenant_id = @tenant AND t.repository = ANY(@repos::text[])
      AND g.created_at >= @inicio AND g.created_at < @fim AND g.created_at <= @snapshot
), contagens AS (
    SELECT m->>'id' AS id, m->>'nivel' AS nivel, max(left(m->>'descricao', 300)) AS descricao,
           count(DISTINCT g.id) AS acionamentos,
           count(DISTINCT g.id) FILTER (WHERE g.level = 'alto') AS avaliadas_altas,
           count(DISTINCT g.id) FILTER (WHERE g.level = 'alto' AND EXISTS (
               SELECT 1 FROM approvals a
               WHERE (a.tenant_id, a.task_run_id, a.commit_sha) = (g.tenant_id, g.task_run_id, g.head_sha)
                 AND a.decision = 'aprovada'
           )) AS aprovadas
    FROM gates g CROSS JOIN LATERAL jsonb_array_elements(g.rules_hit) m
    WHERE m->>'nivel' IN ('baixo', 'medio', 'alto')
    GROUP BY 1, 2
), calibracoes AS (
    SELECT task_run_id AS id, repository AS repo, pr_number AS pr, deterministic_level AS inicial,
           level AS final, calibration_status AS resultado, created_at
    FROM gates
    ORDER BY created_at DESC, id DESC
    LIMIT 50
)
SELECT jsonb_build_object(
    'regras', (SELECT coalesce(jsonb_agg(to_jsonb(c) ORDER BY id, nivel), '[]') FROM contagens c),
    'calibracoes', (SELECT coalesce(jsonb_agg(to_jsonb(c) - 'created_at' ORDER BY created_at DESC, id DESC), '[]') FROM calibracoes c)
);
