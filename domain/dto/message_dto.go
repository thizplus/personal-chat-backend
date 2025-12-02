package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/types"
)

// ============ Request DTOs ============

// TextMessageRequest สำหรับการส่งข้อความประเภทข้อความ
type TextMessageRequest struct {
	TempID   string      `json:"temp_id,omitempty"`
	Content  string      `json:"content" validate:"required"`
	Metadata types.JSONB `json:"metadata,omitempty"`
}

// StickerMessageRequest สำหรับการส่งข้อความประเภทสติกเกอร์
type StickerMessageRequest struct {
	TempID            string      `json:"temp_id,omitempty"`
	StickerID         uuid.UUID   `json:"sticker_id" validate:"required"`
	StickerSetID      uuid.UUID   `json:"sticker_set_id" validate:"required"`
	MediaURL          string      `json:"media_url" validate:"required"`
	MediaThumbnailURL string      `json:"media_thumbnail_url,omitempty"`
	Metadata          types.JSONB `json:"metadata,omitempty"`
}

// ImageMessageRequest สำหรับการส่งข้อความประเภทรูปภาพ
type ImageMessageRequest struct {
	TempID            string      `json:"temp_id,omitempty"`
	MediaURL          string      `json:"media_url" validate:"required"`
	MediaThumbnailURL string      `json:"media_thumbnail_url,omitempty"`
	Caption           string      `json:"caption,omitempty"`
	Metadata          types.JSONB `json:"metadata,omitempty"`
}

// FileMessageRequest สำหรับการส่งข้อความประเภทไฟล์
type FileMessageRequest struct {
	TempID   string      `json:"temp_id,omitempty"`
	MediaURL string      `json:"media_url" validate:"required"`
	FileName string      `json:"file_name" validate:"required"`
	FileSize int64       `json:"file_size" validate:"required"`
	FileType string      `json:"file_type" validate:"required"`
	Metadata types.JSONB `json:"metadata,omitempty"`
}

// EditMessageRequest สำหรับการแก้ไขข้อความ
type EditMessageRequest struct {
	Content string `json:"content" validate:"required"`
}

// ReplyMessageRequest สำหรับการตอบกลับข้อความ
type ReplyMessageRequest struct {
	MessageType       string      `json:"message_type" validate:"required,oneof=text image file sticker"`
	Content           string      `json:"content,omitempty"`
	MediaURL          string      `json:"media_url,omitempty"`
	MediaThumbnailURL string      `json:"media_thumbnail_url,omitempty"`
	Metadata          types.JSONB `json:"metadata,omitempty"`
}

// BulkMessageRequest สำหรับส่งหลายไฟล์ใน 1 message (Album/Group Message)
type BulkMessageRequest struct {
	Messages []BulkMessageItem `json:"messages" validate:"required,min=1,max=10,dive"`
}

// BulkMessageItem รายละเอียดของแต่ละไฟล์ในกลุ่ม
type BulkMessageItem struct {
	TempID            string      `json:"temp_id,omitempty"`
	MessageType       string      `json:"message_type" validate:"required,oneof=image video file"`
	MediaURL          string      `json:"media_url" validate:"required"`
	MediaThumbnailURL string      `json:"media_thumbnail_url,omitempty"`
	Caption           string      `json:"caption,omitempty"` // ใช้ได้เฉพาะ item แรก
	FileName          string      `json:"file_name,omitempty"`
	FileSize          int64       `json:"file_size,omitempty"`
	FileType          string      `json:"file_type,omitempty"`
	Metadata          types.JSONB `json:"metadata,omitempty"`
}

// BusinessMessageRequest สำหรับการส่งข้อความในนามธุรกิจ
type BusinessTextMessageRequest struct {
	Content   string      `json:"content" validate:"required"`
	Metadata  types.JSONB `json:"metadata,omitempty"`
	ReplyToID *uuid.UUID  `json:"reply_to_id,omitempty"`
}

// BusinessStickerMessageRequest สำหรับการส่งสติกเกอร์ในนามธุรกิจ
type BusinessStickerMessageRequest struct {
	StickerID         uuid.UUID   `json:"sticker_id" validate:"required"`
	StickerSetID      uuid.UUID   `json:"sticker_set_id" validate:"required"`
	MediaURL          string      `json:"media_url" validate:"required"`
	MediaThumbnailURL string      `json:"media_thumbnail_url,omitempty"`
	Metadata          types.JSONB `json:"metadata,omitempty"`
	ReplyToID         *uuid.UUID  `json:"reply_to_id,omitempty"`
}

// BusinessImageMessageRequest สำหรับการส่งรูปภาพในนามธุรกิจ
type BusinessImageMessageRequest struct {
	MediaURL          string      `json:"media_url" validate:"required"`
	MediaThumbnailURL string      `json:"media_thumbnail_url,omitempty"`
	Caption           string      `json:"caption,omitempty"`
	Metadata          types.JSONB `json:"metadata,omitempty"`
	ReplyToID         *uuid.UUID  `json:"reply_to_id,omitempty"`
}

// BusinessFileMessageRequest สำหรับการส่งไฟล์ในนามธุรกิจ
type BusinessFileMessageRequest struct {
	MediaURL  string      `json:"media_url" validate:"required"`
	FileName  string      `json:"file_name" validate:"required"`
	FileSize  int64       `json:"file_size" validate:"required"`
	FileType  string      `json:"file_type" validate:"required"`
	Metadata  types.JSONB `json:"metadata,omitempty"`
	ReplyToID *uuid.UUID  `json:"reply_to_id,omitempty"`
}

// ============ Response DTOs ============

// MessageDTO ข้อมูลข้อความ
type MessageDTO struct {
	ID                uuid.UUID  `json:"id"`
	TempID            string     `json:"temp_id,omitempty"` // Temporary ID from frontend
	ConversationID    uuid.UUID  `json:"conversation_id"`
	SenderID          *uuid.UUID `json:"sender_id"`
	SenderType        string     `json:"sender_type"` // user, business, system
	SenderName        string     `json:"sender_name,omitempty"`
	SenderAvatar      string     `json:"sender_avatar,omitempty"`
	MessageType       string     `json:"message_type"` // text, image, file, sticker, album
	Content           string     `json:"content"`
	MediaURL          string     `json:"media_url,omitempty"`
	MediaThumbnailURL string     `json:"media_thumbnail_url,omitempty"`

	// ข้อมูลเพิ่มเติมสำหรับอัลบั้ม
	AlbumFiles interface{} `json:"album_files,omitempty"` // For album messages: [{"id": "uuid", "file_type": "image", "media_url": "...", "position": 0}]

	// ข้อมูลเพิ่มเติมสำหรับไฟล์
	FileName string `json:"file_name,omitempty"`
	FileSize int64  `json:"file_size,omitempty"`
	FileType string `json:"file_type,omitempty"`

	// ข้อมูลเพิ่มเติมสำหรับสติกเกอร์
	StickerID    *uuid.UUID `json:"sticker_id,omitempty"`
	StickerSetID *uuid.UUID `json:"sticker_set_id,omitempty"`

	// ข้อมูลหลัก
	Metadata  types.JSONB `json:"metadata,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`

	// สถานะข้อความ
	IsDeleted bool       `json:"is_deleted"`
	IsEdited  bool       `json:"is_edited"`
	EditCount int        `json:"edit_count"`
	DeletedBy *uuid.UUID `json:"deleted_by,omitempty"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`

	// ข้อมูลการอ่าน
	IsRead         bool        `json:"is_read"`
	ReadCount      int         `json:"read_count"`
	Status         string      `json:"status"` // sent, delivered, read, failed
	ReadByIDs      []uuid.UUID `json:"read_by_ids,omitempty"`
	DeliveredToIDs []uuid.UUID `json:"delivered_to_ids,omitempty"`

	// ข้อมูลการตอบกลับ
	ReplyToID      *uuid.UUID    `json:"reply_to_id,omitempty"`
	ReplyToMessage *ReplyInfoDTO `json:"reply_to_message,omitempty"`

	// ข้อมูลธุรกิจ
	BusinessID *uuid.UUID `json:"business_id,omitempty"`
	AdminID    *uuid.UUID `json:"admin_id,omitempty"`

	// ข้อมูลเพิ่มเติมที่อาจต้องการในอนาคต
	SenderInfo   *UserBasicDTO     `json:"sender_info,omitempty"`
	BusinessInfo *BusinessBasicDTO `json:"business_info,omitempty"`
	AdminInfo    *UserBasicDTO     `json:"admin_info,omitempty"`
}

// UserBasicDTO ข้อมูลพื้นฐานของผู้ใช้
type UserBasicDTO struct {
	ID              uuid.UUID `json:"id"`
	Username        string    `json:"username"`
	DisplayName     string    `json:"display_name"`
	ProfileImageURL string    `json:"profile_image_url,omitempty"`
}

// BusinessBasicDTO ข้อมูลพื้นฐานของธุรกิจ
type BusinessBasicDTO struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	LogoURL     string    `json:"logo_url,omitempty"`
}

// MessageEditHistoryDTO ข้อมูลประวัติการแก้ไขข้อความ
type MessageEditHistoryDTO struct {
	ID         uuid.UUID     `json:"id"`
	MessageID  uuid.UUID     `json:"message_id"`
	Content    string        `json:"content"`
	EditedBy   uuid.UUID     `json:"edited_by"`
	EditorInfo *UserBasicDTO `json:"editor_info,omitempty"`
	EditedAt   time.Time     `json:"edited_at"`
}

// MessageDeleteHistoryDTO ข้อมูลประวัติการลบข้อความ
type MessageDeleteHistoryDTO struct {
	ID          uuid.UUID     `json:"id"`
	MessageID   uuid.UUID     `json:"message_id"`
	DeletedBy   uuid.UUID     `json:"deleted_by"`
	DeleterInfo *UserBasicDTO `json:"deleter_info,omitempty"`
	DeletedAt   time.Time     `json:"deleted_at"`
	Reason      string        `json:"reason,omitempty"`
}

// MessageEditHistoryListDTO รายการประวัติการแก้ไขข้อความ
type MessageEditHistoryListDTO struct {
	MessageID uuid.UUID               `json:"message_id"`
	History   []MessageEditHistoryDTO `json:"history"`
}

// MessageDeleteHistoryListDTO รายการประวัติการลบข้อความ
type MessageDeleteHistoryListDTO struct {
	MessageID uuid.UUID                 `json:"message_id"`
	History   []MessageDeleteHistoryDTO `json:"history"`
}

// MessageResponse สำหรับผลลัพธ์การดำเนินการกับข้อความ
type MessageResponse struct {
	GenericResponse
	Data MessageDTO `json:"data"`
}

// MessageEditHistoryResponse สำหรับผลลัพธ์การดึงประวัติการแก้ไขข้อความ
type MessageEditHistoryResponse struct {
	GenericResponse
	Data MessageEditHistoryListDTO `json:"data"`
}

// MessageDeleteHistoryResponse สำหรับผลลัพธ์การดึงประวัติการลบข้อความ
type MessageDeleteHistoryResponse struct {
	GenericResponse
	Data MessageDeleteHistoryListDTO `json:"data"`
}

// MessageDeleteResponse สำหรับผลลัพธ์การลบข้อความ
type MessageDeleteResponse struct {
	GenericResponse
}

// BulkMessageResponse สำหรับผลลัพธ์การส่งข้อความแบบ bulk
type BulkMessageResponse struct {
	Messages []*MessageDTO `json:"messages"`
	AlbumID  string        `json:"album_id"`
}

// ReplyInfoDTO ข้อมูลย่อของข้อความที่ถูกตอบกลับ
type ReplyInfoDTO struct {
	ID          string     `json:"id"`
	MessageType string     `json:"message_type"`
	Content     string     `json:"content"`
	SenderName  string     `json:"sender_name"`
	SenderID    *uuid.UUID `json:"sender_id,omitempty"`
}
