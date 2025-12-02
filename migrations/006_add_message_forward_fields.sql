-- migrations/006_add_message_forward_fields.sql
-- Add forward fields to messages table for forwarding messages functionality

ALTER TABLE messages
ADD COLUMN IF NOT EXISTS is_forwarded BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS forwarded_from JSONB;

-- Create index for better query performance when filtering forwarded messages
CREATE INDEX IF NOT EXISTS idx_messages_forwarded ON messages(is_forwarded) WHERE is_forwarded = TRUE;
CREATE INDEX IF NOT EXISTS idx_messages_forwarded_from ON messages USING gin(forwarded_from) WHERE forwarded_from IS NOT NULL;

-- Add comment to explain the format
COMMENT ON COLUMN messages.is_forwarded IS 'Indicates if this message was forwarded from another message';
COMMENT ON COLUMN messages.forwarded_from IS 'Original message info: {"message_id": "uuid", "sender_id": "uuid", "conversation_id": "uuid", "original_timestamp": "..."}';
