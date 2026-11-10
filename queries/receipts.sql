-- name: AppendReceipt :one
INSERT INTO receipts (id, tenant_id, task_run_id, previous_receipt_id, payload, content_hash, signature, key_id, signed_at)
SELECT COALESCE(sqlc.narg(id)::uuid, gen_random_uuid()), tr.tenant_id, tr.id, sqlc.narg(previous_receipt_id), sqlc.arg(payload), sqlc.arg(content_hash),
       sqlc.arg(signature), sqlc.arg(key_id), sqlc.arg(signed_at)
FROM task_runs tr
WHERE tr.id = sqlc.arg(task_run_id)
  AND (sqlc.narg(previous_receipt_id)::uuid IS NULL OR EXISTS (
      SELECT 1 FROM receipts anterior
      WHERE anterior.id = sqlc.narg(previous_receipt_id)::uuid AND anterior.tenant_id = tr.tenant_id))
RETURNING *;

-- name: GetReceipt :one
SELECT * FROM receipts WHERE id = $1;

-- name: GetReceiptByTaskRun :one
SELECT * FROM receipts WHERE task_run_id = $1;

-- name: LockReceiptChain :exec
SELECT pg_advisory_xact_lock(hashtextextended('receipts', sqlc.arg(tenant_id)::bigint));

-- name: GetLastReceiptOfTenant :one
SELECT r.* FROM receipts r
WHERE r.tenant_id = $1
  AND NOT EXISTS (SELECT 1 FROM receipts sucessor WHERE sucessor.previous_receipt_id = r.id)
ORDER BY r.signed_at DESC, r.created_at DESC, r.id DESC
LIMIT 1;
