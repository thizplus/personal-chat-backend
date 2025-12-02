// domain/repository/user_friendship_repository.go
package repository

import (
	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
)

type UserFriendshipRepository interface {
	// พื้นฐาน CRUD
	Create(userFriendship *models.UserFriendship) error
	FindByID(id uuid.UUID) (*models.UserFriendship, error)
	Update(userFriendship *models.UserFriendship) error
	Delete(id uuid.UUID) error

	// สำหรับฟีเจอร์ของระบบเพื่อน
	FindPendingRequestsByUserID(userID uuid.UUID) ([]*models.UserFriendship, error)
	FindPendingRequestsByFriendID(friendID uuid.UUID) ([]*models.UserFriendship, error)
	FindAcceptedFriendships(userID uuid.UUID) ([]*models.UserFriendship, error)
	FindByUserIDAndFriendID(userID, friendID uuid.UUID) (*models.UserFriendship, error)
	FindByUserIDOrFriendID(userID, friendID uuid.UUID) ([]*models.UserFriendship, error)
	UpdateStatus(id uuid.UUID, status string) error
	DeleteByUserIDAndFriendID(userID, friendID uuid.UUID) error
	FindBlockedUsers(userID uuid.UUID) ([]*models.UserFriendship, error)
	FindBlockedByUsers(userID uuid.UUID) ([]*models.UserFriendship, error) // หาคนที่บล็อกเรา
}
