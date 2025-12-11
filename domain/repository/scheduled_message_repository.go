// domain/repository/scheduled_message_repository.go
package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
)

type ScheduledMessageRepository interface {
	// CRUD operations
	Create(scheduledMsg *models.ScheduledMessage) error
	GetByID(id uuid.UUID) (*models.ScheduledMessage, error)
	Update(scheduledMsg *models.ScheduledMessage) error
	Delete(id uuid.UUID) error

	// Query operations
	FindByUserID(userID uuid.UUID, limit, offset int) ([]*models.ScheduledMessage, int64, error)
	FindByConversationID(conversationID uuid.UUID, limit, offset int) ([]*models.ScheduledMessage, int64, error)
	FindByConversationAndUser(conversationID, userID uuid.UUID, limit, offset int) ([]*models.ScheduledMessage, int64, error)
	FindPendingMessages(beforeTime time.Time, limit int) ([]*models.ScheduledMessage, error)

	// Status updates
	UpdateStatus(id uuid.UUID, status string, sentAt *time.Time, messageID *uuid.UUID, errorReason string) error
	CancelScheduledMessage(id uuid.UUID) error
}
