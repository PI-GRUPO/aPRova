-- name: CreateRiskGate :one
INSERT INTO risk_gates (tenant_id, task_run_id, deterministic_level, level, reason, rules_hit, new_dependencies,
                        calibration_status, calibration_reason, calibration_neighbors)
SELECT tr.tenant_id, tr.id, sqlc.arg(deterministic_level), sqlc.arg(level), sqlc.arg(reason), sqlc.arg(rules_hit), sqlc.arg(new_dependencies),
       sqlc.arg(calibration_status), sqlc.arg(calibration_reason), sqlc.arg(calibration_neighbors)
FROM task_runs tr
WHERE tr.id = sqlc.arg(task_run_id)
RETURNING *;

-- name: GetRiskGateByTaskRun :one
SELECT * FROM risk_gates WHERE task_run_id = $1;
