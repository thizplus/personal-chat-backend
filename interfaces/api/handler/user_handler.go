// interfaces/api/handler/user_handlers.go
package handler

import (
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid" // เพิ่ม import uuid
	"github.com/thizplus/gofiber-chat-api/domain/service"
	"github.com/thizplus/gofiber-chat-api/domain/types"
	"github.com/thizplus/gofiber-chat-api/interfaces/api/middleware"
	"github.com/thizplus/gofiber-chat-api/pkg/utils"
)

type UserHandler struct {
	userService    service.UserService
	authService    service.AuthService
	storageService service.FileStorageService
}

func NewUserHandler(
	userService service.UserService,
	authService service.AuthService,
	storageService service.FileStorageService,
) *UserHandler {
	return &UserHandler{
		userService:    userService,
		authService:    authService,
		storageService: storageService,
	}
}

// GetCurrentUser ดึงข้อมูลผู้ใช้ปัจจุบัน
func (h *UserHandler) GetCurrentUser(c *fiber.Ctx) error {
	// ดึง userID จาก middleware เป็น UUID
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// ดึงข้อมูลผู้ใช้
	user, err := h.userService.GetCurrentUser(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Error getting user data: " + err.Error(),
		})
	}

	// อัปเดตเวลาใช้งานล่าสุด
	h.userService.UpdateLastActive(userID)

	// ส่งข้อมูลกลับ
	return c.JSON(fiber.Map{
		"success": true,
		"user":    user,
	})
}

// GetProfile ดึงข้อมูลโปรไฟล์ผู้ใช้ตามไอดี
func (h *UserHandler) GetProfile(c *fiber.Ctx) error {
	// ดึงและแปลง userId จาก URL parameter เป็น UUID
	userID, err := utils.ParseUUIDParam(c, "userId")
	if err != nil {
		return err // error response ถูกจัดการในฟังก์ชันแล้ว
	}

	// ดึงข้อมูลผู้ใช้ที่ร้องขอจาก middleware
	requesterID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// ดึงข้อมูลผู้ใช้
	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "User not found",
		})
	}

	// ถ้าดูโปรไฟล์ตัวเอง ส่งข้อมูลเพิ่มเติมได้
	if userID == requesterID {
		// ส่งข้อมูลเต็ม
		return c.JSON(fiber.Map{
			"success": true,
			"data":    user,
		})
	}

	// ถ้าดูโปรไฟล์คนอื่น ส่งเฉพาะข้อมูลสาธารณะ
	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"id":                user.ID,
			"username":          user.Username,
			"display_name":      user.DisplayName,
			"profile_image_url": user.ProfileImageURL,
			"bio":               user.Bio,
			"last_active_at":    user.LastActiveAt,
			"status":            user.Status,
		},
	})
}

// UpdateProfile อัปเดตข้อมูลโปรไฟล์ผู้ใช้
func (h *UserHandler) UpdateProfile(c *fiber.Ctx) error {
	// ดึงและแปลง userId จาก URL parameter เป็น UUID
	userID, err := utils.ParseUUIDParam(c, "userId")
	if err != nil {
		return err
	}

	// ดึงข้อมูลผู้ใช้ที่ร้องขอจาก middleware
	requesterID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// ตรวจสอบว่าเป็นการอัปเดตของตัวเองหรือไม่
	if userID != requesterID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": "You can only update your own profile",
		})
	}

	// รับข้อมูลที่ต้องการอัปเดต
	var input types.JSONB
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request data",
		})
	}

	// ใช้ service อัปเดตข้อมูล
	user, err := h.userService.UpdateProfile(userID, input)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Error updating profile: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Profile updated successfully",
		"data":    user,
	})
}

// UploadProfileImage อัปโหลดรูปโปรไฟล์
func (h *UserHandler) UploadProfileImage(c *fiber.Ctx) error {
	// ดึงและแปลง userId จาก URL parameter เป็น UUID
	userID, err := utils.ParseUUIDParam(c, "userId")
	if err != nil {
		return err
	}

	// ดึงข้อมูลผู้ใช้ที่ร้องขอจาก middleware
	requesterID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// ตรวจสอบว่าเป็นการอัปเดตของตัวเองหรือไม่
	if userID != requesterID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": "You can only update your own profile image",
		})
	}

	// รับไฟล์ที่อัปโหลด
	file, err := c.FormFile("profile_image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "No image file uploaded",
		})
	}

	// ตรวจสอบขนาดไฟล์ (เช่น ไม่เกิน 100MB)
	maxSize := 100 * 1024 * 1024 // 100MB
	if file.Size > int64(maxSize) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "File too large (max 100MB)",
		})
	}

	// ตรวจสอบประเภทไฟล์
	ext := strings.ToLower(filepath.Ext(file.Filename))
	validExt := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
	}

	if !validExt[ext] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid file type. Only JPG, JPEG, PNG and GIF are allowed",
		})
	}

	// อัปโหลดไฟล์โดยใช้ storageService
	result, err := h.storageService.UploadImage(file, "profile_images")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to upload image to cloud storage",
			"error":   err.Error(),
		})
	}

	// ใช้ URL ที่ได้จากการอัปโหลด
	imageURL := result.URL

	// อัปเดต URL ในฐานข้อมูล
	if err := h.userService.UploadProfileImage(userID, imageURL); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Error updating profile image: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Profile image uploaded successfully",
		"data": fiber.Map{
			"profile_image_url": imageURL,
			"public_id":         result.PublicID,
		},
	})
}

// SearchUsers ค้นหาผู้ใช้
func (h *UserHandler) SearchUsers(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Search query is required",
		})
	}

	// ดึง limit และ offset จาก query params
	limit := utils.ParseIntWithLimit(c.Query("limit"), 20, 1, 50)
	offset := utils.ParseInt(c.Query("offset"), 0)

	// ค้นหาผู้ใช้
	users, total, err := h.userService.SearchUsers(query, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Error searching users: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"users":  users,
			"count":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetStatus ดึงสถานะผู้ใช้
func (h *UserHandler) GetStatus(c *fiber.Ctx) error {
	userIDsStr := c.Query("ids")
	if userIDsStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "User IDs are required",
		})
	}

	// แยก IDs เป็น array
	userIDsStrArray := utils.SplitCommaString(userIDsStr)

	// แปลง string array เป็น UUID array
	var userIDs []uuid.UUID
	for _, idStr := range userIDsStrArray {
		id, err := uuid.Parse(idStr)
		if err != nil {
			// ข้าม ID ที่ไม่ถูกต้อง
			continue
		}
		userIDs = append(userIDs, id)
	}

	if len(userIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "No valid user IDs provided",
		})
	}

	// ดึงสถานะผู้ใช้
	statuses, err := h.userService.GetUserStatuses(userIDs)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Error getting user statuses: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    statuses,
	})
}

// เพิ่มใน UserHandler
func (h *UserHandler) SearchUserByEmail(c *fiber.Ctx) error {
	email := c.Query("email")
	if email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Email is required",
		})
	}

	user, err := h.userService.GetUserByEmail(email)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "User not found with this email",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"user": map[string]interface{}{
			"id":                user.ID,
			"username":          user.Username,
			"email":             user.Email,
			"display_name":      user.DisplayName,
			"profile_image_url": user.ProfileImageURL,
		},
	})
}
