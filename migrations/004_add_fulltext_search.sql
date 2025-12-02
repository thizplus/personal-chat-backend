-- Migration: Add full-text search support for messages
-- Date: 2025-11-29
-- Purpose: Add content_tsvector column and GIN index for fast text search

-- Step 1: Add tsvector column for full-text search
ALTER TABLE messages
ADD COLUMN IF NOT EXISTS content_tsvector tsvector;

-- Step 2: Populate existing data with tsvector values
UPDATE messages
SET content_tsvector = to_tsvector('english', COALESCE(content, ''))
WHERE content_tsvector IS NULL;

-- Step 3: Create GIN index for fast full-text search
CREATE INDEX IF NOT EXISTS idx_messages_content_tsvector
ON messages USING GIN (content_tsvector);

-- Step 4: Create trigger function to auto-update tsvector on INSERT/UPDATE
CREATE OR REPLACE FUNCTION messages_content_tsvector_update()
RETURNS trigger AS $$
BEGIN
  NEW.content_tsvector := to_tsvector('english', COALESCE(NEW.content, ''));
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Step 5: Create trigger to call the function
DROP TRIGGER IF EXISTS tsvector_update ON messages;
CREATE TRIGGER tsvector_update
BEFORE INSERT OR UPDATE OF content ON messages
FOR EACH ROW
EXECUTE FUNCTION messages_content_tsvector_update();

-- Verify
-- SELECT COUNT(*) FROM messages WHERE content_tsvector IS NOT NULL;
