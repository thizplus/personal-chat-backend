// infrastructure/persistence/postgres/user_friendship_repository.go
package postgres

import (
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/repository"
	"gorm.io/gorm"
)

type userFriendshipRepository struct {
	db *gorm.DB
}

func NewUserFriendshipRepository(db *gorm.DB) repository.UserFriendshipRepository {
	return &userFriendshipRepository{db: db}
}

func (r *userFriendshipRepository) Create(userFriendship *models.UserFriendship) error {
	if userFriendship.ID == uuid.Nil {
		userFriendship.ID = uuid.New()
	}
	if userFriendship.RequestedAt.IsZero() {
		userFriendship.RequestedAt = time.Now()
	}
	if userFriendship.UpdatedAt.IsZero() {
		userFriendship.UpdatedAt = time.Now()
	}

	return r.db.Create(userFriendship).Error
}

func (r *userFriendshipRepository) FindByID(id uuid.UUID) (*models.UserFriendship, error) {
	var userFriendship models.UserFriendship
	if err := r.db.Where("id = ?", id).First(&userFriendship).Error; err != nil {
		return nil, err
	}
	return &userFriendship, nil
}

func (r *userFriendshipRepository) Update(userFriendship *models.UserFriendship) error {
	userFriendship.UpdatedAt = time.Now()
	return r.db.Save(userFriendship).Error
}

func (r *userFriendshipRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.UserFriendship{}, "id = ?", id).Error
}

func (r *userFriendshipRepository) FindPendingRequestsByUserID(userID uuid.UUID) ([]*models.UserFriendship, error) {
	var userFriendships []*models.UserFriendship
	if err := r.db.Where("user_id = ? AND status = ?", userID, "pending").Find(&userFriendships).Error; err != nil {
		return nil, err
	}
	return userFriendships, nil
}

func (r *userFriendshipRepository) FindPendingRequestsByFriendID(friendID uuid.UUID) ([]*models.UserFriendship, error) {
	var userFriendships []*models.UserFriendship
	if err := r.db.Where("friend_id = ? AND status = ?", friendID, "pending").Find(&userFriendships).Error; err != nil {
		return nil, err
	}
	return userFriendships, nil
}

func (r *userFriendshipRepository) FindAcceptedFriendships(userID uuid.UUID) ([]*models.UserFriendship, error) {
	var userFriendships []*models.UserFriendship
	if err := r.db.Where("(user_id = ? OR friend_id = ?) AND status = ?", userID, userID, "accepted").Find(&userFriendships).Error; err != nil {
		return nil, err
	}
	return userFriendships, nil
}

func (r *userFriendshipRepository) FindByUserIDAndFriendID(userID, friendID uuid.UUID) (*models.UserFriendship, error) {
	var userFriendship models.UserFriendship
	if err := r.db.Where("user_id = ? AND friend_id = ?", userID, friendID).First(&userFriendship).Error; err != nil {
		return nil, err
	}
	return &userFriendship, nil
}

func (r *userFriendshipRepository) FindByUserIDOrFriendID(userID, friendID uuid.UUID) ([]*models.UserFriendship, error) {
	var userFriendships []*models.UserFriendship
	if err := r.db.Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
		userID, friendID, friendID, userID).Find(&userFriendships).Error; err != nil {
		return nil, err
	}
	return userFriendships, nil
}

func (r *userFriendshipRepository) UpdateStatus(id uuid.UUID, status string) error {
	return r.db.Model(&models.UserFriendship{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

func (r *userFriendshipRepository) DeleteByUserIDAndFriendID(userID, friendID uuid.UUID) error {
	return r.db.Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
		userID, friendID, friendID, userID).Delete(&models.UserFriendship{}).Error
}

func (r *userFriendshipRepository) FindBlockedUsers(userID uuid.UUID) ([]*models.UserFriendship, error) {
	var userFriendships []*models.UserFriendship
	if err := r.db.Where("user_id = ? AND status = ?", userID, "blocked").Find(&userFriendships).Error; err != nil {
		return nil, err
	}
	return userFriendships, nil
}

// FindBlockedByUsers หาคนที่บล็อกเรา (เราคือ friend_id ในบันทึก)
func (r *userFriendshipRepository) FindBlockedByUsers(userID uuid.UUID) ([]*models.UserFriendship, error) {
	var userFriendships []*models.UserFriendship
	if err := r.db.Where("friend_id = ? AND status = ?", userID, "blocked").Find(&userFriendships).Error; err != nil {
		return nil, err
	}
	return userFriendships, nil
}
