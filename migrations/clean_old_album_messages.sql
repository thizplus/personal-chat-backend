-- ลบ album messages แบบเก่า (ที่มี metadata.album_id)
-- วันที่: 2025-11-28

-- 1. แสดงจำนวน messages ที่จะลบก่อน
SELECT COUNT(*) as total_messages_to_delete
FROM messages
WHERE metadata->>'album_id' IS NOT NULL;

-- 2. แสดงตัวอย่าง messages ที่จะลบ (5 อันแรก)
SELECT
    id,
    message_type,
    metadata->>'album_id' as album_id,
    metadata->>'album_position' as position,
    created_at
FROM messages
WHERE metadata->>'album_id' IS NOT NULL
ORDER BY created_at DESC
LIMIT 5;

-- 3. ลบข้อมูล (uncomment เพื่อใช้งาน)
DELETE FROM messages
WHERE metadata->>'album_id' IS NOT NULL;

-- 4. ตรวจสอบผลลัพธ์
SELECT COUNT(*) as remaining_messages
FROM messages;
