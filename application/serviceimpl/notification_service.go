// application/serviceimpl/notification_service.go
package serviceimpl

import (
	"encoding/json"
	"fmt"

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

	// สร้าง MessageDTO พื้นฐาน
	messageDTO := &dto.MessageDTO{
		ID:                message.ID,
		ConversationID:    message.ConversationID,
		SenderID:          message.SenderID,
		SenderType:        message.SenderType,
		MessageType:       message.MessageType,
		Content:           message.Content,
		MediaURL:          message.MediaURL,
		MediaThumbnailURL: message.MediaThumbnailURL,
		Metadata:          message.Metadata,
		CreatedAt:         message.CreatedAt,
		UpdatedAt:         message.UpdatedAt,
		IsDeleted:         message.IsDeleted,
		IsEdited:          message.IsEdited,
		EditCount:         message.EditCount,
		BusinessID:        message.BusinessID,
		ReplyToID:         message.ReplyToID,
		IsRead:            true, // ผู้ส่งอ่านแล้ว
		ReadCount:         1,    // เริ่มต้นที่ 1 (ผู้ส่ง)
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

	// เพิ่มข้อมูลธุรกิจ (ถ้ามี)
	if message.BusinessID != nil {
		if err == nil && business != nil {
			messageDTO.BusinessInfo = &dto.BusinessBasicDTO{
				ID:      business.ID,
				Name:    business.Name,
				LogoURL: business.ProfileImageURL,
			}
		}
	}

	// เพิ่มข้อมูลการตอบกลับ (ถ้ามี)
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
			if replyMsg.SenderType == "business" && replyMsg.BusinessID != nil {
				// ถ้าเป็นข้อความจากธุรกิจ ใช้ชื่อธุรกิจเป็น sender_name
				if err == nil && business != nil {
					replyInfo.SenderName = business.Name
				} else {
					// กรณีหาข้อมูลธุรกิจไม่พบ ใช้ชื่อธุรกิจจากข้อความปัจจุบัน (ถ้ามี)
					if message.BusinessID != nil && *message.BusinessID == *replyMsg.BusinessID &&
						messageDTO.BusinessInfo != nil {
						replyInfo.SenderName = messageDTO.BusinessInfo.Name
					}
				}
			} else if replyMsg.SenderID != nil {
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

	data, _ := json.MarshalIndent(messageDTO, "", "  ")
	fmt.Println("[DEBUGXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX] CHECK REPLY TO MESSAGE messageDTO:", string(data))

	// ส่งแจ้งเตือนผ่าน WebSocket
	s.wsPort.BroadcastNewMessage(message.ConversationID, messageDTO)
}

// NotifyMessageRead แจ้งเตือนการอ่านข้อความ
func (s *notificationService) NotifyMessageRead(conversationID uuid.UUID, message interface{}) {
	s.wsPort.BroadcastMessageRead(conversationID, message)
}

// NotifyMessageReadAll แจ้งเตือนการอ่านข้อความทั้งหมด
func (s *notificationService) NotifyMessageReadAll(conversationID uuid.UUID, message interface{}) {
	s.wsPort.BroadcastMessageReadAll(conversationID, message)
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

	// สร้างข้อมูลพื้นฐานสำหรับการแจ้งเตือน
	notificationData := map[string]interface{}{
		"request_id":   friendshipData.ID.String(),
		"user_id":      friendshipData.UserID.String(),
		"friend_id":    friendshipData.FriendID.String(),
		"status":       friendshipData.Status,
		"requested_at": friendshipData.RequestedAt,
	}

	// ดึงข้อมูลผู้ส่งคำขอเพิ่มเติม
	sender, err := s.userRepo.FindByID(friendshipData.UserID)
	if err == nil && sender != nil {
		notificationData["sender"] = map[string]interface{}{
			"id":                sender.ID.String(),
			"username":          sender.Username,
			"display_name":      sender.DisplayName,
			"profile_image_url": sender.ProfileImageURL,
		}
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

	// สร้างข้อมูลพื้นฐานสำหรับการแจ้งเตือน
	notificationData := map[string]interface{}{
		"friendship_id": friendshipData.ID.String(),
		"user_id":       friendshipData.UserID.String(),
		"friend_id":     friendshipData.FriendID.String(),
		"status":        friendshipData.Status,
		"accepted_at":   friendshipData.UpdatedAt,
	}

	// ดึงข้อมูลผู้ยอมรับคำขอ (friend)
	acceptor, err := s.userRepo.FindByID(friendshipData.FriendID)
	if err == nil && acceptor != nil {
		notificationData["acceptor"] = map[string]interface{}{
			"id":                acceptor.ID.String(),
			"username":          acceptor.Username,
			"display_name":      acceptor.DisplayName,
			"profile_image_url": acceptor.ProfileImageURL,
			"last_active_at":    acceptor.LastActiveAt,
		}
	}

	// ส่งการแจ้งเตือนไปยังผู้ส่งคำขอเดิม (userID)
	return s.wsPort.BroadcastFriendRequestAccepted(friendshipData.UserID, notificationData)
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
