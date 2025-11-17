// domain/service/user_service.go
package service

import (
	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/types"
)

type UserService interface {
	GetUserByID(id uuid.UUID) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error) // เพิ่มเมธอดนี้
	GetCurrentUser(id uuid.UUID) (*models.User, error)
	UpdateProfile(id uuid.UUID, data types.JSONB) (*models.User, error)
	UpdateLastActive(id uuid.UUID) error
	SearchUsers(query string, limit, offset int) ([]*models.User, int, error)
	GetUserStatuses(userIDs []uuid.UUID) ([]types.JSONB, error) // เปลี่ยนจาก types.JSONB เป็น []types.JSONB
	UploadProfileImage(userID uuid.UUID, imageURL string) error
	SearchUsersExact(query string, limit, offset int) ([]*models.User, int64, error)
}
