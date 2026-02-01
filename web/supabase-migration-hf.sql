-- Migration: Switch to Hugging Face all-MiniLM-L6-v2 (384 dimensions)
-- Run this in Supabase SQL Editor

-- Clear existing embeddings (they have different dimensions)
DELETE FROM memo_embeddings;

-- Drop existing indexes
DROP INDEX IF EXISTS memo_embeddings_embedding_idx;
DROP INDEX IF EXISTS memo_embeddings_embedding_hnsw_idx;
DROP INDEX IF EXISTS memo_embeddings_intent_embedding_idx;

-- Alter column to 384 dimensions
ALTER TABLE memo_embeddings
  ALTER COLUMN embedding TYPE vector(384);

-- Recreate index with HNSW (faster for kNN)
CREATE INDEX memo_embeddings_embedding_hnsw_idx
  ON memo_embeddings USING hnsw (embedding vector_cosine_ops);

-- Verify
SELECT column_name, data_type, udt_name
FROM information_schema.columns
WHERE table_name = 'memo_embeddings' AND column_name = 'embedding';
