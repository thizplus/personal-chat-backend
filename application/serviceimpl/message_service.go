// application/serviceimpl/message_service.go
package serviceimpl

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/repository"
	"github.com/thizplus/gofiber-chat-api/domain/service"
	"github.com/thizplus/gofiber-chat-api/domain/types"
)

// urlRegex สำหรับตรวจจับ URL ในข้อความ
var urlRegex = regexp.MustCompile(`https?://[^\s]+`)

// messageService เป็น implementation ของ MessageService interface
type messageService struct {
	messageRepo         repository.MessageRepository
	messageReadRepo     repository.MessageReadRepository
	conversationRepo    repository.ConversationRepository
	userRepo            repository.UserRepository
	notificationService service.NotificationService
	mentionRepo         repository.MessageMentionRepository
}

// NewMessageService สร้าง instance ใหม่ของ MessageService
func NewMessageService(
	messageRepo repository.MessageRepository,
	messageReadRepo repository.MessageReadRepository,
	conversationRepo repository.ConversationRepository,
	userRepo repository.UserRepository,
	notificationService service.NotificationService,
	mentionRepo repository.MessageMentionRepository,
) service.MessageService {
	return &messageService{
		messageRepo:         messageRepo,
		messageReadRepo:     messageReadRepo,
		conversationRepo:    conversationRepo,
		userRepo:            userRepo,
		notificationService: notificationService,
		mentionRepo:         mentionRepo,
	}
}

// CheckBusinessAdmin ตรวจสอบว่าผู้ใช้เป็นแอดมินของธุรกิจหรือไม่

// createMessageRead สร้างบันทึกการอ่านข้อความ
func (s *messageService) createMessageRead(messageID, userID uuid.UUID) error {
	// ตรวจสอบว่ามีบันทึกการอ่านแล้วหรือไม่

	isRead, err := s.messageRepo.IsMessageRead(messageID, userID)
	if err != nil {
		return err
	}

	if isRead {
		return nil // ถ้าอ่านแล้ว ไม่ต้องทำอะไร
	}

	// สร้างบันทึกการอ่าน
	now := time.Now()
	messageRead := &models.MessageRead{
		ID:        uuid.New(),
		MessageID: messageID,
		UserID:    userID,
		ReadAt:    now,
	}

	return s.messageReadRepo.CreateRead(messageRead)
}

// updateConversationLastRead อัปเดต last_read_at ในข้อมูลสมาชิกการสนทนา
func (s *messageService) updateConversationLastRead(conversationID, userID uuid.UUID, readTime time.Time) error {

	return s.conversationRepo.UpdateMemberLastRead(conversationID, userID, readTime)
}

func (s *messageService) convertMetadataToJSON(metadata map[string]interface{}) types.JSONB {
	if metadata == nil {
		return types.JSONB{} // คืนค่า JSONB ที่เป็น empty map
	}

	// สร้าง types.JSONB ใหม่จาก metadata
	jsonb := types.JSONB{}
	for k, v := range metadata {
		jsonb[k] = v
	}

	return jsonb
}

// extractLinks ดึง URLs จากข้อความ
func (s *messageService) extractLinks(content string) []string {
	if content == "" {
		return nil
	}

	links := urlRegex.FindAllString(content, -1)
	if len(links) == 0 {
		return nil
	}

	// Remove duplicates
	uniqueLinks := make(map[string]bool)
	result := []string{}

	for _, link := range links {
		if !uniqueLinks[link] {
			uniqueLinks[link] = true
			result = append(result, link)
		}
	}

	return result
}

// PinMessage ปักหมุดข้อความ (เฉพาะ owner/admin ที่ pin ได้สำหรับกลุ่ม)
func (s *messageService) PinMessage(messageID, conversationID, userID uuid.UUID) error {
	// ตรวจสอบว่า message อยู่ในการสนทนานี้
	message, err := s.messageRepo.GetByID(messageID)
	if err != nil {
		return err
	}
	if message == nil {
		return errors.New("message not found")
	}
	if message.ConversationID != conversationID {
		return errors.New("message does not belong to this conversation")
	}

	// ตรวจสอบว่า user เป็นสมาชิกของการสนทนา
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return errors.New("user is not a member of this conversation")
	}

	// ดึงข้อมูลการสนทนา
	conversation, err := s.conversationRepo.GetByID(conversationID)
	if err != nil {
		return err
	}

	// ถ้าเป็นกลุ่ม ต้องเป็น owner/admin เท่านั้น
	if conversation.Type == "group" {
		member, err := s.conversationRepo.GetMember(conversationID, userID)
		if err != nil {
			return err
		}
		if member == nil {
			return errors.New("user is not a member of this conversation")
		}
		if member.Role != "owner" && member.Role != "admin" {
			return errors.New("only owner/admin can pin messages in group conversations")
		}
	}

	// ปักหมุดข้อความ
	return s.messageRepo.PinMessage(messageID, userID)
}

// UnpinMessage ยกเลิกการปักหมุดข้อความ
func (s *messageService) UnpinMessage(messageID, conversationID, userID uuid.UUID) error {
	// ตรวจสอบว่า message อยู่ในการสนทนานี้
	message, err := s.messageRepo.GetByID(messageID)
	if err != nil {
		return err
	}
	if message == nil {
		return errors.New("message not found")
	}
	if message.ConversationID != conversationID {
		return errors.New("message does not belong to this conversation")
	}

	// ตรวจสอบว่า user เป็นสมาชิกของการสนทนา
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return errors.New("user is not a member of this conversation")
	}

	// ดึงข้อมูลการสนทนา
	conversation, err := s.conversationRepo.GetByID(conversationID)
	if err != nil {
		return err
	}

	// ถ้าเป็นกลุ่ม ต้องเป็น owner/admin หรือคนที่ pin เอง
	if conversation.Type == "group" {
		member, err := s.conversationRepo.GetMember(conversationID, userID)
		if err != nil {
			return err
		}
		if member == nil {
			return errors.New("user is not a member of this conversation")
		}

		// อนุญาตให้ unpin ได้ถ้าเป็น owner/admin หรือคนที่ pin เอง
		isAuthorized := member.Role == "owner" || member.Role == "admin" ||
			(message.PinnedBy != nil && *message.PinnedBy == userID)

		if !isAuthorized {
			return errors.New("only owner/admin or the user who pinned can unpin messages")
		}
	}

	// ยกเลิกการปักหมุด
	return s.messageRepo.UnpinMessage(messageID)
}

// GetPinnedMessages ดึงรายการข้อความที่ปักหมุดในการสนทนา
func (s *messageService) GetPinnedMessages(conversationID, userID uuid.UUID, limit, offset int) ([]*models.Message, int64, error) {
	// ตรวจสอบว่า user เป็นสมาชิกของการสนทนา
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return nil, 0, err
	}
	if !isMember {
		return nil, 0, errors.New("user is not a member of this conversation")
	}

	// ดึงข้อความที่ปักหมุด
	return s.messageRepo.GetPinnedMessages(conversationID, limit, offset)
}

// GetMessagesByDate ดึงข้อความตามวันที่กำหนด
func (s *messageService) GetMessagesByDate(conversationID, userID uuid.UUID, dateStr string, limit int) ([]*models.Message, int64, bool, bool, error) {
	// ตรวจสอบว่า user เป็นสมาชิกของการสนทนา
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return nil, 0, false, false, err
	}
	if !isMember {
		return nil, 0, false, false, errors.New("user is not a member of this conversation")
	}

	// Parse date string (YYYY-MM-DD)
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, 0, false, false, errors.New("invalid date format, use YYYY-MM-DD")
	}

	// สร้างช่วงเวลาสำหรับวันนั้น
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// ดึงข้อความในวันนั้น
	messages, total, err := s.messageRepo.FindByDateRange(conversationID, startOfDay, endOfDay, limit)
	if err != nil {
		return nil, 0, false, false, err
	}

	// ตรวจสอบว่ามีข้อความก่อนหน้าหรือไม่
	hasMoreBefore := false
	beforeMessages, _, err := s.messageRepo.FindByDateRange(conversationID, time.Time{}, startOfDay, 1)
	if err == nil && len(beforeMessages) > 0 {
		hasMoreBefore = true
	}

	// ตรวจสอบว่ามีข้อความหลังจากนี้หรือไม่
	hasMoreAfter := false
	afterMessages, _, err := s.messageRepo.FindByDateRange(conversationID, endOfDay, time.Now().Add(24*time.Hour), 1)
	if err == nil && len(afterMessages) > 0 {
		hasMoreAfter = true
	}

	return messages, total, hasMoreBefore, hasMoreAfter, nil
}

// SearchMessages ค้นหาข้อความ (CURSOR-BASED)
func (s *messageService) SearchMessages(
	query string,
	conversationID *uuid.UUID,
	userID uuid.UUID,
	limit int,
	cursor *string,
	direction string,
) ([]*models.Message, *string, bool, error) {
	// ถ้าระบุ conversation_id ให้ตรวจสอบว่า user เป็นสมาชิกหรือไม่
	if conversationID != nil {
		isMember, err := s.conversationRepo.IsMember(*conversationID, userID)
		if err != nil {
			return nil, nil, false, err
		}
		if !isMember {
			return nil, nil, false, errors.New("user is not a member of this conversation")
		}
	}

	// ค้นหาข้อความ
	return s.messageRepo.SearchMessages(query, conversationID, limit, cursor, direction)
}

// notifyMentionedUsers ส่งการแจ้งเตือนและบันทึกลง database
func (s *messageService) notifyMentionedUsers(message *models.Message, mentions interface{}, senderID uuid.UUID) {
	// แปลง mentions เป็น array ของ mention objects
	var mentionsList []interface{}

	if mentionsArray, ok := mentions.([]interface{}); ok {
		mentionsList = mentionsArray
	} else if mentionsMap, ok := mentions.(map[string]interface{}); ok {
		// ถ้าเป็น map ให้ดึง data array ออกมา
		if data, ok := mentionsMap["data"].([]interface{}); ok {
			mentionsList = data
		}
	}

	// ดึงข้อมูลผู้ส่ง
	sender, err := s.userRepo.FindByID(senderID)
	if err != nil || sender == nil {
		return // ถ้าไม่พบข้อมูลผู้ส่งก็ข้าม
	}

	// รวม user IDs ที่ต้องการส่งการแจ้งเตือน และเตรียมบันทึกลง database
	var mentionedUserIDs []uuid.UUID
	var mentionRecords []*models.MessageMention

	// ส่งการแจ้งเตือนไปยังแต่ละคนที่ถูก mention
	for _, mention := range mentionsList {
		mentionMap, ok := mention.(map[string]interface{})
		if !ok {
			continue
		}

		// ดึง user_id จาก mention object
		userIDStr, ok := mentionMap["user_id"].(string)
		if !ok {
			continue
		}

		mentionedUserID, err := uuid.Parse(userIDStr)
		if err != nil {
			continue
		}

		// ไม่ส่งการแจ้งเตือนให้ตัวเอง
		if mentionedUserID == senderID {
			continue
		}

		mentionedUserIDs = append(mentionedUserIDs, mentionedUserID)

		// สร้าง mention record สำหรับบันทึกลง database
		mentionRecord := &models.MessageMention{
			ID:              uuid.New(),
			MessageID:       message.ID,
			MentionedUserID: mentionedUserID,
		}

		// Optional: บันทึก start_index และ length ถ้ามี
		if startIndex, ok := mentionMap["start_index"].(float64); ok {
			idx := int(startIndex)
			mentionRecord.StartIndex = &idx
		}
		if length, ok := mentionMap["length"].(float64); ok {
			len := int(length)
			mentionRecord.Length = &len
		}

		mentionRecords = append(mentionRecords, mentionRecord)
	}

	// บันทึก mentions ลง database
	if len(mentionRecords) > 0 && s.mentionRepo != nil {
		if err := s.mentionRepo.CreateBatch(mentionRecords); err != nil {
			// Log error แต่ไม่ fail การส่งข้อความ
			fmt.Printf("Warning: Failed to save mentions: %v\n", err)
		} else {
			fmt.Printf("✅ Successfully saved %d mentions to database\n", len(mentionRecords))
		}
	} else {
		fmt.Printf("⚠️ Skipping mention save: mentionRecords=%d, mentionRepo=%v\n", len(mentionRecords), s.mentionRepo != nil)
	}

	// ส่งการแจ้งเตือนถ้ามีคนที่ถูก mention
	if len(mentionedUserIDs) > 0 {
		notificationData := map[string]interface{}{
			"type":            "mention",
			"message_id":      message.ID.String(),
			"conversation_id": message.ConversationID.String(),
			"sender_id":       senderID.String(),
			"sender_name":     sender.DisplayName,
			"message_preview": truncateString(message.Content, 100),
		}

		s.notificationService.SendNotification(mentionedUserIDs, notificationData)
	}
}

// truncateString ตัดข้อความให้สั้นลง
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ForwardMessage ส่งต่อข้อความไปยังการสนทนาอื่น
func (s *messageService) ForwardMessage(messageID, targetConversationID, userID uuid.UUID) (*models.Message, error) {
	// ดึงข้อความต้นฉบับ
	originalMsg, err := s.messageRepo.GetByID(messageID)
	if err != nil {
		return nil, err
	}
	if originalMsg == nil {
		return nil, errors.New("message not found")
	}

	// ตรวจสอบว่า user เป็นสมาชิกของการสนทนาต้นทาง
	isMember, err := s.conversationRepo.IsMember(originalMsg.ConversationID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("user is not a member of the source conversation")
	}

	// ตรวจสอบว่า user เป็นสมาชิกของการสนทนาปลายทาง
	isMember, err = s.conversationRepo.IsMember(targetConversationID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("user is not a member of the target conversation")
	}

	// สร้างข้อมูล forwarded_from
	forwardedFrom := types.JSONB{
		"message_id":         originalMsg.ID.String(),
		"conversation_id":    originalMsg.ConversationID.String(),
		"original_timestamp": originalMsg.CreatedAt.Format(time.RFC3339),
	}
	if originalMsg.SenderID != nil {
		forwardedFrom["sender_id"] = originalMsg.SenderID.String()

		// ดึงข้อมูลผู้ส่งต้นฉบับเพื่อเอา sender_name
		if s.userRepo != nil {
			originalSender, err := s.userRepo.FindByID(*originalMsg.SenderID)
			if err == nil && originalSender != nil {
				senderName := originalSender.DisplayName
				if senderName == "" {
					senderName = originalSender.Username
				}
				forwardedFrom["sender_name"] = senderName
			}
		}
	}

	// สร้างข้อความใหม่
	now := time.Now()
	forwardedMsg := &models.Message{
		ID:                uuid.New(),
		ConversationID:    targetConversationID,
		SenderID:          &userID,
		SenderType:        "user",
		MessageType:       originalMsg.MessageType,
		Content:           originalMsg.Content,
		MediaURL:          originalMsg.MediaURL,
		MediaThumbnailURL: originalMsg.MediaThumbnailURL,
		AlbumFiles:        originalMsg.AlbumFiles, // Copy album files for album messages
		Metadata:          originalMsg.Metadata,
		IsForwarded:       true,
		ForwardedFrom:     forwardedFrom,
		CreatedAt:         now,
		UpdatedAt:         now,
		IsDeleted:         false,
	}

	// บันทึกข้อความ
	if err := s.messageRepo.Create(forwardedMsg); err != nil {
		return nil, err
	}

	// สร้างบันทึกการอ่านสำหรับผู้ส่ง
	messageRead := &models.MessageRead{
		ID:        uuid.New(),
		MessageID: forwardedMsg.ID,
		UserID:    userID,
		ReadAt:    now,
	}
	_ = s.messageReadRepo.CreateRead(messageRead)

	// อัปเดต last_read_at สำหรับผู้ส่ง
	_ = s.conversationRepo.UpdateMemberLastRead(targetConversationID, userID, now)

	// อัปเดตข้อความล่าสุดของการสนทนา
	lastMsgText := "[Forwarded] "
	if originalMsg.MessageType == "text" {
		lastMsgText += originalMsg.Content
	} else {
		lastMsgText += "[" + originalMsg.MessageType + "]"
	}
	_ = s.messageRepo.UpdateConversationLastMessage(targetConversationID, lastMsgText, now, forwardedMsg.ID)

	// ส่ง WebSocket event แจ้งการอัปเดต conversation พร้อม mention data
	s.notifyConversationUpdated(targetConversationID, lastMsgText, now, forwardedMsg.ID)

	return forwardedMsg, nil
}

// ForwardMessages ส่งต่อหลายข้อความไปยังหลายการสนทนา
func (s *messageService) ForwardMessages(messageIDs []uuid.UUID, targetConversationIDs []uuid.UUID, userID uuid.UUID) (map[uuid.UUID][]*models.Message, error) {
	if len(messageIDs) == 0 {
		return nil, errors.New("no messages to forward")
	}
	if len(targetConversationIDs) == 0 {
		return nil, errors.New("no target conversations specified")
	}

	// Map สำหรับเก็บผลลัพธ์ [conversationID] => [messages]
	results := make(map[uuid.UUID][]*models.Message)

	// Forward แต่ละข้อความไปยังทุกการสนทนาปลายทาง
	for _, msgID := range messageIDs {
		for _, targetConvID := range targetConversationIDs {
			forwardedMsg, err := s.ForwardMessage(msgID, targetConvID, userID)
			if err != nil {
				// ถ้า forward ไม่สำเร็จก็ข้ามไป (fail silently หรือจะ log error ก็ได้)
				continue
			}

			// เก็บผลลัพธ์
			results[targetConvID] = append(results[targetConvID], forwardedMsg)
		}
	}

	if len(results) == 0 {
		return nil, errors.New("failed to forward any messages")
	}

	return results, nil
}

// notifyConversationUpdated ส่ง WebSocket event แจ้งการอัปเดต conversation พร้อม mention data
// สำหรับแต่ละ member (personalized per user)
func (s *messageService) notifyConversationUpdated(conversationID uuid.UUID, lastMessageText string, lastMessageAt time.Time, lastMessageID uuid.UUID) {
	// ดึงรายชื่อสมาชิกทั้งหมดในการสนทนา
	members, err := s.conversationRepo.GetMembers(conversationID)
	if err != nil {
		fmt.Printf("Error getting conversation members: %v\n", err)
		return
	}

	// ส่ง notification ให้แต่ละ member พร้อมข้อมูล mention ที่เป็น personalized
	for _, member := range members {
		// คำนวณ mention data สำหรับ user นี้
		var hasMention bool
		var mentionCount int
		var lastMessageHasMention bool

		// 1. นับ unread mentions
		if member.LastReadAt != nil {
			mentionCount, _ = s.mentionRepo.CountUnreadMentionsByConversation(
				conversationID,
				member.UserID,
				member.LastReadAt,
			)
		} else {
			mentionCount, _ = s.mentionRepo.CountUnreadMentionsByConversation(
				conversationID,
				member.UserID,
				nil,
			)
		}

		if mentionCount > 0 {
			hasMention = true
		}

		// 2. ตรวจสอบว่า last message มี mention หรือไม่
		if lastMessageID != uuid.Nil {
			lastMessageHasMention, _ = s.mentionRepo.CheckLastMessageHasMention(
				lastMessageID,
				member.UserID,
			)
		}

		// สร้าง notification payload สำหรับ user นี้
		updateData := map[string]interface{}{
			"conversation_id":          conversationID.String(),
			"last_message_text":        lastMessageText,
			"last_message_at":          lastMessageAt.Format(time.RFC3339),
			"has_unread_mention":       hasMention,
			"unread_mention_count":     mentionCount,
			"last_message_has_mention": lastMessageHasMention,
		}

		// ส่ง WebSocket event แบบ personalized ไปยัง user นี้
		s.notificationService.NotifyConversationUpdatedToUser(member.UserID, updateData)
	}
}
