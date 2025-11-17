// application/serviceimpl/user_service.go
package serviceimpl

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/repository"
	serviceInterfaces "github.com/thizplus/gofiber-chat-api/domain/service"
	"github.com/thizplus/gofiber-chat-api/domain/types"
)

type userService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) serviceInterfaces.UserService {
	return &userService{
		userRepo: userRepo,
	}
}

func (s *userService) GetUserByID(id uuid.UUID) (*models.User, error) {
	return s.userRepo.FindByID(id)
}

// เพิ่มเมธอด GetCurrentUser
func (s *userService) GetCurrentUser(id uuid.UUID) (*models.User, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// อัปเดต last_active_at
	now := time.Now()
	user.LastActiveAt = &now
	if err := s.userRepo.Update(user); err != nil {
		// บันทึกข้อผิดพลาด แต่ไม่ส่งผลกระทบต่อการดึงข้อมูลผู้ใช้
		// log.Printf("Failed to update last_active_at: %v", err)
	}

	return user, nil
}

func (s *userService) UpdateLastActive(id uuid.UUID) error {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return err
	}

	now := time.Now()
	user.LastActiveAt = &now
	return s.userRepo.Update(user)
}

// เพิ่มเมธอดที่ยังขาดอยู่...

// UpdateProfile อัปเดตข้อมูลโปรไฟล์ผู้ใช้
func (s *userService) UpdateProfile(id uuid.UUID, data types.JSONB) (*models.User, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// อัปเดตข้อมูลตามที่รับมา
	if displayName, ok := data["display_name"].(string); ok {
		user.DisplayName = displayName
	}

	if bio, ok := data["bio"].(string); ok {
		user.Bio = bio
	}

	// อัปเดต settings ถ้ามี
	if settings, ok := data["settings"].(types.JSONB); ok && user.Settings != nil {
		// ต้องจัดการตามประเภทข้อมูลของ Settings ในโมเดล
		// หากใช้ common.JSONB:
		for key, value := range settings {
			user.Settings[key] = value
		}
	}

	// บันทึกการเปลี่ยนแปลง
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	return user, nil
}

// SearchUsers ค้นหาผู้ใช้
func (s *userService) SearchUsers(query string, limit, offset int) ([]*models.User, int, error) {
	// ต้องมีเมธอด SearchUsers ใน UserRepository
	return s.userRepo.SearchUsers(query, limit, offset)
}

// GetUserStatuses ดึงสถานะของผู้ใช้หลายคน
func (s *userService) GetUserStatuses(userIDs []uuid.UUID) ([]types.JSONB, error) {
	// เปลี่ยนจาก map เป็น array
	statuses := make([]types.JSONB, 0, len(userIDs))

	for _, id := range userIDs {
		user, err := s.userRepo.FindByID(id)
		if err != nil {
			continue // ข้ามกรณีไม่พบผู้ใช้
		}

		// เพิ่ม user_id เข้าไปใน object แทนการใช้เป็น key
		statuses = append(statuses, types.JSONB{
			"user_id":        id.String(),
			"status":         user.Status,
			"last_active_at": user.LastActiveAt,
		})
	}

	return statuses, nil
}

// UploadProfileImage อัปเดตรูปโปรไฟล์
func (s *userService) UploadProfileImage(userID uuid.UUID, imageURL string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return err
	}

	user.ProfileImageURL = imageURL
	return s.userRepo.Update(user)
}

// SearchUsersExact ค้นหาผู้ใช้แบบตรงกับทั้งหมด
func (s *userService) SearchUsersExact(query string, limit, offset int) ([]*models.User, int64, error) {
	// ตรวจสอบว่า query ไม่ว่าง
	if query == "" {
		return nil, 0, errors.New("search query is required")
	}

	// เรียกใช้ repository สำหรับการค้นหาแบบตรงกับทั้งหมด
	return s.userRepo.SearchUsersExact(query, limit, offset)
}

// เพิ่มเมธอดนี้
func (s *userService) GetUserByEmail(email string) (*models.User, error) {
	return s.userRepo.FindByEmail(email)
}
