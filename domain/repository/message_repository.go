// domain/repository/message_repository.go
package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
)

// MessageRepository เป็น interface สำหรับจัดการข้อมูลข้อความ
type MessageRepository interface {
	// การดึงข้อมูลข้อความ
	GetByID(id uuid.UUID) (*models.Message, error)
	GetMessagesByConversationID(conversationID uuid.UUID, limit, offset int) ([]*models.Message, int64, error)

	// การสร้างและแก้ไขข้อความ
	Create(message *models.Message) error
	BulkCreate(messages []*models.Message) error
	Update(message *models.Message) error
	UpdateFields(messageID uuid.UUID, updates map[string]interface{}) error
	Delete(id uuid.UUID) error

	// การจัดการประวัติการแก้ไขและลบ
	CreateEditHistory(history *models.MessageEditHistory) error
	GetEditHistory(messageID uuid.UUID) ([]*models.MessageEditHistory, error)
	CreateDeleteHistory(history *models.MessageDeleteHistory) error
	GetDeleteHistory(messageID uuid.UUID) ([]*models.MessageDeleteHistory, error)

	// การจัดการการอ่านข้อความ
	MarkAsRead(messageID, userID uuid.UUID, readAt time.Time) error
	GetReads(messageID uuid.UUID) ([]*models.MessageRead, error)
	IsMessageRead(messageID, userID uuid.UUID) (bool, error)
	MarkAllAsRead(conversationID, userID uuid.UUID, readAt time.Time) error

	// ตรวจสอบความเป็นเจ้าของและสิทธิ์
	IsSender(messageID, userID uuid.UUID) (bool, error)
	IsConversationAdmin(conversationID, userID uuid.UUID) (bool, error)

	// อัพเดตข้อความล่าสุดในการสนทนา
	UpdateConversationLastMessage(conversationID uuid.UUID, lastMessageText string, lastMessageAt time.Time) error

	GetLastMessageByConversation(conversationID uuid.UUID) (*models.Message, error)
	GetLastNonDeletedMessageByConversation(conversationID uuid.UUID) (*models.Message, error)

	// GetMessagesBefore ดึงข้อความที่เก่ากว่า ID ที่ระบุ
	GetMessagesBefore(conversationID, messageID uuid.UUID, limit int) ([]*models.Message, error)

	// GetMessagesAfter ดึงข้อความที่ใหม่กว่า ID ที่ระบุ
	GetMessagesAfter(conversationID, messageID uuid.UUID, limit int) ([]*models.Message, error)

	// CountAllMessages นับจำนวนข้อความทั้งหมดในการสนทนา
	CountAllMessages(conversationID uuid.UUID) (int64, error)

	// เมธอดสำหรับดึงข้อความหลังเวลาที่กำหนดและไม่ใช่ของผู้ใช้
	GetMessagesAfterTime(conversationID uuid.UUID, afterTime time.Time, excludeUserID uuid.UUID) ([]*models.Message, error)

	// เมธอดสำหรับดึงข้อความทั้งหมดที่ไม่ใช่ของผู้ใช้ (สำหรับกรณีไม่มี LastReadAt)
	GetAllUnreadMessages(conversationID uuid.UUID, excludeUserID uuid.UUID) ([]*models.Message, error)

	// สรุปข้อมูล media และ link
	GetMessageTypeSummary(conversationID uuid.UUID) (map[string]int64, error)
	CountMessagesWithLinks(conversationID uuid.UUID) (int64, error)
	GetMediaByType(conversationID uuid.UUID, messageType string, limit, offset int) ([]*models.Message, int64, error)

	// Pin messages
	PinMessage(messageID, userID uuid.UUID) error
	UnpinMessage(messageID uuid.UUID) error
	GetPinnedMessages(conversationID uuid.UUID, limit, offset int) ([]*models.Message, int64, error)

	// Jump to date
	FindByDateRange(conversationID uuid.UUID, startDate, endDate time.Time, limit int) ([]*models.Message, int64, error)

	// Search messages (CURSOR-BASED)
	// Returns: messages, nextCursor, hasMore, error
	SearchMessages(searchQuery string, conversationID *uuid.UUID, limit int, cursor *string, direction string) ([]*models.Message, *string, bool, error)

	// Bulk/Album messages
	GetMessagesByAlbumID(albumID string) ([]*models.Message, error)
}
