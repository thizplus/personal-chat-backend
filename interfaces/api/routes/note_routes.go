// interfaces/api/routes/note_routes.go
package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/thizplus/gofiber-chat-api/interfaces/api/handler"
	"github.com/thizplus/gofiber-chat-api/interfaces/api/middleware"
)

// SetupNoteRoutes กำหนดเส้นทาง API สำหรับบันทึก
func SetupNoteRoutes(router fiber.Router, noteHandler *handler.NoteHandler) {
	// Protected routes
	notes := router.Group("/notes")
	notes.Use(middleware.Protected())

	// CRUD operations
	notes.Post("/", noteHandler.CreateNote)           // สร้างบันทึกใหม่
	notes.Get("/", noteHandler.GetNotes)              // ดึงรายการบันทึกทั้งหมด

	// Special routes (ต้องมาก่อน /:id เพื่อไม่ให้ conflict)
	notes.Get("/pinned", noteHandler.GetPinnedNotes)  // ดึงรายการบันทึกที่ปักหมุด
	notes.Get("/search", noteHandler.SearchNotes)     // ค้นหาบันทึก
	notes.Get("/by-tag", noteHandler.GetNotesByTag)   // ดึงบันทึกตาม tag

	// Pin operations (ต้องมาก่อน /:id เพราะมี sub-path)
	notes.Put("/:id/pin", noteHandler.PinNote)        // ปักหมุดบันทึก (PUT)
	notes.Post("/:id/pin", noteHandler.PinNote)       // ปักหมุดบันทึก (POST - รองรับทั้ง 2 method)
	notes.Delete("/:id/pin", noteHandler.UnpinNote)   // ยกเลิกการปักหมุด

	// Dynamic routes (ต้องมาหลังสุด)
	notes.Get("/:id", noteHandler.GetNote)            // ดึงบันทึกเฉพาะ
	notes.Put("/:id", noteHandler.UpdateNote)         // อัปเดตบันทึก
	notes.Delete("/:id", noteHandler.DeleteNote)      // ลบบันทึก
}
