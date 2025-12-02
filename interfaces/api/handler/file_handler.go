// interfaces/api/handler/file_handler.go
package handler

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/repository"
	"github.com/thizplus/gofiber-chat-api/domain/service"
	"github.com/thizplus/gofiber-chat-api/interfaces/api/middleware"
)

const (
	// MaxFileSize is 100MB (for most files)
	MaxFileSize = 100 * 1024 * 1024
	// DefaultPresignedExpiry is 15 minutes
	DefaultPresignedExpiry = 15 * time.Minute
	// MaxUploadsPerHour per user (rate limiting)
	MaxUploadsPerHour = 100
)

// FileHandler จัดการ API endpoints เกี่ยวกับการอัปโหลดไฟล์
type FileHandler struct {
	storageService   service.FileStorageService
	fileUploadRepo   repository.FileUploadRepository
}

// NewFileHandler สร้าง FileHandler ใหม่
func NewFileHandler(storageService service.FileStorageService, fileUploadRepo repository.FileUploadRepository) *FileHandler {
	return &FileHandler{
		storageService:   storageService,
		fileUploadRepo:   fileUploadRepo,
	}
}

// UploadImage จัดการการอัปโหลดรูปภาพ
func (h *FileHandler) UploadImage(c *fiber.Ctx) error {
	// รับไฟล์จาก request
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "ไม่พบไฟล์รูปภาพในคำขอ",
		})
	}

	// กำหนด folder สำหรับเก็บรูปภาพ (ถ้ามีการส่งมา)
	folder := c.FormValue("folder", "images")

	// อัปโหลดรูปภาพโดยใช้ storage service
	result, err := h.storageService.UploadImage(file, folder)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "ไม่สามารถอัปโหลดรูปภาพได้: " + err.Error(),
		})
	}

	// ส่งผลลัพธ์กลับไป
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "อัปโหลดรูปภาพสำเร็จ",
		"data":    result,
	})
}

// UploadFile จัดการการอัปโหลดไฟล์ทั่วไป
func (h *FileHandler) UploadFile(c *fiber.Ctx) error {
	// รับไฟล์จาก request
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "ไม่พบไฟล์ในคำขอ",
		})
	}

	// กำหนด folder สำหรับเก็บไฟล์ (ถ้ามีการส่งมา)
	folder := c.FormValue("folder", "files")

	// อัปโหลดไฟล์โดยใช้ storage service
	result, err := h.storageService.UploadFile(file, folder)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "ไม่สามารถอัปโหลดไฟล์ได้: " + err.Error(),
		})
	}

	// ส่งผลลัพธ์กลับไป
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "อัปโหลดไฟล์สำเร็จ",
		"data":    result,
	})
}

// GeneratePresignedUploadURL สร้าง presigned URL สำหรับให้ client upload ตรง
func (h *FileHandler) GeneratePresignedUploadURL(c *fiber.Ctx) error {
	// Parse request body
	var req struct {
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
		Folder      string `json:"folder"`
		ExpiryMins  int    `json:"expiry_mins"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	// Validate
	if req.Filename == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Filename is required",
		})
	}
	if req.ContentType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Content type is required",
		})
	}

	// Default values
	if req.Folder == "" {
		req.Folder = "uploads"
	}
	if req.ExpiryMins == 0 {
		req.ExpiryMins = 15 // 15 minutes default
	}

	// สร้าง path สำหรับไฟล์
	ext := filepath.Ext(req.Filename)
	nameWithoutExt := req.Filename[:len(req.Filename)-len(ext)]
	uniqueID := uuid.New().String()[:8]
	filename := nameWithoutExt + "_" + uniqueID + ext

	var path string
	if req.Folder != "" {
		path = filepath.Join(req.Folder, filename)
	} else {
		path = filename
	}
	path = filepath.ToSlash(path)

	// สร้าง presigned URL
	expiry := time.Duration(req.ExpiryMins) * time.Minute
	result, err := h.storageService.GeneratePresignedUploadURL(path, req.ContentType, expiry)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to generate presigned URL: " + err.Error(),
		})
	}

	// ส่งผลลัพธ์กลับไป
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Presigned URL generated successfully",
		"data": fiber.Map{
			"url":        result.URL,
			"method":     result.Method,
			"path":       result.Path,
			"expires_at": result.ExpiresAt.Format(time.RFC3339),
			"fields":     result.Fields,
			"headers": fiber.Map{
				"Content-Type": req.ContentType,
			},
		},
	})
}

// DeleteFile ลบไฟล์
func (h *FileHandler) DeleteFile(c *fiber.Ctx) error {
	// Parse request body
	var req struct {
		Path string `json:"path"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	// Validate
	if req.Path == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Path is required",
		})
	}

	// ลบไฟล์
	err := h.storageService.DeleteFile(req.Path)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to delete file: " + err.Error(),
		})
	}

	// ส่งผลลัพธ์กลับไป
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "File deleted successfully",
	})
}

// PrepareUpload เตรียมการอัปโหลดและสร้าง presigned URL
func (h *FileHandler) PrepareUpload(c *fiber.Ctx) error {
	// Get user ID from JWT
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized",
		})
	}

	// Parse request body
	var req struct {
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
		Size        int64  `json:"size"`
		Folder      string `json:"folder"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	// Validate
	if req.Filename == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Filename is required",
		})
	}
	if req.ContentType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Content type is required",
		})
	}
	if req.Size <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "File size must be greater than 0",
		})
	}
	if req.Size > MaxFileSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": fmt.Sprintf("File size exceeds maximum allowed size of %d MB", MaxFileSize/(1024*1024)),
		})
	}

	// Rate limiting: Check uploads in last hour
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	uploadCount, err := h.fileUploadRepo.CountByUserID(userID, oneHourAgo)
	if err != nil {
		log.Printf("Error counting uploads: %v", err)
	} else if uploadCount >= MaxUploadsPerHour {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"success": false,
			"message": fmt.Sprintf("Upload limit exceeded. Maximum %d uploads per hour", MaxUploadsPerHour),
		})
	}

	// Default values
	if req.Folder == "" {
		req.Folder = "uploads"
	}

	// สร้าง unique filename
	ext := filepath.Ext(req.Filename)
	nameWithoutExt := req.Filename[:len(req.Filename)-len(ext)]
	uniqueID := uuid.New().String()[:8]
	filename := nameWithoutExt + "_" + uniqueID + ext

	// สร้าง path
	path := filepath.ToSlash(filepath.Join(req.Folder, filename))

	// สร้าง presigned URL
	result, err := h.storageService.GeneratePresignedUploadURL(path, req.ContentType, DefaultPresignedExpiry)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to generate presigned URL: " + err.Error(),
		})
	}

	// สร้าง FileUpload record
	upload := &models.FileUpload{
		ID:          uuid.New(),
		UserID:      userID,
		Filename:    req.Filename,
		ContentType: req.ContentType,
		Size:        req.Size,
		Status:      models.FileUploadStatusPending,
		Path:        path,
		ExpiresAt:   result.ExpiresAt,
	}

	if err := h.fileUploadRepo.Create(upload); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to create upload record: " + err.Error(),
		})
	}

	// ส่งผลลัพธ์กลับไป
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Upload prepared successfully",
		"data": fiber.Map{
			"upload_id":  upload.ID,
			"upload_url": result.URL,
			"method":     result.Method,
			"path":       path,
			"expires_at": result.ExpiresAt.Format(time.RFC3339),
			"headers": fiber.Map{
				"Content-Type": req.ContentType,
			},
		},
	})
}

// ConfirmUpload ยืนยันว่าการอัปโหลดสำเร็จแล้ว
func (h *FileHandler) ConfirmUpload(c *fiber.Ctx) error {
	// Get user ID from JWT
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized",
		})
	}

	// Parse request body
	var req struct {
		UploadID string `json:"upload_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	// Parse upload ID
	uploadID, err := uuid.Parse(req.UploadID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid upload ID",
		})
	}

	// Find upload record
	upload, err := h.fileUploadRepo.FindByID(uploadID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Upload not found",
		})
	}

	// Verify ownership
	if upload.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": "You don't have permission to confirm this upload",
		})
	}

	// Check if already completed
	if upload.Status == models.FileUploadStatusCompleted {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Upload already confirmed",
		})
	}

	// Check if expired
	if time.Now().After(upload.ExpiresAt) {
		// Mark as failed and cleanup
		h.fileUploadRepo.UpdateStatus(uploadID, models.FileUploadStatusFailed)
		h.storageService.DeleteFile(upload.Path)

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Upload expired",
		})
	}

	// Get public URL
	publicURL := h.storageService.GetPublicURL(upload.Path)

	// Mark as completed
	if err := h.fileUploadRepo.MarkAsCompleted(uploadID, publicURL); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to confirm upload: " + err.Error(),
		})
	}

	// Reload to get updated data
	upload, _ = h.fileUploadRepo.FindByID(uploadID)

	// ส่งผลลัพธ์กลับไป
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Upload confirmed successfully",
		"data": fiber.Map{
			"id":           upload.ID,
			"url":          upload.URL,
			"path":         upload.Path,
			"filename":     upload.Filename,
			"content_type": upload.ContentType,
			"size":         upload.Size,
			"status":       upload.Status,
			"uploaded_at":  upload.CompletedAt,
		},
	})
}
