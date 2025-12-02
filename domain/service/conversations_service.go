// domain/service/conversation_service.go
package service

import (
	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/dto"
	"github.com/thizplus/gofiber-chat-api/domain/types"
)

type ConversationService interface {
	// CreateDirectConversation สร้างการสนทนาแบบส่วนตัวระหว่างผู้ใช้สองคน
	CreateDirectConversation(userID, friendID uuid.UUID) (*dto.ConversationDTO, error)

	// CreateGroupConversation สร้างการสนทนาแบบกลุ่ม
	CreateGroupConversation(userID uuid.UUID, title, iconURL string, memberIDs []uuid.UUID) (*dto.ConversationDTO, error)

	// GetUserConversations ดึงรายการการสนทนาทั้งหมดของผู้ใช้
	GetUserConversations(userID uuid.UUID, limit, offset int, convType string, pinned bool) ([]*dto.ConversationDTO, int, error)

	// GetConversationMessages ดึงข้อความทั้งหมดในการสนทนา
	GetConversationMessages(conversationID, userID uuid.UUID, limit, offset int) ([]*dto.MessageDTO, int64, error)

	// SetPinStatus กำหนดสถานะการปักหมุดของการสนทนา
	SetPinStatus(conversationID, userID uuid.UUID, isPinned bool) error

	// SetMuteStatus กำหนดสถานะการปิดเสียงของการสนทนา
	SetMuteStatus(conversationID, userID uuid.UUID, isMuted bool) error

	// CheckMembership ตรวจสอบว่าผู้ใช้เป็นสมาชิกของการสนทนาหรือไม่
	CheckMembership(userID, conversationID uuid.UUID) (bool, error)

	// GetMessageContext ดึงข้อความเป้าหมายพร้อมข้อความก่อนหน้าและถัดไป
	GetMessageContext(conversationID, userID uuid.UUID, targetID string,
		beforeCount, afterCount int) ([]*dto.MessageDTO, bool, bool, error)

	// GetMessagesBeforeID ดึงข้อความที่เก่ากว่า ID ที่ระบุ
	GetMessagesBeforeID(conversationID, userID uuid.UUID, beforeID string,
		limit int) ([]*dto.MessageDTO, int64, error)

	// GetMessagesAfterID ดึงข้อความที่ใหม่กว่า ID ที่ระบุ
	GetMessagesAfterID(conversationID, userID uuid.UUID, afterID string,
		limit int) ([]*dto.MessageDTO, int64, error)

	// GetConversationsBeforeTime ดึงการสนทนาที่เก่ากว่าเวลาที่ระบุ
	GetConversationsBeforeTime(userID uuid.UUID, beforeTime string, limit int, convType string, pinned bool) ([]*dto.ConversationDTO, int, error)

	// GetConversationsAfterTime ดึงการสนทนาที่ใหม่กว่าเวลาที่ระบุ
	GetConversationsAfterTime(userID uuid.UUID, afterTime string, limit int, convType string, pinned bool) ([]*dto.ConversationDTO, int, error)

	// GetConversationsBeforeID ดึงการสนทนาที่เก่ากว่า ID ที่ระบุ
	GetConversationsBeforeID(userID, beforeID uuid.UUID, limit int, convType string, pinned bool) ([]*dto.ConversationDTO, int, error)

	// GetConversationsAfterID ดึงการสนทนาที่ใหม่กว่า ID ที่ระบุ
	GetConversationsAfterID(userID, afterID uuid.UUID, limit int, convType string, pinned bool) ([]*dto.ConversationDTO, int, error)

	// UpdateConversation อัปเดตข้อมูลการสนทนา
	UpdateConversation(id uuid.UUID, updateData types.JSONB) error

	// GetConversationMediaSummary ดึงสรุปจำนวน media และ link ในการสนทนา
	GetConversationMediaSummary(conversationID, userID uuid.UUID) (*dto.MediaSummaryDTO, error)

	// GetConversationMediaByType ดึงรายละเอียด media ตามประเภทพร้อม pagination
	GetConversationMediaByType(conversationID, userID uuid.UUID, mediaType string, limit, offset int) (*dto.MediaListDTO, error)

	// SetHiddenStatus ตั้งค่าสถานะการซ่อนการสนทนา
	SetHiddenStatus(conversationID, userID uuid.UUID, isHidden bool) error

	// DeleteConversation ลบการสนทนา (smart delete - hide for direct, leave for group)
	DeleteConversation(conversationID, userID uuid.UUID) (string, error)

	// TransferOwnership โอนความเป็นเจ้าของกลุ่มให้สมาชิกคนอื่น
	TransferOwnership(conversationID, currentOwnerID, newOwnerID uuid.UUID) error
}
