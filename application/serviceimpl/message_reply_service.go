// application/serviceimpl/message_reply_service.go
package serviceimpl

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
)

// ReplyToMessage ตอบกลับข้อความ
func (s *messageService) ReplyToMessage(replyToID, userID uuid.UUID, messageType, content, mediaURL, thumbnailURL string, metadata map[string]interface{}) (*models.Message, error) {

	// ดึงข้อมูลข้อความที่ตอบกลับ
	replyToMessage, err := s.messageRepo.GetByID(replyToID)
	if err != nil {
		return nil, fmt.Errorf("error fetching reply-to message: %w", err)
	}

	if replyToMessage == nil {
		return nil, fmt.Errorf("message not found")
	}

	// ตรวจสอบว่าข้อความไม่ได้ถูกลบไป
	if replyToMessage.IsDeleted {
		return nil, fmt.Errorf("cannot reply to deleted message")
	}

	// ตรวจสอบว่าผู้ใช้เป็นสมาชิกของการสนทนา
	isMember, err := s.conversationRepo.IsMember(replyToMessage.ConversationID, userID)
	if err != nil {
		return nil, fmt.Errorf("error checking membership: %w", err)
	}

	if !isMember {
		return nil, fmt.Errorf("you are not a member of this conversation")
	}

	// ตรวจสอบตามประเภทข้อความ
	switch messageType {
	case "text":
		if strings.TrimSpace(content) == "" {
			return nil, fmt.Errorf("message content is required")
		}
	case "sticker":
		if mediaURL == "" {
			return nil, fmt.Errorf("sticker URL is required")
		}
	case "image", "file":
		if mediaURL == "" {
			return nil, fmt.Errorf("media URL is required")
		}
	default:
		return nil, fmt.Errorf("invalid message type")
	}

	senderType := "user"

	// ตรวจสอบว่ามี business_id ใน metadata หรือไม่

	// สร้างข้อความใหม่
	now := time.Now()
	message := &models.Message{
		ID:                uuid.New(),
		ConversationID:    replyToMessage.ConversationID,
		SenderID:          &userID,
		SenderType:        senderType, // ใช้ค่าที่กำหนดจากเงื่อนไข
		MessageType:       messageType,
		Content:           content,
		MediaURL:          mediaURL,
		MediaThumbnailURL: thumbnailURL,
		ReplyToID:         &replyToID,
		Metadata:          s.convertMetadataToJSON(metadata),
		CreatedAt:         now,
		UpdatedAt:         now,
	}


	// บันทึกข้อความ
	if err := s.messageRepo.Create(message); err != nil {
		return nil, fmt.Errorf("error creating message: %w", err)
	}

	// อัปเดตข้อความล่าสุดของการสนทนา
	lastMessageText := ""
	switch messageType {
	case "text":
		lastMessageText = content
	case "sticker":
		lastMessageText = "[Sticker]"
	case "image":
		lastMessageText = "[Image]"
		if content != "" {
			lastMessageText = "[Image] " + content
		}
	case "file":
		lastMessageText = "[File]"
		if content != "" {
			lastMessageText = "[File] " + content
		}
	default:
		lastMessageText = "[Message]"
	}

	if err := s.messageRepo.UpdateConversationLastMessage(replyToMessage.ConversationID, lastMessageText, now, message.ID); err != nil {
		fmt.Printf("Error updating conversation last message: %v, conversationID: %s", err, replyToMessage.ConversationID.String())
	} else {
		// ส่ง WebSocket event แจ้งการอัปเดต conversation พร้อม mention data
		s.notifyConversationUpdated(replyToMessage.ConversationID, lastMessageText, now, message.ID)
	}

	// อัปเดตเวลาอ่านล่าสุดของผู้ส่ง
	if err := s.updateConversationLastRead(replyToMessage.ConversationID, userID, now); err != nil {
		fmt.Printf("Error updating last read time: %v, conversationID: %s, userID: %s", err, replyToMessage.ConversationID.String(), userID)
	}

	// สร้างบันทึกการอ่านสำหรับผู้ส่ง
	if err := s.createMessageRead(message.ID, userID); err != nil {
		fmt.Printf("Error creating read record: %v, messageID: %s, userID: %s", err, message.ID.String(), userID)
	}

	return message, nil
}
