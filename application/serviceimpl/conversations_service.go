// application/serviceimpl/conversation_service.go
package serviceimpl

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/dto"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/repository"
	"github.com/thizplus/gofiber-chat-api/domain/service"
	"github.com/thizplus/gofiber-chat-api/domain/types"
)

type conversationService struct {
	conversationRepo    repository.ConversationRepository
	userRepo            repository.UserRepository
	messageRepo         repository.MessageRepository
}

// NewConversationService สร้าง service ใหม่
func NewConversationService(
	conversationRepo repository.ConversationRepository,
	userRepo repository.UserRepository,
	messageRepo repository.MessageRepository,

) service.ConversationService {
	return &conversationService{
		conversationRepo:    conversationRepo,
		userRepo:            userRepo,
		messageRepo:         messageRepo,
	}
}

// CreateDirectConversation สร้างการสนทนาแบบส่วนตัวระหว่างผู้ใช้สองคน
func (s *conversationService) CreateDirectConversation(userID, friendID uuid.UUID) (*dto.ConversationDTO, error) {

	// 1. ตรวจสอบว่าผู้ใช้ที่จะสนทนาด้วยมีอยู่จริงไหม
	friend, err := s.userRepo.FindByID(friendID)
	if err != nil || friend == nil {
		return nil, errors.New("friend not found")
	}

	// 2. ตรวจสอบความเป็นเพื่อน (เพิ่มความเข้มงวด)
	isFriend, err := s.checkFriendship(userID, friendID)
	if err != nil {
		return nil, err
	}
	if !isFriend {
		return nil, errors.New("you must be friends to start a chat")
	}

	// 3. ตรวจสอบว่ามีการสนทนาอยู่แล้วหรือไม่
	existingConv, err := s.conversationRepo.FindDirectConversation(userID, friendID)
	if err == nil && existingConv != nil {
		// ถ้ามีการสนทนาอยู่แล้ว ดึงข้อมูลการสนทนาและส่งกลับ
		return s.convertToConversationDTO(existingConv, userID)
	}

	// 4. สร้างการสนทนาใหม่
	now := time.Now()
	conversation := &models.Conversation{
		ID:        uuid.New(),
		Type:      "direct",
		CreatedAt: now,
		UpdatedAt: now,
		CreatorID: &userID,
		IsActive:  true,
	}

	if err := s.conversationRepo.Create(conversation); err != nil {
		return nil, err
	}

	// เพิ่มผู้สร้างเป็นสมาชิก
	member1 := &models.ConversationMember{
		ID:             uuid.New(),
		ConversationID: conversation.ID,
		UserID:         userID,
		IsAdmin:        true,
		JoinedAt:       now,
	}
	if err := s.conversationRepo.AddMember(member1); err != nil {
		return nil, err
	}

	// เพิ่มผู้ใช้อีกคนเป็นสมาชิก
	member2 := &models.ConversationMember{
		ID:             uuid.New(),
		ConversationID: conversation.ID,
		UserID:         friendID,
		IsAdmin:        false,
		JoinedAt:       now,
	}
	if err := s.conversationRepo.AddMember(member2); err != nil {
		// ข้อผิดพลาดไม่ร้ายแรง แต่เราควรบันทึกลงในล็อก
	}

	// สร้างข้อความระบบแจ้งการสร้างการสนทนา
	welcomeMessageText := "Conversation created."
	err = s.createSystemMessage(conversation.ID, welcomeMessageText)
	if err != nil {
		// ไม่คืนค่าข้อผิดพลาด แต่ควรบันทึกลงในล็อก
	}

	// ดึงข้อมูลการสนทนาที่สร้างเสร็จแล้ว
	createdConv, err := s.conversationRepo.GetByID(conversation.ID)
	if err != nil {
		return nil, err
	}

	// แปลงเป็น DTO สำหรับผู้สร้าง
	creatorDTO, err := s.convertToConversationDTO(createdConv, userID)
	if err != nil {
		return nil, err
	}

	return creatorDTO, nil
}

// GetUserConversations ดึงรายการการสนทนาทั้งหมดของผู้ใช้ พร้อมตัวกรอง
func (s *conversationService) GetUserConversations(userID uuid.UUID, limit, offset int,
	convType string, pinned bool) ([]*dto.ConversationDTO, int, error) {

	// เรียกใช้ repository
	conversations, total, err := s.conversationRepo.GetUserConversationsWithFilter(
		userID, limit, offset, convType, pinned)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]*dto.ConversationDTO, 0, len(conversations))
	filteredCount := 0 // นับจำนวนที่ถูกกรอง

	for _, conversation := range conversations {
		dto, err := s.convertToConversationDTO(conversation, userID)
		if err != nil {
			filteredCount++
			continue
		}
		dtos = append(dtos, dto)
	}

	// ปรับ total ให้ตรงกับจำนวนที่แสดงจริง
	adjustedTotal := total - filteredCount

	return dtos, adjustedTotal, nil
}

func (s *conversationService) convertToConversationDTO(conversation *models.Conversation, userID uuid.UUID) (*dto.ConversationDTO, error) {
	if conversation == nil {
		return nil, errors.New("conversation is nil")
	}


	convDTO := &dto.ConversationDTO{
		ID:              conversation.ID,
		Type:            conversation.Type,
		Title:           conversation.Title,
		IconURL:         conversation.IconURL,
		CreatedAt:       conversation.CreatedAt,
		UpdatedAt:       conversation.UpdatedAt,
		LastMessageText: conversation.LastMessageText,
		LastMessageAt:   conversation.LastMessageAt,
		CreatorID:       conversation.CreatorID,
		IsActive:        conversation.IsActive,
		Metadata:        conversation.Metadata,
	}

	// ดึงข้อมูลเพิ่มเติมตามประเภทการสนทนา
	if conversation.Type == "direct" {
		// ... โค้ดเดิมสำหรับ direct conversation
		members, err := s.conversationRepo.GetMembers(conversation.ID)
		if err == nil && len(members) > 0 {
			var otherMember *models.ConversationMember
			for _, member := range members {
				if member.UserID != userID {
					otherMember = member
					break
				}
			}

			if otherMember != nil {
				friend, err := s.userRepo.FindByID(otherMember.UserID)
				if err == nil && friend != nil {
					if convDTO.Title == "" {
						if friend.DisplayName != "" {
							convDTO.Title = friend.DisplayName
						} else {
							convDTO.Title = friend.Username
						}
					}

					if convDTO.IconURL == "" {
						convDTO.IconURL = friend.ProfileImageURL
					}

					contactInfo := types.JSONB{
						"user_id":           friend.ID.String(),
						"username":          friend.Username,
						"display_name":      friend.DisplayName,
						"profile_image_url": friend.ProfileImageURL,
					}
					convDTO.ContactInfo = contactInfo
				}
			}
		}
	}

	// ตรวจสอบสถานะ pin/mute
	member, err := s.conversationRepo.GetMember(conversation.ID, userID)
	if err == nil && member != nil {
		convDTO.IsPinned = member.IsPinned
		convDTO.IsMuted = member.IsMuted

		// คำนวณ unread_count
		var unreadCount int
		if member.LastReadAt != nil {
			messages, err := s.messageRepo.GetMessagesAfterTime(
				conversation.ID, *member.LastReadAt, userID)
			if err == nil {
				unreadCount = len(messages)
			}
		} else {
			messages, err := s.messageRepo.GetAllUnreadMessages(
				conversation.ID, userID)
			if err == nil {
				unreadCount = len(messages)
			}
		}

		convDTO.UnreadCount = unreadCount
	} else {
		convDTO.IsPinned = false
		convDTO.IsMuted = false
		convDTO.UnreadCount = 0
	}

	// จำนวนสมาชิก
	members, err := s.conversationRepo.GetMembers(conversation.ID)
	if err == nil {
		convDTO.MemberCount = len(members)
	} else {
		convDTO.MemberCount = 0
	}

	return convDTO, nil
}

func (s *conversationService) checkFriendship(userID, friendID uuid.UUID) (bool, error) {
	// ต้องมีฟังก์ชันในระบบของคุณสำหรับการตรวจสอบความเป็นเพื่อน
	// ในตัวอย่างนี้จะให้ค่าจริงเสมอ
	return true, nil
}

func (s *conversationService) createSystemMessage(conversationID uuid.UUID, content string) error {
	// ควรเรียกใช้ MessageRepository เพื่อสร้างข้อความระบบ
	// ในตัวอย่างนี้จะไม่ทำการสร้างข้อความจริง
	return nil
}

func (s *conversationService) getUserName(userID uuid.UUID) (string, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return "", err
	}

	if user.DisplayName != "" {
		return user.DisplayName, nil
	}
	return user.Username, nil
}

func (s *conversationService) userExists(userID uuid.UUID) (bool, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return false, err
	}
	return user != nil, nil
}

// application/serviceimpl/conversations_service.go

// CreateBusinessConversation สร้างการสนทนากับธุรกิจ



// 🔧 ต้องเพิ่ม repositories ใน conversationService struct
// type conversationService struct {
// 	conversationRepo     repository.ConversationRepository
// 	businessRepo         repository.BusinessRepository
// 	userRepo             repository.UserRepository
// 	// ... repositories อื่นๆ
// }

// application/serviceimpl/conversations_service.go
// เพิ่มเมธอด CreateGroupConversation

// CreateGroupConversation สร้างการสนทนาแบบกลุ่ม
func (s *conversationService) CreateGroupConversation(userID uuid.UUID, title, iconURL string, memberIDs []uuid.UUID) (*dto.ConversationDTO, error) {
	// 1. ตรวจสอบข้อมูลที่จำเป็น
	if title == "" {
		return nil, errors.New("group conversation requires a title")
	}

	// 2. ตรวจสอบว่ามีสมาชิกอย่างน้อย 1 คน (นอกเหนือจากผู้สร้าง)
	if len(memberIDs) == 0 {
		return nil, errors.New("at least one member is required for group conversation")
	}

	// 3. ตรวจสอบว่าสมาชิกทุกคนมีอยู่จริงและเป็นเพื่อนกับผู้สร้าง
	validMemberIDs := []uuid.UUID{}
	for _, memberID := range memberIDs {
		// ข้ามถ้าเป็น ID ของผู้สร้าง
		if memberID == userID {
			continue
		}

		// ตรวจสอบว่าผู้ใช้มีอยู่จริง
		user, err := s.userRepo.FindByID(memberID)
		if err != nil || user == nil {
			// ข้ามผู้ใช้ที่ไม่มีอยู่จริง
			continue
		}

		// ตรวจสอบความเป็นเพื่อน (เพิ่มความเข้มงวด)
		isFriend, err := s.checkFriendship(userID, memberID)
		if err != nil || !isFriend {
			// ข้ามผู้ใช้ที่ไม่ใช่เพื่อน
			continue
		}

		validMemberIDs = append(validMemberIDs, memberID)
	}

	// 4. ตรวจสอบว่ามีสมาชิกที่ถูกต้องอย่างน้อย 1 คน
	if len(validMemberIDs) == 0 {
		return nil, errors.New("no valid members found for group conversation")
	}

	// 5. สร้างการสนทนาใหม่
	now := time.Now()
	conversation := &models.Conversation{
		ID:        uuid.New(),
		Type:      "group",
		Title:     title,
		IconURL:   iconURL,
		CreatedAt: now,
		UpdatedAt: now,
		CreatorID: &userID,
		IsActive:  true,
	}

	if err := s.conversationRepo.Create(conversation); err != nil {
		return nil, err
	}

	// 6. เพิ่มผู้สร้างเป็นสมาชิกและ owner
	creator := &models.ConversationMember{
		ID:             uuid.New(),
		ConversationID: conversation.ID,
		UserID:         userID,
		Role:           models.RoleOwner, // ✅ ตั้งเป็น owner
		IsAdmin:        true,             // Keep for backward compatibility
		JoinedAt:       now,
	}
	if err := s.conversationRepo.AddMember(creator); err != nil {
		return nil, err
	}

	// 7. เพิ่มสมาชิกอื่นๆ (ที่ผ่านการตรวจสอบแล้ว)
	allMemberIDs := []uuid.UUID{userID} // เริ่มด้วยผู้สร้าง
	for _, memberID := range validMemberIDs {
		// เพิ่มเป็นสมาชิก
		member := &models.ConversationMember{
			ID:             uuid.New(),
			ConversationID: conversation.ID,
			UserID:         memberID,
			Role:           models.RoleMember, // ✅ ตั้งเป็น member
			IsAdmin:        false,
			JoinedAt:       now,
		}
		if err := s.conversationRepo.AddMember(member); err != nil {
			// ไม่คืนค่าข้อผิดพลาด แต่ควรบันทึกลงในล็อก
			continue
		}
		allMemberIDs = append(allMemberIDs, memberID)
	}

	// 8. สร้างข้อความระบบแจ้งการสร้างกลุ่ม
	welcomeMessageText := "Group created."
	creatorName, err := s.getUserName(userID)
	if err == nil && creatorName != "" {
		welcomeMessageText = creatorName + " created the group."
	}
	s.createSystemMessage(conversation.ID, welcomeMessageText)

	// 9. ดึงข้อมูลการสนทนาที่สร้างเสร็จแล้ว
	createdConv, err := s.conversationRepo.GetByID(conversation.ID)
	if err != nil {
		return nil, err
	}

	// 10. แปลงเป็น DTO
	convDTO, err := s.convertToConversationDTO(createdConv, userID)
	if err != nil {
		return nil, err
	}

	// 11. เพิ่มข้อมูลเพิ่มเติม
	convDTO.MemberCount = len(allMemberIDs)
	convDTO.IsPinned = false
	convDTO.IsMuted = false
	convDTO.UnreadCount = 0

	return convDTO, nil
}

// GetConversationMessages ดึงข้อความทั้งหมดในการสนทนา
func (s *conversationService) GetConversationMessages(conversationID, userID uuid.UUID, limit, offset int) ([]*dto.MessageDTO, int64, error) {
	// ตรวจสอบว่าผู้ใช้เป็นสมาชิกของการสนทนานี้
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return nil, 0, err
	}

	if !isMember {
		return nil, 0, errors.New("you are not a member of this conversation")
	}

	// ดึงข้อความทั้งหมดในการสนทนา
	messages, total, err := s.messageRepo.GetMessagesByConversationID(conversationID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// แปลงข้อความเป็น DTOs
	messageDTOs := make([]*dto.MessageDTO, 0, len(messages))
	for _, msg := range messages {
		messageDTO, err := s.ConvertToMessageDTO(msg, userID)
		if err != nil {
			// ข้ามข้อความที่มีปัญหา
			continue
		}
		messageDTOs = append(messageDTOs, messageDTO)
	}

	return messageDTOs, total, nil
}

// SetPinStatus กำหนดสถานะการปักหมุดของการสนทนา
func (s *conversationService) SetPinStatus(conversationID, userID uuid.UUID, isPinned bool) error {
	// ตรวจสอบว่าผู้ใช้เป็นสมาชิกของการสนทนานี้
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return err
	}

	if !isMember {
		return errors.New("you are not a member of this conversation")
	}

	// อัพเดตสถานะปักหมุด
	return s.conversationRepo.SetPinStatus(conversationID, userID, isPinned)
}

// SetMuteStatus กำหนดสถานะการปิดเสียงของการสนทนา
func (s *conversationService) SetMuteStatus(conversationID, userID uuid.UUID, isMuted bool) error {
	// ตรวจสอบว่าผู้ใช้เป็นสมาชิกของการสนทนานี้
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return err
	}

	if !isMember {
		return errors.New("you are not a member of this conversation")
	}

	// อัพเดตสถานะปิดเสียง
	return s.conversationRepo.SetMuteStatus(conversationID, userID, isMuted)
}

// CheckMembership ตรวจสอบว่าผู้ใช้เป็นสมาชิกของการสนทนาหรือไม่
func (s *conversationService) CheckMembership(userID, conversationID uuid.UUID) (bool, error) {
	return s.conversationRepo.IsMember(conversationID, userID)
}

// ConvertToMessageDTO แปลง Message model เป็น MessageDTO
func (s *conversationService) ConvertToMessageDTO(msg *models.Message, userID uuid.UUID) (*dto.MessageDTO, error) {
	if msg == nil {
		return nil, errors.New("message is nil")
	}

	// ดึง temp_id จาก metadata ถ้ามี (JSONB เป็น map[string]interface{} อยู่แล้ว)
	tempID := ""
	if msg.Metadata != nil {
		if val, ok := msg.Metadata["tempId"].(string); ok {
			tempID = val
		} else if val, ok := msg.Metadata["temp_id"].(string); ok {
			tempID = val
		}
	}

	messageDTO := &dto.MessageDTO{
		ID:                msg.ID,
		TempID:            tempID,
		ConversationID:    msg.ConversationID,
		SenderID:          msg.SenderID,
		SenderType:        msg.SenderType,
		MessageType:       msg.MessageType,
		Content:           msg.Content,
		MediaURL:          msg.MediaURL,
		MediaThumbnailURL: msg.MediaThumbnailURL,
		AlbumFiles:        msg.AlbumFiles,  // Copy album_files สำหรับ album messages
		Metadata:          msg.Metadata,
		CreatedAt:         msg.CreatedAt,
		UpdatedAt:         msg.UpdatedAt,
		IsDeleted:         msg.IsDeleted,
		IsEdited:          msg.IsEdited,
		EditCount:         msg.EditCount,
		ReplyToID:         msg.ReplyToID,
		ReadCount:         0,     // ค่าเริ่มต้น จะอัปเดตทีหลัง
		IsRead:            false, // ค่าเริ่มต้น จะอัปเดตทีหลัง
	}

	// ดึง file info และ sticker info จาก metadata ถ้ามี
	if msg.Metadata != nil {
		if fileName, ok := msg.Metadata["file_name"].(string); ok {
			messageDTO.FileName = fileName
		}
		if fileSize, ok := msg.Metadata["file_size"].(float64); ok {
			messageDTO.FileSize = int64(fileSize)
		}
		if fileType, ok := msg.Metadata["file_type"].(string); ok {
			messageDTO.FileType = fileType
		}
		if stickerIDStr, ok := msg.Metadata["sticker_id"].(string); ok {
			if stickerID, err := uuid.Parse(stickerIDStr); err == nil {
				messageDTO.StickerID = &stickerID
			}
		}
		if stickerSetIDStr, ok := msg.Metadata["sticker_set_id"].(string); ok {
			if stickerSetID, err := uuid.Parse(stickerSetIDStr); err == nil {
				messageDTO.StickerSetID = &stickerSetID
			}
		}
	}

	// 1. เพิ่มข้อมูลผู้ส่ง
	s.addSenderInfoToDTO(messageDTO)

	// 2. เพิ่มข้อมูลสถานะการอ่าน (และคำนวณ status)
	s.addReadStatusToDTO(messageDTO, userID)

	// 3. เพิ่มข้อมูลข้อความที่ตอบกลับ (ถ้ามี)
	if msg.ReplyToID != nil {
		s.addReplyToInfoToDTO(messageDTO)
	}

	return messageDTO, nil
}

// addSenderInfoToDTO เพิ่มข้อมูลผู้ส่งใน DTO
func (s *conversationService) addSenderInfoToDTO(msgDTO *dto.MessageDTO) {
	if msgDTO.SenderID == nil {
		return
	}

		// ดึงข้อมูลผู้ใช้
		user, err := s.userRepo.FindByID(*msgDTO.SenderID)
		if err == nil && user != nil {
			if user.DisplayName != "" {
				msgDTO.SenderName = user.DisplayName
			} else {
				msgDTO.SenderName = user.Username
			}
			msgDTO.SenderAvatar = user.ProfileImageURL
		}
}

// addReadStatusToDTO เพิ่มข้อมูลสถานะการอ่านใน DTO
func (s *conversationService) addReadStatusToDTO(msgDTO *dto.MessageDTO, userID uuid.UUID) {
	// ดึงข้อมูลการอ่านทั้งหมดของข้อความนี้
	reads, err := s.messageRepo.GetReads(msgDTO.ID)
	if err != nil {
		return
	}

	// คำนวณ ReadCount
	msgDTO.ReadCount = len(reads)

	// คำนวณ Status จาก read_count
	if msgDTO.ReadCount >= 2 {
		msgDTO.Status = "read" // มีคนอ่านแล้ว (นอกจากผู้ส่ง)
	} else if msgDTO.ReadCount == 1 {
		msgDTO.Status = "sent" // ผู้ส่งอ่านเองแล้ว
	} else {
		msgDTO.Status = "sent" // default
	}

	// ตรวจสอบว่าผู้ใช้อ่านข้อความแล้วหรือยัง
	for _, read := range reads {
		if read.UserID == userID {
			msgDTO.IsRead = true
			break
		}
	}
}

// addReplyToInfoToDTO เพิ่มข้อมูลข้อความที่ตอบกลับใน DTO
func (s *conversationService) addReplyToInfoToDTO(msgDTO *dto.MessageDTO) {
	if msgDTO.ReplyToID == nil {
		return
	}

	// ดึงข้อความที่ตอบกลับ
	replyMsg, err := s.messageRepo.GetByID(*msgDTO.ReplyToID)
	if err != nil || replyMsg == nil {
		return
	}

	// สร้างข้อมูลย่อของข้อความที่ตอบกลับ
	replyInfo := &dto.ReplyInfoDTO{
		ID:          replyMsg.ID.String(),
		MessageType: replyMsg.MessageType,
		Content:     replyMsg.Content,
		SenderID:    replyMsg.SenderID,
	}

	// เพิ่มข้อมูลผู้ส่งของข้อความที่ตอบกลับ
	if replyMsg.SenderID != nil {
			user, err := s.userRepo.FindByID(*replyMsg.SenderID) // แก้ไขตรงนี้: เพิ่ม * เพื่อดึงค่าจาก pointer
			if err == nil && user != nil {
				if user.DisplayName != "" {
					replyInfo.SenderName = user.DisplayName
				} else {
					replyInfo.SenderName = user.Username
				}
			}
	}

	msgDTO.ReplyToMessage = replyInfo
}

// GetMessageContext ดึงข้อความเป้าหมายพร้อมข้อความก่อนหน้าและถัดไป
func (s *conversationService) GetMessageContext(conversationID, userID uuid.UUID, targetID string,
	beforeCount, afterCount int) ([]*dto.MessageDTO, bool, bool, error) {

	// แปลง targetID เป็น uuid
	targetUUID, err := uuid.Parse(targetID)
	if err != nil {
		return nil, false, false, fmt.Errorf("invalid target message ID: %w", err)
	}

	// ตรวจสอบสิทธิ์การเข้าถึง
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return nil, false, false, err
	}

	if !isMember {
		return nil, false, false, errors.New("you are not a member of this conversation")
	}

	// ตรวจสอบว่ามีข้อความเป้าหมายจริงหรือไม่
	targetMsg, err := s.messageRepo.GetByID(targetUUID)
	if err != nil {
		return nil, false, false, fmt.Errorf("error fetching target message: %w", err)
	}

	if targetMsg == nil {
		return nil, false, false, errors.New("target message not found")
	}

	// ตรวจสอบว่าข้อความเป้าหมายอยู่ในการสนทนานี้หรือไม่
	if targetMsg.ConversationID != conversationID {
		return nil, false, false, errors.New("target message does not belong to this conversation")
	}

	// ดึงข้อความก่อนหน้าเป้าหมาย
	beforeMessages, err := s.messageRepo.GetMessagesBefore(conversationID, targetUUID, beforeCount+1) // +1 เพื่อตรวจสอบ hasMore
	if err != nil {
		return nil, false, false, fmt.Errorf("error fetching messages before target: %w", err)
	}

	// ตรวจสอบว่ามีข้อความเพิ่มเติมก่อนหน้าหรือไม่
	hasMoreBefore := len(beforeMessages) > beforeCount
	if hasMoreBefore {
		// ตัดข้อความส่วนเกิน
		beforeMessages = beforeMessages[:beforeCount]
	}

	// ดึงข้อความหลังเป้าหมาย
	afterMessages, err := s.messageRepo.GetMessagesAfter(conversationID, targetUUID, afterCount+1) // +1 เพื่อตรวจสอบ hasMore
	if err != nil {
		return nil, false, false, fmt.Errorf("error fetching messages after target: %w", err)
	}

	// ตรวจสอบว่ามีข้อความเพิ่มเติมหลังหรือไม่
	hasMoreAfter := len(afterMessages) > afterCount
	if hasMoreAfter {
		// ตัดข้อความส่วนเกิน
		afterMessages = afterMessages[:afterCount]
	}

	// รวมข้อความทั้งหมดและจัดเรียงตามเวลา
	allMessages := make([]*models.Message, 0, len(beforeMessages)+1+len(afterMessages))
	allMessages = append(allMessages, beforeMessages...)
	allMessages = append(allMessages, targetMsg)
	allMessages = append(allMessages, afterMessages...)

	// จัดเรียงข้อความตามเวลา (จากเก่าไปใหม่)
	sort.Slice(allMessages, func(i, j int) bool {
		return allMessages[i].CreatedAt.Before(allMessages[j].CreatedAt)
	})

	// แปลงเป็น DTOs โดยใช้ฟังก์ชันที่มีอยู่แล้ว
	messageDTOs := make([]*dto.MessageDTO, 0, len(allMessages))
	for _, msg := range allMessages {
		messageDTO, err := s.ConvertToMessageDTO(msg, userID)
		if err != nil {
			// ข้ามข้อความที่มีปัญหา
			continue
		}

		// เพิ่มการเน้นสำหรับข้อความเป้าหมาย (ถ้าต้องการ)
		if msg.ID == targetUUID {
			// ถ้ามีฟิลด์ IsHighlighted ใน MessageDTO ให้กำหนดค่าเป็น true
			// messageDTO.IsHighlighted = true
		}

		messageDTOs = append(messageDTOs, messageDTO)
	}

	return messageDTOs, hasMoreBefore, hasMoreAfter, nil
}

// GetMessagesBeforeID ดึงข้อความที่เก่ากว่า ID ที่ระบุ
func (s *conversationService) GetMessagesBeforeID(conversationID, userID uuid.UUID, beforeID string,
	limit int) ([]*dto.MessageDTO, int64, error) {

	// แปลง beforeID เป็น uuid
	beforeUUID, err := uuid.Parse(beforeID)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid before message ID: %w", err)
	}

	// ตรวจสอบสิทธิ์การเข้าถึง
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return nil, 0, err
	}

	if !isMember {
		return nil, 0, errors.New("you are not a member of this conversation")
	}

	// ดึงข้อความที่เก่ากว่า ID ที่ระบุ
	messages, err := s.messageRepo.GetMessagesBefore(conversationID, beforeUUID, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("error fetching messages before ID: %w", err)
	}

	// ดึงจำนวนข้อความทั้งหมดในการสนทนา (หรือจะใช้จำนวนที่น้อยกว่า ID นี้ก็ได้)
	total, err := s.messageRepo.CountAllMessages(conversationID)
	if err != nil {
		// ถ้ามีข้อผิดพลาดในการนับข้อความ ใช้ค่าประมาณ
		total = int64(len(messages))
	}

	// แปลงเป็น DTOs โดยใช้ฟังก์ชันที่มีอยู่แล้ว
	messageDTOs := make([]*dto.MessageDTO, 0, len(messages))
	for _, msg := range messages {
		messageDTO, err := s.ConvertToMessageDTO(msg, userID)
		if err != nil {
			// ข้ามข้อความที่มีปัญหา
			continue
		}
		messageDTOs = append(messageDTOs, messageDTO)
	}

	return messageDTOs, total, nil
}

// GetMessagesAfterID ดึงข้อความที่ใหม่กว่า ID ที่ระบุ
func (s *conversationService) GetMessagesAfterID(conversationID, userID uuid.UUID, afterID string,
	limit int) ([]*dto.MessageDTO, int64, error) {

	// แปลง afterID เป็น uuid
	afterUUID, err := uuid.Parse(afterID)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid after message ID: %w", err)
	}

	// ตรวจสอบสิทธิ์การเข้าถึง
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return nil, 0, err
	}

	if !isMember {
		return nil, 0, errors.New("you are not a member of this conversation")
	}

	// ดึงข้อความที่ใหม่กว่า ID ที่ระบุ
	messages, err := s.messageRepo.GetMessagesAfter(conversationID, afterUUID, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("error fetching messages after ID: %w", err)
	}

	// ดึงจำนวนข้อความทั้งหมดในการสนทนา
	total, err := s.messageRepo.CountAllMessages(conversationID)
	if err != nil {
		// ถ้ามีข้อผิดพลาดในการนับข้อความ ใช้ค่าประมาณ
		total = int64(len(messages))
	}

	// แปลงเป็น DTOs โดยใช้ฟังก์ชันที่มีอยู่แล้ว
	messageDTOs := make([]*dto.MessageDTO, 0, len(messages))
	for _, msg := range messages {
		messageDTO, err := s.ConvertToMessageDTO(msg, userID)
		if err != nil {
			// ข้ามข้อความที่มีปัญหา
			continue
		}
		messageDTOs = append(messageDTOs, messageDTO)
	}

	return messageDTOs, total, nil
}

// GetConversationsBeforeTime ดึงการสนทนาที่เก่ากว่าเวลาที่ระบุ
func (s *conversationService) GetConversationsBeforeTime(userID uuid.UUID, beforeTime string, limit int,
	convType string, pinned bool) ([]*dto.ConversationDTO, int, error) {

	// แปลง string เป็น time.Time
	parsedTime, err := time.Parse(time.RFC3339, beforeTime)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid time format: %w", err)
	}

	// เรียกใช้ repository
	conversations, total, err := s.conversationRepo.GetConversationsBeforeTime(
		userID, parsedTime, limit, convType, pinned)
	if err != nil {
		return nil, 0, err
	}

	// แปลงเป็น DTOs
	dtos := make([]*dto.ConversationDTO, 0, len(conversations))
	for _, conversation := range conversations {
		dto, err := s.convertToConversationDTO(conversation, userID)
		if err != nil {
			// ข้ามการสนทนาที่มีปัญหา
			continue
		}
		dtos = append(dtos, dto)
	}

	return dtos, total, nil
}

// GetConversationsAfterTime ดึงการสนทนาที่ใหม่กว่าเวลาที่ระบุ
func (s *conversationService) GetConversationsAfterTime(userID uuid.UUID, afterTime string, limit int,
	convType string, pinned bool) ([]*dto.ConversationDTO, int, error) {

	// แปลง string เป็น time.Time
	parsedTime, err := time.Parse(time.RFC3339, afterTime)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid time format: %w", err)
	}

	// เรียกใช้ repository
	conversations, total, err := s.conversationRepo.GetConversationsAfterTime(
		userID, parsedTime, limit, convType, pinned)
	if err != nil {
		return nil, 0, err
	}

	// แปลงเป็น DTOs
	dtos := make([]*dto.ConversationDTO, 0, len(conversations))
	for _, conversation := range conversations {
		dto, err := s.convertToConversationDTO(conversation, userID)
		if err != nil {
			// ข้ามการสนทนาที่มีปัญหา
			continue
		}
		dtos = append(dtos, dto)
	}

	return dtos, total, nil
}

// GetConversationsBeforeID ดึงการสนทนาที่เก่ากว่า ID ที่ระบุ
func (s *conversationService) GetConversationsBeforeID(userID, beforeID uuid.UUID, limit int,
	convType string, pinned bool) ([]*dto.ConversationDTO, int, error) {

	// เรียกใช้ repository
	conversations, total, err := s.conversationRepo.GetConversationsBeforeID(
		userID, beforeID, limit, convType, pinned)
	if err != nil {
		return nil, 0, err
	}

	// แปลงเป็น DTOs
	dtos := make([]*dto.ConversationDTO, 0, len(conversations))
	for _, conversation := range conversations {
		dto, err := s.convertToConversationDTO(conversation, userID)
		if err != nil {
			// ข้ามการสนทนาที่มีปัญหา
			continue
		}
		dtos = append(dtos, dto)
	}

	return dtos, total, nil
}

// GetConversationsAfterID ดึงการสนทนาที่ใหม่กว่า ID ที่ระบุ
func (s *conversationService) GetConversationsAfterID(userID, afterID uuid.UUID, limit int,
	convType string, pinned bool) ([]*dto.ConversationDTO, int, error) {

	// เรียกใช้ repository
	conversations, total, err := s.conversationRepo.GetConversationsAfterID(
		userID, afterID, limit, convType, pinned)
	if err != nil {
		return nil, 0, err
	}

	// แปลงเป็น DTOs
	dtos := make([]*dto.ConversationDTO, 0, len(conversations))
	for _, conversation := range conversations {
		dto, err := s.convertToConversationDTO(conversation, userID)
		if err != nil {
			// ข้ามการสนทนาที่มีปัญหา
			continue
		}
		dtos = append(dtos, dto)
	}

	return dtos, total, nil
}

// UpdateConversation อัปเดตข้อมูลการสนทนา
func (s *conversationService) UpdateConversation(id uuid.UUID, updateData types.JSONB) error {
	return s.conversationRepo.UpdateConversation(id, updateData)
}

// GetConversationMediaSummary ดึงสรุปจำนวน media และ link ในการสนทนา
func (s *conversationService) GetConversationMediaSummary(conversationID, userID uuid.UUID) (*dto.MediaSummaryDTO, error) {
	// ตรวจสอบว่า user เป็นสมาชิกของการสนทนาหรือไม่
	isMember, err := s.CheckMembership(userID, conversationID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("user is not a member of this conversation")
	}

	// ดึงข้อมูลสรุปจาก repository
	typeSummary, err := s.messageRepo.GetMessageTypeSummary(conversationID)
	if err != nil {
		return nil, err
	}

	linkCount, err := s.messageRepo.CountMessagesWithLinks(conversationID)
	if err != nil {
		return nil, err
	}

	// สร้าง DTO
	summary := &dto.MediaSummaryDTO{
		ImageCount: typeSummary["image"],
		VideoCount: typeSummary["video"],
		FileCount:  typeSummary["file"],
		LinkCount:  linkCount,
		TotalMedia: typeSummary["image"] + typeSummary["video"] + typeSummary["file"],
	}

	return summary, nil
}

// GetConversationMediaByType ดึงรายละเอียด media ตามประเภทพร้อม pagination
func (s *conversationService) GetConversationMediaByType(conversationID, userID uuid.UUID, mediaType string, limit, offset int) (*dto.MediaListDTO, error) {
	// ตรวจสอบว่า user เป็นสมาชิกของการสนทนาหรือไม่
	isMember, err := s.CheckMembership(userID, conversationID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("user is not a member of this conversation")
	}

	// ตรวจสอบ media type ที่รองรับ
	validTypes := map[string]bool{
		"image": true,
		"video": true,
		"file":  true,
		"link":  true,
	}
	if !validTypes[mediaType] {
		return nil, fmt.Errorf("invalid media type: %s", mediaType)
	}

	// ดึงข้อมูลจาก repository
	messages, total, err := s.messageRepo.GetMediaByType(conversationID, mediaType, limit, offset)
	if err != nil {
		return nil, err
	}

	// แปลงเป็น DTO
	items := make([]*dto.MediaItemDTO, 0)
	for _, msg := range messages {
		// ตรวจสอบว่าเป็น album หรือ single media
		if msg.MessageType == "album" {
			// Album message: แยก files ออกมา
			if msg.AlbumFiles != nil {
				// Parse album_files (เป็น []interface{})
				if filesArray, ok := msg.AlbumFiles.([]interface{}); ok {
					for _, fileData := range filesArray {
						if fileMap, ok := fileData.(map[string]interface{}); ok {
							fileType, _ := fileMap["file_type"].(string)

							// เอาเฉพาะ file type ที่ต้องการ
							if fileType == mediaType {
								item := &dto.MediaItemDTO{
									MessageID:    msg.ID.String(),
									MessageType:  fileType, // ใช้ file_type จาก album_files
									Content:      msg.Content,
									CreatedAt:    msg.CreatedAt,
									IsAlbum:      true, // บอกว่ามาจาก album
								}

								// ดึง media URLs
								if mediaURL, ok := fileMap["media_url"].(string); ok {
									item.MediaURL = mediaURL
								}
								if thumbnailURL, ok := fileMap["media_thumbnail_url"].(string); ok {
									item.ThumbnailURL = thumbnailURL
								}

								// เพิ่มข้อมูล file (ถ้ามี)
								if fileName, ok := fileMap["file_name"].(string); ok {
									item.FileName = fileName
								}
								if fileSize, ok := fileMap["file_size"].(float64); ok {
									item.FileSize = int64(fileSize)
								}

								items = append(items, item)
							}
						}
					}
				}
			}
		} else {
			// Single media message (แบบเดิม)
			item := &dto.MediaItemDTO{
				MessageID:    msg.ID.String(),
				MessageType:  msg.MessageType,
				Content:      msg.Content,
				MediaURL:     msg.MediaURL,
				ThumbnailURL: msg.MediaThumbnailURL,
				CreatedAt:    msg.CreatedAt,
				IsAlbum:      false,
			}

			// เพิ่มข้อมูลไฟล์ถ้าเป็น file type
			if msg.MessageType == "file" && msg.Metadata != nil {
				if fileName, ok := msg.Metadata["file_name"].(string); ok {
					item.FileName = fileName
				}
				if fileSize, ok := msg.Metadata["file_size"].(float64); ok {
					item.FileSize = int64(fileSize)
				}
			}

			// เพิ่ม metadata สำหรับ link type
			if mediaType == "link" {
				item.Metadata = msg.Metadata
			}

			items = append(items, item)
		}
	}

	// สร้าง pagination
	hasMore := int64(offset+limit) < total

	result := &dto.MediaListDTO{
		Data: items,
		Pagination: dto.PaginationDTO{
			Total:   total,
			Limit:   limit,
			Offset:  offset,
			HasMore: hasMore,
		},
	}

	return result, nil
}

// SetHiddenStatus ตั้งค่าสถานะการซ่อนการสนทนา
func (s *conversationService) SetHiddenStatus(conversationID, userID uuid.UUID, isHidden bool) error {
	// 1. ตรวจสอบว่าเป็นสมาชิก
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return errors.New("you are not a member of this conversation")
	}

	// 2. ตั้งค่า hidden status
	return s.conversationRepo.SetHiddenStatus(conversationID, userID, isHidden)
}

// DeleteConversation ลบการสนทนา (smart delete)
// - Direct conversation: Hide
// - Group conversation: Leave (Remove member)
func (s *conversationService) DeleteConversation(conversationID, userID uuid.UUID) (string, error) {
	// 1. ตรวจสอบว่าเป็นสมาชิก
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return "", err
	}
	if !isMember {
		return "", errors.New("you are not a member of this conversation")
	}

	// 2. ดึงข้อมูล conversation
	conversation, err := s.conversationRepo.GetByID(conversationID)
	if err != nil {
		return "", err
	}

	// 3. จัดการตามประเภท
	if conversation.Type == "direct" {
		// Direct: Hide conversation
		err = s.conversationRepo.SetHiddenStatus(conversationID, userID, true)
		if err != nil {
			return "", err
		}
		return "hidden", nil
	} else {
		// Group: Leave group - just hide for now (not remove member)
		// เพื่อความง่ายและปลอดภัย ให้ hide เหมือน direct แทนการ remove member
		err = s.conversationRepo.SetHiddenStatus(conversationID, userID, true)
		if err != nil {
			return "", err
		}
		return "hidden", nil
	}
}

// TransferOwnership โอนความเป็นเจ้าของกลุ่มให้สมาชิกคนอื่น
func (s *conversationService) TransferOwnership(conversationID, currentOwnerID, newOwnerID uuid.UUID) error {
	// 1. ตรวจสอบว่าการสนทนานี้มีอยู่จริง
	conversation, err := s.conversationRepo.GetByID(conversationID)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}
	if conversation == nil {
		return errors.New("conversation not found")
	}

	// 2. ตรวจสอบว่าเป็น group conversation (ไม่สามารถโอนความเป็นเจ้าของใน direct chat ได้)
	if conversation.Type != "group" {
		return errors.New("ownership transfer is only available for group conversations")
	}

	// 3. ตรวจสอบว่า current owner เป็น owner จริงหรือไม่
	currentOwner, err := s.conversationRepo.GetMember(conversationID, currentOwnerID)
	if err != nil {
		return fmt.Errorf("failed to get current owner: %w", err)
	}
	if currentOwner == nil {
		return errors.New("current owner is not a member of this conversation")
	}
	if currentOwner.Role != models.RoleOwner {
		return errors.New("only the owner can transfer ownership")
	}

	// 4. ตรวจสอบว่าผู้รับโอนเป็นสมาชิกของกลุ่มหรือไม่
	newOwner, err := s.conversationRepo.GetMember(conversationID, newOwnerID)
	if err != nil {
		return fmt.Errorf("failed to get new owner: %w", err)
	}
	if newOwner == nil {
		return errors.New("new owner is not a member of this conversation")
	}

	// 5. ไม่สามารถโอนให้ตัวเองได้
	if currentOwnerID == newOwnerID {
		return errors.New("cannot transfer ownership to yourself")
	}

	// 6. เปลี่ยน role ของ current owner เป็น admin
	currentOwner.Role = models.RoleAdmin
	currentOwner.IsAdmin = true
	if err := s.conversationRepo.UpdateMember(currentOwner); err != nil {
		return fmt.Errorf("failed to update current owner role: %w", err)
	}

	// 7. เปลี่ยน role ของ new owner เป็น owner
	newOwner.Role = models.RoleOwner
	newOwner.IsAdmin = true
	if err := s.conversationRepo.UpdateMember(newOwner); err != nil {
		// Rollback: เปลี่ยน current owner กลับเป็น owner
		currentOwner.Role = models.RoleOwner
		s.conversationRepo.UpdateMember(currentOwner)
		return fmt.Errorf("failed to update new owner role: %w", err)
	}

	// 8. อัปเดต creator_id ของ conversation (optional - depends on your business logic)
	conversation.CreatorID = &newOwnerID
	if err := s.conversationRepo.Update(conversation); err != nil {
		// ไม่ rollback เพราะ role ได้เปลี่ยนแล้ว
		// แค่บันทึกลงใน log
		fmt.Printf("Warning: Failed to update conversation creator_id: %v\n", err)
	}

	return nil
}

// application/serviceimpl/conversation_service.go
// เพิ่มเมธอดเหล่านี้ใน conversationService struct ที่มีอยู่แล้ว

// ========================================
// ========================================

// GetBusinessConversations ดึงการสนทนาทั้งหมดของธุรกิจ

// GetBusinessConversationsBeforeTime ดึงการสนทนาธุรกิจที่เก่ากว่าเวลาที่ระบุ

// GetBusinessConversationsAfterTime ดึงการสนทนาธุรกิจที่ใหม่กว่าเวลาที่ระบุ

// GetBusinessConversationsBeforeID ดึงการสนทนาธุรกิจที่เก่ากว่า ID ที่ระบุ

// GetBusinessConversationsAfterID ดึงการสนทนาธุรกิจที่ใหม่กว่า ID ที่ระบุ

// GetBusinessConversationMessages ดึงข้อความในการสนทนาธุรกิจ

// GetBusinessMessageContext ดึงข้อความเป้าหมายพร้อมบริบทสำหรับธุรกิจ

// GetBusinessMessagesBeforeID ดึงข้อความธุรกิจที่เก่ากว่า ID ที่ระบุ

// GetBusinessMessagesAfterID ดึงข้อความธุรกิจที่ใหม่กว่า ID ที่ระบุ

// CheckConversationBelongsToBusiness ตรวจสอบว่าการสนทนาเป็นของธุรกิจ

// ========================================
// 🔧 HELPER FUNCTIONS สำหรับ Business Context
// ========================================

// convertToBusinessConversationDTO แปลง Conversation model เป็น DTO สำหรับ business context

// ConvertToBusinessMessageDTO แปลง Message model เป็น DTO สำหรับ business context

// addBusinessReadStatusToDTO เพิ่มข้อมูลสถานะการอ่านสำหรับ business context
