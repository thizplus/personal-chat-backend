// domain/repository/pinned_message_repository.go
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
)

// PinnedMessageRepository defines methods for pinned message operations
type PinnedMessageRepository interface {
	// Create a pinned message
	Create(ctx context.Context, pinnedMessage *models.PinnedMessage) error

	// Delete a pinned message
	Delete(ctx context.Context, messageID, userID uuid.UUID, pinType string) error

	// Delete by ID
	DeleteByID(ctx context.Context, id uuid.UUID) error

	// Get pinned message by ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.PinnedMessage, error)

	// Check if message is pinned by user with specific type
	IsPinned(ctx context.Context, messageID, userID uuid.UUID, pinType string) (bool, error)

	// Check if message is pinned by anyone (public only)
	IsPublicPinned(ctx context.Context, messageID uuid.UUID) (bool, error)

	// Get all pinned messages in a conversation (for a user)
	// Returns both personal pins by user AND public pins
	GetPinnedMessages(ctx context.Context, conversationID, userID uuid.UUID, pinType string, limit, offset int) ([]*models.PinnedMessage, int64, error)

	// Get public pinned messages count
	GetPublicPinnedCount(ctx context.Context, conversationID uuid.UUID) (int64, error)

	// Delete all pinned entries for a specific message (when message is deleted)
	DeleteAllByMessageID(ctx context.Context, messageID uuid.UUID) error

	// Delete all pinned entries for a conversation (when conversation is deleted)
	DeleteAllByConversationID(ctx context.Context, conversationID uuid.UUID) error
}
