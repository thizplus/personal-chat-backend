-- Fix Existing Group Roles
-- วันที่: 2025-11-28
-- ปัญหา: กลุ่มเก่าที่สร้างก่อนหน้านี้ creator มี role = 'member' แทนที่จะเป็น 'owner'

-- 1. อัพเดท creator ของทุกกลุ่มให้เป็น 'owner'
UPDATE conversation_members cm
SET role = 'owner'
FROM conversations c
WHERE cm.conversation_id = c.id
  AND c.type = 'group'
  AND cm.user_id = c.creator_id
  AND cm.role != 'owner';

-- 2. ตรวจสอบผลลัพธ์
SELECT
    c.id as conversation_id,
    c.title as group_name,
    cm.user_id as owner_user_id,
    cm.role as current_role,
    cm.is_admin
FROM conversations c
INNER JOIN conversation_members cm ON c.id = cm.conversation_id AND c.creator_id = cm.user_id
WHERE c.type = 'group'
ORDER BY c.created_at DESC;

-- 3. ตรวจสอบว่ายังมีกลุ่มไหนที่ creator ไม่ใช่ owner
SELECT
    c.id as conversation_id,
    c.title as group_name,
    c.creator_id,
    cm.user_id,
    cm.role,
    cm.is_admin
FROM conversations c
LEFT JOIN conversation_members cm ON c.id = cm.conversation_id AND c.creator_id = cm.user_id
WHERE c.type = 'group'
  AND (cm.role IS NULL OR cm.role != 'owner')
ORDER BY c.created_at DESC;

-- คาดว่าจะได้ 0 rows หลังรัน query 1
