-- Migration: Switch from OpenAI (1536 dims) to BGE-small (384 dims)
-- Run this if you want to use the local bge-small-en-v1.5 model

-- Option 1: Drop and recreate the table (loses existing data)
-- DROP TABLE IF EXISTS memo_embeddings;

-- Option 2: Alter the column type (keeps data but invalidates existing embeddings)
-- You'll need to re-embed all notes after this

-- First, drop dependent objects
DROP INDEX IF EXISTS memo_embeddings_embedding_idx;
DROP INDEX IF EXISTS memo_embeddings_embedding_hnsw_idx;
DROP INDEX IF EXISTS memo_embeddings_intent_embedding_idx;

-- Alter the embedding column to support 384 dimensions
ALTER TABLE memo_embeddings
  ALTER COLUMN embedding TYPE vector(384)
  USING embedding::vector(384);

-- Recreate index with HNSW (faster for kNN)
CREATE INDEX IF NOT EXISTS memo_embeddings_embedding_hnsw_idx
  ON memo_embeddings USING hnsw (embedding vector_cosine_ops);

-- Note: After running this migration, you need to:
-- 1. Delete all existing rows: DELETE FROM memo_embeddings;
-- 2. Re-save all your notes to generate new embeddings with the local model
