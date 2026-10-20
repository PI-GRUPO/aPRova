DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM code_memory) THEN
        RAISE EXCEPTION 'code_memory tem % linha(s) com embedding de 1536 dimensões; regere os embeddings com 768 dimensões antes de aplicar esta migration', (SELECT count(*) FROM code_memory);
    END IF;
END
$$;

DROP INDEX code_memory_embedding;
ALTER TABLE code_memory ALTER COLUMN embedding TYPE vector(768);
CREATE INDEX code_memory_embedding ON code_memory USING hnsw (embedding vector_cosine_ops);
