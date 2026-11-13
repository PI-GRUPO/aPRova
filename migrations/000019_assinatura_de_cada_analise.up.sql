DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM code_memory) THEN
        RAISE EXCEPTION 'code_memory tem % linha(s) no formato antigo, com o desfecho congelado na linha; o histórico passa a derivar o desfecho dos fatos do PR, então apague ou regere essas linhas antes de aplicar esta migration', (SELECT count(*) FROM code_memory);
    END IF;
END
$$;

DROP INDEX code_memory_task_run_id;

ALTER TABLE code_memory
    DROP COLUMN level,
    DROP COLUMN outcome,
    DROP CONSTRAINT code_memory_task_run_id_fkey,
    ALTER COLUMN task_run_id SET NOT NULL,
    ADD CONSTRAINT code_memory_task_run_id_fkey FOREIGN KEY (task_run_id) REFERENCES task_runs (id) ON DELETE RESTRICT,
    ADD CONSTRAINT code_memory_task_run_id_key UNIQUE (task_run_id),
    ADD CONSTRAINT code_memory_summary_check CHECK (summary <> '');
