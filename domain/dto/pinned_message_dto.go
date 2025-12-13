// domain/dto/pinned_message_dto.go
package dto

import (
	"time"

	"github.com/google/uuid"
)

// ============ Request DTOs ============

// PinMessageRequest สำหรับการปักหมุดข้อความ
type PinMessageRequest struct {
	PinType string `json:"pin_type" validate:"required,oneof=personal public"`
}

// UnpinMessageRequest สำหรับการยกเลิกปักหมุดข้อความ
type UnpinMessageRequest struct {
	PinType string `json:"pin_type" validate:"required,oneof=personal public"`
}

// GetPinnedMessagesRequest สำหรับการดึงข้อมูลข้อความที่ปักหมุด
type GetPinnedMessagesRequest struct {
	PinType string `json:"pin_type,omitempty" validate:"omitempty,oneof=personal public all"`
	Limit   int    `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
	Offset  int    `json:"offset,omitempty" validate:"omitempty,min=0"`
}

// ============ Response DTOs ============

// PinnedMessageDTO ข้อมูลข้อความที่ปักหมุด
type PinnedMessageDTO struct {
	ID             uuid.UUID     `json:"id"`
	MessageID      uuid.UUID     `json:"message_id"`
	ConversationID uuid.UUID     `json:"conversation_id"`
	UserID         uuid.UUID     `json:"user_id"`
	PinType        string        `json:"pin_type"`
	PinnedAt       time.Time     `json:"pinned_at"`
	PinnedBy       *UserBasicDTO `json:"pinned_by,omitempty"`
	Message        *MessageDTO   `json:"message,omitempty"`
}

// PinnedMessagesListDTO รายการข้อความที่ปักหมุด
type PinnedMessagesListDTO struct {
	ConversationID uuid.UUID          `json:"conversation_id"`
	Total          int64              `json:"total"`
	PinnedMessages []PinnedMessageDTO `json:"pinned_messages"`
}
