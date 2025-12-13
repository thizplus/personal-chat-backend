// interfaces/api/routes/pinned_message_routes.go
package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/thizplus/gofiber-chat-api/interfaces/api/handler"
	"github.com/thizplus/gofiber-chat-api/interfaces/api/middleware"
)

// SetupPinnedMessageRoutes sets up routes for pinned messages
func SetupPinnedMessageRoutes(router fiber.Router, pinnedHandler *handler.PinnedMessageHandler) {
	// Protected routes under /conversations
	conversations := router.Group("/conversations")
	conversations.Use(middleware.Protected())

	// Pin/Unpin message
	conversations.Post("/:conversation_id/messages/:message_id/pin", pinnedHandler.PinMessage)
	conversations.Delete("/:conversation_id/messages/:message_id/pin", pinnedHandler.UnpinMessage)

	// Get pinned messages
	conversations.Get("/:conversation_id/pinned-messages", pinnedHandler.GetPinnedMessages)
}
