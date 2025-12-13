// application/serviceimpl/pinned_message_service.go
package serviceimpl

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/dto"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/port"
	"github.com/thizplus/gofiber-chat-api/domain/repository"
	"github.com/thizplus/gofiber-chat-api/domain/service"
)

// Maximum public pins per conversation
const MaxPublicPinnedMessages = 5

type pinnedMessageService struct {
	pinnedRepo       repository.PinnedMessageRepository
	messageRepo      repository.MessageRepository
	conversationRepo repository.ConversationRepository
	wsPort           port.WebSocketPort
}

// NewPinnedMessageService creates a new pinned message service
func NewPinnedMessageService(
	pinnedRepo repository.PinnedMessageRepository,
	messageRepo repository.MessageRepository,
	conversationRepo repository.ConversationRepository,
	wsPort port.WebSocketPort,
) service.PinnedMessageService {
	return &pinnedMessageService{
		pinnedRepo:       pinnedRepo,
		messageRepo:      messageRepo,
		conversationRepo: conversationRepo,
		wsPort:           wsPort,
	}
}

// PinMessage pins a message
func (s *pinnedMessageService) PinMessage(ctx context.Context, conversationID, messageID, userID uuid.UUID, pinType string) (*dto.PinnedMessageDTO, error) {
	// Validate pin type
	if pinType != models.PinTypePersonal && pinType != models.PinTypePublic {
		return nil, errors.New("invalid pin type, must be 'personal' or 'public'")
	}

	// Check if message exists and belongs to conversation
	message, err := s.messageRepo.GetByID(messageID)
	if err != nil {
		return nil, err
	}
	if message == nil {
		return nil, errors.New("message not found")
	}
	if message.ConversationID != conversationID {
		return nil, errors.New("message does not belong to this conversation")
	}
	if message.IsDeleted {
		return nil, errors.New("cannot pin deleted message")
	}

	// Check if user is member of conversation
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("user is not a member of this conversation")
	}

	// For public pins, check permissions (only owner/admin in group)
	if pinType == models.PinTypePublic {
		conversation, err := s.conversationRepo.GetByID(conversationID)
		if err != nil {
			return nil, err
		}

		if conversation.Type == "group" {
			member, err := s.conversationRepo.GetMember(conversationID, userID)
			if err != nil {
				return nil, err
			}
			if member == nil || (member.Role != "owner" && member.Role != "admin") {
				return nil, errors.New("only owner/admin can create public pins in group conversations")
			}
		}

		// Check max public pins limit
		publicCount, err := s.pinnedRepo.GetPublicPinnedCount(ctx, conversationID)
		if err != nil {
			return nil, err
		}
		if publicCount >= MaxPublicPinnedMessages {
			return nil, errors.New("maximum public pins limit reached (5)")
		}
	}

	// Check if already pinned
	isPinned, err := s.pinnedRepo.IsPinned(ctx, messageID, userID, pinType)
	if err != nil {
		return nil, err
	}
	if isPinned {
		return nil, errors.New("message is already pinned with this type")
	}

	// Create pinned message
	now := time.Now()
	pinnedMessage := &models.PinnedMessage{
		ID:             uuid.New(),
		MessageID:      messageID,
		ConversationID: conversationID,
		UserID:         userID,
		PinType:        pinType,
		PinnedAt:       now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.pinnedRepo.Create(ctx, pinnedMessage); err != nil {
		return nil, err
	}

	// Get full pinned message with associations
	fullPinned, err := s.pinnedRepo.GetByID(ctx, pinnedMessage.ID)
	if err != nil {
		return nil, err
	}

	pinnedDTO := s.toPinnedMessageDTO(fullPinned)

	// Broadcast WebSocket event for public pins
	if pinType == models.PinTypePublic && s.wsPort != nil {
		s.wsPort.BroadcastMessagePinned(conversationID, pinnedDTO)
	}

	return pinnedDTO, nil
}

// UnpinMessage unpins a message
func (s *pinnedMessageService) UnpinMessage(ctx context.Context, conversationID, messageID, userID uuid.UUID, pinType string) error {
	// Validate pin type
	if pinType != models.PinTypePersonal && pinType != models.PinTypePublic {
		return errors.New("invalid pin type, must be 'personal' or 'public'")
	}

	// Check if user is member of conversation
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return errors.New("user is not a member of this conversation")
	}

	// For personal pins, user can only unpin their own
	if pinType == models.PinTypePersonal {
		return s.pinnedRepo.Delete(ctx, messageID, userID, pinType)
	}

	// For public pins, check permissions
	conversation, err := s.conversationRepo.GetByID(conversationID)
	if err != nil {
		return err
	}

	if conversation.Type == "group" {
		member, err := s.conversationRepo.GetMember(conversationID, userID)
		if err != nil {
			return err
		}
		if member == nil || (member.Role != "owner" && member.Role != "admin") {
			return errors.New("only owner/admin can remove public pins in group conversations")
		}
	}

	// Delete the public pin (any user who created it)
	if err := s.pinnedRepo.Delete(ctx, messageID, userID, pinType); err != nil {
		return err
	}

	// Broadcast WebSocket event for public unpins
	if s.wsPort != nil {
		s.wsPort.BroadcastMessageUnpinned(conversationID, messageID, userID)
	}

	return nil
}

// GetPinnedMessages gets pinned messages for a conversation
func (s *pinnedMessageService) GetPinnedMessages(ctx context.Context, conversationID, userID uuid.UUID, pinType string, limit, offset int) (*dto.PinnedMessagesListDTO, error) {
	// Check if user is member of conversation
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("user is not a member of this conversation")
	}

	// Get pinned messages
	pinnedMessages, total, err := s.pinnedRepo.GetPinnedMessages(ctx, conversationID, userID, pinType, limit, offset)
	if err != nil {
		return nil, err
	}

	// Convert to DTOs
	pinnedDTOs := make([]dto.PinnedMessageDTO, len(pinnedMessages))
	for i, pm := range pinnedMessages {
		pinnedDTOs[i] = *s.toPinnedMessageDTO(pm)
	}

	return &dto.PinnedMessagesListDTO{
		ConversationID: conversationID,
		Total:          total,
		PinnedMessages: pinnedDTOs,
	}, nil
}

// IsPinned checks if a message is pinned by user
func (s *pinnedMessageService) IsPinned(ctx context.Context, messageID, userID uuid.UUID, pinType string) (bool, error) {
	return s.pinnedRepo.IsPinned(ctx, messageID, userID, pinType)
}

// Helper function to convert model to DTO
func (s *pinnedMessageService) toPinnedMessageDTO(pm *models.PinnedMessage) *dto.PinnedMessageDTO {
	if pm == nil {
		return nil
	}

	result := &dto.PinnedMessageDTO{
		ID:             pm.ID,
		MessageID:      pm.MessageID,
		ConversationID: pm.ConversationID,
		UserID:         pm.UserID,
		PinType:        pm.PinType,
		PinnedAt:       pm.PinnedAt,
	}

	// Add pinned by info
	if pm.User != nil {
		result.PinnedBy = &dto.UserBasicDTO{
			ID:              pm.User.ID,
			Username:        pm.User.Username,
			DisplayName:     pm.User.DisplayName,
			ProfileImageURL: pm.User.ProfileImageURL,
		}
	}

	// Add message info
	if pm.Message != nil {
		result.Message = &dto.MessageDTO{
			ID:             pm.Message.ID,
			ConversationID: pm.Message.ConversationID,
			SenderID:       pm.Message.SenderID,
			MessageType:    pm.Message.MessageType,
			Content:        pm.Message.Content,
			MediaURL:       pm.Message.MediaURL,
			CreatedAt:      pm.Message.CreatedAt,
			UpdatedAt:      pm.Message.UpdatedAt,
		}

		if pm.Message.Sender != nil {
			result.Message.SenderInfo = &dto.UserBasicDTO{
				ID:              pm.Message.Sender.ID,
				Username:        pm.Message.Sender.Username,
				DisplayName:     pm.Message.Sender.DisplayName,
				ProfileImageURL: pm.Message.Sender.ProfileImageURL,
			}
		}
	}

	return result
}
