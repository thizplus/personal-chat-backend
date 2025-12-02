-- Migration: Create group_activities table
-- This table tracks all activities/changes in group conversations

CREATE TABLE IF NOT EXISTS group_activities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    actor_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_id UUID REFERENCES users(id) ON DELETE SET NULL,
    old_value JSONB,
    new_value JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_group_activities_conversation ON group_activities(conversation_id);
CREATE INDEX IF NOT EXISTS idx_group_activities_type ON group_activities(type);
CREATE INDEX IF NOT EXISTS idx_group_activities_created_at ON group_activities(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_group_activities_actor ON group_activities(actor_id);

-- Add comment to table
COMMENT ON TABLE group_activities IS 'Tracks all activities and changes in group conversations';
COMMENT ON COLUMN group_activities.type IS 'Activity type: group.created, group.name_changed, member.added, etc.';
COMMENT ON COLUMN group_activities.actor_id IS 'User who performed the action';
COMMENT ON COLUMN group_activities.target_id IS 'User who was affected by the action (optional)';
COMMENT ON COLUMN group_activities.old_value IS 'Previous value before the change (JSON format)';
COMMENT ON COLUMN group_activities.new_value IS 'New value after the change (JSON format)';
