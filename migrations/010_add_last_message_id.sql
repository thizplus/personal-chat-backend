-- Migration: Add last_message_id to conversations table
-- Date: 2025-12-03
-- Description: Add last_message_id column to track the last message in a conversation
--              This enables efficient mention detection in conversation list

-- Add last_message_id column
ALTER TABLE conversations
ADD COLUMN IF NOT EXISTS last_message_id UUID;

-- Add comment for documentation
COMMENT ON COLUMN conversations.last_message_id IS 'Reference to the last message in the conversation';

-- Update existing conversations with their last message ID
UPDATE conversations c
SET last_message_id = (
    SELECT m.id
    FROM messages m
    WHERE m.conversation_id = c.id
    ORDER BY m.created_at DESC
    LIMIT 1
)
WHERE c.last_message_id IS NULL;

-- Add index for better query performance
CREATE INDEX IF NOT EXISTS idx_conversations_last_message_id
ON conversations(last_message_id);

-- Note: We intentionally do NOT add a foreign key constraint here
-- to avoid circular dependency issues and maintain flexibility
-- The relationship is maintained through application logic
