DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM code_memory) THEN
        RAISE EXCEPTION 'code_memory tem % linha(s) com embedding de 768 dimensões; não há como voltar a 1536 sem regerar os embeddings', (SELECT count(*) FROM code_memory);
    END IF;
END
$$;

DROP INDEX code_memory_embedding;
ALTER TABLE code_memory ALTER COLUMN embedding TYPE vector(1536);
CREATE INDEX code_memory_embedding ON code_memory USING hnsw (embedding vector_cosine_ops);
