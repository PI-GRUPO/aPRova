-- name: GetRiskExplanation :one
SELECT * FROM risk_explanations WHERE task_run_id = $1;

-- name: CreateRiskExplanation :exec
INSERT INTO risk_explanations (task_run_id, result, text, model, reason)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (task_run_id) DO NOTHING;
