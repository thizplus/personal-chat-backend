package dto

import (
	"time"

	"github.com/google/uuid"
)

// ============ Request DTOs ============

// FileUploadRequest สำหรับการอัปโหลดไฟล์
type FileUploadRequest struct {
	Folder string `form:"folder"`
	// ไฟล์จะถูกส่งมาในรูปแบบ multipart/form-data ซึ่งไม่สามารถระบุใน struct ได้โดยตรง
}

// ============ Response DTOs ============

// FileUploadDTO ข้อมูลผลลัพธ์การอัปโหลดไฟล์
type FileUploadDTO struct {
	ID          uuid.UUID              `json:"id"`
	FileName    string                 `json:"file_name"`
	FileSize    int64                  `json:"file_size"`
	FileType    string                 `json:"file_type"`
	MimeType    string                 `json:"mime_type"`
	URL         string                 `json:"url"`
	Path        string                 `json:"path"`
	Folder      string                 `json:"folder"`
	StorageType string                 `json:"storage_type"`
	UploadedAt  time.Time              `json:"uploaded_at"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`

	// สำหรับรูปภาพโดยเฉพาะ
	Width        *int    `json:"width,omitempty"`
	Height       *int    `json:"height,omitempty"`
	ThumbnailURL *string `json:"thumbnail_url,omitempty"`
}

// FileUploadResponse สำหรับผลลัพธ์การอัปโหลดไฟล์
type FileUploadResponse struct {
	GenericResponse
	Data FileUploadDTO `json:"data"`
}

// FileError สำหรับข้อผิดพลาดในการอัปโหลดไฟล์
type FileError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// FileErrorResponse สำหรับการตอบกลับกรณีเกิดข้อผิดพลาด
type FileErrorResponse struct {
	GenericResponse
	Error FileError `json:"error"`
}

// FilesListDTO ข้อมูลรายการไฟล์
type FilesListDTO struct {
	Files  []FileUploadDTO `json:"files"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

// FilesListResponse สำหรับผลลัพธ์การดึงรายการไฟล์
type FilesListResponse struct {
	GenericResponse
	Data FilesListDTO `json:"data"`
}

// ============ Presigned Upload DTOs ============

// PresignedUploadRequest สำหรับขอ presigned URL
type PresignedUploadRequest struct {
	Filename    string `json:"filename" validate:"required"`
	ContentType string `json:"content_type" validate:"required"`
	Folder      string `json:"folder"`
	ExpiryMins  int    `json:"expiry_mins"` // จำนวนนาทีที่ URL จะหมดอายุ (default: 15)
}

// PresignedUploadDTO ข้อมูล presigned URL
type PresignedUploadDTO struct {
	URL        string            `json:"url"`         // URL สำหรับ upload
	Method     string            `json:"method"`      // HTTP method (PUT, POST)
	Path       string            `json:"path"`        // Path ของไฟล์ใน storage
	ExpiresAt  string            `json:"expires_at"`  // เวลาหมดอายุ (ISO 8601)
	Fields     map[string]string `json:"fields,omitempty"` // Fields สำหรับ POST (S3/R2)
	Headers    map[string]string `json:"headers,omitempty"` // Headers ที่ต้องส่งไปด้วย
}

// PresignedUploadResponse สำหรับผลลัพธ์การขอ presigned URL
type PresignedUploadResponse struct {
	GenericResponse
	Data PresignedUploadDTO `json:"data"`
}

// ============ File Delete DTOs ============

// DeleteFileRequest สำหรับลบไฟล์
type DeleteFileRequest struct {
	Path string `json:"path" validate:"required"` // Path ของไฟล์ที่ต้องการลบ
}

// DeleteFileResponse สำหรับผลลัพธ์การลบไฟล์
type DeleteFileResponse struct {
	GenericResponse
}

// ============ File Upload Workflow DTOs ============

// PrepareUploadRequest สำหรับเตรียม upload และขอ presigned URL
type PrepareUploadRequest struct {
	Filename    string `json:"filename" validate:"required"`
	ContentType string `json:"content_type" validate:"required"`
	Size        int64  `json:"size" validate:"required,min=1"`
	Folder      string `json:"folder"`
}

// PrepareUploadDTO ข้อมูลที่ return จากการเตรียม upload
type PrepareUploadDTO struct {
	UploadID   uuid.UUID         `json:"upload_id"`   // ID สำหรับ track upload นี้
	UploadURL  string            `json:"upload_url"`  // Presigned URL สำหรับ upload
	Method     string            `json:"method"`      // HTTP method (PUT)
	Path       string            `json:"path"`        // Path ของไฟล์ใน storage
	ExpiresAt  string            `json:"expires_at"`  // เวลาหมดอายุ
	Headers    map[string]string `json:"headers,omitempty"` // Headers ที่ต้องส่ง
}

// PrepareUploadResponse สำหรับผลลัพธ์การเตรียม upload
type PrepareUploadResponse struct {
	GenericResponse
	Data PrepareUploadDTO `json:"data"`
}

// ConfirmUploadRequest สำหรับยืนยันว่า upload สำเร็จแล้ว
type ConfirmUploadRequest struct {
	UploadID uuid.UUID `json:"upload_id" validate:"required"`
}

// ConfirmUploadDTO ข้อมูลไฟล์ที่ upload สำเร็จ
type ConfirmUploadDTO struct {
	ID          uuid.UUID `json:"id"`
	URL         string    `json:"url"`          // Public URL
	Path        string    `json:"path"`         // Storage path
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	Status      string    `json:"status"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

// ConfirmUploadResponse สำหรับผลลัพธ์การยืนยัน upload
type ConfirmUploadResponse struct {
	GenericResponse
	Data ConfirmUploadDTO `json:"data"`
}
