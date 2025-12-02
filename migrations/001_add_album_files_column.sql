-- Migration: Add album_files column to messages table
-- Date: 2025-11-28
-- Description: เพิ่ม column album_files สำหรับ album messages และ migrate ข้อมูลเก่า

-- 1. เพิ่ม column album_files (ถ้ายังไม่มี)
-- GORM AutoMigrate จะทำให้อัตโนมัติ แต่ถ้าต้องการ manual migration:
ALTER TABLE messages
ADD COLUMN IF NOT EXISTS album_files JSONB DEFAULT NULL;

-- 2. สร้าง index สำหรับ album_files
CREATE INDEX IF NOT EXISTS idx_messages_album_files ON messages USING GIN (album_files);

-- 3. Migrate ข้อมูลเก่า: รวม album messages เก่าเป็น message เดียว
-- (ใช้เฉพาะถ้ามีข้อมูล album แบบเก่าในระบบ)

-- Step 3.1: สร้าง temporary table เพื่อเก็บ album groups
CREATE TEMP TABLE IF NOT EXISTS temp_album_groups AS
SELECT
  metadata->>'album_id' as album_id,
  MIN(id) as first_message_id,
  MIN(created_at) as created_at,
  MIN(conversation_id) as conversation_id,
  MIN(sender_id) as sender_id,
  COUNT(*) as total_files,
  MAX(CASE WHEN metadata->>'album_caption' IS NOT NULL THEN metadata->>'album_caption' ELSE content END) as caption,
  jsonb_agg(
    jsonb_build_object(
      'id', id::text,
      'file_type', message_type,
      'media_url', media_url,
      'media_thumbnail_url', media_thumbnail_url,
      'position', (metadata->>'album_position')::int,
      'file_name', metadata->>'file_name',
      'file_size', (metadata->>'file_size')::bigint,
      'file_type', metadata->>'file_type'
    ) ORDER BY (metadata->>'album_position')::int
  ) as album_files_data
FROM messages
WHERE metadata->>'album_id' IS NOT NULL
  AND metadata->>'album_position' IS NOT NULL
GROUP BY metadata->>'album_id';

-- Step 3.2: Update message แรกของแต่ละ album ให้เป็น type "album"
UPDATE messages m
SET
  message_type = 'album',
  content = ag.caption,
  album_files = ag.album_files_data,
  media_url = NULL,  -- Clear single media fields
  media_thumbnail_url = NULL,
  metadata = COALESCE(metadata, '{}'::jsonb) - 'album_id' - 'album_position' - 'album_total' - 'album_caption' || jsonb_build_object('album_total', ag.total_files),
  updated_at = NOW()
FROM temp_album_groups ag
WHERE m.id = ag.first_message_id;

-- Step 3.3: ลบ messages ที่เหลือ (position > 0)
-- ⚠️ คำเตือน: คำสั่งนี้จะลบข้อมูลจริง
-- ตรวจสอบให้แน่ใจก่อนว่า step 3.2 ทำงานถูกต้องแล้ว

-- Uncomment เพื่อใช้งาน:
-- DELETE FROM messages
-- WHERE id IN (
--   SELECT m.id
--   FROM messages m
--   WHERE m.metadata->>'album_id' IS NOT NULL
--     AND m.metadata->>'album_position' IS NOT NULL
--     AND (m.metadata->>'album_position')::int > 0
-- );

-- Step 3.4: ลบ temporary table
DROP TABLE IF EXISTS temp_album_groups;

-- 4. ตรวจสอบผลลัพธ์
-- SELECT id, message_type, content, album_files, metadata
-- FROM messages
-- WHERE message_type = 'album'
-- LIMIT 5;

-- หมายเหตุ:
-- 1. GORM AutoMigrate จะเพิ่ม column album_files ให้อัตโนมัติ
-- 2. Migration script นี้ใช้สำหรับ migrate ข้อมูลเก่าเท่านั้น
-- 3. ถ้าไม่มีข้อมูล album แบบเก่า ให้ข้าม step 3
-- 4. ควร backup database ก่อนรัน migration
-- 5. ทดสอบใน development environment ก่อน
