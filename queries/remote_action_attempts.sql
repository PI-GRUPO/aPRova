-- name: CreateRemoteActionAttempt :one
INSERT INTO remote_action_attempts (
    action_id, tenant_id, task_run_id, action_type, slack_user_id, github_login,
    repository, pr_number, decision, reason, attempted_at
)
VALUES (
    sqlc.arg(action_id), sqlc.arg(tenant_id),
    (SELECT tr.id FROM task_runs tr
     WHERE tr.tenant_id = sqlc.arg(tenant_id) AND tr.repository = sqlc.arg(repository) AND tr.pr_number = sqlc.arg(pr_number)
     ORDER BY tr.created_at DESC, tr.id DESC
     LIMIT 1),
    sqlc.arg(action_type), sqlc.arg(slack_user_id), sqlc.narg(github_login),
    sqlc.arg(repository), sqlc.arg(pr_number), sqlc.arg(decision), sqlc.arg(reason), sqlc.arg(attempted_at)
)
RETURNING id;

-- name: MarkRemoteActionAttemptFailed :execrows
UPDATE remote_action_attempts
SET decision = 'falha', reason = sqlc.arg(reason)
WHERE id = sqlc.arg(id) AND decision = 'aceita';
