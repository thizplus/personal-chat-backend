// domain/models/conversation.go

package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/types"
)

// Conversation - การสนทนาระหว่างผู้ใช้หรือกลุ่ม
type Conversation struct {
	ID              uuid.UUID   `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Type            string      `json:"type" gorm:"type:varchar(20);not null"` // private, group
	Title           string      `json:"title,omitempty" gorm:"type:varchar(100)"`
	IconURL         string      `json:"icon_url,omitempty" gorm:"type:text"`
	CreatedAt       time.Time   `json:"created_at" gorm:"type:timestamp with time zone;default:now()"`
	UpdatedAt       time.Time   `json:"updated_at" gorm:"type:timestamp with time zone;default:now()"`
	LastMessageText string      `json:"last_message_text,omitempty" gorm:"type:text"`
	LastMessageAt   *time.Time  `json:"last_message_at,omitempty" gorm:"type:timestamp with time zone"`
	LastMessageID   *uuid.UUID  `json:"last_message_id,omitempty" gorm:"type:uuid"`
	CreatorID       *uuid.UUID  `json:"creator_id,omitempty" gorm:"type:uuid"`
	IsActive        bool        `json:"is_active" gorm:"default:true"`
	Metadata        types.JSONB `json:"metadata,omitempty" gorm:"type:jsonb;default:'{}'::jsonb"`

	// Associations
	Creator  *User                 `json:"creator,omitempty" gorm:"foreignkey:CreatorID"`
	Members  []*ConversationMember `json:"members,omitempty" gorm:"foreignkey:ConversationID"`
	Messages []*Message            `json:"messages,omitempty" gorm:"foreignkey:ConversationID"`
}

// TableName - ระบุชื่อตารางใน database
func (Conversation) TableName() string {
	return "conversations"
}
