-- migrations/009_add_conversation_to_notes.sql
-- Add conversation_id to notes table for conversation-scoped notes

-- Add conversation_id column (nullable for backward compatibility)
ALTER TABLE notes
ADD COLUMN IF NOT EXISTS conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE;

-- Add index for filtering notes by conversation
CREATE INDEX IF NOT EXISTS idx_notes_conversation ON notes(user_id, conversation_id)
WHERE conversation_id IS NOT NULL;

-- Add index for global notes (conversation_id is NULL)
CREATE INDEX IF NOT EXISTS idx_notes_global ON notes(user_id)
WHERE conversation_id IS NULL;

-- Add index for conversation notes only
CREATE INDEX IF NOT EXISTS idx_notes_by_conversation ON notes(conversation_id)
WHERE conversation_id IS NOT NULL;

-- Update comment
COMMENT ON COLUMN notes.conversation_id IS 'Optional conversation link. NULL = global note, UUID = conversation-specific note';

-- Migration complete
-- Notes with conversation_id = NULL are "Personal/Global Notes"
-- Notes with conversation_id = UUID are "Conversation Notes"
