-- migrations/004_add_message_fulltext_search.sql
-- Add full-text search capability to messages table

-- เพิ่ม column สำหรับ full-text search (tsvector)
ALTER TABLE messages
ADD COLUMN IF NOT EXISTS content_tsvector tsvector;

-- สร้าง GIN index สำหรับ full-text search
CREATE INDEX IF NOT EXISTS idx_messages_fulltext ON messages USING gin(content_tsvector);

-- สร้าง function สำหรับ update tsvector โดยอัตโนมัติ
CREATE OR REPLACE FUNCTION messages_tsvector_trigger() RETURNS trigger AS $$
BEGIN
  NEW.content_tsvector := to_tsvector('english', COALESCE(NEW.content, ''));
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

-- สร้าง trigger ที่จะ update tsvector ทุกครั้งที่มีการ insert หรือ update
DROP TRIGGER IF EXISTS tsvectorupdate ON messages;
CREATE TRIGGER tsvectorupdate
  BEFORE INSERT OR UPDATE OF content
  ON messages
  FOR EACH ROW
  EXECUTE FUNCTION messages_tsvector_trigger();

-- Update tsvector สำหรับข้อมูลที่มีอยู่แล้ว
UPDATE messages SET content_tsvector = to_tsvector('english', COALESCE(content, ''))
WHERE content IS NOT NULL AND content != '';
