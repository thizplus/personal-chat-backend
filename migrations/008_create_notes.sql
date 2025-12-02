-- migrations/008_create_notes.sql
-- Create notes table for personal note-taking

CREATE TABLE IF NOT EXISTS notes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255),
    content TEXT,
    tags JSONB DEFAULT '[]'::jsonb,
    is_pinned BOOLEAN DEFAULT FALSE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_notes_user ON notes(user_id);
CREATE INDEX IF NOT EXISTS idx_notes_pinned ON notes(user_id, is_pinned) WHERE is_pinned = TRUE;
CREATE INDEX IF NOT EXISTS idx_notes_tags ON notes USING gin(tags);
CREATE INDEX IF NOT EXISTS idx_notes_created_at ON notes(created_at DESC);

-- Add full-text search for note content
ALTER TABLE notes ADD COLUMN IF NOT EXISTS content_tsvector tsvector;

CREATE INDEX IF NOT EXISTS idx_notes_fulltext ON notes USING gin(content_tsvector);

-- Create trigger to update search vector
CREATE OR REPLACE FUNCTION notes_tsvector_trigger() RETURNS trigger AS $$
BEGIN
  NEW.content_tsvector :=
    setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(NEW.content, '')), 'B');
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE TRIGGER tsvectorupdate
  BEFORE INSERT OR UPDATE OF title, content ON notes
  FOR EACH ROW EXECUTE FUNCTION notes_tsvector_trigger();

-- Add comments
COMMENT ON TABLE notes IS 'Stores personal notes created by users';
COMMENT ON COLUMN notes.tags IS 'Array of tags: ["tag1", "tag2"]';
COMMENT ON COLUMN notes.is_pinned IS 'Whether this note is pinned to the top';
