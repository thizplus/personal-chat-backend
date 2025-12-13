-- Migration: Add initial_message fields to user_friendships table
-- For Message Request feature (like Instagram/Facebook)

-- Add initial_message column (text for longer messages)
ALTER TABLE user_friendships
ADD COLUMN IF NOT EXISTS initial_message TEXT;

-- Add initial_message_at timestamp column
ALTER TABLE user_friendships
ADD COLUMN IF NOT EXISTS initial_message_at TIMESTAMP WITH TIME ZONE;

-- Add comment for documentation
COMMENT ON COLUMN user_friendships.initial_message IS 'Initial message sent with friend request (Message Request feature)';
COMMENT ON COLUMN user_friendships.initial_message_at IS 'Timestamp when initial message was sent';
