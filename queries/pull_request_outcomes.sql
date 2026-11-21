-- name: UpsertPullRequestOutcome :exec
INSERT INTO pull_request_outcomes (tenant_id, repository, pr_number, merged, head_sha, merge_commit_sha, commits, closed_at, delivery_id)
VALUES (sqlc.arg(tenant_id), sqlc.arg(repository), sqlc.arg(pr_number), sqlc.arg(merged), sqlc.arg(head_sha),
        sqlc.narg(merge_commit_sha), sqlc.arg(commits), sqlc.arg(closed_at), sqlc.arg(delivery_id))
ON CONFLICT (tenant_id, repository, pr_number) DO UPDATE
SET merged = EXCLUDED.merged, head_sha = EXCLUDED.head_sha, merge_commit_sha = EXCLUDED.merge_commit_sha,
    commits = EXCLUDED.commits, closed_at = EXCLUDED.closed_at, delivery_id = EXCLUDED.delivery_id, updated_at = now()
WHERE pull_request_outcomes.closed_at < EXCLUDED.closed_at;

-- name: GetPullRequestOutcome :one
SELECT merged, head_sha, merge_commit_sha, commits FROM pull_request_outcomes
WHERE tenant_id = sqlc.arg(tenant_id) AND repository = sqlc.arg(repository) AND pr_number = sqlc.arg(pr_number);

-- name: CreatePullRequestRevert :exec
INSERT INTO pull_request_reverts (tenant_id, repository, pr_number, reverted_by_pr, reverted_at, delivery_id)
VALUES (sqlc.arg(tenant_id), sqlc.arg(repository), sqlc.arg(pr_number), sqlc.arg(reverted_by_pr), sqlc.arg(reverted_at), sqlc.arg(delivery_id))
ON CONFLICT DO NOTHING;
