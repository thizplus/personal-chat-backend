// application/serviceimpl/message_delete_service.go
package serviceimpl

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/types"
)

// DeleteMessage ลบข้อความ (soft delete)
func (s *messageService) DeleteMessage(messageID, userID uuid.UUID) error {

	// ดึงข้อมูลข้อความ
	message, err := s.messageRepo.GetByID(messageID)
	if err != nil {
		return fmt.Errorf("error fetching message: %w", err)
	}

	if message == nil {
		return fmt.Errorf("message not found")
	}

	// ตรวจสอบว่าข้อความถูกลบไปแล้วหรือไม่
	if message.IsDeleted {
		return fmt.Errorf("message is already deleted")
	}

	// ตรวจสอบว่าผู้ใช้เป็นเจ้าของข้อความหรือไม่
	isSender := message.SenderID != nil && *message.SenderID == userID

	// ถ้าไม่ใช่เจ้าของ ให้ตรวจสอบว่าเป็นแอดมินหรือไม่
	isAdmin := false
	if !isSender {
		var err error
		isAdmin, err = s.messageRepo.IsConversationAdmin(message.ConversationID, userID)
		if err != nil {
			return fmt.Errorf("error checking admin status: %w", err)
		}

		if !isAdmin {
			return fmt.Errorf("only message owner or conversation admin can delete messages")
		}
	}

	// สร้าง metadata สำหรับประวัติการลบ
	now := time.Now()
	metadataObj := map[string]interface{}{
		"deleted_by_id": userID,
		"deleted_at":    now.Format(time.RFC3339),
		"message_type":  message.MessageType,
	}

	// บันทึกประวัติการลบ
	deleteHistory := &models.MessageDeleteHistory{
		ID:                uuid.New(),
		MessageID:         messageID,
		Content:           message.Content,
		MediaURL:          message.MediaURL,
		MediaThumbnailURL: message.MediaThumbnailURL,
		Metadata:          s.convertMetadataToJSON(metadataObj),
		DeletedAt:         now,
		DeletedBy:         userID,
	}

	if err := s.messageRepo.CreateDeleteHistory(deleteHistory); err != nil {
		fmt.Printf("Failed to save delete history: %v\n", err)
	}

	// "ลบ" ข้อความ (soft delete)
	message.IsDeleted = true
	message.Content = ""
	message.MediaURL = ""
	message.MediaThumbnailURL = ""
	message.Metadata = types.JSONB{} // empty JSONB
	message.UpdatedAt = now

	if err := s.messageRepo.Update(message); err != nil {
		return fmt.Errorf("error updating message: %w", err)
	}

	// ตรวจสอบว่าเป็นข้อความล่าสุดของการสนทนาหรือไม่ และอัพเดทหากจำเป็น
	lastMessage, err := s.messageRepo.GetLastMessageByConversation(message.ConversationID)
	if err == nil && lastMessage != nil && lastMessage.ID == message.ID {
		// ดึงข้อความล่าสุดที่ไม่ถูกลบ
		newLastMessage, err := s.messageRepo.GetLastNonDeletedMessageByConversation(message.ConversationID)
		if err == nil && newLastMessage != nil {
			// มีข้อความล่าสุดใหม่
			lastMessageText := ""
			switch newLastMessage.MessageType {
			case "text":
				lastMessageText = newLastMessage.Content
			case "sticker":
				lastMessageText = "[Sticker]"
			case "image":
				lastMessageText = "[Image]"
				if newLastMessage.Content != "" {
					lastMessageText = "[Image] " + newLastMessage.Content
				}
			case "file":
				lastMessageText = "[File]"
				if newLastMessage.Content != "" {
					lastMessageText = "[File] " + newLastMessage.Content
				}
			default:
				lastMessageText = "[Message]"
			}

			if err := s.messageRepo.UpdateConversationLastMessage(message.ConversationID, lastMessageText, newLastMessage.CreatedAt, newLastMessage.ID); err != nil {
				fmt.Printf("Error updating conversation last message: %v\n", err)
			} else {
				// ส่ง WebSocket event แจ้งการอัปเดต conversation พร้อม mention data
				s.notifyConversationUpdated(message.ConversationID, lastMessageText, newLastMessage.CreatedAt, newLastMessage.ID)
			}
		} else {
			// ไม่มีข้อความเหลือแล้ว - ใช้ zero UUID สำหรับกรณีไม่มีข้อความ
			if err := s.messageRepo.UpdateConversationLastMessage(message.ConversationID, "[No messages]", now, uuid.Nil); err != nil {
				fmt.Printf("Error updating conversation last message: %v\n", err)
			} else {
				// ส่ง WebSocket event แจ้งการอัปเดต conversation พร้อม mention data
				s.notifyConversationUpdated(message.ConversationID, "[No messages]", now, uuid.Nil)
			}
		}
	}

	// ส่ง WebSocket notification แจ้งว่าข้อความถูกลบ
	if s.notificationService != nil {
		s.notificationService.NotifyMessageDeleted(message.ConversationID, messageID)
	}

	return nil
}

// GetMessageDeleteHistory ดึงประวัติการลบข้อความ (สำหรับแอดมินเท่านั้น)
func (s *messageService) GetMessageDeleteHistory(messageID, userID uuid.UUID) ([]*models.MessageDeleteHistory, error) {

	// ดึงข้อมูลข้อความ
	message, err := s.messageRepo.GetByID(messageID)
	if err != nil {
		return nil, fmt.Errorf("error fetching message: %w", err)
	}

	if message == nil {
		return nil, fmt.Errorf("message not found")
	}

	// ตรวจสอบว่าผู้ใช้เป็นแอดมินของการสนทนานี้
	isAdmin, err := s.messageRepo.IsConversationAdmin(message.ConversationID, userID)
	if err != nil {
		return nil, fmt.Errorf("error checking admin status: %w", err)
	}

	if !isAdmin {
		return nil, fmt.Errorf("only admins can view delete history")
	}

	// ดึงประวัติการลบ
	history, err := s.messageRepo.GetDeleteHistory(messageID)
	if err != nil {
		return nil, fmt.Errorf("error fetching delete history: %w", err)
	}

	// เพิ่มข้อมูลเพิ่มเติมให้แต่ละรายการ
	for _, deletion := range history {
		// ดึงข้อมูลผู้ลบ
		deleter, err := s.userRepo.FindByID(deletion.DeletedBy)
		if err == nil && deleter != nil {
			// สร้าง metadata ใหม่ที่มีข้อมูลเพิ่มเติม
			metadataMap := types.JSONB{}

			// ถ้ามี Metadata เดิม ให้คัดลอกค่าเดิมมาก่อน
			for k, v := range deletion.Metadata {
				metadataMap[k] = v
			}

			// เพิ่มข้อมูลผู้ลบ
			metadataMap["deleter_name"] = deleter.DisplayName
			if metadataMap["deleter_name"] == "" {
				metadataMap["deleter_name"] = deleter.Username
			}
			metadataMap["deleter_avatar"] = deleter.ProfileImageURL

			// บันทึกกลับไปที่ metadata
			deletion.Metadata = metadataMap
		}
	}

	return history, nil
}
