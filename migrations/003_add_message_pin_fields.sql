-- migrations/003_add_message_pin_fields.sql
-- Add pin fields to messages table

ALTER TABLE messages
ADD COLUMN IF NOT EXISTS is_pinned BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS pinned_by UUID REFERENCES users(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS pinned_at TIMESTAMP WITH TIME ZONE;

-- Create index for better query performance
CREATE INDEX IF NOT EXISTS idx_messages_pinned ON messages(conversation_id, is_pinned) WHERE is_pinned = TRUE;
CREATE INDEX IF NOT EXISTS idx_messages_pinned_by ON messages(pinned_by) WHERE pinned_by IS NOT NULL;
