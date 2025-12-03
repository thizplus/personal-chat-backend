// infrastructure/persistence/postgres/message_repository.go
package postgres

import (
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/repository"
	"gorm.io/gorm"
)

type messageRepository struct {
	db *gorm.DB
}

// NewMessageRepository สร้าง repository ใหม่
func NewMessageRepository(db *gorm.DB) repository.MessageRepository {
	return &messageRepository{
		db: db,
	}
}

// GetByID ดึงข้อมูลข้อความตาม ID
func (r *messageRepository) GetByID(id uuid.UUID) (*models.Message, error) {
	var message models.Message
	if err := r.db.First(&message, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &message, nil
}

// GetMessagesByConversationID ดึงข้อความทั้งหมดในการสนทนา
func (r *messageRepository) GetMessagesByConversationID(conversationID uuid.UUID, limit, offset int) ([]*models.Message, int64, error) {
	var count int64
	if err := r.db.Model(&models.Message{}).Where("conversation_id = ?", conversationID).Count(&count).Error; err != nil {
		return nil, 0, err
	}

	var messages []*models.Message
	// Fetch ข้อความล่าสุดก่อน (DESC) แล้วค่อย reverse เป็น ASC
	if err := r.db.Where("conversation_id = ?", conversationID).
		Order("created_at DESC"). // ดึงข้อความล่าสุดก่อน
		Limit(limit).
		Offset(offset).
		Find(&messages).Error; err != nil {
		return nil, 0, err
	}

	// Reverse array เพื่อให้เป็น ASC (เก่า → ใหม่) ก่อน return
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, count, nil
}

// Create สร้างข้อความใหม่
func (r *messageRepository) Create(message *models.Message) error {
	return r.db.Create(message).Error
}

// BulkCreate สร้างหลายข้อความพร้อมกัน (สำหรับ Album/Bulk Upload)
func (r *messageRepository) BulkCreate(messages []*models.Message) error {
	return r.db.CreateInBatches(messages, 100).Error
}

// GetMessagesByAlbumID ดึงข้อความทั้งหมดในอัลบั้มเดียวกัน
func (r *messageRepository) GetMessagesByAlbumID(albumID string) ([]*models.Message, error) {
	var messages []*models.Message
	err := r.db.
		Where("metadata->>'album_id' = ?", albumID).
		Order("(metadata->>'album_position')::int ASC").
		Find(&messages).Error
	return messages, err
}

// Update อัพเดตข้อความ
func (r *messageRepository) Update(message *models.Message) error {
	// ใช้ Updates แทน Save เพื่อหลีกเลี่ยง nil pointer ใน AlbumFiles
	// Updates จะอัปเดตเฉพาะ non-zero fields
	return r.db.Model(&models.Message{}).Where("id = ?", message.ID).Updates(message).Error
}

// UpdateFields อัพเดตเฉพาะ fields ที่ระบุ
func (r *messageRepository) UpdateFields(messageID uuid.UUID, updates map[string]interface{}) error {
	return r.db.Model(&models.Message{}).Where("id = ?", messageID).Updates(updates).Error
}

// Delete ลบข้อความ (soft delete)
func (r *messageRepository) Delete(id uuid.UUID) error {
	result := r.db.Model(&models.Message{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_deleted":          true,
			"content":             nil,
			"media_url":           nil,
			"media_thumbnail_url": nil,
			"metadata":            "{}",
			"updated_at":          time.Now(),
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("message not found")
	}
	return nil
}

// CreateEditHistory บันทึกประวัติการแก้ไขข้อความ
func (r *messageRepository) CreateEditHistory(history *models.MessageEditHistory) error {
	return r.db.Create(history).Error
}

// GetEditHistory ดึงประวัติการแก้ไขข้อความ
func (r *messageRepository) GetEditHistory(messageID uuid.UUID) ([]*models.MessageEditHistory, error) {
	var history []*models.MessageEditHistory
	if err := r.db.Where("message_id = ?", messageID).
		Order("edited_at DESC").
		Find(&history).Error; err != nil {
		return nil, err
	}
	return history, nil
}

// CreateDeleteHistory บันทึกประวัติการลบข้อความ
func (r *messageRepository) CreateDeleteHistory(history *models.MessageDeleteHistory) error {
	return r.db.Create(history).Error
}

// GetDeleteHistory ดึงประวัติการลบข้อความ
func (r *messageRepository) GetDeleteHistory(messageID uuid.UUID) ([]*models.MessageDeleteHistory, error) {
	var history []*models.MessageDeleteHistory
	if err := r.db.Where("message_id = ?", messageID).
		Order("deleted_at DESC").
		Find(&history).Error; err != nil {
		return nil, err
	}
	return history, nil
}

// MarkAsRead ทำเครื่องหมายว่าข้อความถูกอ่านแล้ว
func (r *messageRepository) MarkAsRead(messageID, userID uuid.UUID, readAt time.Time) error {
	// ตรวจสอบว่ามีการอ่านแล้วหรือไม่
	var count int64
	if err := r.db.Model(&models.MessageRead{}).
		Where("message_id = ? AND user_id = ?", messageID, userID).
		Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return nil // มีการอ่านแล้ว ไม่ต้องทำอะไร
	}

	// สร้างบันทึกการอ่าน
	read := models.MessageRead{
		ID:        uuid.New(),
		MessageID: messageID,
		UserID:    userID,
		ReadAt:    readAt,
	}

	return r.db.Create(&read).Error
}

// GetReads ดึงรายการการอ่านข้อความ
func (r *messageRepository) GetReads(messageID uuid.UUID) ([]*models.MessageRead, error) {
	var reads []*models.MessageRead
	if err := r.db.Where("message_id = ?", messageID).
		Order("read_at ASC").
		Find(&reads).Error; err != nil {
		return nil, err
	}
	return reads, nil
}

// IsMessageRead ตรวจสอบว่าข้อความถูกอ่านโดยผู้ใช้แล้วหรือไม่
func (r *messageRepository) IsMessageRead(messageID, userID uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.Model(&models.MessageRead{}).
		Where("message_id = ? AND user_id = ?", messageID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// MarkAllAsRead ทำเครื่องหมายว่าข้อความทั้งหมดในการสนทนาถูกอ่านแล้ว
func (r *messageRepository) MarkAllAsRead(conversationID, userID uuid.UUID, readAt time.Time) error {
	// ดึงรายการข้อความที่ยังไม่ได้อ่าน
	rows, err := r.db.Raw(`
		SELECT m.id 
		FROM messages m 
		LEFT JOIN message_reads mr ON m.id = mr.message_id AND mr.user_id = ? 
		WHERE m.conversation_id = ? AND m.sender_id != ? AND mr.id IS NULL AND m.is_deleted = false
	`, userID, conversationID, userID).Rows()

	if err != nil {
		return err
	}
	defer rows.Close()

	var messageIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return err
		}
		messageIDs = append(messageIDs, id)
	}

	if len(messageIDs) == 0 {
		return nil // ไม่มีข้อความที่ต้องมาร์ค
	}

	// สร้างบันทึกการอ่านเป็นชุด
	reads := make([]models.MessageRead, 0, len(messageIDs))
	for _, messageID := range messageIDs {
		reads = append(reads, models.MessageRead{
			ID:        uuid.New(),
			MessageID: messageID,
			UserID:    userID,
			ReadAt:    readAt,
		})
	}

	return r.db.CreateInBatches(reads, 100).Error
}

// IsSender ตรวจสอบว่าผู้ใช้เป็นผู้ส่งข้อความหรือไม่
func (r *messageRepository) IsSender(messageID, userID uuid.UUID) (bool, error) {
	var message models.Message
	err := r.db.Select("sender_id").First(&message, "id = ?", messageID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, errors.New("message not found")
		}
		return false, err
	}
	return message.SenderID != nil && *message.SenderID == userID, nil
}

// IsConversationAdmin ตรวจสอบว่าผู้ใช้เป็นแอดมินของการสนทนาหรือไม่
func (r *messageRepository) IsConversationAdmin(conversationID, userID uuid.UUID) (bool, error) {
	var member models.ConversationMember
	err := r.db.Select("is_admin").First(&member, "conversation_id = ? AND user_id = ?", conversationID, userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return member.IsAdmin, nil
}

// UpdateConversationLastMessage อัพเดตข้อความล่าสุดในการสนทนา
func (r *messageRepository) UpdateConversationLastMessage(conversationID uuid.UUID, lastMessageText string, lastMessageAt time.Time, messageID uuid.UUID) error {
	return r.db.Model(&models.Conversation{}).
		Where("id = ?", conversationID).
		Updates(map[string]interface{}{
			"last_message_id":   messageID,
			"last_message_text": lastMessageText,
			"last_message_at":   lastMessageAt,
			"updated_at":        time.Now(),
		}).Error
}

// GetLastMessageByConversation ดึงข้อความล่าสุดของการสนทนา
func (r *messageRepository) GetLastMessageByConversation(conversationID uuid.UUID) (*models.Message, error) {
	var message models.Message
	err := r.db.Where("conversation_id = ?", conversationID).
		Order("created_at DESC").
		First(&message).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &message, nil
}

// GetLastNonDeletedMessageByConversation ดึงข้อความล่าสุดที่ไม่ถูกลบของการสนทนา
func (r *messageRepository) GetLastNonDeletedMessageByConversation(conversationID uuid.UUID) (*models.Message, error) {
	var message models.Message
	err := r.db.Where("conversation_id = ? AND is_deleted = ?", conversationID, false).
		Order("created_at DESC").
		First(&message).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &message, nil
}

// GetMessagesBefore ดึงข้อความที่เก่ากว่า ID ที่ระบุ
func (r *messageRepository) GetMessagesBefore(conversationID, messageID uuid.UUID, limit int) ([]*models.Message, error) {
	var targetMessage models.Message
	if err := r.db.First(&targetMessage, "id = ?", messageID).Error; err != nil {
		return nil, err
	}

	var messages []*models.Message

	// ดึงข้อความที่เก่ากว่าข้อความเป้าหมาย โดยเรียงจากใหม่ไปเก่า (DESC) ก่อน
	// ใช้ composite cursor (created_at + id) เพื่อป้องกัน overlap เมื่อมี messages ที่มี timestamp เดียวกัน
	if err := r.db.Where("conversation_id = ? AND (created_at < ? OR (created_at = ? AND id < ?))",
		conversationID, targetMessage.CreatedAt, targetMessage.CreatedAt, messageID).
		Order("created_at DESC, id DESC"). // Query DESC ก่อน
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}

	// ✅ Reverse เป็น ASC (เก่า → ใหม่) ก่อน return
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// GetMessagesAfter ดึงข้อความที่ใหม่กว่า ID ที่ระบุ
func (r *messageRepository) GetMessagesAfter(conversationID, messageID uuid.UUID, limit int) ([]*models.Message, error) {
	var targetMessage models.Message
	if err := r.db.First(&targetMessage, "id = ?", messageID).Error; err != nil {
		return nil, err
	}

	var messages []*models.Message

	// ดึงข้อความที่ใหม่กว่าข้อความเป้าหมาย โดยเรียงจากเก่าไปใหม่ (ASC)
	// ใช้ composite cursor (created_at + id) เพื่อป้องกัน overlap เมื่อมี messages ที่มี timestamp เดียวกัน
	if err := r.db.Where("conversation_id = ? AND (created_at > ? OR (created_at = ? AND id > ?))",
		conversationID, targetMessage.CreatedAt, targetMessage.CreatedAt, messageID).
		Order("created_at ASC, id ASC"). // ✅ Query ASC (เก่า → ใหม่)
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}

	return messages, nil
}

// CountAllMessages นับจำนวนข้อความทั้งหมดในการสนทนา
func (r *messageRepository) CountAllMessages(conversationID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&models.Message{}).Where("conversation_id = ?", conversationID).Count(&count).Error
	return count, err
}

// infrastructure/persistence/postgres/message_repository.go
func (r *messageRepository) GetMessagesAfterTime(conversationID uuid.UUID, afterTime time.Time, excludeUserID uuid.UUID) ([]*models.Message, error) {
	var messages []*models.Message

	// ดึงข้อความที่สร้างหลังเวลาที่กำหนด ไม่ใช่ของผู้ใช้ที่กำหนด และไม่ถูกลบ
	err := r.db.Where("conversation_id = ? AND created_at > ? AND sender_id != ? AND is_deleted = ?",
		conversationID, afterTime, excludeUserID, false).
		Find(&messages).Error

	return messages, err
}

func (r *messageRepository) GetAllUnreadMessages(conversationID uuid.UUID, excludeUserID uuid.UUID) ([]*models.Message, error) {
	var messages []*models.Message

	// ดึงข้อความทั้งหมดในการสนทนาที่ไม่ใช่ของผู้ใช้ที่กำหนด และไม่ถูกลบ
	err := r.db.Where("conversation_id = ? AND sender_id != ? AND is_deleted = ?",
		conversationID, excludeUserID, false).
		Find(&messages).Error

	return messages, err
}

// GetMessageTypeSummary นับจำนวนข้อความแต่ละประเภทในการสนทนา
func (r *messageRepository) GetMessageTypeSummary(conversationID uuid.UUID) (map[string]int64, error) {
	type Result struct {
		MessageType string
		Count       int64
	}

	// 1. นับ single media messages (แบบเดิม)
	var singleMediaResults []Result
	err := r.db.Model(&models.Message{}).
		Select("message_type, COUNT(*) as count").
		Where("conversation_id = ? AND is_deleted = ? AND message_type IN (?)",
			conversationID,
			false,
			[]string{"image", "video", "file"}).
		Group("message_type").
		Find(&singleMediaResults).Error

	if err != nil {
		return nil, err
	}

	// สร้าง summary map
	summary := make(map[string]int64)
	for _, result := range singleMediaResults {
		summary[result.MessageType] = result.Count
	}

	// 2. นับ files ใน albums โดยใช้ JSONB functions
	type AlbumFileCount struct {
		FileType string
		Count    int64
	}

	var albumFileCounts []AlbumFileCount
	err = r.db.Raw(`
		SELECT
			file->>'file_type' as file_type,
			COUNT(*) as count
		FROM messages,
			 jsonb_array_elements(album_files) as file
		WHERE conversation_id = ?
		  AND is_deleted = false
		  AND message_type = 'album'
		  AND album_files IS NOT NULL
		GROUP BY file->>'file_type'
	`, conversationID).Scan(&albumFileCounts).Error

	if err != nil {
		return nil, err
	}

	// รวมจำนวนจาก albums เข้าไปใน summary
	for _, albumCount := range albumFileCounts {
		summary[albumCount.FileType] += albumCount.Count
	}

	return summary, nil
}

// CountMessagesWithLinks นับจำนวนข้อความที่มีลิงก์ในการสนทนา
func (r *messageRepository) CountMessagesWithLinks(conversationID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&models.Message{}).
		Where("conversation_id = ? AND is_deleted = ? AND metadata->>'links' IS NOT NULL AND metadata->>'links' != '[]'",
			conversationID,
			false).
		Count(&count).Error

	return count, err
}

// GetMediaByType ดึงรายละเอียดข้อความตามประเภทพร้อม pagination
func (r *messageRepository) GetMediaByType(conversationID uuid.UUID, messageType string, limit, offset int) ([]*models.Message, int64, error) {
	var messages []*models.Message
	var total int64

	// กรณี link (ไม่เปลี่ยนแปลง)
	if messageType == "link" {
		query := r.db.Model(&models.Message{}).
			Where("conversation_id = ? AND is_deleted = ?", conversationID, false).
			Where("metadata->>'links' IS NOT NULL AND metadata->>'links' != '[]'")

		if err := query.Count(&total).Error; err != nil {
			return nil, 0, err
		}

		if err := query.Order("created_at DESC").
			Limit(limit).
			Offset(offset).
			Find(&messages).Error; err != nil {
			return nil, 0, err
		}

		return messages, total, nil
	}

	// สำหรับ image/video/file: ดึงทั้ง single messages และ album messages

	// 1. นับจำนวนรวม (single + album files)
	var singleCount int64
	err := r.db.Model(&models.Message{}).
		Where("conversation_id = ? AND is_deleted = ? AND message_type = ?",
			conversationID, false, messageType).
		Count(&singleCount).Error
	if err != nil {
		return nil, 0, err
	}

	var albumFileCount int64
	err = r.db.Raw(`
		SELECT COUNT(*)
		FROM messages,
			 jsonb_array_elements(album_files) as file
		WHERE conversation_id = ?
		  AND is_deleted = false
		  AND message_type = 'album'
		  AND album_files IS NOT NULL
		  AND file->>'file_type' = ?
	`, conversationID, messageType).Scan(&albumFileCount).Error
	if err != nil {
		return nil, 0, err
	}

	total = singleCount + albumFileCount

	// 2. ดึง messages ทั้ง 2 ประเภท
	// Query for single media messages
	err = r.db.Model(&models.Message{}).
		Where("conversation_id = ? AND is_deleted = ? AND message_type = ?",
			conversationID, false, messageType).
		Order("created_at DESC").
		Find(&messages).Error
	if err != nil {
		return nil, 0, err
	}

	// Query for album messages that contain this media type
	var albumMessages []*models.Message
	err = r.db.Raw(`
		SELECT DISTINCT m.*
		FROM messages m,
			 jsonb_array_elements(m.album_files) as file
		WHERE m.conversation_id = ?
		  AND m.is_deleted = false
		  AND m.message_type = 'album'
		  AND m.album_files IS NOT NULL
		  AND file->>'file_type' = ?
		ORDER BY m.created_at DESC
	`, conversationID, messageType).Scan(&albumMessages).Error
	if err != nil {
		return nil, 0, err
	}

	// รวม messages ทั้ง 2 ประเภท
	messages = append(messages, albumMessages...)

	// เรียงตาม created_at DESC
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].CreatedAt.After(messages[j].CreatedAt)
	})

	// Apply pagination
	if offset >= len(messages) {
		return []*models.Message{}, total, nil
	}

	end := offset + limit
	if end > len(messages) {
		end = len(messages)
	}

	messages = messages[offset:end]

	return messages, total, nil
}

// PinMessage ปักหมุดข้อความ
func (r *messageRepository) PinMessage(messageID, userID uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&models.Message{}).
		Where("id = ?", messageID).
		Updates(map[string]interface{}{
			"is_pinned": true,
			"pinned_by": userID,
			"pinned_at": now,
		}).Error
}

// UnpinMessage ยกเลิกการปักหมุดข้อความ
func (r *messageRepository) UnpinMessage(messageID uuid.UUID) error {
	return r.db.Model(&models.Message{}).
		Where("id = ?", messageID).
		Updates(map[string]interface{}{
			"is_pinned": false,
			"pinned_by": nil,
			"pinned_at": nil,
		}).Error
}

// GetPinnedMessages ดึงรายการข้อความที่ปักหมุดในการสนทนา
func (r *messageRepository) GetPinnedMessages(conversationID uuid.UUID, limit, offset int) ([]*models.Message, int64, error) {
	var messages []*models.Message
	var total int64

	// นับจำนวนทั้งหมด
	if err := r.db.Model(&models.Message{}).
		Where("conversation_id = ? AND is_pinned = ? AND is_deleted = ?", conversationID, true, false).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// ดึงข้อมูล - เรียงตามเวลาที่ปักหมุด (ล่าสุดก่อน)
	if err := r.db.
		Preload("Sender").
		Preload("Pinner").
		Preload("ReplyTo").
		Where("conversation_id = ? AND is_pinned = ? AND is_deleted = ?", conversationID, true, false).
		Order("pinned_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error; err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}

// FindByDateRange ดึงข้อความในช่วงวันที่กำหนด
func (r *messageRepository) FindByDateRange(conversationID uuid.UUID, startDate, endDate time.Time, limit int) ([]*models.Message, int64, error) {
	var messages []*models.Message
	var total int64

	// นับจำนวนข้อความทั้งหมดในช่วงเวลานี้
	if err := r.db.Model(&models.Message{}).
		Where("conversation_id = ? AND created_at >= ? AND created_at < ? AND is_deleted = ?",
			conversationID, startDate, endDate, false).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// ดึงข้อความ - เรียงตามเวลา (เก่าสุดก่อน)
	if err := r.db.
		Preload("Sender").
		Preload("ReplyTo").
		Preload("ReplyTo.Sender").
		Where("conversation_id = ? AND created_at >= ? AND created_at < ? AND is_deleted = ?",
			conversationID, startDate, endDate, false).
		Order("created_at ASC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}

// SearchMessages ค้นหาข้อความโดยใช้ full-text search (CURSOR-BASED)
func (r *messageRepository) SearchMessages(
	searchQuery string,
	conversationID *uuid.UUID,
	limit int,
	cursor *string,
	direction string,
) ([]*models.Message, *string, bool, error) {
	var messages []*models.Message

	// สร้าง base query
	baseQuery := r.db.Model(&models.Message{}).
		Where("is_deleted = ?", false).
		Where("content_tsvector @@ plainto_tsquery('english', ?)", searchQuery)

	// Filter by conversation if specified
	if conversationID != nil {
		baseQuery = baseQuery.Where("conversation_id = ?", *conversationID)
	}

	// Apply cursor pagination
	if cursor != nil && *cursor != "" {
		cursorID, err := uuid.Parse(*cursor)
		if err != nil {
			return nil, nil, false, errors.New("invalid cursor")
		}

		// Get cursor message to compare timestamp
		var cursorMsg models.Message
		if err := r.db.Where("id = ?", cursorID).First(&cursorMsg).Error; err != nil {
			return nil, nil, false, errors.New("cursor message not found")
		}

		if direction == "after" {
			// Get newer messages
			baseQuery = baseQuery.Where(
				"(created_at > ?) OR (created_at = ? AND id > ?)",
				cursorMsg.CreatedAt, cursorMsg.CreatedAt, cursorID,
			)
		} else {
			// Get older messages (default)
			baseQuery = baseQuery.Where(
				"(created_at < ?) OR (created_at = ? AND id < ?)",
				cursorMsg.CreatedAt, cursorMsg.CreatedAt, cursorID,
			)
		}
	}

	// Order by time (DESC for "before", ASC for "after")
	if direction == "after" {
		baseQuery = baseQuery.Order("created_at ASC, id ASC")
	} else {
		baseQuery = baseQuery.Order("created_at DESC, id DESC")
	}

	// Fetch limit + 1 to check if there are more results
	if err := baseQuery.
		Preload("Sender").
		Preload("Conversation").
		Preload("ReplyTo").
		Limit(limit + 1).
		Find(&messages).Error; err != nil {
		return nil, nil, false, err
	}

	// Check if there are more results
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit] // Remove the extra message
	}

	// Reverse if direction is "before" to maintain chronological order
	if direction != "after" && len(messages) > 0 {
		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
			messages[i], messages[j] = messages[j], messages[i]
		}
	}

	// Get next cursor (last message ID)
	var nextCursor *string
	if len(messages) > 0 {
		lastID := messages[len(messages)-1].ID.String()
		nextCursor = &lastID
	}

	return messages, nextCursor, hasMore, nil
}

