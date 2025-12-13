// domain/dto/user_friendship_dto.go
package dto

import (
	"time"

	"github.com/google/uuid"
)

// ============ Constants ============

// FriendshipStatus สถานะความสัมพันธ์
type FriendshipStatus string

const (
	FriendshipStatusPending  FriendshipStatus = "pending"
	FriendshipStatusAccepted FriendshipStatus = "accepted"
	FriendshipStatusRejected FriendshipStatus = "rejected"
	FriendshipStatusBlocked  FriendshipStatus = "blocked"
	FriendshipStatusNone     FriendshipStatus = "none"
)

// ============ Request DTOs ============

// SearchUsersRequest สำหรับการค้นหาผู้ใช้
type SearchUsersRequest struct {
	Query  string `json:"q" query:"q" validate:"required"`
	Limit  int    `json:"limit" query:"limit"`
	Offset int    `json:"offset" query:"offset"`
}

// SendFriendRequestParam สำหรับพารามิเตอร์การส่งคำขอเป็นเพื่อน
type SendFriendRequestParam struct {
	FriendID       uuid.UUID `json:"friend_id" validate:"required"`
	InitialMessage *string   `json:"initial_message,omitempty"` // ข้อความแรกที่ส่งพร้อมคำขอ (Message Request feature)
}

// FriendRequestParam สำหรับพารามิเตอร์การจัดการคำขอเป็นเพื่อน
type FriendRequestParam struct {
	RequestID uuid.UUID `json:"request_id" validate:"required"`
}

// BlockUserParam สำหรับพารามิเตอร์การบล็อกผู้ใช้
type BlockUserParam struct {
	UserID uuid.UUID `json:"user_id" validate:"required"`
}

// ============ Response Data DTOs ============

// FriendshipData ข้อมูลความสัมพันธ์
type FriendshipData struct {
	ID               uuid.UUID        `json:"id"`
	UserID           uuid.UUID        `json:"user_id"`
	FriendID         uuid.UUID        `json:"friend_id"`
	Status           FriendshipStatus `json:"status"`
	RequestedAt      time.Time        `json:"requested_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	InitialMessage   *string          `json:"initial_message,omitempty"`
	InitialMessageAt *time.Time       `json:"initial_message_at,omitempty"`
}

// FriendItem ข้อมูลเพื่อน
type FriendItem struct {
	ID               uuid.UUID        `json:"id"`
	Username         string           `json:"username"`
	DisplayName      string           `json:"display_name"`
	ProfileImageURL  *string          `json:"profile_image_url,omitempty"`
	Bio              *string          `json:"bio,omitempty"`
	Status           string           `json:"status"`
	LastActiveAt     *time.Time       `json:"last_active_at,omitempty"`
	FriendshipID     uuid.UUID        `json:"friendship_id"`
	FriendshipStatus FriendshipStatus `json:"friendship_status"`
	ConversationID   *string          `json:"conversation_id,omitempty"`
}

// FriendSearchResultItem ผลลัพธ์การค้นหาผู้ใช้
type FriendSearchResultItem struct {
	ID               uuid.UUID        `json:"id"`
	Username         string           `json:"username"`
	DisplayName      string           `json:"display_name"`
	ProfileImageURL  *string          `json:"profile_image_url,omitempty"`
	Bio              *string          `json:"bio,omitempty"`
	FriendshipStatus FriendshipStatus `json:"friendship_status"`
	FriendshipID     *uuid.UUID       `json:"friendship_id,omitempty"`
}

// PendingRequestItem ข้อมูลคำขอเป็นเพื่อนที่รออยู่
type PendingRequestItem struct {
	RequestID        uuid.UUID  `json:"request_id"`
	UserID           uuid.UUID  `json:"user_id"`
	Username         string     `json:"username"`
	DisplayName      string     `json:"display_name"`
	ProfileImageURL  string     `json:"profile_image_url,omitempty"`
	RequestedAt      time.Time  `json:"requested_at"`
	InitialMessage   *string    `json:"initial_message,omitempty"`
	InitialMessageAt *time.Time `json:"initial_message_at,omitempty"`
}

// BlockedUserItem ข้อมูลผู้ใช้ที่ถูกบล็อก
type BlockedUserItem struct {
	ID              uuid.UUID `json:"id"`
	Username        string    `json:"username"`
	DisplayName     string    `json:"display_name"`
	ProfileImageURL *string   `json:"profile_image_url,omitempty"`
}

// ============ Response Wrapper DTOs ============

// FriendsListResponse สำหรับผลลัพธ์รายชื่อเพื่อน
type FriendsListResponse struct {
	GenericResponse
	Data []FriendItem `json:"data"`
}

// FriendSearchResponse สำหรับผลลัพธ์การค้นหาผู้ใช้
type FriendSearchResponse struct {
	GenericResponse
	Data []FriendSearchResultItem `json:"data"`
}

// FriendshipInfoResponse สำหรับข้อมูลความสัมพันธ์
type FriendshipInfoResponse struct {
	GenericResponse
	Data FriendshipData `json:"data"`
}

// PendingRequestsResponse สำหรับผลลัพธ์รายการคำขอเป็นเพื่อนที่รออยู่
type PendingRequestsResponse struct {
	GenericResponse
	Data []PendingRequestItem `json:"data"`
}

// BlockedUsersResponse สำหรับผลลัพธ์รายชื่อผู้ใช้ที่ถูกบล็อก
type BlockedUsersResponse struct {
	GenericResponse
	Data []BlockedUserItem `json:"data"`
}

// ============ Specific Response DTOs ============

// SendFriendRequestResponse การตอบกลับสำหรับการส่งคำขอเป็นเพื่อน
type SendFriendRequestResponse struct {
	GenericResponse
	Data FriendshipData `json:"data"`
}

// AcceptFriendRequestResponse การตอบกลับสำหรับการยอมรับคำขอเป็นเพื่อน
type AcceptFriendRequestResponse struct {
	GenericResponse
	Data FriendshipData `json:"data"`
}

// RejectFriendRequestResponse การตอบกลับสำหรับการปฏิเสธคำขอเป็นเพื่อน
type RejectFriendRequestResponse struct {
	GenericResponse
	Data FriendshipData `json:"data"`
}

// RemoveFriendResponse การตอบกลับสำหรับการลบเพื่อน
type RemoveFriendResponse struct {
	GenericResponse
}

// BlockUserResponse การตอบกลับสำหรับการบล็อกผู้ใช้
type BlockUserResponse struct {
	GenericResponse
}

// UnblockUserResponse การตอบกลับสำหรับการเลิกบล็อกผู้ใช้
type UnblockUserResponse struct {
	GenericResponse
}

// สำหรับ NotifyFriendRequestReceived

type FriendRequestDetailDTO struct {
	FriendshipData          // ฝัง FriendshipData ไว้
	SenderInfo     UserInfo `json:"sender_info"` // ข้อมูลผู้ส่งคำขอ
	FriendInfo     UserInfo `json:"friend_info"` // ข้อมูลผู้รับคำขอ
}
type UserInfo struct {
	ID              uuid.UUID `json:"id"`
	Username        string    `json:"username"`
	DisplayName     string    `json:"display_name"`
	ProfileImageURL *string   `json:"profile_image_url,omitempty"`
}
