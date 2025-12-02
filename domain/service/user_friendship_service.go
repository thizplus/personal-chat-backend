// domain/service/user_friendship_service.go
package service

import (
	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
)

type UserFriendshipService interface {
	// ฟีเจอร์หลักของระบบเพื่อน
	SendFriendRequest(userID, friendID uuid.UUID) (*models.UserFriendship, error)
	AcceptFriendRequest(requestID, userID uuid.UUID) (*models.UserFriendship, error)
	RejectFriendRequest(requestID, userID uuid.UUID) (*models.UserFriendship, error)
	CancelFriendRequest(requestID, userID uuid.UUID) error
	RemoveFriend(userID, friendID uuid.UUID) error
	GetFriends(userID uuid.UUID) ([]*models.User, error)
	GetPendingRequests(userID uuid.UUID) ([]*models.UserFriendship, error)
	GetSentRequests(userID uuid.UUID) ([]*models.UserFriendship, error)

	// ฟีเจอร์การบล็อก
	BlockUser(userID, targetID uuid.UUID) error
	UnblockUser(userID, targetID uuid.UUID) error
	GetBlockedUsers(userID uuid.UUID) ([]*models.User, error)
	GetBlockedByUsers(userID uuid.UUID) ([]*models.User, error) // หาคนที่บล็อกเรา

	// ตรวจสอบความสัมพันธ์
	GetFriendshipStatus(userID, otherUserID uuid.UUID) (string, uuid.UUID, error) // returns status, friendshipID, error
	IsFriend(userID, otherUserID uuid.UUID) (bool, error)
	HasPendingRequest(userID, otherUserID uuid.UUID) (bool, string, error) // returns exists, direction, error

	// ตรวจสอบสถานะการบล็อก
	IsBlocked(userID, targetID uuid.UUID) (bool, error)                                   // ตรวจสอบว่า userID บล็อค targetID หรือไม่
	IsBlockedBy(userID, targetID uuid.UUID) (bool, error)                                 // ตรวจสอบว่า userID ถูก targetID บล็อคหรือไม่
	CheckBlockStatus(user1ID, user2ID uuid.UUID) (isBlocked bool, isBlockedBy bool, err error) // ตรวจสอบ block status แบบ bidirectional
}
