DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM code_memory) THEN
        RAISE EXCEPTION 'code_memory tem % linha(s) sem desfecho gravado; o formato antigo exige level e outcome em cada linha, então apague essas linhas antes de reverter esta migration', (SELECT count(*) FROM code_memory);
    END IF;
END
$$;

ALTER TABLE code_memory
    DROP CONSTRAINT code_memory_summary_check,
    DROP CONSTRAINT code_memory_task_run_id_key,
    DROP CONSTRAINT code_memory_task_run_id_fkey,
    ALTER COLUMN task_run_id DROP NOT NULL,
    ADD CONSTRAINT code_memory_task_run_id_fkey FOREIGN KEY (task_run_id) REFERENCES task_runs (id) ON DELETE SET NULL,
    ADD COLUMN level text NOT NULL CHECK (level IN ('baixo', 'medio', 'alto')),
    ADD COLUMN outcome text NOT NULL CHECK (outcome IN (
        'revertida_apos_merge', 'alto_risco_aprovada', 'fechada_sem_merge',
        'ajustada_e_aprovada', 'sem_reversao'
    ));

CREATE INDEX code_memory_task_run_id ON code_memory (task_run_id);
