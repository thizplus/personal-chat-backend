// interfaces/api/handler/scheduled_message_handler.go
package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/thizplus/gofiber-chat-api/domain/service"
	"github.com/thizplus/gofiber-chat-api/interfaces/api/middleware"
	"github.com/thizplus/gofiber-chat-api/pkg/utils"
)

type ScheduledMessageHandler struct {
	scheduledMessageService service.ScheduledMessageService
}

func NewScheduledMessageHandler(scheduledMessageService service.ScheduledMessageService) *ScheduledMessageHandler {
	return &ScheduledMessageHandler{
		scheduledMessageService: scheduledMessageService,
	}
}

// ScheduleMessage กำหนดเวลาส่งข้อความ
func (h *ScheduledMessageHandler) ScheduleMessage(c *fiber.Ctx) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	conversationID, err := utils.ParseUUIDParam(c, "conversationId")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid conversation ID: " + err.Error(),
		})
	}

	// Parse request body
	var input struct {
		MessageType string                 `json:"message_type"`
		Content     string                 `json:"content"`
		MediaURL    string                 `json:"media_url"`
		Metadata    map[string]interface{} `json:"metadata"`
		ScheduledAt string                 `json:"scheduled_at"` // RFC3339 format
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body: " + err.Error(),
		})
	}

	// Validate message type
	if input.MessageType == "" {
		input.MessageType = "text"
	}

	// Parse scheduled_at
	scheduledAt, err := time.Parse(time.RFC3339, input.ScheduledAt)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid scheduled_at format (use RFC3339): " + err.Error(),
		})
	}

	// Schedule the message
	scheduledMsg, err := h.scheduledMessageService.ScheduleMessage(
		conversationID,
		userID,
		input.MessageType,
		input.Content,
		input.MediaURL,
		input.Metadata,
		scheduledAt,
	)

	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if err.Error() == "user is not a member of this conversation" {
			statusCode = fiber.StatusForbidden
		} else if err.Error() == "scheduled_at must be in the future" {
			statusCode = fiber.StatusBadRequest
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Message scheduled successfully",
		"data":    scheduledMsg,
	})
}

// GetScheduledMessage ดึงข้อมูลข้อความที่กำหนดเวลาส่ง
func (h *ScheduledMessageHandler) GetScheduledMessage(c *fiber.Ctx) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	scheduledMsgID, err := utils.ParseUUIDParam(c, "id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid scheduled message ID: " + err.Error(),
		})
	}

	scheduledMsg, err := h.scheduledMessageService.GetScheduledMessage(scheduledMsgID, userID)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if err.Error() == "scheduled message not found" {
			statusCode = fiber.StatusNotFound
		} else if err.Error() == "unauthorized to access this scheduled message" {
			statusCode = fiber.StatusForbidden
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    scheduledMsg,
	})
}

// GetUserScheduledMessages ดึงรายการข้อความที่กำหนดเวลาส่งของผู้ใช้
func (h *ScheduledMessageHandler) GetUserScheduledMessages(c *fiber.Ctx) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	limit := c.QueryInt("limit", 20)
	if limit > 100 {
		limit = 100
	}
	offset := c.QueryInt("offset", 0)

	scheduledMsgs, total, err := h.scheduledMessageService.GetUserScheduledMessages(userID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"scheduled_messages": scheduledMsgs,
			"pagination": fiber.Map{
				"total":  total,
				"limit":  limit,
				"offset": offset,
			},
		},
	})
}

// GetConversationScheduledMessages ดึงรายการข้อความที่กำหนดเวลาส่งในการสนทนา
func (h *ScheduledMessageHandler) GetConversationScheduledMessages(c *fiber.Ctx) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	conversationID, err := utils.ParseUUIDParam(c, "conversationId")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid conversation ID: " + err.Error(),
		})
	}

	limit := c.QueryInt("limit", 20)
	if limit > 100 {
		limit = 100
	}
	offset := c.QueryInt("offset", 0)

	scheduledMsgs, total, err := h.scheduledMessageService.GetConversationScheduledMessages(conversationID, userID, limit, offset)
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

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"scheduled_messages": scheduledMsgs,
			"pagination": fiber.Map{
				"total":  total,
				"limit":  limit,
				"offset": offset,
			},
		},
	})
}

// CancelScheduledMessage ยกเลิกข้อความที่กำหนดเวลาส่ง
func (h *ScheduledMessageHandler) CancelScheduledMessage(c *fiber.Ctx) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	scheduledMsgID, err := utils.ParseUUIDParam(c, "id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid scheduled message ID: " + err.Error(),
		})
	}

	if err := h.scheduledMessageService.CancelScheduledMessage(scheduledMsgID, userID); err != nil {
		statusCode := fiber.StatusInternalServerError
		if err.Error() == "scheduled message not found" {
			statusCode = fiber.StatusNotFound
		} else if err.Error() == "unauthorized to cancel this scheduled message" {
			statusCode = fiber.StatusForbidden
		} else if err.Error() == "can only cancel pending scheduled messages" {
			statusCode = fiber.StatusBadRequest
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Scheduled message cancelled successfully",
	})
}

// UpdateScheduledTime อัปเดตเวลาที่กำหนดส่งข้อความ
func (h *ScheduledMessageHandler) UpdateScheduledTime(c *fiber.Ctx) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	scheduledMsgID, err := utils.ParseUUIDParam(c, "id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid scheduled message ID: " + err.Error(),
		})
	}

	// Parse request body
	var input struct {
		ScheduledAt string `json:"scheduled_at"` // RFC3339 format
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body: " + err.Error(),
		})
	}

	// Parse scheduled_at
	newScheduledAt, err := time.Parse(time.RFC3339, input.ScheduledAt)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid scheduled_at format (use RFC3339): " + err.Error(),
		})
	}

	// Update scheduled time
	scheduledMsg, err := h.scheduledMessageService.UpdateScheduledTime(scheduledMsgID, userID, newScheduledAt)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		switch err.Error() {
		case "scheduled message not found":
			statusCode = fiber.StatusNotFound
		case "unauthorized to update this scheduled message":
			statusCode = fiber.StatusForbidden
		case "can only update pending scheduled messages":
			statusCode = fiber.StatusBadRequest
		case "scheduled_at must be in the future":
			statusCode = fiber.StatusBadRequest
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Scheduled time updated successfully",
		"data":    scheduledMsg,
	})
}
