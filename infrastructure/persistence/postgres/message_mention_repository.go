package postgres

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/repository"
	"gorm.io/gorm"
)

type messageMentionRepository struct {
	db *gorm.DB
}

// NewMessageMentionRepository creates a new message mention repository
func NewMessageMentionRepository(db *gorm.DB) repository.MessageMentionRepository {
	return &messageMentionRepository{db: db}
}

func (r *messageMentionRepository) Create(mention *models.MessageMention) error {
	return r.db.Create(mention).Error
}

func (r *messageMentionRepository) CreateBatch(mentions []*models.MessageMention) error {
	if len(mentions) == 0 {
		return nil
	}
	return r.db.CreateInBatches(mentions, 100).Error
}

func (r *messageMentionRepository) GetByUserID(
	userID uuid.UUID,
	limit int,
	cursor *string,
	direction string,
) ([]*models.MessageMention, *string, bool, error) {
	var mentions []*models.MessageMention

	query := r.db.Model(&models.MessageMention{}).
		Where("mentioned_user_id = ?", userID)

	// Apply cursor pagination
	if cursor != nil && *cursor != "" {
		cursorID, err := uuid.Parse(*cursor)
		if err != nil {
			return nil, nil, false, errors.New("invalid cursor")
		}

		var cursorMention models.MessageMention
		if err := r.db.Where("id = ?", cursorID).First(&cursorMention).Error; err != nil {
			return nil, nil, false, errors.New("cursor not found")
		}

		if direction == "after" {
			query = query.Where(
				"(created_at > ?) OR (created_at = ? AND id > ?)",
				cursorMention.CreatedAt, cursorMention.CreatedAt, cursorID,
			)
		} else {
			query = query.Where(
				"(created_at < ?) OR (created_at = ? AND id < ?)",
				cursorMention.CreatedAt, cursorMention.CreatedAt, cursorID,
			)
		}
	}

	// Order
	if direction == "after" {
		query = query.Order("created_at ASC, id ASC")
	} else {
		query = query.Order("created_at DESC, id DESC")
	}

	// Fetch limit + 1 to check hasMore
	if err := query.
		Preload("Message").
		Preload("Message.Sender").
		Preload("Message.Conversation").
		Limit(limit + 1).
		Find(&mentions).Error; err != nil {
		return nil, nil, false, err
	}

	hasMore := len(mentions) > limit
	if hasMore {
		mentions = mentions[:limit]
	}

	// ไม่ต้อง reverse เพราะเราต้องการให้ล่าสุดขึ้นก่อน (ORDER BY created_at DESC)
	// การ reverse จะทำให้เก่าสุดขึ้นก่อนแทน

	// Next cursor
	var nextCursor *string
	if len(mentions) > 0 {
		lastID := mentions[len(mentions)-1].ID.String()
		nextCursor = &lastID
	}

	return mentions, nextCursor, hasMore, nil
}

func (r *messageMentionRepository) DeleteByMessageID(messageID uuid.UUID) error {
	return r.db.Where("message_id = ?", messageID).Delete(&models.MessageMention{}).Error
}

func (r *messageMentionRepository) GetByMessageID(messageID uuid.UUID) ([]*models.MessageMention, error) {
	var mentions []*models.MessageMention
	err := r.db.Where("message_id = ?", messageID).
		Preload("MentionedUser").
		Find(&mentions).Error
	return mentions, err
}

// CountUnreadMentionsByConversation counts unread mentions in a conversation for a user
func (r *messageMentionRepository) CountUnreadMentionsByConversation(
	conversationID uuid.UUID,
	userID uuid.UUID,
	lastReadAt *time.Time,
) (int, error) {
	var count int64

	// Base query: join with messages to get conversation context
	query := r.db.Model(&models.MessageMention{}).
		Joins("JOIN messages ON messages.id = message_mentions.message_id").
		Where("messages.conversation_id = ?", conversationID).
		Where("message_mentions.mentioned_user_id = ?", userID).
		Where("messages.sender_id != ?", userID) // Don't count self-mentions

	// If lastReadAt is provided, only count mentions after that time
	if lastReadAt != nil {
		query = query.Where("messages.created_at > ?", *lastReadAt)
	}

	err := query.Count(&count).Error
	if err != nil {
		return 0, err
	}

	return int(count), nil
}

// CheckLastMessageHasMention checks if a specific message has a mention for a user
func (r *messageMentionRepository) CheckLastMessageHasMention(
	messageID uuid.UUID,
	userID uuid.UUID,
) (bool, error) {
	var count int64

	err := r.db.Model(&models.MessageMention{}).
		Where("message_id = ?", messageID).
		Where("mentioned_user_id = ?", userID).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}
