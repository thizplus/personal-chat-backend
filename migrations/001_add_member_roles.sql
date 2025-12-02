-- Migration: Add role field to conversation_members
-- This will be auto-added by GORM, but use this to set initial values

-- Update existing data: set role based on is_admin
UPDATE conversation_members
SET role = CASE
    WHEN is_admin = true THEN 'admin'
    ELSE 'member'
END
WHERE role IS NULL OR role = '';

-- Set owners (conversation creators) as 'owner'
UPDATE conversation_members cm
SET role = 'owner'
FROM conversations c
WHERE cm.conversation_id = c.id
  AND cm.user_id = c.creator_id;

-- Create index (GORM should do this, but just in case)
CREATE INDEX IF NOT EXISTS idx_conversation_members_role ON conversation_members(role);
