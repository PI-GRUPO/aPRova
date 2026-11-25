WITH autorizadas AS NOT MATERIALIZED (
    SELECT id, tenant_id, repository, pr_number, head_sha, state, received_at, started_at
    FROM task_runs
    WHERE tenant_id = @tenant AND repository = ANY(@repos::text[])
), periodo AS (
    SELECT a.*, g.level
    FROM autorizadas a
    LEFT JOIN risk_gates g ON (g.tenant_id, g.task_run_id) = (a.tenant_id, a.id)
    WHERE a.received_at >= @inicio AND a.received_at < @fim AND a.received_at <= @snapshot
), dias AS (
    SELECT (received_at AT TIME ZONE 'UTC')::date AS data,
           count(*) FILTER (WHERE level = 'alto') AS alto,
           count(*) FILTER (WHERE level = 'medio') AS medio,
           count(*) FILTER (WHERE level = 'baixo') AS baixo,
           count(*) FILTER (WHERE level IS NULL) AS nao_avaliado
    FROM periodo
    GROUP BY 1
), tentativas AS (
    SELECT s.failure_type
    FROM periodo p
    JOIN sandbox_results s ON (s.tenant_id, s.task_run_id) = (p.tenant_id, p.id)
), pendentes AS (
    SELECT (
        SELECT max(s.created_at)
        FROM task_steps s
        WHERE (s.tenant_id, s.task_run_id) = (a.tenant_id, a.id)
          AND s.to_state = 'aguardando_aprovacao'
    ) AS desde
    FROM autorizadas a
    WHERE a.state = 'aguardando_aprovacao'
), bloqueios AS (
    SELECT p.state, ultima.decision
    FROM periodo p
    LEFT JOIN LATERAL (
        SELECT a.decision
        FROM approvals a
        WHERE (a.tenant_id, a.task_run_id, a.commit_sha) = (p.tenant_id, p.id, p.head_sha)
        ORDER BY a.decided_at DESC, a.id DESC
        LIMIT 1
    ) ultima ON true
    WHERE EXISTS (
        SELECT 1 FROM task_steps s
        WHERE (s.tenant_id, s.task_run_id) = (p.tenant_id, p.id)
          AND s.to_state = 'aguardando_aprovacao'
    )
), duracoes AS (
    SELECT extract(epoch FROM (min(s.created_at) - p.started_at)) AS segundos
    FROM periodo p
    JOIN task_steps s ON (s.tenant_id, s.task_run_id, s.to_state) = (p.tenant_id, p.id, p.state)
    WHERE p.state IN ('concluido', 'rejeitado', 'falhou') AND s.created_at >= p.started_at
    GROUP BY p.id, p.started_at
), ultimo AS (
    SELECT r.id, a.repository AS repo, a.pr_number AS pr,
           r.signed_at AS emissao, encode(r.content_hash, 'hex') AS hash
    FROM autorizadas a
    JOIN receipts r ON (r.tenant_id, r.task_run_id) = (a.tenant_id, a.id)
    WHERE r.signed_at <= @snapshot AND r.created_at <= @snapshot
    ORDER BY r.signed_at DESC, r.id DESC
    LIMIT 1
)
SELECT jsonb_build_object(
    'recebidas', count(*),
    'avaliadas', count(level),
    'alto', count(*) FILTER (WHERE level = 'alto'),
    'pendentes', (SELECT count(*) FROM pendentes),
    'desde', (SELECT min(desde) FROM pendentes),
    'pendentes_sem_inicio', (SELECT count(*) FROM pendentes WHERE desde IS NULL),
    'tempo_medio_segundos', (SELECT avg(segundos) FROM duracoes),
    'bloqueios', (SELECT jsonb_build_object(
        'aguardando', count(*) FILTER (WHERE state = 'aguardando_aprovacao'),
        'aprovadas', count(*) FILTER (WHERE state <> 'aguardando_aprovacao' AND decision = 'aprovada'),
        'alteracoes', count(*) FILTER (WHERE state <> 'aguardando_aprovacao' AND decision = 'alteracoes_solicitadas'),
        'sem_decisao', count(*) FILTER (WHERE state <> 'aguardando_aprovacao' AND decision IS NULL)
    ) FROM bloqueios),
    'tentativas', (SELECT count(*) FROM tentativas),
    'falhas', (SELECT jsonb_build_object(
        'infraestrutura', count(*) FILTER (WHERE failure_type = 'infraestrutura'),
        'teste', count(*) FILTER (WHERE failure_type = 'teste'),
        'compilacao', count(*) FILTER (WHERE failure_type = 'compilacao')
    ) FROM tentativas),
    'mapa', (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY data), '[]') FROM dias d),
    'ultimo_recibo', (SELECT to_jsonb(u) FROM ultimo u)
)
FROM periodo;
