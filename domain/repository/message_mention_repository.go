package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
)

// MessageMentionRepository defines methods for managing message mentions
type MessageMentionRepository interface {
	// Create a single mention
	Create(mention *models.MessageMention) error

	// Create multiple mentions at once
	CreateBatch(mentions []*models.MessageMention) error

	// Get mentions for a specific user (cursor-based pagination)
	// Returns: mentions, nextCursor, hasMore, error
	GetByUserID(
		userID uuid.UUID,
		limit int,
		cursor *string,
		direction string,
	) ([]*models.MessageMention, *string, bool, error)

	// Delete mentions for a message (when message is deleted)
	DeleteByMessageID(messageID uuid.UUID) error

	// Get all mentions in a message
	GetByMessageID(messageID uuid.UUID) ([]*models.MessageMention, error)

	// CountUnreadMentionsByConversation counts unread mentions in a conversation for a user
	// If lastReadAt is nil, counts all mentions where the user is not the sender
	// If lastReadAt is provided, counts only mentions created after that time
	CountUnreadMentionsByConversation(
		conversationID uuid.UUID,
		userID uuid.UUID,
		lastReadAt *time.Time,
	) (int, error)

	// CheckLastMessageHasMention checks if a specific message has a mention for a user
	// Returns true if the message mentions the user, false otherwise
	CheckLastMessageHasMention(
		messageID uuid.UUID,
		userID uuid.UUID,
	) (bool, error)
}
