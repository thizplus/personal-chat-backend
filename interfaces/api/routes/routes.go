// interfaces/api/routes/routes.go - คงไว้แบบเดิม ไม่ต้องแก้ไข
package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/thizplus/gofiber-chat-api/interfaces/api/handler"
)

// SetupRoutes กำหนดเส้นทาง API ทั้งหมดของแอปพลิเคชัน
func SetupRoutes(
	app *fiber.App,
	authHandler *handler.AuthHandler,
	fileHandler *handler.FileHandler,
	userFriendshipHandler *handler.UserFriendshipHandler,

	userHandler *handler.UserHandler,
	conversationHandler *handler.ConversationHandler,
	conversationMemberHandler *handler.ConversationMemberHandler,

	messageHandler *handler.MessageHandler,
	messageReadHandler *handler.MessageReadHandler,
	mentionHandler *handler.MentionHandler,
	scheduledMessageHandler *handler.ScheduledMessageHandler,

	stickerHandler *handler.StickerHandler,
	noteHandler *handler.NoteHandler,

	searchHandler *handler.SearchHandler,
	presenceHandler *handler.PresenceHandler,
	pinnedMessageHandler *handler.PinnedMessageHandler,

) {
	// สร้าง API group
	api := app.Group("/api/v1")

	// Health check route
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "API is running",
		})
	})

	// กำหนดเส้นทางต่างๆ
	SetupAuthRoutes(api, authHandler)
	SetupFileRoutes(api, fileHandler)


	SetupUserRoutes(api, userHandler)
	SetupUserFriendshipRoutes(api, userFriendshipHandler)
	SetupConversationRoutes(api, conversationHandler, conversationMemberHandler)

	SetupMessageRoutes(api, messageHandler)
	SetupMessageReadRoutes(api, messageReadHandler)
	SetupMentionRoutes(api, mentionHandler)
	SetupScheduledMessageRoutes(api, scheduledMessageHandler)

	SetupStickerRoutes(api, stickerHandler)
	SetupNoteRoutes(api, noteHandler)



	SetupSearchRoutes(api, searchHandler)
	SetupPresenceRoutes(api, presenceHandler)
	SetupPinnedMessageRoutes(api, pinnedMessageHandler)

}
