// interfaces/api/routes/conversation_routes.go
package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/thizplus/gofiber-chat-api/interfaces/api/handler"
	"github.com/thizplus/gofiber-chat-api/interfaces/api/middleware"
)

// SetupConversationRoutes กำหนดเส้นทางสำหรับการสนทนาส่วนตัว
func SetupConversationRoutes(
	router fiber.Router,
	conversationHandler *handler.ConversationHandler,
	conversationMemberHandler *handler.ConversationMemberHandler,
) {
	// สร้างกลุ่มเส้นทางการสนทนา
	conversations := router.Group("/conversations")
	conversations.Use(middleware.Protected())

	// เส้นทางหลัก - เฉพาะการสนทนาส่วนตัว
	conversations.Post("/", conversationHandler.Create)              // [success] 8.1 การสร้างการสนทนา [direct,group,business]
	conversations.Get("/", conversationHandler.GetUserConversations) // [success] 8.2 การดึงรายการการสนทนา [Y]
	conversations.Get("/unread-counts", conversationHandler.GetUnreadCounts) // ดึงจำนวนข้อความที่ยังไม่ได้อ่านในทุกการสนทนา

	// เส้นทางเฉพาะการสนทนา
	conversations.Post("/:conversationId/read", conversationHandler.MarkConversationAsRead) // ทำเครื่องหมายว่าอ่านแล้ว
	conversations.Patch("/:conversationId", conversationHandler.UpdateConversation)             // [success] 8.3 การอัปเดตข้อมูลการสนทนา [Y]
	conversations.Get("/:conversationId/messages", conversationHandler.GetConversationMessages) // [succcess] 8.4 การดึงข้อความในการสนทนา [Y]
	conversations.Patch("/:conversationId/pin", conversationHandler.TogglePinConversation)      // [success] 8.5 การเปลี่ยนสถานะปักหมุดของการสนทนา [Y]
	conversations.Patch("/:conversationId/mute", conversationHandler.ToggleMuteConversation)    // [success] 8.6 การเปลี่ยนสถานะการปิดเสียงของการสนทนา [Y]
	conversations.Patch("/:conversationId/hide", conversationHandler.HideConversation)          // การซ่อน/แสดงการสนทนา
	conversations.Delete("/:conversationId", conversationHandler.DeleteConversation)            // การลบการสนทนา (smart delete)

	// Media Gallery & Jump to Message
	conversations.Get("/:conversationId/media/summary", conversationHandler.GetMediaSummary)      // ดึงสรุปจำนวน media และ link
	conversations.Get("/:conversationId/media", conversationHandler.GetMediaByType)               // ดึงรายละเอียด media ตามประเภท
	conversations.Get("/:conversationId/messages/context", conversationHandler.GetMessageContext) // Jump to Message

	// การจัดการสมาชิกในกลุ่ม
	conversations.Post("/:conversationId/members", conversationMemberHandler.AddConversationMember)              // [success] 9.1 การเพิ่มสมาชิกในการสนทนา [Y]
	conversations.Post("/:conversationId/members/bulk", conversationMemberHandler.BulkAddConversationMembers)    // การเพิ่มสมาชิกหลายคนพร้อมกัน
	conversations.Get("/:conversationId/members", conversationMemberHandler.GetConversationMembers)              // [success] 9.2 การดึงรายชื่อสมาชิกในการสนทนา [Y]
	conversations.Delete("/:conversationId/members/:userId", conversationMemberHandler.RemoveConversationMember) // [success] 9.4 การลบสมาชิกจากการสนทนา [Y]
	conversations.Patch("/:conversationId/members/:userId/admin", conversationMemberHandler.ToggleMemberAdmin)   // [success] 9.3 การเปลี่ยนสถานะแอดมินของสมาชิก [Y]

	// การจัดการ role และ ownership
	conversations.Patch("/:conversationId/members/:userId/role", conversationMemberHandler.ChangeRole)           // เปลี่ยน role ของสมาชิก (owner/admin/member)
	conversations.Post("/:conversationId/transfer-ownership", conversationHandler.TransferOwnership)             // โอนความเป็นเจ้าของกลุ่มให้สมาชิกคนอื่น

	// Group Activity Log
	conversations.Get("/:conversationId/activities", conversationHandler.GetActivities) // ดึง activity log ของกลุ่ม

	// Jump to Date
	conversations.Get("/:conversationId/messages/by-date", conversationHandler.GetMessagesByDate) // ดึงข้อความตามวันที่
}
