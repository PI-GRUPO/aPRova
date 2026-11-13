ALTER TABLE task_runs
    ADD COLUMN check_run_id bigint CONSTRAINT task_runs_check_run_id_positivo CHECK (check_run_id > 0);

CREATE TABLE pull_request_comments (
    tenant_id  bigint      NOT NULL CHECK (tenant_id > 0),
    repository text        NOT NULL,
    pr_number  integer     NOT NULL CHECK (pr_number > 0),
    comment_id bigint      NOT NULL CHECK (comment_id > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, repository, pr_number)
);

CREATE INDEX pull_request_comments_tenant_id ON pull_request_comments (tenant_id);
