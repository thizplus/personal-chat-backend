// application/serviceimpl/scheduled_message_service.go
package serviceimpl

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/repository"
	"github.com/thizplus/gofiber-chat-api/domain/service"
)

type scheduledMessageService struct {
	scheduledMessageRepo    repository.ScheduledMessageRepository
	conversationRepo        repository.ConversationRepository
	messageService          service.MessageService
	notificationService     service.NotificationService
	processor               service.ScheduledMessageProcessor
}

// NewScheduledMessageService สร้าง instance ใหม่ของ ScheduledMessageService
func NewScheduledMessageService(
	scheduledMessageRepo repository.ScheduledMessageRepository,
	conversationRepo repository.ConversationRepository,
	messageService service.MessageService,
	notificationService service.NotificationService,
) service.ScheduledMessageService {
	return &scheduledMessageService{
		scheduledMessageRepo:    scheduledMessageRepo,
		conversationRepo:        conversationRepo,
		messageService:          messageService,
		notificationService:     notificationService,
	}
}

// SetProcessor ตั้งค่า processor reference (เรียกหลังจากสร้าง processor แล้ว)
func (s *scheduledMessageService) SetProcessor(processor service.ScheduledMessageProcessor) {
	s.processor = processor
	log.Println("[ScheduledMessageService] Processor connected for precise timing")
}

// ScheduleMessage กำหนดเวลาส่งข้อความ
func (s *scheduledMessageService) ScheduleMessage(
	conversationID, userID uuid.UUID,
	messageType, content, mediaURL string,
	metadata map[string]interface{},
	scheduledAt time.Time,
) (*models.ScheduledMessage, error) {
	// ตรวจสอบว่า user เป็นสมาชิกของการสนทนา
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("user is not a member of this conversation")
	}

	// ตรวจสอบว่า scheduled_at ต้องอยู่ในอนาคต
	if scheduledAt.Before(time.Now()) {
		return nil, errors.New("scheduled_at must be in the future")
	}

	// สร้าง metadata JSONB
	metadataJSON := make(map[string]interface{})
	if metadata != nil {
		for k, v := range metadata {
			metadataJSON[k] = v
		}
	}

	// สร้าง scheduled message
	scheduledMsg := &models.ScheduledMessage{
		ID:             uuid.New(),
		ConversationID: conversationID,
		SenderID:       userID,
		MessageType:    messageType,
		Content:        content,
		MediaURL:       mediaURL,
		Metadata:       metadataJSON,
		ScheduledAt:    scheduledAt,
		Status:         "pending",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// บันทึกลงฐานข้อมูล
	if err := s.scheduledMessageRepo.Create(scheduledMsg); err != nil {
		return nil, err
	}

	// สร้าง in-memory timer สำหรับ precise timing
	if s.processor != nil {
		s.processor.ScheduleMessage(scheduledMsg.ID, scheduledAt)
		log.Printf("[ScheduledMessageService] Created timer for message %s at %s", scheduledMsg.ID, scheduledAt.Format(time.RFC3339))
	}

	return scheduledMsg, nil
}

// GetScheduledMessage ดึงข้อมูลข้อความที่กำหนดเวลาส่ง
func (s *scheduledMessageService) GetScheduledMessage(id, userID uuid.UUID) (*models.ScheduledMessage, error) {
	scheduledMsg, err := s.scheduledMessageRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if scheduledMsg == nil {
		return nil, errors.New("scheduled message not found")
	}

	// ตรวจสอบว่า user เป็นเจ้าของข้อความ
	if scheduledMsg.SenderID != userID {
		return nil, errors.New("unauthorized to access this scheduled message")
	}

	return scheduledMsg, nil
}

// GetUserScheduledMessages ดึงรายการข้อความที่กำหนดเวลาส่งของผู้ใช้
func (s *scheduledMessageService) GetUserScheduledMessages(userID uuid.UUID, limit, offset int) ([]*models.ScheduledMessage, int64, error) {
	return s.scheduledMessageRepo.FindByUserID(userID, limit, offset)
}

// GetConversationScheduledMessages ดึงรายการข้อความที่กำหนดเวลาส่งในการสนทนา (เฉพาะของ user คนนั้นเท่านั้น)
func (s *scheduledMessageService) GetConversationScheduledMessages(conversationID, userID uuid.UUID, limit, offset int) ([]*models.ScheduledMessage, int64, error) {
	// ตรวจสอบว่า user เป็นสมาชิกของการสนทนา
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return nil, 0, err
	}
	if !isMember {
		return nil, 0, errors.New("user is not a member of this conversation")
	}

	// ดึงเฉพาะข้อความที่ตั้งเวลาโดย user คนนี้เท่านั้น (ไม่ให้เห็นของคนอื่น)
	return s.scheduledMessageRepo.FindByConversationAndUser(conversationID, userID, limit, offset)
}

// CancelScheduledMessage ยกเลิกข้อความที่กำหนดเวลาส่ง
func (s *scheduledMessageService) CancelScheduledMessage(id, userID uuid.UUID) error {
	// ดึงข้อมูล scheduled message
	scheduledMsg, err := s.scheduledMessageRepo.GetByID(id)
	if err != nil {
		return err
	}
	if scheduledMsg == nil {
		return errors.New("scheduled message not found")
	}

	// ตรวจสอบว่า user เป็นเจ้าของข้อความ
	if scheduledMsg.SenderID != userID {
		return errors.New("unauthorized to cancel this scheduled message")
	}

	// ตรวจสอบสถานะ
	if scheduledMsg.Status != "pending" {
		return errors.New("can only cancel pending scheduled messages")
	}

	// ยกเลิก in-memory timer
	if s.processor != nil {
		s.processor.CancelMessage(id)
		log.Printf("[ScheduledMessageService] Cancelled timer for message %s", id)
	}

	return s.scheduledMessageRepo.CancelScheduledMessage(id)
}

// UpdateScheduledTime เปลี่ยนเวลาที่กำหนดส่ง
func (s *scheduledMessageService) UpdateScheduledTime(id, userID uuid.UUID, newScheduledAt time.Time) (*models.ScheduledMessage, error) {
	// ดึงข้อมูล scheduled message
	scheduledMsg, err := s.scheduledMessageRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if scheduledMsg == nil {
		return nil, errors.New("scheduled message not found")
	}

	// ตรวจสอบว่า user เป็นเจ้าของข้อความ
	if scheduledMsg.SenderID != userID {
		return nil, errors.New("unauthorized to update this scheduled message")
	}

	// ตรวจสอบสถานะ
	if scheduledMsg.Status != "pending" {
		return nil, errors.New("can only update pending scheduled messages")
	}

	// ตรวจสอบว่าเวลาใหม่ต้องอยู่ในอนาคต
	if newScheduledAt.Before(time.Now()) {
		return nil, errors.New("scheduled_at must be in the future")
	}

	// อัปเดตในฐานข้อมูล
	scheduledMsg.ScheduledAt = newScheduledAt
	scheduledMsg.UpdatedAt = time.Now()
	if err := s.scheduledMessageRepo.Update(scheduledMsg); err != nil {
		return nil, err
	}

	// อัปเดต in-memory timer
	if s.processor != nil {
		s.processor.RescheduleMessage(id, newScheduledAt)
		log.Printf("[ScheduledMessageService] Rescheduled message %s to %s", id, newScheduledAt.Format(time.RFC3339))
	}

	return scheduledMsg, nil
}

// GetPendingMessagesForProcessor ดึง pending messages สำหรับ processor
func (s *scheduledMessageService) GetPendingMessagesForProcessor(beforeTime time.Time, limit int) ([]*models.ScheduledMessage, error) {
	return s.scheduledMessageRepo.FindPendingMessages(beforeTime, limit)
}

// ProcessSingleScheduledMessage ประมวลผลข้อความเดียว (เรียกจาก timer callback)
func (s *scheduledMessageService) ProcessSingleScheduledMessage(messageID uuid.UUID) error {
	// ดึงข้อมูล scheduled message
	scheduledMsg, err := s.scheduledMessageRepo.GetByID(messageID)
	if err != nil {
		return fmt.Errorf("failed to get scheduled message: %w", err)
	}
	if scheduledMsg == nil {
		return errors.New("scheduled message not found")
	}

	// ตรวจสอบสถานะ
	if scheduledMsg.Status != "pending" {
		log.Printf("[ScheduledMessageService] Message %s is not pending (status: %s), skipping", messageID, scheduledMsg.Status)
		return nil
	}

	// ส่งข้อความ
	if err := s.sendScheduledMessage(scheduledMsg); err != nil {
		// อัปเดตสถานะเป็น failed
		_ = s.scheduledMessageRepo.UpdateStatus(
			scheduledMsg.ID,
			"failed",
			nil,
			nil,
			err.Error(),
		)
		return err
	}

	return nil
}

// ProcessScheduledMessages ประมวลผลข้อความที่ถึงเวลาส่ง (legacy method - kept for compatibility)
func (s *scheduledMessageService) ProcessScheduledMessages() error {
	// ดึงข้อความที่ถึงเวลาส่งแล้ว
	now := time.Now()
	scheduledMessages, err := s.scheduledMessageRepo.FindPendingMessages(now, 100)
	if err != nil {
		return fmt.Errorf("failed to fetch pending messages: %w", err)
	}

	if len(scheduledMessages) == 0 {
		return nil
	}

	log.Printf("[ScheduledMessageService] Processing %d scheduled messages (legacy mode)", len(scheduledMessages))

	// ส่งแต่ละข้อความ
	for _, scheduledMsg := range scheduledMessages {
		if err := s.sendScheduledMessage(scheduledMsg); err != nil {
			log.Printf("[ScheduledMessageService] Failed to send message %s: %v", scheduledMsg.ID, err)
			// อัปเดตสถานะเป็น failed
			_ = s.scheduledMessageRepo.UpdateStatus(
				scheduledMsg.ID,
				"failed",
				nil,
				nil,
				err.Error(),
			)
		}
	}

	return nil
}

// sendScheduledMessage ส่งข้อความที่กำหนดเวลาส่ง
func (s *scheduledMessageService) sendScheduledMessage(scheduledMsg *models.ScheduledMessage) error {
	var message *models.Message
	var err error

	// แปลง metadata กลับเป็น map[string]interface{}
	metadata := make(map[string]interface{})
	for k, v := range scheduledMsg.Metadata {
		metadata[k] = v
	}

	// ส่งข้อความตามประเภท
	switch scheduledMsg.MessageType {
	case "text":
		message, err = s.messageService.SendTextMessage(
			scheduledMsg.ConversationID,
			scheduledMsg.SenderID,
			scheduledMsg.Content,
			metadata,
		)
	case "image":
		message, err = s.messageService.SendImageMessage(
			scheduledMsg.ConversationID,
			scheduledMsg.SenderID,
			scheduledMsg.MediaURL,
			"", // thumbnailURL
			scheduledMsg.Content, // caption
			metadata,
		)
	case "file":
		message, err = s.messageService.SendFileMessage(
			scheduledMsg.ConversationID,
			scheduledMsg.SenderID,
			scheduledMsg.MediaURL,
			scheduledMsg.Content, // fileName
			0,                    // fileSize
			"",                   // fileType
			metadata,
		)
	case "sticker":
		// สติกเกอร์ต้องมี stickerID ใน metadata
		var stickerID, stickerSetID uuid.UUID
		if stickerIDStr, ok := metadata["sticker_id"].(string); ok {
			stickerID, _ = uuid.Parse(stickerIDStr)
		}
		if stickerSetIDStr, ok := metadata["sticker_set_id"].(string); ok {
			stickerSetID, _ = uuid.Parse(stickerSetIDStr)
		}

		message, err = s.messageService.SendStickerMessage(
			scheduledMsg.ConversationID,
			scheduledMsg.SenderID,
			stickerID,
			stickerSetID,
			scheduledMsg.MediaURL,
			"", // thumbnailURL
			metadata,
		)
	case "album":
		// Album - หลายไฟล์ในข้อความเดียว
		// ดึง album_files จาก metadata
		albumFilesRaw, ok := metadata["album_files"]
		if !ok {
			return fmt.Errorf("album_files is required for album message type")
		}

		// แปลง album_files เป็น []map[string]interface{}
		var albumFiles []map[string]interface{}
		switch v := albumFilesRaw.(type) {
		case []interface{}:
			for _, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					albumFiles = append(albumFiles, itemMap)
				}
			}
		case []map[string]interface{}:
			albumFiles = v
		default:
			return fmt.Errorf("invalid album_files format")
		}

		if len(albumFiles) == 0 {
			return fmt.Errorf("album_files cannot be empty")
		}

		// ใช้ SendBulkMessages สำหรับส่ง album
		message, err = s.messageService.SendBulkMessages(
			scheduledMsg.ConversationID,
			scheduledMsg.SenderID,
			scheduledMsg.Content, // caption
			albumFiles,
		)
	default:
		return fmt.Errorf("unsupported message type: %s", scheduledMsg.MessageType)
	}

	if err != nil {
		return err
	}

	// ✅ ส่ง WebSocket notification เหมือนการส่งข้อความปกติ
	if s.notificationService != nil {
		s.notificationService.NotifyNewMessage(scheduledMsg.ConversationID, message)
		log.Printf("[ScheduledMessage] WebSocket notification sent for message %s", message.ID)
	}

	// อัปเดตสถานะเป็น sent
	now := time.Now()
	return s.scheduledMessageRepo.UpdateStatus(
		scheduledMsg.ID,
		"sent",
		&now,
		&message.ID,
		"",
	)
}
