CREATE TABLE risk_explanations (
    task_run_id uuid PRIMARY KEY REFERENCES task_runs (id) ON DELETE CASCADE,
    result      text        NOT NULL CHECK (result IN ('gerada', 'descartada', 'falha')),
    text        text        NOT NULL DEFAULT '' CHECK (char_length(text) <= 900),
    model       text        NOT NULL CHECK (char_length(model) BETWEEN 1 AND 100),
    reason      text        NOT NULL DEFAULT '' CHECK (char_length(reason) <= 1000),
    created_at  timestamptz NOT NULL DEFAULT now(),
    CHECK ((result = 'gerada') = (text <> ''))
);
