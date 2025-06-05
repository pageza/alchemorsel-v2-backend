-- Add vector extension and embedding column
CREATE EXTENSION IF NOT EXISTS vector;

ALTER TABLE recipes
    ADD COLUMN IF NOT EXISTS embedding vector(3);

-- Index for nearest neighbor search
CREATE INDEX IF NOT EXISTS idx_recipes_embedding ON recipes USING ivfflat (embedding vector_l2_ops);
