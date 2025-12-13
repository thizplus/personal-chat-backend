// domain/models/pinned_message.go
package models

import (
	"time"

	"github.com/google/uuid"
)

// Pin type constants
const (
	PinTypePersonal = "personal"
	PinTypePublic   = "public"
)

// PinnedMessage represents a pinned message in a conversation
type PinnedMessage struct {
	ID             uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	MessageID      uuid.UUID  `json:"message_id" gorm:"type:uuid;not null"`
	ConversationID uuid.UUID  `json:"conversation_id" gorm:"type:uuid;not null"`
	UserID         uuid.UUID  `json:"user_id" gorm:"type:uuid;not null"` // User who pinned
	PinType        string     `json:"pin_type" gorm:"type:varchar(20);not null"`
	PinnedAt       time.Time  `json:"pinned_at" gorm:"type:timestamp with time zone;default:now()"`
	CreatedAt      time.Time  `json:"created_at" gorm:"type:timestamp with time zone;default:now()"`
	UpdatedAt      time.Time  `json:"updated_at" gorm:"type:timestamp with time zone;default:now()"`

	// Associations
	Message      *Message      `json:"message,omitempty" gorm:"foreignkey:MessageID"`
	Conversation *Conversation `json:"conversation,omitempty" gorm:"foreignkey:ConversationID"`
	User         *User         `json:"user,omitempty" gorm:"foreignkey:UserID"`
}

// TableName returns the table name for GORM
func (PinnedMessage) TableName() string {
	return "pinned_messages"
}
