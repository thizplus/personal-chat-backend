// domain/models/note.go

package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/types"
)

// NoteVisibility - ระดับการมองเห็น Note
type NoteVisibility string

const (
	NoteVisibilityPrivate NoteVisibility = "private" // เห็นเฉพาะเจ้าของ
	NoteVisibilityShared  NoteVisibility = "shared"  // เห็นทุกคนใน conversation (เฉพาะ conversation notes)
)

// Note - บันทึกส่วนตัวของผู้ใช้
// รองรับทั้ง Personal Notes (conversation_id = NULL) และ Conversation Notes (conversation_id = UUID)
type Note struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	UserID         uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;index"`
	ConversationID *uuid.UUID     `json:"conversation_id,omitempty" gorm:"type:uuid;index"` // NULL = Personal Note, UUID = Conversation Note
	Title          string         `json:"title" gorm:"type:varchar(255)"`
	Content        string         `json:"content" gorm:"type:text"`
	Tags           types.JSONB    `json:"tags,omitempty" gorm:"type:jsonb;default:'[]'::jsonb"` // Format: ["tag1", "tag2"]
	IsPinned       bool           `json:"is_pinned" gorm:"default:false"`
	Visibility     NoteVisibility `json:"visibility" gorm:"type:varchar(20);default:'private'"` // private = เห็นเฉพาะเจ้าของ, shared = เห็นทุกคนใน conversation

	CreatedAt time.Time `json:"created_at" gorm:"type:timestamp with time zone;default:now()"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:timestamp with time zone;default:now()"`

	// Associations
	User         *User         `json:"user,omitempty" gorm:"foreignkey:UserID"`
	Conversation *Conversation `json:"conversation,omitempty" gorm:"foreignkey:ConversationID"`
}

// TableName - ระบุชื่อตารางใน database
func (Note) TableName() string {
	return "notes"
}
