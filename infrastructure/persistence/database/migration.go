// database/migration.go
package database

import (
	"log"

	"github.com/thizplus/gofiber-chat-api/domain/models"
	"gorm.io/gorm"
)

// RunMigration ทำการ migrate โมเดลทั้งหมดไปยังฐานข้อมูล
func RunMigration(db *gorm.DB) error {
	log.Println("กำลังทำ Auto Migration...")

	// ทำการ migrate โมเดลทั้งหมด
	// การเรียงลำดับมีความสำคัญ - ควรเริ่มจากตารางหลักก่อน แล้วค่อยไปตารางที่มี foreign key
	err := db.AutoMigrate(
		// โมเดลหลัก (ไม่มี FK ไปหาตารางอื่น)
		&models.User{},
		&models.StickerSet{},

		// โมเดลที่มี FK ไปหาตารางหลัก
		&models.Conversation{},
		&models.Sticker{},
		&models.UserFriendship{},
		&models.UserStickerSet{},
		&models.RefreshToken{},
		&models.TokenBlacklist{},
		&models.FileUpload{},

		// โมเดลที่มี FK ไปหาตารางที่มี FK
		&models.ConversationMember{},
		&models.Message{},
		&models.UserFavoriteSticker{},
		&models.UserRecentSticker{},

		// โมเดลที่ขึ้นอยู่กับตารางอื่นที่ซับซ้อน
		&models.MessageRead{},
		&models.MessageEditHistory{},
		&models.MessageDeleteHistory{},
		&models.MessageMention{},
		&models.ScheduledMessage{},
		&models.Note{},
		&models.GroupActivity{},
		&models.PinnedMessage{},
	)

	if err != nil {
		log.Printf("Auto Migration ล้มเหลว: %v", err)
		return err
	}

	// เพิ่ม foreign key constraints ที่ไม่ได้ถูกสร้างโดยอัตโนมัติ
	// ถ้าจำเป็น สามารถเพิ่ม Raw SQL queries ได้ที่นี่

	log.Println("Auto Migration สำเร็จ")
	return nil
}

// CreateIndices สร้าง indices เพื่อเพิ่มประสิทธิภาพในการค้นหา
func CreateIndices(db *gorm.DB) error {
	log.Println("กำลังสร้าง indices...")

	// สร้าง indices สำหรับตารางที่มีการค้นหาบ่อย
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id)").Error; err != nil {
		return err
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at)").Error; err != nil {
		return err
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status)").Error; err != nil {
		return err
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_messages_conversation_status ON messages(conversation_id, status)").Error; err != nil {
		return err
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_conversation_members_user_id ON conversation_members(user_id)").Error; err != nil {
		return err
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_conversation_members_is_hidden ON conversation_members(is_hidden)").Error; err != nil {
		return err
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_conversations_updated_at ON conversations(updated_at)").Error; err != nil {
		return err
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_messages_sender_id ON messages(sender_id)").Error; err != nil {
		return err
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_user_friendships_user_id ON user_friendships(user_id)").Error; err != nil {
		return err
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_user_friendships_friend_id ON user_friendships(friend_id)").Error; err != nil {
		return err
	}


	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)").Error; err != nil {
		return err
	}

	// Indices สำหรับ message_mentions (Task 1.2)
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_mentions_user_time ON message_mentions(mentioned_user_id, created_at DESC)").Error; err != nil {
		return err
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_mentions_message ON message_mentions(message_id)").Error; err != nil {
		return err
	}

	// Add unique constraint for message_mentions
	if err := db.Exec("ALTER TABLE message_mentions DROP CONSTRAINT IF EXISTS unique_message_mention").Error; err != nil {
		return err
	}
	if err := db.Exec("ALTER TABLE message_mentions ADD CONSTRAINT unique_message_mention UNIQUE (message_id, mentioned_user_id)").Error; err != nil {
		return err
	}

	log.Println("สร้าง indices สำเร็จ")
	return nil
}

// SetupFullTextSearch ตั้งค่า full-text search สำหรับ messages table
func SetupFullTextSearch(db *gorm.DB) error {
	log.Println("กำลังตั้งค่า full-text search...")

	// Step 1: Add content_tsvector column if not exists
	if err := db.Exec(`
		ALTER TABLE messages
		ADD COLUMN IF NOT EXISTS content_tsvector tsvector
	`).Error; err != nil {
		return err
	}

	// Step 2: Populate existing data
	if err := db.Exec(`
		UPDATE messages
		SET content_tsvector = to_tsvector('english', COALESCE(content, ''))
		WHERE content_tsvector IS NULL
	`).Error; err != nil {
		return err
	}

	// Step 3: Create GIN index
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_messages_content_tsvector
		ON messages USING GIN (content_tsvector)
	`).Error; err != nil {
		return err
	}

	// Step 4: Create trigger function
	if err := db.Exec(`
		CREATE OR REPLACE FUNCTION messages_content_tsvector_update()
		RETURNS trigger AS $$
		BEGIN
		  NEW.content_tsvector := to_tsvector('english', COALESCE(NEW.content, ''));
		  RETURN NEW;
		END;
		$$ LANGUAGE plpgsql
	`).Error; err != nil {
		return err
	}

	// Step 5: Create trigger
	if err := db.Exec(`DROP TRIGGER IF EXISTS tsvector_update ON messages`).Error; err != nil {
		return err
	}
	if err := db.Exec(`
		CREATE TRIGGER tsvector_update
		BEFORE INSERT OR UPDATE OF content ON messages
		FOR EACH ROW
		EXECUTE FUNCTION messages_content_tsvector_update()
	`).Error; err != nil {
		return err
	}

	log.Println("ตั้งค่า full-text search สำเร็จ")
	return nil
}

// SetupDatabase ตั้งค่าฐานข้อมูลทั้งหมด
func SetupDatabase(db *gorm.DB) error {
	// ทำ migration
	if err := RunMigration(db); err != nil {
		return err
	}

	// สร้าง indices
	if err := CreateIndices(db); err != nil {
		return err
	}

	// ตั้งค่า full-text search
	if err := SetupFullTextSearch(db); err != nil {
		return err
	}

	return nil
}
