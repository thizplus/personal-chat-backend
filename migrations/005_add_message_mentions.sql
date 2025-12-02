-- migrations/005_add_message_mentions.sql
-- Add mentions field to messages table for @username functionality

ALTER TABLE messages
ADD COLUMN IF NOT EXISTS mentions JSONB;

-- Create index for better query performance when filtering by mentioned users
CREATE INDEX IF NOT EXISTS idx_messages_mentions ON messages USING gin(mentions);

-- Add comment to explain the format
COMMENT ON COLUMN messages.mentions IS 'Array of mention objects: [{"user_id": "uuid", "start_index": 0, "length": 10}]';
