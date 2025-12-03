// application/serviceimpl/notification_service.go
package serviceimpl

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/dto"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/port"
	"github.com/thizplus/gofiber-chat-api/domain/repository"
	"github.com/thizplus/gofiber-chat-api/domain/service"
)

// notificationService เป็น implementation ของ NotificationService interface
type notificationService struct {
	wsPort              port.WebSocketPort
	userRepo            repository.UserRepository
	messageRepo         repository.MessageRepository
	conversationRepo    repository.ConversationRepository
}

// NewNotificationService สร้าง instance ใหม่ของ NotificationService
func NewNotificationService(
	wsPort port.WebSocketPort,
	userRepo repository.UserRepository,
	messageRepo repository.MessageRepository,
	conversationRepo repository.ConversationRepository,
) service.NotificationService {
	return &notificationService{
		wsPort:              wsPort,
		userRepo:            userRepo,
		messageRepo:         messageRepo,
		conversationRepo:    conversationRepo,
	}
}

// =========== Message Notifications ===========

// NotifyNewMessage แจ้งเตือนข้อความใหม่
func (s *notificationService) NotifyNewMessage(conversationID uuid.UUID, messageData interface{}) {
	// แปลงข้อมูลเป็น models.Message ถ้าเป็นไปได้
	message, ok := messageData.(*models.Message)
	if !ok {
		// ถ้าไม่สามารถแปลงได้ ส่งข้อมูลไปตรงๆ
		s.wsPort.BroadcastNewMessage(conversationID, messageData)
		return
	}

	// คำนวณ read_count จากฐานข้อมูล
	readCount := 1 // เริ่มต้นที่ 1 (ผู้ส่งอ่านเอง)
	if message.ID != uuid.Nil {
		// นับจำนวนคนที่อ่านข้อความจริงๆ จาก database
		reads, err := s.messageRepo.GetReads(message.ID)
		if err == nil && len(reads) > 0 {
			readCount = len(reads)
		}
	}

	// คำนวณ status จาก read_count
	status := "sent" // default: ส่งสำเร็จแล้ว
	if readCount >= 2 {
		status = "read" // มีคนอ่านแล้ว (นอกจากผู้ส่ง)
	}

	// ดึง temp_id จาก metadata ถ้ามี (JSONB เป็น map[string]interface{} อยู่แล้ว)
	tempID := ""
	if message.Metadata != nil {
		if val, ok := message.Metadata["tempId"].(string); ok {
			tempID = val
		} else if val, ok := message.Metadata["temp_id"].(string); ok {
			tempID = val
		}
	}

	// สร้าง MessageDTO พื้นฐาน
	messageDTO := &dto.MessageDTO{
		ID:                message.ID,
		TempID:            tempID,
		ConversationID:    message.ConversationID,
		SenderID:          message.SenderID,
		SenderType:        message.SenderType,
		MessageType:       message.MessageType,
		Content:           message.Content,
		MediaURL:          message.MediaURL,
		MediaThumbnailURL: message.MediaThumbnailURL,
		AlbumFiles:        message.AlbumFiles,  // Copy album_files สำหรับ album messages
		Metadata:          message.Metadata,
		CreatedAt:         message.CreatedAt,
		UpdatedAt:         message.UpdatedAt,
		IsDeleted:         message.IsDeleted,
		IsEdited:          message.IsEdited,
		EditCount:         message.EditCount,
		ReplyToID:         message.ReplyToID,
		IsRead:            readCount >= 1,
		ReadCount:         readCount,
		Status:            status,
	}

	// ดึง file info และ sticker info จาก metadata ถ้ามี
	if message.Metadata != nil {
		if fileName, ok := message.Metadata["file_name"].(string); ok {
			messageDTO.FileName = fileName
		}
		if fileSize, ok := message.Metadata["file_size"].(float64); ok {
			messageDTO.FileSize = int64(fileSize)
		}
		if fileType, ok := message.Metadata["file_type"].(string); ok {
			messageDTO.FileType = fileType
		}
		if stickerIDStr, ok := message.Metadata["sticker_id"].(string); ok {
			if stickerID, err := uuid.Parse(stickerIDStr); err == nil {
				messageDTO.StickerID = &stickerID
			}
		}
		if stickerSetIDStr, ok := message.Metadata["sticker_set_id"].(string); ok {
			if stickerSetID, err := uuid.Parse(stickerSetIDStr); err == nil {
				messageDTO.StickerSetID = &stickerSetID
			}
		}
	}

	// เพิ่มข้อมูลผู้ส่ง
	if message.SenderID != nil {
		sender, err := s.userRepo.FindByID(*message.SenderID)
		if err == nil && sender != nil {
			messageDTO.SenderName = sender.DisplayName
			if messageDTO.SenderName == "" {
				messageDTO.SenderName = sender.Username
			}
			messageDTO.SenderAvatar = sender.ProfileImageURL

			// สร้าง UserBasicDTO
			messageDTO.SenderInfo = &dto.UserBasicDTO{
				ID:              sender.ID,
				Username:        sender.Username,
				DisplayName:     sender.DisplayName,
				ProfileImageURL: sender.ProfileImageURL,
			}
		}
	}


	// เพิ่มข้อมูลการตอบกลับ (ถ้ามี)
	if message.ReplyToID != nil {
		replyMsg, err := s.messageRepo.GetByID(*message.ReplyToID)
		if err == nil && replyMsg != nil {
			replyInfo := &dto.ReplyInfoDTO{
				ID:          replyMsg.ID.String(),
				MessageType: replyMsg.MessageType,
				Content:     replyMsg.Content,
			}

			// ตรวจสอบประเภทผู้ส่งข้อความที่ถูกตอบกลับ
			if replyMsg.SenderID != nil {
				// ถ้าเป็นข้อความจากผู้ใช้ทั่วไป ใช้ชื่อผู้ใช้เป็น sender_name
				replySender, err := s.userRepo.FindByID(*replyMsg.SenderID)
				if err == nil && replySender != nil {
					if replySender.DisplayName != "" {
						replyInfo.SenderName = replySender.DisplayName
					} else {
						replyInfo.SenderName = replySender.Username
					}
				}
			}

			messageDTO.ReplyToMessage = replyInfo
		}
	}

	// เพิ่มข้อมูล Forward (ถ้ามี)
	messageDTO.IsForwarded = message.IsForwarded
	if message.IsForwarded && message.ForwardedFrom != nil {
		forwardedFrom := &dto.ForwardedFromDTO{}

		if msgID, ok := message.ForwardedFrom["message_id"].(string); ok {
			forwardedFrom.MessageID = msgID
		}
		if senderID, ok := message.ForwardedFrom["sender_id"].(string); ok {
			forwardedFrom.SenderID = senderID
		}
		if senderName, ok := message.ForwardedFrom["sender_name"].(string); ok {
			forwardedFrom.SenderName = senderName
		}
		if convID, ok := message.ForwardedFrom["conversation_id"].(string); ok {
			forwardedFrom.ConversationID = convID
		}
		if timestamp, ok := message.ForwardedFrom["original_timestamp"].(string); ok {
			if parsedTime, err := time.Parse(time.RFC3339, timestamp); err == nil {
				forwardedFrom.OriginalTimestamp = parsedTime
			}
		}

		messageDTO.ForwardedFrom = forwardedFrom
	}

	data, _ := json.MarshalIndent(messageDTO, "", "  ")
	fmt.Println("[DEBUGXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX] CHECK REPLY TO MESSAGE messageDTO:", string(data))

	// ส่งแจ้งเตือนผ่าน WebSocket
	s.wsPort.BroadcastNewMessage(message.ConversationID, messageDTO)
}

// NotifyMessageRead แจ้งเตือนการอ่านข้อความ (เก่า - broadcast ไปทุกคน)
func (s *notificationService) NotifyMessageRead(conversationID uuid.UUID, message interface{}) {
	s.wsPort.BroadcastMessageRead(conversationID, message)
}

// NotifyMessageReadAll แจ้งเตือนการอ่านข้อความทั้งหมด (เก่า - broadcast ไปทุกคน)
func (s *notificationService) NotifyMessageReadAll(conversationID uuid.UUID, message interface{}) {
	s.wsPort.BroadcastMessageReadAll(conversationID, message)
}

// NotifyMessageReadToSender ส่ง message.read event ไปยังผู้ส่งข้อความเท่านั้น (ใช้สำหรับ group chat)
func (s *notificationService) NotifyMessageReadToSender(senderID uuid.UUID, message interface{}) {
	s.wsPort.SendMessageReadToSender(senderID, message)
}

// NotifyMessageReadAllToUser ส่ง message.read_all event ไปยัง user ที่อ่าน (สำหรับ multi-device sync)
func (s *notificationService) NotifyMessageReadAllToUser(userID uuid.UUID, message interface{}) {
	s.wsPort.SendMessageReadAllToUser(userID, message)
}

// NotifyMessageDelivered แจ้งเตือนการส่งข้อความสำเร็จ
func (s *notificationService) NotifyMessageDelivered(conversationID uuid.UUID, message interface{}) {
	s.wsPort.BroadcastMessageDelivered(conversationID, message)
}

// NotifyMessageEdited แจ้งเตือนการแก้ไขข้อความ
func (s *notificationService) NotifyMessageEdited(conversationID uuid.UUID, message interface{}) {
	s.wsPort.BroadcastMessageEdited(conversationID, message)
}

// NotifyMessageReply แจ้งเตือนการตอบกลับข้อความ
func (s *notificationService) NotifyMessageReply(conversationID uuid.UUID, message interface{}) {
	s.wsPort.BroadcastMessageReply(conversationID, message)
}

// NotifyMessageDeleted แจ้งเตือนการลบข้อความ
func (s *notificationService) NotifyMessageDeleted(conversationID uuid.UUID, messageID uuid.UUID) {
	s.wsPort.BroadcastMessageDeleted(conversationID, messageID)
}

// NotifyMessageReaction แจ้งเตือนการแสดงความรู้สึกต่อข้อความ
func (s *notificationService) NotifyMessageReaction(conversationID uuid.UUID, reaction interface{}) {
	s.wsPort.BroadcastMessageReaction(conversationID, reaction)
}

// =========== Conversation Notifications ===========

// NotifyConversationCreated แจ้งเตือนการสร้างการสนทนาใหม่

func (s *notificationService) NotifyConversationCreated(userIDs []uuid.UUID, conversation interface{}) error {
	// ตรวจสอบประเภทข้อมูล
	conversationData, ok := conversation.(*dto.ConversationDTO)
	if !ok {
		return fmt.Errorf("invalid conversation type, expected *dto.ConversationDTO")
	}

	// สำหรับการสนทนาแบบตัวต่อตัว (direct)
	if conversationData.Type == "direct" {
		// ดึงข้อมูลผู้สร้าง
		var creator *models.User
		if conversationData.CreatorID != nil {
			var err error
			creator, err = s.userRepo.FindByID(*conversationData.CreatorID)
			if err != nil {
				return fmt.Errorf("failed to get creator info: %w", err)
			}
		}

		// ดึงข้อมูลผู้รับ
		var contactUser *models.User
		if contactUserIDStr, ok := conversationData.ContactInfo["user_id"].(string); ok {
			contactUserID, err := uuid.Parse(contactUserIDStr)
			if err == nil {
				contactUser, err = s.userRepo.FindByID(contactUserID)
				if err != nil {
					return fmt.Errorf("failed to get contact user info: %w", err)
				}
			}
		}

		// แยกผู้ใช้ออกเป็นสองกลุ่ม: ผู้สร้างและผู้รับ
		creatorIDGroup := []uuid.UUID{}
		contactIDGroup := []uuid.UUID{}

		for _, userID := range userIDs {
			if creator != nil && creator.ID == userID {
				creatorIDGroup = append(creatorIDGroup, userID)
			} else if contactUser != nil && contactUser.ID == userID {
				contactIDGroup = append(contactIDGroup, userID)
			}
		}

		// สร้างข้อมูลสำหรับผู้สร้าง (แสดงข้อมูลของผู้รับ)
		if len(creatorIDGroup) > 0 && contactUser != nil {
			creatorView := *conversationData // Clone ข้อมูลเดิม
			creatorView.Title = contactUser.DisplayName
			creatorView.IconURL = contactUser.ProfileImageURL

			creatorView.ContactInfo = map[string]interface{}{
				"user_id":           contactUser.ID.String(),
				"username":          contactUser.Username,
				"display_name":      contactUser.DisplayName,
				"profile_image_url": contactUser.ProfileImageURL,
			}

			// ส่งข้อมูลไปยังผู้สร้าง
			err := s.wsPort.BroadcastConversationCreated(creatorIDGroup, &creatorView)
			if err != nil {
				return fmt.Errorf("failed to broadcast to creators: %w", err)
			}
		}

		// สร้างข้อมูลสำหรับผู้รับ (แสดงข้อมูลของผู้สร้าง)
		if len(contactIDGroup) > 0 && creator != nil {
			contactView := *conversationData // Clone ข้อมูลเดิม
			contactView.Title = creator.DisplayName
			contactView.IconURL = creator.ProfileImageURL

			contactView.ContactInfo = map[string]interface{}{
				"user_id":           creator.ID.String(),
				"username":          creator.Username,
				"display_name":      creator.DisplayName,
				"profile_image_url": creator.ProfileImageURL,
			}

			// ส่งข้อมูลไปยังผู้รับ
			err := s.wsPort.BroadcastConversationCreated(contactIDGroup, &contactView)
			if err != nil {
				return fmt.Errorf("failed to broadcast to contacts: %w", err)
			}
		}

		return nil
	} else {
		// สำหรับการสนทนาแบบกลุ่ม ไม่ต้องปรับแต่งข้อมูล
		return s.wsPort.BroadcastConversationCreated(userIDs, conversation)
	}
}

// NotifyConversationUpdated แจ้งเตือนการอัปเดตการสนทนา
func (s *notificationService) NotifyConversationUpdated(conversationID uuid.UUID, update interface{}) {
	s.wsPort.BroadcastConversationUpdated(conversationID, update)
}

// NotifyConversationUpdatedToUser ส่ง conversation.update ไปยัง user คนใดคนหนึ่ง (personalized)
func (s *notificationService) NotifyConversationUpdatedToUser(userID uuid.UUID, update interface{}) {
	s.wsPort.BroadcastToUser(userID, "conversation.update", update)
}

// NotifyConversationDeleted แจ้งเตือนการลบการสนทนา
func (s *notificationService) NotifyConversationDeleted(conversationID uuid.UUID, memberIDs []uuid.UUID) {
	s.wsPort.BroadcastConversationDeleted(conversationID, memberIDs)
}

// NotifyUserAddedToConversation แจ้งเตือนการเพิ่มผู้ใช้เข้าการสนทนา
func (s *notificationService) NotifyUserAddedToConversation(conversationID uuid.UUID, userID uuid.UUID) {
	s.wsPort.BroadcastUserAddedToConversation(conversationID, userID)
}

// NotifyUserRemovedFromConversation แจ้งเตือนการลบผู้ใช้ออกจากการสนทนา
func (s *notificationService) NotifyUserRemovedFromConversation(userID uuid.UUID, conversationID uuid.UUID) {
	s.wsPort.BroadcastUserRemovedFromConversation(userID, conversationID)
}

// NotifyNewConversation แจ้งเตือนการสนทนาใหม่
func (s *notificationService) NotifyNewConversation(conversation interface{}) error {
	conversationData, ok := conversation.(*dto.ConversationDTO)
	if !ok {
		return fmt.Errorf("invalid conversation type, expected *dto.ConversationDTO")
	}

	// ตรวจสอบว่า ContactInfo มีอยู่จริง
	if conversationData.ContactInfo == nil {
		return fmt.Errorf("conversation has no contact info")
	}

	// ตรวจสอบว่ามี user_id ใน ContactInfo หรือไม่
	userIDValue, exists := conversationData.ContactInfo["user_id"]
	if !exists {
		return fmt.Errorf("contact info does not contain user_id")
	}

	// ตรวจสอบประเภทของ user_id และแปลงเป็น UUID
	var userID uuid.UUID

	switch v := userIDValue.(type) {
	case string:
		var err error
		userID, err = uuid.Parse(v)
		if err != nil {
			return fmt.Errorf("invalid user ID format: %v", err)
		}
	case uuid.UUID:
		userID = v
	default:
		return fmt.Errorf("user_id is not in a valid format, got %T", userIDValue)
	}

	return s.wsPort.BroadcastNewConversation(userID, conversation)
}

// =========== Business Notifications ===========






// =========== Friend Notifications ===========

// NotifyFriendRequestReceived แจ้งเตือนการได้รับคำขอเป็นเพื่อน
func (s *notificationService) NotifyFriendRequestReceived(request interface{}) error {

	friendshipData, ok := request.(*models.UserFriendship)
	if !ok {
		return fmt.Errorf("invalid request type, expected *models.UserFriendship")
	}

	// ดึงข้อมูลผู้ส่งคำขอ
	sender, err := s.userRepo.FindByID(friendshipData.UserID)
	if err != nil {
		return fmt.Errorf("failed to get sender info: %w", err)
	}

	// สร้างข้อมูลตาม spec
	notificationData := map[string]interface{}{
		"request_id": friendshipData.ID.String(),
		"from": map[string]interface{}{
			"id":                sender.ID.String(),
			"username":          sender.Username,
			"display_name":      sender.DisplayName,
			"profile_image_url": sender.ProfileImageURL,
		},
		"created_at": friendshipData.RequestedAt.Format(time.RFC3339),
	}

	return s.wsPort.BroadcastFriendRequestReceived(friendshipData.FriendID, notificationData)
}

// NotifyFriendRequestAccepted แจ้งเตือนการยอมรับคำขอเป็นเพื่อน
func (s *notificationService) NotifyFriendRequestAccepted(friendship interface{}) error {
	// แปลง interface{} เป็น *models.UserFriendship
	friendshipData, ok := friendship.(*models.UserFriendship)
	if !ok {
		return fmt.Errorf("invalid friendship type, expected *models.UserFriendship")
	}

	// ดึงข้อมูลผู้ยอมรับคำขอ (friend)
	acceptor, err := s.userRepo.FindByID(friendshipData.FriendID)
	if err != nil {
		return fmt.Errorf("failed to get acceptor info: %w", err)
	}

	// สร้างข้อมูลตาม spec
	notificationData := map[string]interface{}{
		"request_id": friendshipData.ID.String(),
		"by": map[string]interface{}{
			"id":                acceptor.ID.String(),
			"username":          acceptor.Username,
			"display_name":      acceptor.DisplayName,
			"profile_image_url": acceptor.ProfileImageURL,
		},
		"accepted_at": friendshipData.UpdatedAt.Format(time.RFC3339),
	}

	// ส่งการแจ้งเตือนไปยังผู้ส่งคำขอเดิม (userID)
	return s.wsPort.BroadcastFriendRequestAccepted(friendshipData.UserID, notificationData)
}

// NotifyFriendRequestRejected แจ้งเตือนการปฏิเสธคำขอเป็นเพื่อน
func (s *notificationService) NotifyFriendRequestRejected(friendship interface{}) error {
	// แปลง interface{} เป็น *models.UserFriendship
	friendshipData, ok := friendship.(*models.UserFriendship)
	if !ok {
		return fmt.Errorf("invalid friendship type, expected *models.UserFriendship")
	}

	// สร้างข้อมูลตาม spec (เฉพาะ request_id และ rejected_at)
	notificationData := map[string]interface{}{
		"request_id":  friendshipData.ID.String(),
		"rejected_at": friendshipData.UpdatedAt.Format(time.RFC3339),
	}

	// ส่งการแจ้งเตือนไปยังผู้ส่งคำขอเดิม (userID)
	return s.wsPort.BroadcastFriendRequestRejected(friendshipData.UserID, notificationData)
}

// NotifyFriendRemoved แจ้งเตือนการลบเพื่อน
func (s *notificationService) NotifyFriendRemoved(userID uuid.UUID, friendID uuid.UUID) {
	s.wsPort.BroadcastFriendRemoved(userID, friendID)
}

// =========== User Notifications ===========

// NotifyUserBlocked แจ้งเตือนการบล็อกผู้ใช้
func (s *notificationService) NotifyUserBlocked(blockerID uuid.UUID, blockedID uuid.UUID) {
	s.wsPort.BroadcastUserBlocked(blockerID, blockedID)
}

// NotifyUserUnblocked แจ้งเตือนการยกเลิกการบล็อกผู้ใช้
func (s *notificationService) NotifyUserUnblocked(unblockerID uuid.UUID, unblockedID uuid.UUID) {
	s.wsPort.BroadcastUserUnblocked(unblockerID, unblockedID)
}

// =========== General Notifications ===========

// SendNotification ส่งการแจ้งเตือนทั่วไปไปยังผู้ใช้หลายคน
func (s *notificationService) SendNotification(userIDs []uuid.UUID, notification interface{}) {
	s.wsPort.BroadcastNotification(userIDs, notification)
}

// SendAlert ส่งการแจ้งเตือนสำคัญไปยังผู้ใช้
func (s *notificationService) SendAlert(userID uuid.UUID, alert interface{}) {
	s.wsPort.BroadcastAlert(userID, alert)
}

// NotifySystemMessage ส่งข้อความระบบไปยังผู้ใช้หลายคน
func (s *notificationService) NotifySystemMessage(userIDs []uuid.UUID, message interface{}) {
	s.wsPort.BroadcastSystemMessage(userIDs, message)
}

// =========== Member Role Notifications ===========

// NotifyMemberRoleChanged แจ้งเตือนการเปลี่ยนแปลง role ของสมาชิก
func (s *notificationService) NotifyMemberRoleChanged(conversationID, userID uuid.UUID, oldRole, newRole string, changedByUserID uuid.UUID) {
	// ดึงข้อมูลผู้ใช้ที่ถูกเปลี่ยน role
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		// ถ้าไม่พบผู้ใช้ ให้ส่งข้อมูลพื้นฐาน
		notificationData := map[string]interface{}{
			"conversation_id":   conversationID.String(),
			"user_id":          userID.String(),
			"old_role":         oldRole,
			"new_role":         newRole,
			"changed_by":       changedByUserID.String(),
			"changed_at":       time.Now().Format(time.RFC3339),
		}
		s.wsPort.BroadcastMemberRoleChanged(conversationID, notificationData)
		return
	}

	// ดึงข้อมูลผู้ที่ทำการเปลี่ยน
	changedBy, _ := s.userRepo.FindByID(changedByUserID)

	// สร้างข้อมูลการแจ้งเตือน
	notificationData := map[string]interface{}{
		"conversation_id": conversationID.String(),
		"user": map[string]interface{}{
			"id":                userID.String(),
			"username":          user.Username,
			"display_name":      user.DisplayName,
			"profile_image_url": user.ProfileImageURL,
		},
		"old_role":   oldRole,
		"new_role":   newRole,
		"changed_at": time.Now().Format(time.RFC3339),
	}

	if changedBy != nil {
		notificationData["changed_by"] = map[string]interface{}{
			"id":                changedByUserID.String(),
			"username":          changedBy.Username,
			"display_name":      changedBy.DisplayName,
			"profile_image_url": changedBy.ProfileImageURL,
		}
	}

	// ส่ง notification ไปยังสมาชิกทุกคนในกลุ่ม
	s.wsPort.BroadcastMemberRoleChanged(conversationID, notificationData)
}

// NotifyOwnershipTransferred แจ้งเตือนการโอนความเป็นเจ้าของ
func (s *notificationService) NotifyOwnershipTransferred(conversationID, previousOwnerID, newOwnerID uuid.UUID) {
	// ดึงข้อมูล owner เดิม
	previousOwner, err := s.userRepo.FindByID(previousOwnerID)
	if err != nil {
		previousOwner = nil
	}

	// ดึงข้อมูล owner ใหม่
	newOwner, err := s.userRepo.FindByID(newOwnerID)
	if err != nil {
		newOwner = nil
	}

	// สร้างข้อมูลการแจ้งเตือน
	notificationData := map[string]interface{}{
		"conversation_id":    conversationID.String(),
		"previous_owner_id": previousOwnerID.String(),
		"new_owner_id":      newOwnerID.String(),
		"transferred_at":    time.Now().Format(time.RFC3339),
	}

	if previousOwner != nil {
		notificationData["previous_owner"] = map[string]interface{}{
			"id":                previousOwnerID.String(),
			"username":          previousOwner.Username,
			"display_name":      previousOwner.DisplayName,
			"profile_image_url": previousOwner.ProfileImageURL,
		}
	}

	if newOwner != nil {
		notificationData["new_owner"] = map[string]interface{}{
			"id":                newOwnerID.String(),
			"username":          newOwner.Username,
			"display_name":      newOwner.DisplayName,
			"profile_image_url": newOwner.ProfileImageURL,
		}
	}

	// ส่ง notification ไปยังสมาชิกทุกคนในกลุ่ม
	s.wsPort.BroadcastOwnershipTransferred(conversationID, notificationData)
}

// NotifyNewActivity แจ้งเตือน activity ใหม่ในกลุ่ม
func (s *notificationService) NotifyNewActivity(conversationID uuid.UUID, activity *dto.ActivityDTO) {
	// ส่ง activity ไปยังสมาชิกทุกคนในกลุ่ม
	s.wsPort.BroadcastNewActivity(conversationID, activity)
}

// =========== Customer Profile Notifications ===========

// NotifyProfileUpdate แจ้งเตือนการอัพเดทโปรไฟล์ลูกค้า
func (s *notificationService) NotifyProfileUpdate(businessID, userID uuid.UUID, profile interface{}) {
	s.wsPort.BroadcastProfileUpdate(businessID, userID, profile)

}

// NotifyProfileUpdateTags แจ้งเตือนการอัพเดทแท็กของลูกค้า
func (s *notificationService) NotifyProfileUpdateTags(businessID, userID uuid.UUID, tagId uuid.UUID, action string) {
	// สร้าง payload ที่มีข้อมูลครบถ้วน
	payload := map[string]interface{}{
		"user_id":     userID.String(),
		"business_id": businessID.String(),
		"tag_id":      tagId.String(),
		"action":      action, // "add" หรือ "remove"
	}

	// ส่งข้อมูลไปยังทุกคนในธุรกิจโดยใช้ BroadcastToBusiness โดยตรง
	s.wsPort.BroadcastProfileUpdateTags(businessID, userID, payload)
}
