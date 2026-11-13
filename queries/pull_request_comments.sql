-- name: GetPullRequestComment :one
SELECT comment_id FROM pull_request_comments
WHERE tenant_id = $1 AND repository = $2 AND pr_number = $3;

-- name: UpsertPullRequestComment :exec
INSERT INTO pull_request_comments (tenant_id, repository, pr_number, comment_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (tenant_id, repository, pr_number)
DO UPDATE SET comment_id = EXCLUDED.comment_id, updated_at = now();
