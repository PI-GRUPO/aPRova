WITH filtrados AS (
    SELECT r.id, t.repository AS repo, t.pr_number AS pr, t.head_sha AS sha,
           r.signed_at AS emissao, r.key_id, encode(r.content_hash, 'hex') AS hash_sha256
    FROM receipts r
    JOIN task_runs t ON (t.tenant_id, t.id) = (r.tenant_id, r.task_run_id)
    WHERE r.tenant_id = @tenant AND t.repository = ANY(@repos::text[])
      AND r.signed_at >= @inicio AND r.signed_at < @fim
      AND r.created_at <= @snapshot AND r.signed_at <= @snapshot
), pagina AS (
    SELECT * FROM filtrados
    ORDER BY emissao DESC, id DESC
    LIMIT @limite OFFSET @offset
)
SELECT jsonb_build_object(
    'total', (SELECT count(*) FROM filtrados),
    'itens', coalesce(jsonb_agg(to_jsonb(p) ORDER BY emissao DESC, id DESC), '[]')
)
FROM pagina p;
