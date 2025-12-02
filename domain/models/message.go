// domain/models/message.go

package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/types"
)

// Message - ข้อความในการสนทนา
type Message struct {
	ID                uuid.UUID   `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	ConversationID    uuid.UUID   `json:"conversation_id" gorm:"type:uuid;not null"`
	SenderID          *uuid.UUID  `json:"sender_id,omitempty" gorm:"type:uuid"`
	SenderType        string      `json:"sender_type" gorm:"type:varchar(20);default:'user'"`
	MessageType       string      `json:"message_type" gorm:"type:varchar(20);not null"` // text, image, file, sticker, album
	Content           string      `json:"content,omitempty" gorm:"type:text"`
	MediaURL          string      `json:"media_url,omitempty" gorm:"type:text"`
	MediaThumbnailURL string      `json:"media_thumbnail_url,omitempty" gorm:"type:text"`
	AlbumFiles        interface{} `json:"album_files,omitempty" gorm:"type:jsonb;serializer:json"` // For album messages: [{"id": "uuid", "file_type": "image", "media_url": "...", "position": 0}]
	Metadata          types.JSONB `json:"metadata,omitempty" gorm:"type:jsonb;default:'{}'::jsonb"`
	Mentions          types.JSONB `json:"mentions,omitempty" gorm:"type:jsonb"` // Format: [{"user_id": "uuid", "start_index": 0, "length": 10}]

	// Status tracking
	Status      string     `json:"status" gorm:"type:varchar(20);default:'sent'"` // sent, delivered, read
	DeliveredAt *time.Time `json:"delivered_at,omitempty" gorm:"type:timestamp with time zone"`
	ReadAt      *time.Time `json:"read_at,omitempty" gorm:"type:timestamp with time zone"`

	CreatedAt         time.Time   `json:"created_at" gorm:"type:timestamp with time zone;default:now()"`
	UpdatedAt         time.Time   `json:"updated_at" gorm:"type:timestamp with time zone;default:now()"`
	IsDeleted         bool        `json:"is_deleted" gorm:"default:false"`
	ReplyToID         *uuid.UUID  `json:"reply_to_id,omitempty" gorm:"type:uuid"`
	IsEdited          bool        `json:"is_edited" gorm:"default:false"`
	EditCount         int         `json:"edit_count" gorm:"default:0"`

	// Pin fields
	IsPinned bool        `json:"is_pinned" gorm:"default:false"`
	PinnedBy *uuid.UUID  `json:"pinned_by,omitempty" gorm:"type:uuid"`
	PinnedAt *time.Time  `json:"pinned_at,omitempty" gorm:"type:timestamp with time zone"`

	// Forward fields
	IsForwarded   bool        `json:"is_forwarded" gorm:"default:false"`
	ForwardedFrom types.JSONB `json:"forwarded_from,omitempty" gorm:"type:jsonb"` // Format: {"message_id": "uuid", "sender_id": "uuid", "conversation_id": "uuid", "original_timestamp": "..."}

	// Associations
	Conversation  *Conversation         `json:"conversation,omitempty" gorm:"foreignkey:ConversationID"`
	Sender        *User                 `json:"sender,omitempty" gorm:"foreignkey:SenderID"`
	ReplyTo       *Message              `json:"reply_to,omitempty" gorm:"foreignkey:ReplyToID"`
	Reads         []*MessageRead        `json:"reads,omitempty" gorm:"foreignkey:MessageID"`
	EditHistory   []*MessageEditHistory `json:"edit_history,omitempty" gorm:"foreignkey:MessageID"`
	DeleteHistory *MessageDeleteHistory `json:"delete_history,omitempty" gorm:"foreignkey:MessageID"`
	Pinner        *User                 `json:"pinner,omitempty" gorm:"foreignkey:PinnedBy"`
}

// TableName - ระบุชื่อตารางใน database
func (Message) TableName() string {
	return "messages"
}
