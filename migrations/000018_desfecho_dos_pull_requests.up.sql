CREATE TABLE pull_request_outcomes (
    tenant_id        bigint      NOT NULL CHECK (tenant_id > 0),
    repository       text        NOT NULL,
    pr_number        integer     NOT NULL CHECK (pr_number > 0),
    merged           boolean     NOT NULL,
    head_sha         text        NOT NULL CHECK (head_sha ~ '^[0-9a-f]{40}([0-9a-f]{24})?$'),
    merge_commit_sha text        CHECK (merge_commit_sha ~ '^[0-9a-f]{40}([0-9a-f]{24})?$'),
    closed_at        timestamptz NOT NULL,
    delivery_id      text        NOT NULL UNIQUE,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, repository, pr_number),
    CONSTRAINT pull_request_outcomes_merge_coerente CHECK (merged = (merge_commit_sha IS NOT NULL))
);

CREATE INDEX pull_request_outcomes_tenant_id ON pull_request_outcomes (tenant_id);

CREATE TABLE pull_request_reverts (
    tenant_id      bigint      NOT NULL CHECK (tenant_id > 0),
    repository     text        NOT NULL,
    pr_number      integer     NOT NULL CHECK (pr_number > 0),
    reverted_by_pr integer     NOT NULL CHECK (reverted_by_pr > 0),
    reverted_at    timestamptz NOT NULL,
    delivery_id    text        NOT NULL UNIQUE,
    created_at     timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, repository, pr_number)
);

CREATE INDEX pull_request_reverts_tenant_id ON pull_request_reverts (tenant_id);
