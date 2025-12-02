// domain/repository/conversation_repository.go
package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/types"
)

type ConversationRepository interface {
	// GetByID ดึงข้อมูลการสนทนาตาม ID
	GetByID(id uuid.UUID) (*models.Conversation, error)

	// Create สร้างการสนทนาใหม่
	Create(conversation *models.Conversation) error

	// UpdateConversation อัพเดตข้อมูลการสนทนา
	UpdateConversation(id uuid.UUID, updateData types.JSONB) error

	// Update อัปเดตการสนทนาทั้งหมด
	Update(conversation *models.Conversation) error

	// Delete ลบการสนทนา (เปลี่ยนสถานะเป็น inactive)
	Delete(id uuid.UUID) error

	// FindDirectConversation หาการสนทนาโดยตรงระหว่างผู้ใช้สองคน
	FindDirectConversation(user1ID, user2ID uuid.UUID) (*models.Conversation, error)

	// GetUserConversations ดึงการสนทนาทั้งหมดของผู้ใช้
	GetUserConversations(userID uuid.UUID, limit, offset int) ([]*models.Conversation, int, error)

	// AddMember เพิ่มสมาชิกในการสนทนา
	AddMember(member *models.ConversationMember) error

	// GetMember ดึงข้อมูลสมาชิกในการสนทนา
	GetMember(conversationID, userID uuid.UUID) (*models.ConversationMember, error)

	// GetMembers ดึงรายการสมาชิกทั้งหมดในการสนทนา
	GetMembers(conversationID uuid.UUID) ([]*models.ConversationMember, error)

	// UpdateMember อัปเดตข้อมูลสมาชิก
	UpdateMember(member *models.ConversationMember) error

	// RemoveMember ลบสมาชิกออกจากการสนทนา
	RemoveMember(conversationID, userID uuid.UUID) error

	// UpdateMemberAdmin อัพเดตสถานะแอดมินของสมาชิก
	UpdateMemberAdmin(conversationID, userID uuid.UUID, isAdmin bool) error

	// UpdateLastMessage อัพเดต last_message สำหรับการสนทนา
	UpdateLastMessage(conversationID uuid.UUID, text string, messageTime time.Time) error

	// SetPinStatus กำหนดสถานะการปักหมุดของการสนทนา
	SetPinStatus(conversationID, userID uuid.UUID, isPinned bool) error

	// SetMuteStatus กำหนดสถานะการปิดเสียงของการสนทนา
	SetMuteStatus(conversationID, userID uuid.UUID, isMuted bool) error

	// SetHiddenStatus กำหนดสถานะการซ่อนการสนทนา
	SetHiddenStatus(conversationID, userID uuid.UUID, isHidden bool) error

	// IsHidden ตรวจสอบว่าการสนทนาถูกซ่อนหรือไม่
	IsHidden(conversationID, userID uuid.UUID) (bool, error)

	// MarkAllMessagesAsRead มาร์คข้อความทั้งหมดในการสนทนาว่าอ่านแล้ว
	MarkAllMessagesAsRead(conversationID, userID uuid.UUID) error

	UpdateMemberLastRead(conversationID uuid.UUID, userID uuid.UUID, readTime time.Time) error

	// GetUserMemberships ดึงข้อมูลสมาชิกการสนทนาทั้งหมดของผู้ใช้
	GetUserMemberships(userID uuid.UUID) ([]*models.ConversationMember, error)

	// GetConversationsByIDs ดึงการสนทนาจาก IDs
	GetConversationsByIDs(ids []uuid.UUID) ([]*models.Conversation, error)

	// IsMember ตรวจสอบว่าผู้ใช้เป็นสมาชิกของการสนทนาหรือไม่
	IsMember(conversationID, userID uuid.UUID) (bool, error)

	// GetLastMessage ดึงข้อความล่าสุดของการสนทนา
	GetLastMessage(conversationID uuid.UUID) (*models.Message, error)

	// GetLastNonDeletedMessage ดึงข้อความล่าสุดที่ไม่ถูกลบ
	GetLastNonDeletedMessage(conversationID uuid.UUID) (*models.Message, error)

	// GetUserConversationsWithFilter ดึงการสนทนาทั้งหมดของผู้ใช้พร้อมตัวกรอง
	GetUserConversationsWithFilter(userID uuid.UUID, limit, offset int, convType string, pinned bool) ([]*models.Conversation, int, error)

	// GetConversationsBeforeTime ดึงการสนทนาที่เก่ากว่าเวลาที่ระบุ
	GetConversationsBeforeTime(userID uuid.UUID, beforeTime time.Time, limit int, convType string, pinned bool) ([]*models.Conversation, int, error)

	// GetConversationsAfterTime ดึงการสนทนาที่ใหม่กว่าเวลาที่ระบุ
	GetConversationsAfterTime(userID uuid.UUID, afterTime time.Time, limit int, convType string, pinned bool) ([]*models.Conversation, int, error)

	// GetConversationsBeforeID ดึงการสนทนาที่เก่ากว่า ID ที่ระบุ
	GetConversationsBeforeID(userID, beforeID uuid.UUID, limit int, convType string, pinned bool) ([]*models.Conversation, int, error)

	// GetConversationsAfterID ดึงการสนทนาที่ใหม่กว่า ID ที่ระบุ
	GetConversationsAfterID(userID, afterID uuid.UUID, limit int, convType string, pinned bool) ([]*models.Conversation, int, error)


}
