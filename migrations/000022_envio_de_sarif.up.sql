CREATE TABLE sarif_uploads (
    tenant_id   bigint      NOT NULL CHECK (tenant_id > 0),
    task_run_id uuid        PRIMARY KEY REFERENCES task_runs (id) ON DELETE RESTRICT,
    sarif_id    text        CHECK (sarif_id <> ''),
    status      text        NOT NULL CHECK (status IN ('aceito', 'processado', 'recusado', 'falhou')),
    detail      text        NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT sarif_uploads_id_so_quando_aceito CHECK ((status = 'falhou') = (sarif_id IS NULL))
);

CREATE INDEX sarif_uploads_tenant_id ON sarif_uploads (tenant_id);

CREATE TABLE sarif_disabled_repositories (
    tenant_id  bigint      NOT NULL CHECK (tenant_id > 0),
    repository text        NOT NULL,
    reason     text        NOT NULL CHECK (reason <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, repository)
);

CREATE INDEX sarif_disabled_repositories_tenant_id ON sarif_disabled_repositories (tenant_id);
