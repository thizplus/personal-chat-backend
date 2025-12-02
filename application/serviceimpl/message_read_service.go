// application/serviceimpl/message_read_service.go
package serviceimpl

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/repository"
	"github.com/thizplus/gofiber-chat-api/domain/service"
)

// messageReadService เป็น implementation ของ MessageReadService
type messageReadService struct {
	messageRepo      repository.MessageRepository
	messageReadRepo  repository.MessageReadRepository
	conversationRepo repository.ConversationRepository
}

// NewMessageReadService สร้าง instance ใหม่ของ MessageReadService
func NewMessageReadService(
	messageRepo repository.MessageRepository,
	messageReadRepo repository.MessageReadRepository,
	conversationRepo repository.ConversationRepository,
) service.MessageReadService {
	return &messageReadService{
		messageRepo:      messageRepo,
		messageReadRepo:  messageReadRepo,
		conversationRepo: conversationRepo,
	}
}

// MarkMessageAsRead ทำเครื่องหมายว่าข้อความถูกอ่านแล้ว
func (s *messageReadService) MarkMessageAsRead(messageID, userID uuid.UUID) (uuid.UUID, error) {

	// ดึงข้อมูลข้อความ
	message, err := s.messageRepo.GetByID(messageID)
	if err != nil {
		return uuid.Nil, err
	}

	if message == nil {
		return uuid.Nil, errors.New("message not found")
	}

	// ตรวจสอบว่าผู้ใช้เป็นสมาชิกของการสนทนา
	isMember, err := s.conversationRepo.IsMember(message.ConversationID, userID)
	if err != nil {
		return uuid.Nil, err
	}

	if !isMember {
		return uuid.Nil, errors.New("you are not a member of this conversation")
	}

	// ถ้าผู้ใช้เป็นผู้ส่งข้อความ ไม่ต้องทำอะไร
	if message.SenderID != nil && *message.SenderID == userID {
		return uuid.Nil, errors.New("you are not a member of this conversation")
	}

	// ตรวจสอบว่าอ่านแล้วหรือยัง
	isRead, err := s.messageRepo.IsMessageRead(messageID, userID)
	if err != nil {
		return uuid.Nil, err
	}

	if isRead {
		return message.ConversationID, nil
	}

	// สร้างบันทึกการอ่าน
	now := time.Now()
	read := &models.MessageRead{
		ID:        uuid.New(),
		MessageID: messageID,
		UserID:    userID,
		ReadAt:    now,
	}

	if err := s.messageReadRepo.CreateRead(read); err != nil {
		return uuid.Nil, err
	}

	// อ่านข้อความเก่ากว่าทั้งหมดในการสนทนาเดียวกัน
	unreadMessages, err := s.messageReadRepo.GetUnreadMessageIDs(message.ConversationID, userID)
	if err == nil && len(unreadMessages) > 0 {
		// กรองเอาเฉพาะข้อความที่เก่ากว่า
		olderMessages := make([]uuid.UUID, 0)
		for _, msgID := range unreadMessages {
			msg, err := s.messageRepo.GetByID(msgID)
			if err != nil {
				continue
			}

			if msg.CreatedAt.Before(message.CreatedAt) || msg.CreatedAt.Equal(message.CreatedAt) {
				olderMessages = append(olderMessages, msgID)
			}
		}

		// มาร์คข้อความที่เก่ากว่าทั้งหมดเป็นอ่านแล้ว
		for _, msgID := range olderMessages {
			// สร้างบันทึกการอ่าน
			read := &models.MessageRead{
				ID:        uuid.New(),
				MessageID: msgID,
				UserID:    userID,
				ReadAt:    now,
			}
			s.messageReadRepo.CreateRead(read)
		}
	}

	// อัปเดตเวลาอ่านล่าสุดในตาราง conversation_members
	err = s.conversationRepo.UpdateMemberLastRead(message.ConversationID, userID, message.CreatedAt)
	if err != nil {
		return uuid.Nil, err
	}

	// คืนค่า conversationID สำหรับการแจ้งเตือน
	return message.ConversationID, nil
}

// GetMessageReads ดึงข้อมูลผู้ที่อ่านข้อความแล้ว
func (s *messageReadService) GetMessageReads(messageID, userID uuid.UUID) ([]*models.MessageRead, error) {

	// ดึงข้อมูลข้อความ
	message, err := s.messageRepo.GetByID(messageID)
	if err != nil {
		return nil, err
	}

	if message == nil {
		return nil, errors.New("message not found")
	}

	// ตรวจสอบว่าผู้ใช้เป็นสมาชิกของการสนทนา
	isMember, err := s.conversationRepo.IsMember(message.ConversationID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("you are not a member of this conversation")
	}

	// ดึงข้อมูลการอ่านทั้งหมด
	return s.messageReadRepo.GetByMessageID(messageID)
}

// MarkAllMessagesAsRead ทำเครื่องหมายว่าข้อความทั้งหมดในการสนทนาถูกอ่านแล้ว
func (s *messageReadService) MarkAllMessagesAsRead(conversationID, userID uuid.UUID) (int, error) {

	// ตรวจสอบว่าผู้ใช้เป็นสมาชิกของการสนทนา
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return 0, err
	}

	if !isMember {
		return 0, errors.New("you are not a member of this conversation")
	}

	// อัปเดต last_read_at ก่อน
	now := time.Now()
	if err := s.conversationRepo.UpdateMemberLastRead(conversationID, userID, now); err != nil {
		// บันทึกข้อผิดพลาดแต่ไม่หยุดการทำงาน
	}

	// ดึงข้อความทั้งหมดที่ยังไม่ได้อ่าน
	unreadMessageIDs, err := s.messageReadRepo.GetUnreadMessageIDs(conversationID, userID)
	if err != nil {
		return 0, err
	}

	if len(unreadMessageIDs) == 0 {
		return 0, nil
	}

	// มาร์คข้อความทั้งหมดเป็นอ่านแล้ว
	for _, messageID := range unreadMessageIDs {
		// สร้างบันทึกการอ่าน
		read := &models.MessageRead{
			ID:        uuid.New(),
			MessageID: messageID,
			UserID:    userID,
			ReadAt:    now,
		}
		if err := s.messageReadRepo.CreateRead(read); err != nil {
			// บันทึกข้อผิดพลาดแต่ไม่หยุดการทำงาน
		}
	}

	return len(unreadMessageIDs), nil
}

// GetUnreadCount ดึงจำนวนข้อความที่ยังไม่ได้อ่านในการสนทนา
func (s *messageReadService) GetUnreadCount(conversationID, userID uuid.UUID) (int, error) {

	// ตรวจสอบว่าผู้ใช้เป็นสมาชิกของการสนทนา
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return 0, err
	}

	if !isMember {
		return 0, errors.New("you are not a member of this conversation")
	}

	// ดึงข้อความทั้งหมดที่ยังไม่ได้อ่าน
	unreadMessageIDs, err := s.messageReadRepo.GetUnreadMessageIDs(conversationID, userID)
	if err != nil {
		return 0, err
	}

	return len(unreadMessageIDs), nil
}

// MarkConversationAsRead ทำเครื่องหมายข้อความทั้งหมดจนถึง lastReadMessageID ว่าอ่านแล้ว
func (s *messageReadService) MarkConversationAsRead(conversationID, userID, lastReadMessageID uuid.UUID) (int, error) {
	// ตรวจสอบว่าผู้ใช้เป็นสมาชิกของการสนทนา
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return 0, err
	}

	if !isMember {
		return 0, errors.New("you are not a member of this conversation")
	}

	// ดึงข้อมูลของ lastReadMessage
	lastReadMessage, err := s.messageRepo.GetByID(lastReadMessageID)
	if err != nil {
		return 0, err
	}

	if lastReadMessage == nil {
		return 0, errors.New("message not found")
	}

	// ตรวจสอบว่าข้อความอยู่ในการสนทนานี้จริง
	if lastReadMessage.ConversationID != conversationID {
		return 0, errors.New("message does not belong to this conversation")
	}

	// ดึงข้อความทั้งหมดที่ยังไม่ได้อ่าน
	unreadMessageIDs, err := s.messageReadRepo.GetUnreadMessageIDs(conversationID, userID)
	if err != nil {
		return 0, err
	}

	// มาร์คข้อความที่เก่ากว่าหรือเท่ากับ lastReadMessage เป็นอ่านแล้ว
	now := time.Now()
	markedCount := 0

	for _, msgID := range unreadMessageIDs {
		msg, err := s.messageRepo.GetByID(msgID)
		if err != nil {
			continue
		}

		// ข้ามข้อความของตัวเอง
		if msg.SenderID != nil && *msg.SenderID == userID {
			continue
		}

		// มาร์คเฉพาะข้อความที่เก่ากว่าหรือเท่ากับ lastReadMessage
		if msg.CreatedAt.Before(lastReadMessage.CreatedAt) || msg.CreatedAt.Equal(lastReadMessage.CreatedAt) {
			read := &models.MessageRead{
				ID:        uuid.New(),
				MessageID: msgID,
				UserID:    userID,
				ReadAt:    now,
			}
			if err := s.messageReadRepo.CreateRead(read); err == nil {
				markedCount++
			}
		}
	}

	// อัปเดต last_read_at
	if err := s.conversationRepo.UpdateMemberLastRead(conversationID, userID, lastReadMessage.CreatedAt); err != nil {
		// บันทึกข้อผิดพลาดแต่ไม่หยุดการทำงาน
	}

	// คำนวณ unread count ที่เหลือ
	remainingUnread := len(unreadMessageIDs) - markedCount
	if remainingUnread < 0 {
		remainingUnread = 0
	}

	return remainingUnread, nil
}

// GetUnreadCounts ดึงจำนวนข้อความที่ยังไม่ได้อ่านในทุกการสนทนา
func (s *messageReadService) GetUnreadCounts(userID uuid.UUID) (map[uuid.UUID]int, int, error) {
	// ดึงการสนทนาทั้งหมดที่ user เป็นสมาชิก (ใช้ limit สูงเพื่อดึงทั้งหมด)
	conversations, _, err := s.conversationRepo.GetUserConversations(userID, 1000, 0)
	if err != nil {
		return nil, 0, err
	}

	unreadCounts := make(map[uuid.UUID]int)
	totalUnread := 0

	for _, conversation := range conversations {
		// ดึงจำนวนข้อความที่ยังไม่ได้อ่านในแต่ละการสนทนา
		unreadMessageIDs, err := s.messageReadRepo.GetUnreadMessageIDs(conversation.ID, userID)
		if err != nil {
			continue
		}

		// นับเฉพาะข้อความที่ไม่ใช่ของตัวเอง
		count := 0
		for _, msgID := range unreadMessageIDs {
			msg, err := s.messageRepo.GetByID(msgID)
			if err != nil {
				continue
			}

			// ข้ามข้อความของตัวเอง
			if msg.SenderID != nil && *msg.SenderID == userID {
				continue
			}

			count++
		}

		if count > 0 {
			unreadCounts[conversation.ID] = count
			totalUnread += count
		}
	}

	return unreadCounts, totalUnread, nil
}

// GetUnreadMessageIDs ดึงรายการ ID ของข้อความที่ยังไม่ได้อ่าน
func (s *messageReadService) GetUnreadMessageIDs(conversationID, userID uuid.UUID) ([]uuid.UUID, error) {
	// ตรวจสอบว่าผู้ใช้เป็นสมาชิกของการสนทนา
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("you are not a member of this conversation")
	}

	// ดึงข้อความทั้งหมดที่ยังไม่ได้อ่าน
	return s.messageReadRepo.GetUnreadMessageIDs(conversationID, userID)
}
