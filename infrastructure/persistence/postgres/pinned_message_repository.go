// infrastructure/persistence/postgres/pinned_message_repository.go
package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/repository"
	"gorm.io/gorm"
)

type pinnedMessageRepository struct {
	db *gorm.DB
}

// NewPinnedMessageRepository creates a new pinned message repository
func NewPinnedMessageRepository(db *gorm.DB) repository.PinnedMessageRepository {
	return &pinnedMessageRepository{db: db}
}

// Create creates a new pinned message
func (r *pinnedMessageRepository) Create(ctx context.Context, pinnedMessage *models.PinnedMessage) error {
	return r.db.WithContext(ctx).Create(pinnedMessage).Error
}

// Delete deletes a pinned message by message_id, user_id, and pin_type
func (r *pinnedMessageRepository) Delete(ctx context.Context, messageID, userID uuid.UUID, pinType string) error {
	return r.db.WithContext(ctx).
		Where("message_id = ? AND user_id = ? AND pin_type = ?", messageID, userID, pinType).
		Delete(&models.PinnedMessage{}).Error
}

// DeleteByID deletes a pinned message by its ID
func (r *pinnedMessageRepository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.PinnedMessage{}, "id = ?", id).Error
}

// GetByID gets a pinned message by ID
func (r *pinnedMessageRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.PinnedMessage, error) {
	var pinnedMessage models.PinnedMessage
	err := r.db.WithContext(ctx).
		Preload("Message").
		Preload("Message.Sender").
		Preload("User").
		First(&pinnedMessage, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &pinnedMessage, nil
}

// IsPinned checks if a message is pinned by a user with a specific type
func (r *pinnedMessageRepository) IsPinned(ctx context.Context, messageID, userID uuid.UUID, pinType string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.PinnedMessage{}).
		Where("message_id = ? AND user_id = ? AND pin_type = ?", messageID, userID, pinType).
		Count(&count).Error
	return count > 0, err
}

// IsPublicPinned checks if a message has a public pin
func (r *pinnedMessageRepository) IsPublicPinned(ctx context.Context, messageID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.PinnedMessage{}).
		Where("message_id = ? AND pin_type = ?", messageID, models.PinTypePublic).
		Count(&count).Error
	return count > 0, err
}

// GetPinnedMessages gets pinned messages for a conversation
// Returns personal pins for the user AND public pins
func (r *pinnedMessageRepository) GetPinnedMessages(ctx context.Context, conversationID, userID uuid.UUID, pinType string, limit, offset int) ([]*models.PinnedMessage, int64, error) {
	var pinnedMessages []*models.PinnedMessage
	var total int64

	query := r.db.WithContext(ctx).Model(&models.PinnedMessage{}).
		Where("conversation_id = ?", conversationID)

	// Filter by pin_type
	switch pinType {
	case "personal":
		// Only personal pins by this user
		query = query.Where("pin_type = ? AND user_id = ?", models.PinTypePersonal, userID)
	case "public":
		// Only public pins (visible to all)
		query = query.Where("pin_type = ?", models.PinTypePublic)
	case "all", "":
		// Both personal (for this user) and public pins
		query = query.Where("(pin_type = ? AND user_id = ?) OR pin_type = ?",
			models.PinTypePersonal, userID, models.PinTypePublic)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Fetch data with preloads
	err := r.db.WithContext(ctx).
		Preload("Message").
		Preload("Message.Sender").
		Preload("Message.ReplyTo").
		Preload("User").
		Where("conversation_id = ?", conversationID).
		Where("(pin_type = ? AND user_id = ?) OR pin_type = ?",
			models.PinTypePersonal, userID, models.PinTypePublic).
		Order("pinned_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&pinnedMessages).Error

	if pinType == "personal" {
		err = r.db.WithContext(ctx).
			Preload("Message").
			Preload("Message.Sender").
			Preload("Message.ReplyTo").
			Preload("User").
			Where("conversation_id = ? AND pin_type = ? AND user_id = ?", conversationID, models.PinTypePersonal, userID).
			Order("pinned_at DESC").
			Limit(limit).
			Offset(offset).
			Find(&pinnedMessages).Error
	} else if pinType == "public" {
		err = r.db.WithContext(ctx).
			Preload("Message").
			Preload("Message.Sender").
			Preload("Message.ReplyTo").
			Preload("User").
			Where("conversation_id = ? AND pin_type = ?", conversationID, models.PinTypePublic).
			Order("pinned_at DESC").
			Limit(limit).
			Offset(offset).
			Find(&pinnedMessages).Error
	}

	if err != nil {
		return nil, 0, err
	}

	return pinnedMessages, total, nil
}

// GetPublicPinnedCount gets the count of public pinned messages in a conversation
func (r *pinnedMessageRepository) GetPublicPinnedCount(ctx context.Context, conversationID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.PinnedMessage{}).
		Where("conversation_id = ? AND pin_type = ?", conversationID, models.PinTypePublic).
		Count(&count).Error
	return count, err
}

// DeleteAllByMessageID deletes all pinned entries for a message
func (r *pinnedMessageRepository) DeleteAllByMessageID(ctx context.Context, messageID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("message_id = ?", messageID).
		Delete(&models.PinnedMessage{}).Error
}

// DeleteAllByConversationID deletes all pinned entries for a conversation
func (r *pinnedMessageRepository) DeleteAllByConversationID(ctx context.Context, conversationID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Delete(&models.PinnedMessage{}).Error
}
