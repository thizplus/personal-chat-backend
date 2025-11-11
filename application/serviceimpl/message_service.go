// application/serviceimpl/message_service.go
package serviceimpl

import (
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/repository"
	"github.com/thizplus/gofiber-chat-api/domain/service"
	"github.com/thizplus/gofiber-chat-api/domain/types"
)

// messageService เป็น implementation ของ MessageService interface
type messageService struct {
	messageRepo         repository.MessageRepository
	messageReadRepo     repository.MessageReadRepository
	conversationRepo    repository.ConversationRepository
	userRepo            repository.UserRepository
}

// NewMessageService สร้าง instance ใหม่ของ MessageService
func NewMessageService(
	messageRepo repository.MessageRepository,
	messageReadRepo repository.MessageReadRepository,
	conversationRepo repository.ConversationRepository,
	userRepo repository.UserRepository,

) service.MessageService {
	return &messageService{
		messageRepo:         messageRepo,
		messageReadRepo:     messageReadRepo,
		conversationRepo:    conversationRepo,
		userRepo:            userRepo,
		businessAccountRepo: businessAccountRepo,
		businessAdminRepo:   businessAdminRepo,
	}
}

// CheckBusinessAdmin ตรวจสอบว่าผู้ใช้เป็นแอดมินของธุรกิจหรือไม่
func (s *messageService) CheckBusinessAdmin(userID, businessID uuid.UUID) (bool, bool, error) {

	admin, err := s.businessAdminRepo.GetByUserAndBusinessID(userID, businessID)
	if err != nil {
		return false, false, err
	}

	if admin == nil {
		return false, false, nil
	}

	// คืนค่า true สำหรับสถานะแอดมิน และค่า true ถ้ามีระดับสูง (owner, admin) หรือ false ถ้าระดับต่ำ (operator)
	isHighLevel := admin.Role == "owner" || admin.Role == "admin"
	return true, isHighLevel, nil
}

// CheckBusinessFollower ตรวจสอบว่าผู้ใช้เป็นผู้ติดตามธุรกิจหรือไม่
func (s *messageService) CheckBusinessFollower(userID, businessID uuid.UUID) (bool, error) {

	follow, err := s.businessAccountRepo.IsFollowing(userID, businessID)
	if err != nil {
		return false, err
	}

	return follow, nil
}

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
