// interfaces/api/handler/pinned_message_handler.go
package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/dto"
	"github.com/thizplus/gofiber-chat-api/domain/service"
	"github.com/thizplus/gofiber-chat-api/interfaces/api/middleware"
)

// PinnedMessageHandler handles pinned message HTTP requests
type PinnedMessageHandler struct {
	pinnedService service.PinnedMessageService
}

// NewPinnedMessageHandler creates a new pinned message handler
func NewPinnedMessageHandler(pinnedService service.PinnedMessageService) *PinnedMessageHandler {
	return &PinnedMessageHandler{pinnedService: pinnedService}
}

// PinMessage pins a message
// POST /api/v1/conversations/:conversation_id/messages/:message_id/pin
func (h *PinnedMessageHandler) PinMessage(c *fiber.Ctx) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	conversationID, err := uuid.Parse(c.Params("conversation_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid conversation ID",
		})
	}

	messageID, err := uuid.Parse(c.Params("message_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid message ID",
		})
	}

	// Parse request body
	var req dto.PinMessageRequest
	if err := c.BodyParser(&req); err != nil {
		// Default to personal if no body
		req.PinType = "personal"
	}

	// Validate pin_type
	if req.PinType == "" {
		req.PinType = "personal"
	}
	if req.PinType != "personal" && req.PinType != "public" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid pin_type, must be 'personal' or 'public'",
		})
	}

	// Pin message
	ctx := c.Context()
	pinnedDTO, err := h.pinnedService.PinMessage(ctx, conversationID, messageID, userID, req.PinType)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		switch err.Error() {
		case "message not found":
			statusCode = fiber.StatusNotFound
		case "user is not a member of this conversation",
			"only owner/admin can create public pins in group conversations":
			statusCode = fiber.StatusForbidden
		case "message is already pinned with this type",
			"maximum public pins limit reached (5)",
			"cannot pin deleted message":
			statusCode = fiber.StatusBadRequest
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Message pinned successfully",
		"data":    pinnedDTO,
	})
}

// UnpinMessage unpins a message
// DELETE /api/v1/conversations/:conversation_id/messages/:message_id/pin
func (h *PinnedMessageHandler) UnpinMessage(c *fiber.Ctx) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	conversationID, err := uuid.Parse(c.Params("conversation_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid conversation ID",
		})
	}

	messageID, err := uuid.Parse(c.Params("message_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid message ID",
		})
	}

	// Get pin_type from query string
	pinType := c.Query("pin_type", "personal")
	if pinType != "personal" && pinType != "public" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid pin_type, must be 'personal' or 'public'",
		})
	}

	// Unpin message
	ctx := c.Context()
	if err := h.pinnedService.UnpinMessage(ctx, conversationID, messageID, userID, pinType); err != nil {
		statusCode := fiber.StatusInternalServerError
		switch err.Error() {
		case "user is not a member of this conversation",
			"only owner/admin can remove public pins in group conversations":
			statusCode = fiber.StatusForbidden
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Message unpinned successfully",
	})
}

// GetPinnedMessages gets pinned messages for a conversation
// GET /api/v1/conversations/:conversation_id/pinned-messages
func (h *PinnedMessageHandler) GetPinnedMessages(c *fiber.Ctx) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	conversationID, err := uuid.Parse(c.Params("conversation_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid conversation ID",
		})
	}

	// Parse query params
	pinType := c.Query("pin_type", "all") // all, personal, public
	limit := c.QueryInt("limit", 50)
	if limit > 100 {
		limit = 100
	}
	offset := c.QueryInt("offset", 0)

	// Get pinned messages
	ctx := c.Context()
	result, err := h.pinnedService.GetPinnedMessages(ctx, conversationID, userID, pinType, limit, offset)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if err.Error() == "user is not a member of this conversation" {
			statusCode = fiber.StatusForbidden
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Pinned messages retrieved successfully",
		"data":    result,
	})
}
