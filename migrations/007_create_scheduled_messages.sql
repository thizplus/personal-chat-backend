-- migrations/007_create_scheduled_messages.sql
-- Create scheduled_messages table for scheduling messages to be sent in the future

CREATE TABLE IF NOT EXISTS scheduled_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message_type VARCHAR(20) NOT NULL,
    content TEXT,
    media_url TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,

    scheduled_at TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    sent_at TIMESTAMP WITH TIME ZONE,
    message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    error_reason TEXT,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_scheduled_messages_conversation ON scheduled_messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_scheduled_messages_sender ON scheduled_messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_scheduled_messages_scheduled_at ON scheduled_messages(scheduled_at);
CREATE INDEX IF NOT EXISTS idx_scheduled_messages_status ON scheduled_messages(status) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_scheduled_messages_pending ON scheduled_messages(scheduled_at, status) WHERE status = 'pending';

-- Add comments
COMMENT ON TABLE scheduled_messages IS 'Stores messages scheduled to be sent at a future time';
COMMENT ON COLUMN scheduled_messages.status IS 'Status: pending, sent, cancelled, failed';
COMMENT ON COLUMN scheduled_messages.message_id IS 'References the actual message after it has been sent';
