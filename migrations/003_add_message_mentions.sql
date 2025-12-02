-- Migration: Add message_mentions table
-- Date: 2025-11-29
-- Purpose: Enable efficient querying of user mentions in messages

-- Create message_mentions table
CREATE TABLE IF NOT EXISTS message_mentions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    mentioned_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    start_index INTEGER,
    length INTEGER,
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT unique_message_mention UNIQUE(message_id, mentioned_user_id)
);

-- Indexes for performance
CREATE INDEX idx_mentions_user_time ON message_mentions(mentioned_user_id, created_at DESC);
CREATE INDEX idx_mentions_message ON message_mentions(message_id);

-- Comments for documentation
COMMENT ON TABLE message_mentions IS 'Stores user mentions in messages for efficient querying';
COMMENT ON COLUMN message_mentions.message_id IS 'Reference to the message containing the mention';
COMMENT ON COLUMN message_mentions.mentioned_user_id IS 'User who was mentioned in the message';
COMMENT ON COLUMN message_mentions.start_index IS 'Character position where mention starts in message content';
COMMENT ON COLUMN message_mentions.length IS 'Length of mention text (e.g., @username)';

-- Display success message
DO $$
BEGIN
    RAISE NOTICE '✅ Migration 003: message_mentions table created successfully';
END $$;
