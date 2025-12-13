-- migrations/014_create_pinned_messages_table.sql
-- Create pinned_messages table for Personal/Public pin feature

-- Create table for tracking pinned messages (supports personal and public pins)
CREATE TABLE IF NOT EXISTS pinned_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    pin_type VARCHAR(20) NOT NULL CHECK (pin_type IN ('personal', 'public')),
    pinned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Unique Constraint: prevent duplicate pins by same user with same type
    CONSTRAINT unique_pinned_message_user_type UNIQUE (message_id, user_id, pin_type)
);

-- Create Indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_pinned_messages_conversation_id ON pinned_messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_pinned_messages_user_id ON pinned_messages(user_id);
CREATE INDEX IF NOT EXISTS idx_pinned_messages_pin_type ON pinned_messages(pin_type);
CREATE INDEX IF NOT EXISTS idx_pinned_messages_message_id ON pinned_messages(message_id);
CREATE INDEX IF NOT EXISTS idx_pinned_messages_pinned_at ON pinned_messages(pinned_at DESC);

-- Composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_pinned_messages_conv_type ON pinned_messages(conversation_id, pin_type);
CREATE INDEX IF NOT EXISTS idx_pinned_messages_conv_user ON pinned_messages(conversation_id, user_id);

-- Add comments for documentation
COMMENT ON TABLE pinned_messages IS 'Stores pinned messages for conversations (personal and public pins)';
COMMENT ON COLUMN pinned_messages.pin_type IS 'Type of pin: personal (visible only to user) or public (visible to all members)';
COMMENT ON COLUMN pinned_messages.user_id IS 'User who pinned the message';
