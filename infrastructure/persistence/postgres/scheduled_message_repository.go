// infrastructure/persistence/postgres/scheduled_message_repository.go
package postgres

import (
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/repository"
	"gorm.io/gorm"
)

type scheduledMessageRepository struct {
	db *gorm.DB
}

// NewScheduledMessageRepository สร้าง instance ใหม่ของ ScheduledMessageRepository
func NewScheduledMessageRepository(db *gorm.DB) repository.ScheduledMessageRepository {
	return &scheduledMessageRepository{db: db}
}

// Create สร้างข้อความที่กำหนดเวลาส่ง
func (r *scheduledMessageRepository) Create(scheduledMsg *models.ScheduledMessage) error {
	return r.db.Create(scheduledMsg).Error
}

// GetByID ดึงข้อมูลข้อความที่กำหนดเวลาส่งตาม ID
func (r *scheduledMessageRepository) GetByID(id uuid.UUID) (*models.ScheduledMessage, error) {
	var scheduledMsg models.ScheduledMessage
	err := r.db.Preload("Conversation").
		Preload("Sender").
		Preload("Message").
		Where("id = ?", id).
		First(&scheduledMsg).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &scheduledMsg, nil
}

// Update อัปเดตข้อมูลข้อความที่กำหนดเวลาส่ง
func (r *scheduledMessageRepository) Update(scheduledMsg *models.ScheduledMessage) error {
	return r.db.Save(scheduledMsg).Error
}

// Delete ลบข้อความที่กำหนดเวลาส่ง
func (r *scheduledMessageRepository) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&models.ScheduledMessage{}).Error
}

// FindByUserID ดึงรายการข้อความที่กำหนดเวลาส่งของผู้ใช้
func (r *scheduledMessageRepository) FindByUserID(userID uuid.UUID, limit, offset int) ([]*models.ScheduledMessage, int64, error) {
	var scheduledMsgs []*models.ScheduledMessage
	var total int64

	// นับจำนวนทั้งหมด
	if err := r.db.Model(&models.ScheduledMessage{}).
		Where("sender_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// ดึงข้อมูล
	err := r.db.Preload("Conversation").
		Preload("Sender").
		Preload("Message").
		Where("sender_id = ?", userID).
		Order("scheduled_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&scheduledMsgs).Error

	if err != nil {
		return nil, 0, err
	}

	return scheduledMsgs, total, nil
}

// FindByConversationID ดึงรายการข้อความที่กำหนดเวลาส่งในการสนทนา
func (r *scheduledMessageRepository) FindByConversationID(conversationID uuid.UUID, limit, offset int) ([]*models.ScheduledMessage, int64, error) {
	var scheduledMsgs []*models.ScheduledMessage
	var total int64

	// นับจำนวนทั้งหมด
	if err := r.db.Model(&models.ScheduledMessage{}).
		Where("conversation_id = ?", conversationID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// ดึงข้อมูล
	err := r.db.Preload("Conversation").
		Preload("Sender").
		Preload("Message").
		Where("conversation_id = ?", conversationID).
		Order("scheduled_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&scheduledMsgs).Error

	if err != nil {
		return nil, 0, err
	}

	return scheduledMsgs, total, nil
}

// FindByConversationAndUser ดึงรายการข้อความที่กำหนดเวลาส่งในการสนทนา เฉพาะของ user คนนั้น
func (r *scheduledMessageRepository) FindByConversationAndUser(conversationID, userID uuid.UUID, limit, offset int) ([]*models.ScheduledMessage, int64, error) {
	var scheduledMsgs []*models.ScheduledMessage
	var total int64

	// นับจำนวนทั้งหมด
	if err := r.db.Model(&models.ScheduledMessage{}).
		Where("conversation_id = ? AND sender_id = ?", conversationID, userID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// ดึงข้อมูล
	err := r.db.Preload("Conversation").
		Preload("Sender").
		Preload("Message").
		Where("conversation_id = ? AND sender_id = ?", conversationID, userID).
		Order("scheduled_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&scheduledMsgs).Error

	if err != nil {
		return nil, 0, err
	}

	return scheduledMsgs, total, nil
}

// FindPendingMessages ดึงรายการข้อความที่ถึงเวลาส่งแล้ว
func (r *scheduledMessageRepository) FindPendingMessages(beforeTime time.Time, limit int) ([]*models.ScheduledMessage, error) {
	var scheduledMsgs []*models.ScheduledMessage

	err := r.db.Preload("Conversation").
		Preload("Sender").
		Where("status = ?", "pending").
		Where("scheduled_at <= ?", beforeTime).
		Order("scheduled_at ASC").
		Limit(limit).
		Find(&scheduledMsgs).Error

	if err != nil {
		return nil, err
	}

	return scheduledMsgs, nil
}

// UpdateStatus อัปเดตสถานะของข้อความที่กำหนดเวลาส่ง
func (r *scheduledMessageRepository) UpdateStatus(id uuid.UUID, status string, sentAt *time.Time, messageID *uuid.UUID, errorReason string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if sentAt != nil {
		updates["sent_at"] = sentAt
	}

	if messageID != nil {
		updates["message_id"] = messageID
	}

	if errorReason != "" {
		updates["error_reason"] = errorReason
	}

	return r.db.Model(&models.ScheduledMessage{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// CancelScheduledMessage ยกเลิกข้อความที่กำหนดเวลาส่ง
func (r *scheduledMessageRepository) CancelScheduledMessage(id uuid.UUID) error {
	return r.db.Model(&models.ScheduledMessage{}).
		Where("id = ?", id).
		Where("status = ?", "pending").
		Updates(map[string]interface{}{
			"status":     "cancelled",
			"updated_at": time.Now(),
		}).Error
}
