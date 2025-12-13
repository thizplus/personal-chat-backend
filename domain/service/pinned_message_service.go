// domain/service/pinned_message_service.go
package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/dto"
)

// PinnedMessageService defines methods for pinned message operations
type PinnedMessageService interface {
	// Pin a message (personal or public)
	PinMessage(ctx context.Context, conversationID, messageID, userID uuid.UUID, pinType string) (*dto.PinnedMessageDTO, error)

	// Unpin a message
	UnpinMessage(ctx context.Context, conversationID, messageID, userID uuid.UUID, pinType string) error

	// Get pinned messages in a conversation
	GetPinnedMessages(ctx context.Context, conversationID, userID uuid.UUID, pinType string, limit, offset int) (*dto.PinnedMessagesListDTO, error)

	// Check if a message is pinned by user
	IsPinned(ctx context.Context, messageID, userID uuid.UUID, pinType string) (bool, error)
}
