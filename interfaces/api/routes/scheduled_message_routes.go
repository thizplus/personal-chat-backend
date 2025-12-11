// interfaces/api/routes/scheduled_message_routes.go
package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/thizplus/gofiber-chat-api/interfaces/api/handler"
	"github.com/thizplus/gofiber-chat-api/interfaces/api/middleware"
)

// SetupScheduledMessageRoutes กำหนดเส้นทาง API สำหรับข้อความที่กำหนดเวลาส่ง
func SetupScheduledMessageRoutes(router fiber.Router, scheduledMessageHandler *handler.ScheduledMessageHandler) {
	// Protected routes
	scheduledMessages := router.Group("/scheduled-messages")
	scheduledMessages.Use(middleware.Protected())

	// CRUD operations
	scheduledMessages.Get("/", scheduledMessageHandler.GetUserScheduledMessages)                                           // ดึงรายการข้อความที่กำหนดเวลาส่งของผู้ใช้
	scheduledMessages.Get("/:id", scheduledMessageHandler.GetScheduledMessage)                                              // ดึงข้อมูลข้อความที่กำหนดเวลาส่ง
	scheduledMessages.Put("/:id", scheduledMessageHandler.UpdateScheduledTime)                                              // อัปเดตเวลาที่กำหนดส่ง
	scheduledMessages.Delete("/:id", scheduledMessageHandler.CancelScheduledMessage)                                        // ยกเลิกข้อความที่กำหนดเวลาส่ง

	// Schedule message in conversation
	conversations := router.Group("/conversations")
	conversations.Use(middleware.Protected())

	conversations.Post("/:conversationId/messages/schedule", scheduledMessageHandler.ScheduleMessage)                      // กำหนดเวลาส่งข้อความ
	conversations.Get("/:conversationId/scheduled-messages", scheduledMessageHandler.GetConversationScheduledMessages)     // ดึงรายการข้อความที่กำหนดเวลาส่งในการสนทนา
}
