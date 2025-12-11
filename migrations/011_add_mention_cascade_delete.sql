-- Migration: Add CASCADE DELETE for message mentions
-- Date: 2025-12-03
-- Description: Ensure mentions are deleted automatically when messages are deleted

-- Drop existing foreign key constraint if exists
ALTER TABLE message_mentions
DROP CONSTRAINT IF EXISTS message_mentions_message_id_fkey;

-- Add foreign key constraint with CASCADE DELETE
ALTER TABLE message_mentions
ADD CONSTRAINT message_mentions_message_id_fkey
FOREIGN KEY (message_id)
REFERENCES messages(id)
ON DELETE CASCADE;

-- Add comment for documentation
COMMENT ON CONSTRAINT message_mentions_message_id_fkey ON message_mentions
IS 'Cascade delete mentions when message is deleted';

-- Verify the constraint was created
SELECT
    tc.constraint_name,
    tc.table_name,
    kcu.column_name,
    ccu.table_name AS foreign_table_name,
    ccu.column_name AS foreign_column_name,
    rc.delete_rule
FROM information_schema.table_constraints AS tc
JOIN information_schema.key_column_usage AS kcu
  ON tc.constraint_name = kcu.constraint_name
  AND tc.table_schema = kcu.table_schema
JOIN information_schema.constraint_column_usage AS ccu
  ON ccu.constraint_name = tc.constraint_name
  AND ccu.table_schema = tc.table_schema
JOIN information_schema.referential_constraints AS rc
  ON tc.constraint_name = rc.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY'
  AND tc.table_name = 'message_mentions'
  AND kcu.column_name = 'message_id';
