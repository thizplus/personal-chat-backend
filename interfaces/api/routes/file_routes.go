// interfaces/api/routes/file_routes.go
package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/thizplus/gofiber-chat-api/interfaces/api/handler"
	"github.com/thizplus/gofiber-chat-api/interfaces/api/middleware"
)

// SetupFileRoutes ตั้งค่าเส้นทางสำหรับการจัดการไฟล์
func SetupFileRoutes(router fiber.Router, fileHandler *handler.FileHandler) {
	// กำหนดกลุ่มเส้นทางไฟล์ (ต้องมีการยืนยันตัวตน)
	files := router.Group("/files")
	files.Use(middleware.Protected())

	// Upload routes (multipart form)
	files.Post("/image", fileHandler.UploadImage) // [success] 3.1 การอัปโหลดรูปภาพ [Y]
	files.Post("/file", fileHandler.UploadFile)   // [success] 3.2 การอัปโหลดไฟล์ [Y]

	// Presigned URL route (JSON body) - Legacy
	files.Post("/presigned-upload", fileHandler.GeneratePresignedUploadURL) // สร้าง presigned URL สำหรับ direct upload

	// New Upload Workflow (Recommended)
	files.Post("/prepare-upload", fileHandler.PrepareUpload) // เตรียม upload และสร้าง presigned URL พร้อม tracking
	files.Post("/confirm-upload", fileHandler.ConfirmUpload) // ยืนยันว่า upload สำเร็จ

	// Delete route (JSON body)
	files.Delete("/", fileHandler.DeleteFile) // ลบไฟล์

	// Legacy routes for backward compatibility
	upload := router.Group("/upload")
	upload.Use(middleware.Protected())
	upload.Post("/image", fileHandler.UploadImage)
	upload.Post("/file", fileHandler.UploadFile)
}
