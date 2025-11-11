// domain/models/user.go

package models

import (
	"time"

	"github.com/thizplus/gofiber-chat-api/domain/types"

	"github.com/google/uuid"
)

// User - ผู้ใช้ในระบบ
type User struct {
	ID              uuid.UUID   `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Username        string      `json:"username" gorm:"type:varchar(50);not null;unique"`
	Email           string      `json:"email,omitempty" gorm:"type:varchar(255);unique"`
	PasswordHash    string      `json:"-" gorm:"type:text"` // ไม่ส่งกลับในการ response JSON
	DisplayName     string      `json:"display_name,omitempty" gorm:"type:varchar(100)"`
	ProfileImageURL string      `json:"profile_image_url,omitempty" gorm:"type:text"`
	Bio             string      `json:"bio,omitempty" gorm:"type:text"`
	CreatedAt       time.Time   `json:"created_at" gorm:"type:timestamp with time zone;default:now()"`
	LastActiveAt    *time.Time  `json:"last_active_at,omitempty" gorm:"type:timestamp with time zone"`
	Settings        types.JSONB `json:"settings,omitempty" gorm:"type:jsonb;default:'{}'::jsonb"`
	Status          string      `json:"status" gorm:"type:varchar(20);default:'active'"`

	// Associations
	ConversationMembers  []*ConversationMember  `json:"conversation_members,omitempty" gorm:"foreignkey:UserID"`
	CreatedConversations []*Conversation        `json:"created_conversations,omitempty" gorm:"foreignkey:CreatorID"`
	Messages             []*Message             `json:"messages,omitempty" gorm:"foreignkey:SenderID"`
	MessageReads         []*MessageRead         `json:"message_reads,omitempty" gorm:"foreignkey:UserID"`
	Events               []*UserEvent           `json:"events,omitempty" gorm:"foreignkey:UserID"`
	FavoriteStickers     []*UserFavoriteSticker `json:"favorite_stickers,omitempty" gorm:"foreignkey:UserID"`
	FriendshipsAsUser    []*UserFriendship      `json:"friendships_as_user,omitempty" gorm:"foreignkey:UserID"`
	FriendshipsAsFriend  []*UserFriendship      `json:"friendships_as_friend,omitempty" gorm:"foreignkey:FriendID"`
	RecentStickers       []*UserRecentSticker   `json:"recent_stickers,omitempty" gorm:"foreignkey:UserID"`
	StickerSets          []*UserStickerSet      `json:"sticker_sets,omitempty" gorm:"foreignkey:UserID"`
	RefreshTokens        []*RefreshToken        `json:"refresh_tokens,omitempty" gorm:"foreignkey:UserID"`
}

// TableName - ระบุชื่อตารางใน database
func (User) TableName() string {
	return "users"
}
