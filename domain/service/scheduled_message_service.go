// domain/service/scheduled_message_service.go
package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
)

// ScheduledMessageProcessor interface สำหรับ processor (เพื่อหลีกเลี่ยง circular dependency)
type ScheduledMessageProcessor interface {
	ScheduleMessage(messageID uuid.UUID, scheduledAt time.Time)
	CancelMessage(messageID uuid.UUID)
	RescheduleMessage(messageID uuid.UUID, newTime time.Time)
}

// ScheduledMessageService เป็น interface ที่กำหนดฟังก์ชันของ Scheduled Message Service
type ScheduledMessageService interface {
	// Create and manage scheduled messages
	ScheduleMessage(conversationID, userID uuid.UUID, messageType, content, mediaURL string, metadata map[string]interface{}, scheduledAt time.Time) (*models.ScheduledMessage, error)
	GetScheduledMessage(id, userID uuid.UUID) (*models.ScheduledMessage, error)
	GetUserScheduledMessages(userID uuid.UUID, limit, offset int) ([]*models.ScheduledMessage, int64, error)
	GetConversationScheduledMessages(conversationID, userID uuid.UUID, limit, offset int) ([]*models.ScheduledMessage, int64, error)
	CancelScheduledMessage(id, userID uuid.UUID) error
	UpdateScheduledTime(id, userID uuid.UUID, newScheduledAt time.Time) (*models.ScheduledMessage, error)

	// For processor to use
	GetPendingMessagesForProcessor(beforeTime time.Time, limit int) ([]*models.ScheduledMessage, error)
	ProcessSingleScheduledMessage(messageID uuid.UUID) error

	// Set processor reference (for timer integration)
	SetProcessor(processor ScheduledMessageProcessor)

	// Legacy method (kept for compatibility)
	ProcessScheduledMessages() error
}
