// domain/dto/conversation_dto.go
package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/types"
)

// domain/dto/conversation_dto.go (เพิ่มเติม)

// ============ Request DTOs ============

// UpdateConversationRequest สำหรับการอัปเดตข้อมูลการสนทนา
type UpdateConversationRequest struct {
	Title   string `json:"title"`
	IconURL string `json:"icon_url"`
}

// CreateDirectConversationRequest สำหรับการสร้างการสนทนาแบบส่วนตัว
type CreateDirectConversationRequest struct {
	Type     string `json:"type" validate:"required,eq=direct"`
	MemberID string `json:"member_id" validate:"required,uuid"`
}

// CreateGroupConversationRequest สำหรับการสร้างการสนทนาแบบกลุ่ม
type CreateGroupConversationRequest struct {
	Type      string   `json:"type" validate:"required,eq=group"`
	Title     string   `json:"title" validate:"required"`
	IconURL   string   `json:"icon_url,omitempty"`
	MemberIDs []string `json:"member_ids" validate:"omitempty,dive,uuid"`
}

// CreateBusinessConversationRequest สำหรับการสร้างการสนทนากับธุรกิจ
type CreateBusinessConversationRequest struct {
	Type       string `json:"type" validate:"required,eq=business"`
	BusinessID string `json:"business_id" validate:"required,uuid"`
}

// ConversationQueryRequest สำหรับการดึงรายการการสนทนา
type ConversationQueryRequest struct {
	Limit      int    `json:"limit,omitempty" validate:"omitempty,min=1,max=50"`
	Offset     int    `json:"offset,omitempty" validate:"omitempty,min=0"`
	BeforeTime string `json:"before_time,omitempty"`
	AfterTime  string `json:"after_time,omitempty"`
	BeforeID   string `json:"before_id,omitempty" validate:"omitempty,uuid"`
	AfterID    string `json:"after_id,omitempty" validate:"omitempty,uuid"`
	Type       string `json:"type,omitempty" validate:"omitempty,oneof=direct group business"`
	Pinned     bool   `json:"pinned,omitempty"`
	Format     string `json:"format,omitempty" validate:"omitempty,oneof=legacy old new"`
}

// ConversationMessagesQueryRequest สำหรับการดึงข้อความในการสนทนา
type ConversationMessagesQueryRequest struct {
	Limit       int    `json:"limit,omitempty" validate:"omitempty,min=1,max=50"`
	Offset      int    `json:"offset,omitempty" validate:"omitempty,min=0"`
	Before      string `json:"before,omitempty" validate:"omitempty,uuid"`
	After       string `json:"after,omitempty" validate:"omitempty,uuid"`
	Target      string `json:"target,omitempty" validate:"omitempty,uuid"`
	BeforeCount int    `json:"before_count,omitempty" validate:"omitempty,min=1,max=100"`
	AfterCount  int    `json:"after_count,omitempty" validate:"omitempty,min=1,max=100"`
}

// ConversationPinRequest สำหรับการปักหมุดการสนทนา
type ConversationPinRequest struct {
	IsPinned bool `json:"is_pinned" validate:"required"`
}

// ConversationMuteRequest สำหรับการปิดเสียงการสนทนา
type ConversationMuteRequest struct {
	IsMuted bool `json:"is_muted" validate:"required"`
}

// ConversationHideRequest สำหรับการซ่อน/แสดงการสนทนา
type ConversationHideRequest struct {
	IsHidden bool `json:"is_hidden" validate:"required"`
}

// ============ Response DTOs ============

// ConversationDTO โครงสร้างข้อมูลสำหรับส่งกลับข้อมูลการสนทนา
type ConversationDTO struct {
	ID              uuid.UUID   `json:"id"`
	Type            string      `json:"type"`
	Title           string      `json:"title,omitempty"`
	IconURL         string      `json:"icon_url,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
	LastMessageText string      `json:"last_message_text,omitempty"`
	LastMessageAt   *time.Time  `json:"last_message_at,omitempty"`
	CreatorID       *uuid.UUID  `json:"creator_id,omitempty"` // เพิ่มฟิลด์นี้
	BusinessID      *uuid.UUID  `json:"business_id,omitempty"`
	IsActive        bool        `json:"is_active"`
	Metadata        types.JSONB `json:"metadata,omitempty"` // เพิ่มฟิลด์นี้
	MemberCount     int         `json:"member_count"`
	UnreadCount     int         `json:"unread_count"`

	// Mention-related fields
	HasUnreadMention      bool `json:"has_unread_mention"`
	UnreadMentionCount    int  `json:"unread_mention_count"`
	LastMessageHasMention bool `json:"last_message_has_mention"`

	IsPinned        bool        `json:"is_pinned"`
	IsMuted         bool        `json:"is_muted"`
	IsHidden        bool        `json:"is_hidden"`
	HiddenAt        *time.Time  `json:"hidden_at,omitempty"`
	ContactInfo     types.JSONB `json:"contact_info,omitempty"`
	BusinessInfo    types.JSONB `json:"business_info,omitempty"`
}

// ConversationCreateResponse สำหรับผลลัพธ์การสร้างการสนทนา
type ConversationCreateResponse struct {
	GenericResponse
	Conversation ConversationDTO `json:"conversation"`
}

// ConversationUpdateResponse สำหรับผลลัพธ์การอัปเดตการสนทนา
type ConversationUpdateResponse struct {
	GenericResponse
}

// ConversationPinResponse สำหรับผลลัพธ์การปักหมุดการสนทนา
type ConversationPinResponse struct {
	GenericResponse
	Data struct {
		IsPinned bool `json:"is_pinned"`
	} `json:"data"`
}

// ConversationMuteResponse สำหรับผลลัพธ์การปิดเสียงการสนทนา
type ConversationMuteResponse struct {
	GenericResponse
	Data struct {
		IsMuted bool `json:"is_muted"`
	} `json:"data"`
}

// ConversationHideResponse สำหรับผลลัพธ์การซ่อน/แสดงการสนทนา
type ConversationHideResponse struct {
	GenericResponse
	Data struct {
		IsHidden bool       `json:"is_hidden"`
		HiddenAt *time.Time `json:"hidden_at,omitempty"`
	} `json:"data"`
}

// ConversationDeleteResponse สำหรับผลลัพธ์การลบการสนทนา
type ConversationDeleteResponse struct {
	GenericResponse
	Data struct {
		ConversationID string `json:"conversation_id"`
		Action         string `json:"action"` // "hidden" or "left"
		Message        string `json:"message"`
	} `json:"data"`
}

// ConversationListData ข้อมูลรายการการสนทนา
type ConversationListData struct {
	Conversations []ConversationDTO `json:"conversations"`
	HasMore       bool              `json:"has_more,omitempty"`
	Pagination    PaginationData    `json:"pagination"`
}

// ConversationListResponse สำหรับผลลัพธ์การดึงรายการการสนทนา
type ConversationListResponse struct {
	GenericResponse
	Data ConversationListData `json:"data"`
}

// LegacyConversationListResponse สำหรับผลลัพธ์การดึงรายการการสนทนาในรูปแบบเก่า
type LegacyConversationListResponse struct {
	Success       bool              `json:"success"`
	Message       string            `json:"message"`
	Conversations []ConversationDTO `json:"conversations"`
	Pagination    PaginationData    `json:"pagination"`
}

// MessageContextData ข้อมูลบริบทของข้อความ
type MessageContextData struct {
	Messages      []MessageDTO `json:"messages"`
	TargetID      string       `json:"target_id"`
	HasMoreBefore bool         `json:"has_more_before"`
	HasMoreAfter  bool         `json:"has_more_after"`
	BusinessID    *uuid.UUID   `json:"business_id,omitempty"`
	UserRole      string       `json:"user_role,omitempty"`
}

// MessagesListData ข้อมูลรายการข้อความ
type MessagesListData struct {
	Messages   []MessageDTO `json:"messages"`
	HasMore    bool         `json:"has_more"`
	Total      int64        `json:"total"`
	BusinessID *uuid.UUID   `json:"business_id,omitempty"`
	UserRole   string       `json:"user_role,omitempty"`
}

// ConversationMessagesResponse สำหรับผลลัพธ์การดึงข้อความในการสนทนา
type ConversationMessagesResponse struct {
	GenericResponse
	Data MessagesListData `json:"data"`
}

// MessageContextResponse สำหรับผลลัพธ์การดึงบริบทของข้อความ
type MessageContextResponse struct {
	GenericResponse
	Data MessageContextData `json:"data"`
}

// BusinessConversationListData ข้อมูลรายการการสนทนาของธุรกิจ
type BusinessConversationListData struct {
	Conversations []ConversationDTO `json:"conversations"`
	HasMore       bool              `json:"has_more"`
	Total         int               `json:"total"`
	BusinessID    uuid.UUID         `json:"business_id"`
	UserRole      string            `json:"user_role"`
	Pagination    PaginationData    `json:"pagination"`
}

// BusinessConversationListResponse สำหรับผลลัพธ์การดึงรายการการสนทนาของธุรกิจ
type BusinessConversationListResponse struct {
	GenericResponse
	Data BusinessConversationListData `json:"data"`
}
