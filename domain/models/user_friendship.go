// domain/models/user_friendship.go

package models

import (
	"time"

	"github.com/google/uuid"
)

// UserFriendship - ความสัมพันธ์ระหว่างผู้ใช้ในฐานะเพื่อน
type UserFriendship struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	UserID      uuid.UUID `json:"user_id" gorm:"type:uuid;not null"`
	FriendID    uuid.UUID `json:"friend_id" gorm:"type:uuid;not null"`
	Status      string    `json:"status" gorm:"type:varchar(20);not null;default:'pending'"` // pending, accepted, rejected, blocked
	RequestedAt time.Time `json:"requested_at" gorm:"type:timestamp with time zone;default:now()"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"type:timestamp with time zone;default:now()"`

	// Initial message sent with friend request (Message Request feature)
	InitialMessage   *string    `json:"initial_message,omitempty" gorm:"type:text"`
	InitialMessageAt *time.Time `json:"initial_message_at,omitempty" gorm:"type:timestamp with time zone"`

	// Associations
	User   *User `json:"user,omitempty" gorm:"foreignkey:UserID"`
	Friend *User `json:"friend,omitempty" gorm:"foreignkey:FriendID"`
}

// TableName - ระบุชื่อตารางใน database
func (UserFriendship) TableName() string {
	return "user_friendships"
}
