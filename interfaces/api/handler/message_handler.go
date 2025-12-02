// interfaces/api/handler/message_handler.go
package handler

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/repository"
	"github.com/thizplus/gofiber-chat-api/domain/service"
	"github.com/thizplus/gofiber-chat-api/domain/types"
	"github.com/thizplus/gofiber-chat-api/interfaces/api/middleware"
	"github.com/thizplus/gofiber-chat-api/pkg/utils"
)

// MessageHandler โครงสร้างของ Handler สำหรับจัดการข้อความ
type MessageHandler struct {
	messageService            service.MessageService
	notificationService       service.NotificationService
	conversationMemberService service.ConversationMemberService
	conversationRepo          repository.ConversationRepository
	userFriendshipService     service.UserFriendshipService
}

// NewMessageHandler สร้าง Handler ใหม่
func NewMessageHandler(
	messageService service.MessageService,
	notificationService service.NotificationService,
	conversationMemberService service.ConversationMemberService,
	conversationRepo repository.ConversationRepository,
	userFriendshipService service.UserFriendshipService,
) *MessageHandler {
	return &MessageHandler{
		messageService:            messageService,
		notificationService:       notificationService,
		conversationMemberService: conversationMemberService,
		conversationRepo:          conversationRepo,
		userFriendshipService:     userFriendshipService,
	}
}

// BlockError custom error type สำหรับ block errors with error code
type BlockError struct {
	Code      string
	Message   string
	BlockerID *uuid.UUID
	BlockedID *uuid.UUID
}

func (e *BlockError) Error() string {
	return e.Message
}

// checkBlockStatusBeforeSend ตรวจสอบว่าผู้ใช้ถูก block หรือไม่ก่อนส่งข้อความ
func (h *MessageHandler) checkBlockStatusBeforeSend(userID, conversationID uuid.UUID) error {
	// ดึงข้อมูล conversation เพื่อตรวจสอบ type
	conversation, err := h.conversationRepo.GetByID(conversationID)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	// ✅ ถ้าเป็น group chat → อนุญาตให้ส่งได้เสมอ (ไม่ตรวจสอบ block)
	if conversation.Type == "group" {
		return nil
	}

	// ✅ ถ้าเป็น direct/private chat → ตรวจสอบ block status
	// ดึงรายชื่อสมาชิกในการสนทนา (สำหรับ direct chat จะมี 2 คน)
	members, _, err := h.conversationMemberService.GetMembers(userID, conversationID, 1, 1000)
	if err != nil {
		return fmt.Errorf("failed to get conversation members: %w", err)
	}

	// ตรวจสอบ block status กับสมาชิกคนอื่น
	for _, member := range members {
		// Parse member UserID from string to UUID
		memberUserID, err := uuid.Parse(member.UserID)
		if err != nil {
			continue // ข้ามถ้า parse ไม่ได้
		}

		if memberUserID == userID {
			continue // ข้ามตัวเอง
		}

		// ตรวจสอบว่าผู้ใช้หรือสมาชิกมีการบล็อกกันหรือไม่
		isBlocked, isBlockedBy, err := h.userFriendshipService.CheckBlockStatus(userID, memberUserID)
		if err != nil {
			return fmt.Errorf("failed to check block status: %w", err)
		}

		// ✅ เราบล็อกคนอื่น
		if isBlocked {
			return &BlockError{
				Code:      "USER_BLOCKED",
				Message:   "คุณได้บล็อกผู้ใช้นี้แล้ว ไม่สามารถส่งข้อความได้",
				BlockerID: &userID,
				BlockedID: &memberUserID,
			}
		}

		// ✅ เราถูกคนอื่นบล็อก
		if isBlockedBy {
			return &BlockError{
				Code:      "BLOCKED_BY_USER",
				Message:   "คุณถูกผู้ใช้นี้บล็อก ไม่สามารถส่งข้อความได้",
				BlockerID: &memberUserID,
				BlockedID: &userID,
			}
		}
	}

	return nil
}

// SendTextMessage จัดการคำขอส่งข้อความประเภทข้อความ
func (h *MessageHandler) SendTextMessage(c *fiber.Ctx) error {
	// ดึง User ID จาก context ที่ตั้งค่าโดย middleware
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	conversationID, err := utils.ParseUUIDParam(c, "conversationId")
	if err != nil {
		return err // error response ถูกจัดการในฟังก์ชันแล้ว
	}

	// ตรวจสอบ block status ก่อนส่งข้อความ
	if err := h.checkBlockStatusBeforeSend(userID, conversationID); err != nil {
		// ✅ ถ้าเป็น BlockError ให้ return พร้อม error_code
		if blockErr, ok := err.(*BlockError); ok {
			response := fiber.Map{
				"success":    false,
				"error_code": blockErr.Code,
				"message":    blockErr.Message,
			}
			// เพิ่ม blocker_id ถ้ามี
			if blockErr.BlockerID != nil {
				response["blocker_id"] = blockErr.BlockerID.String()
			}
			return c.Status(fiber.StatusForbidden).JSON(response)
		}

		// Error อื่นๆ
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	// รับข้อมูลข้อความจาก request body
	var input struct {
		TempID   string      `json:"temp_id"`
		Content  string      `json:"content"`
		Metadata types.JSONB `json:"metadata"`
		Mentions types.JSONB `json:"mentions"` // Format: [{"user_id": "uuid", "start_index": 0, "length": 10}]
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body: " + err.Error(),
		})
	}

	// บันทึก temp_id และ mentions ลงใน metadata ถ้ามี (JSONB เป็น map[string]interface{} อยู่แล้ว)
	metadata := input.Metadata
	if input.TempID != "" {
		if metadata == nil {
			metadata = make(types.JSONB)
		}
		metadata["tempId"] = input.TempID
	}

	// เพิ่ม mentions ลงใน metadata
	if input.Mentions != nil && len(input.Mentions) > 0 {
		if metadata == nil {
			metadata = make(types.JSONB)
		}
		metadata["mentions"] = input.Mentions
	}

	// เรียกใช้ service
	message, err := h.messageService.SendTextMessage(conversationID, userID, input.Content, metadata)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		// ตรวจสอบประเภทข้อผิดพลาดเพื่อกำหนด status code ที่เหมาะสม
		if err.Error() == "user is not a member of this conversation" {
			statusCode = fiber.StatusForbidden
		} else if err.Error() == "message content cannot be empty" {
			statusCode = fiber.StatusBadRequest
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	messageJson, err := json.MarshalIndent(message, "", "  ")
	if err != nil {
		fmt.Printf("[ERROR] Failed to marshal message: %v\n", err)
	} else {
		fmt.Printf("[XXXXXXX]Message sent successfully:\n%s\n", string(messageJson))
	}

	h.notificationService.NotifyNewMessage(conversationID, message)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Message sent successfully",
		"data":    message,
	})
}

// SendStickerMessage จัดการคำขอส่งข้อความประเภทสติกเกอร์
func (h *MessageHandler) SendStickerMessage(c *fiber.Ctx) error {
	// ดึง User ID จาก context
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	conversationID, err := utils.ParseUUIDParam(c, "conversationId")
	if err != nil {
		return err // error response ถูกจัดการในฟังก์ชันแล้ว
	}

	// ตรวจสอบ block status ก่อนส่งข้อความ
	if err := h.checkBlockStatusBeforeSend(userID, conversationID); err != nil {
		// ✅ ถ้าเป็น BlockError ให้ return พร้อม error_code
		if blockErr, ok := err.(*BlockError); ok {
			response := fiber.Map{
				"success":    false,
				"error_code": blockErr.Code,
				"message":    blockErr.Message,
			}
			// เพิ่ม blocker_id ถ้ามี
			if blockErr.BlockerID != nil {
				response["blocker_id"] = blockErr.BlockerID.String()
			}
			return c.Status(fiber.StatusForbidden).JSON(response)
		}

		// Error อื่นๆ
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	// รับข้อมูลสติกเกอร์จาก request body
	var input struct {
		TempID            string      `json:"temp_id"`
		StickerID         uuid.UUID   `json:"sticker_id"`
		StickerSetID      uuid.UUID   `json:"sticker_set_id"`
		MediaURL          string      `json:"media_url"`
		MediaThumbnailURL string      `json:"media_thumbnail_url"`
		Metadata          types.JSONB `json:"metadata"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body: " + err.Error(),
		})
	}

	// บันทึก temp_id ลงใน metadata ถ้ามี (JSONB เป็น map[string]interface{} อยู่แล้ว)
	metadata := input.Metadata
	if input.TempID != "" {
		if metadata == nil {
			metadata = make(types.JSONB)
		}
		metadata["tempId"] = input.TempID
	}

	// เรียกใช้ service
	message, err := h.messageService.SendStickerMessage(
		conversationID,
		userID,
		input.StickerID,
		input.StickerSetID,
		input.MediaURL,
		input.MediaThumbnailURL,
		metadata,
	)

	if err != nil {
		statusCode := fiber.StatusInternalServerError
		// ตรวจสอบประเภทข้อผิดพลาด
		if err.Error() == "user is not a member of this conversation" {
			statusCode = fiber.StatusForbidden
		} else if err.Error() == "sticker URL is required" {
			statusCode = fiber.StatusBadRequest
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	h.notificationService.NotifyNewMessage(conversationID, message)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Sticker sent successfully",
		"data":    message,
	})
}

// SendImageMessage จัดการคำขอส่งข้อความประเภทรูปภาพ
func (h *MessageHandler) SendImageMessage(c *fiber.Ctx) error {
	// ดึง User ID จาก context
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	conversationID, err := utils.ParseUUIDParam(c, "conversationId")
	if err != nil {
		return err // error response ถูกจัดการในฟังก์ชันแล้ว
	}

	// ตรวจสอบ block status ก่อนส่งข้อความ
	if err := h.checkBlockStatusBeforeSend(userID, conversationID); err != nil {
		// ✅ ถ้าเป็น BlockError ให้ return พร้อม error_code
		if blockErr, ok := err.(*BlockError); ok {
			response := fiber.Map{
				"success":    false,
				"error_code": blockErr.Code,
				"message":    blockErr.Message,
			}
			// เพิ่ม blocker_id ถ้ามี
			if blockErr.BlockerID != nil {
				response["blocker_id"] = blockErr.BlockerID.String()
			}
			return c.Status(fiber.StatusForbidden).JSON(response)
		}

		// Error อื่นๆ
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	// รับข้อมูลรูปภาพจาก request body
	var input struct {
		TempID            string      `json:"temp_id"`
		MediaURL          string      `json:"media_url"`
		MediaThumbnailURL string      `json:"media_thumbnail_url"`
		Caption           string      `json:"caption"`
		Metadata          types.JSONB `json:"metadata"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body: " + err.Error(),
		})
	}

	// บันทึก temp_id ลงใน metadata ถ้ามี (JSONB เป็น map[string]interface{} อยู่แล้ว)
	metadata := input.Metadata
	if input.TempID != "" {
		if metadata == nil {
			metadata = make(types.JSONB)
		}
		metadata["tempId"] = input.TempID
	}

	// เรียกใช้ service
	message, err := h.messageService.SendImageMessage(
		conversationID,
		userID,
		input.MediaURL,
		input.MediaThumbnailURL,
		input.Caption,
		metadata,
	)

	if err != nil {
		statusCode := fiber.StatusInternalServerError
		// ตรวจสอบประเภทข้อผิดพลาด
		if err.Error() == "user is not a member of this conversation" {
			statusCode = fiber.StatusForbidden
		} else if err.Error() == "image URL is required" {
			statusCode = fiber.StatusBadRequest
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	h.notificationService.NotifyNewMessage(conversationID, message)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Image sent successfully",
		"data":    message,
	})
}

// SendFileMessage จัดการคำขอส่งข้อความประเภทไฟล์
func (h *MessageHandler) SendFileMessage(c *fiber.Ctx) error {
	// ดึง User ID จาก context
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	conversationID, err := utils.ParseUUIDParam(c, "conversationId")
	if err != nil {
		return err // error response ถูกจัดการในฟังก์ชันแล้ว
	}

	// ตรวจสอบ block status ก่อนส่งข้อความ
	if err := h.checkBlockStatusBeforeSend(userID, conversationID); err != nil {
		// ✅ ถ้าเป็น BlockError ให้ return พร้อม error_code
		if blockErr, ok := err.(*BlockError); ok {
			response := fiber.Map{
				"success":    false,
				"error_code": blockErr.Code,
				"message":    blockErr.Message,
			}
			// เพิ่ม blocker_id ถ้ามี
			if blockErr.BlockerID != nil {
				response["blocker_id"] = blockErr.BlockerID.String()
			}
			return c.Status(fiber.StatusForbidden).JSON(response)
		}

		// Error อื่นๆ
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	// รับข้อมูลไฟล์จาก request body
	var input struct {
		TempID   string      `json:"temp_id"`
		MediaURL string      `json:"media_url"`
		FileName string      `json:"file_name"`
		FileSize int64       `json:"file_size"`
		FileType string      `json:"file_type"`
		Metadata types.JSONB `json:"metadata"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body: " + err.Error(),
		})
	}

	// บันทึก temp_id ลงใน metadata ถ้ามี (JSONB เป็น map[string]interface{} อยู่แล้ว)
	metadata := input.Metadata
	if input.TempID != "" {
		if metadata == nil {
			metadata = make(types.JSONB)
		}
		metadata["tempId"] = input.TempID
	}

	// เรียกใช้ service
	message, err := h.messageService.SendFileMessage(
		conversationID,
		userID,
		input.MediaURL,
		input.FileName,
		input.FileSize,
		input.FileType,
		metadata,
	)

	if err != nil {
		statusCode := fiber.StatusInternalServerError
		// ตรวจสอบประเภทข้อผิดพลาด
		if err.Error() == "user is not a member of this conversation" {
			statusCode = fiber.StatusForbidden
		} else if err.Error() == "file URL is required" {
			statusCode = fiber.StatusBadRequest
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	h.notificationService.NotifyNewMessage(conversationID, message)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "File sent successfully",
		"data":    message,
	})
}

// SendBulkMessages จัดการคำขอส่งหลายข้อความพร้อมกัน (Album/Group Message)
func (h *MessageHandler) SendBulkMessages(c *fiber.Ctx) error {
	// ดึง User ID จาก context
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	conversationID, err := utils.ParseUUIDParam(c, "conversationId")
	if err != nil {
		return err // error response ถูกจัดการในฟังก์ชันแล้ว
	}

	// ตรวจสอบ block status ก่อนส่งข้อความ
	if err := h.checkBlockStatusBeforeSend(userID, conversationID); err != nil {
		// ✅ ถ้าเป็น BlockError ให้ return พร้อม error_code
		if blockErr, ok := err.(*BlockError); ok {
			response := fiber.Map{
				"success":    false,
				"error_code": blockErr.Code,
				"message":    blockErr.Message,
			}
			// เพิ่ม blocker_id ถ้ามี
			if blockErr.BlockerID != nil {
				response["blocker_id"] = blockErr.BlockerID.String()
			}
			return c.Status(fiber.StatusForbidden).JSON(response)
		}

		// Error อื่นๆ
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	// รับข้อมูล bulk messages จาก request body
	var input struct {
		Caption  string                   `json:"caption"`  // caption สำหรับอัลบั้ม
		Messages []map[string]interface{} `json:"messages"`
	}

	if err := c.BodyParser(&input); err != nil {
		fmt.Printf("❌ [SendBulkMessages] Body parse error: %v\n", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body: " + err.Error(),
		})
	}

	// ตรวจสอบว่ามี messages หรือไม่
	if len(input.Messages) == 0 {
		fmt.Printf("❌ [SendBulkMessages] No messages in request\n")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "At least one message is required",
		})
	}

	fmt.Printf("📤 [SendBulkMessages] Received %d files for album in conversation %s (caption: %q)\n", len(input.Messages), conversationID, input.Caption)

	// เรียกใช้ service - ได้ 1 message ที่มี type "album" กลับมา
	message, err := h.messageService.SendBulkMessages(
		conversationID,
		userID,
		input.Caption,  // ส่ง caption เข้าไป
		input.Messages,
	)

	if err != nil {
		fmt.Printf("❌ [SendBulkMessages] Service error: %v\n", err)
		statusCode := fiber.StatusInternalServerError
		// ตรวจสอบประเภทข้อผิดพลาด
		if err.Error() == "user is not a member of this conversation" {
			statusCode = fiber.StatusForbidden
		} else if err.Error() == "maximum 10 files per album" ||
		          err.Error() == "at least one file is required" {
			statusCode = fiber.StatusBadRequest
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	// ส่ง WebSocket notification สำหรับ album message
	h.notificationService.NotifyNewMessage(conversationID, message)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Album sent successfully",
		"data":    message,
	})
}

// EditMessage จัดการคำขอแก้ไขข้อความ
// EditMessage จัดการคำขอแก้ไขข้อความ
func (h *MessageHandler) EditMessage(c *fiber.Ctx) error {
	// Panic recovery เพื่อ debug
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("❌ [EditMessage] PANIC RECOVERED: %v\n", r)
			fmt.Printf("Stack trace:\n%s\n", debug.Stack())
		}
	}()

	fmt.Printf("📝 [EditMessage] Starting edit for message: %s\n", c.Params("messageId"))

	// ดึง User ID จาก context
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		fmt.Printf("❌ [EditMessage] Auth error: %v\n", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	fmt.Printf("📝 [EditMessage] User ID: %s\n", userID)

	messageID, err := utils.ParseUUIDParam(c, "messageId")
	if err != nil {
		fmt.Printf("❌ [EditMessage] Parse UUID error: %v\n", err)
		return err // error response ถูกจัดการในฟังก์ชันแล้ว
	}

	fmt.Printf("📝 [EditMessage] Message ID parsed: %s\n", messageID)

	// รับข้อมูลการแก้ไขจาก request body
	var input struct {
		Content string `json:"content"`
	}

	if err := c.BodyParser(&input); err != nil {
		fmt.Printf("❌ [EditMessage] Body parse error: %v\n", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body: " + err.Error(),
		})
	}

	fmt.Printf("📝 [EditMessage] New content: %q\n", input.Content)

	// ตรวจสอบ messageService
	if h.messageService == nil {
		fmt.Printf("❌ [EditMessage] messageService is nil!\n")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Message service not initialized",
		})
	}

	fmt.Printf("📝 [EditMessage] Calling service.EditMessage...\n")

	// เรียกใช้ service
	message, err := h.messageService.EditMessage(messageID, userID, input.Content)

	fmt.Printf("📝 [EditMessage] Service returned. Error: %v, Message: %v\n", err, message != nil)

	if err != nil {
		fmt.Printf("❌ [EditMessage] Service error: %v\n", err)
		statusCode := fiber.StatusInternalServerError
		// ตรวจสอบประเภทข้อผิดพลาด
		if err.Error() == "message not found" {
			statusCode = fiber.StatusNotFound
		} else if err.Error() == "only message owner can edit messages" {
			statusCode = fiber.StatusForbidden
		} else if err.Error() == "cannot edit deleted message" || err.Error() == "only text messages can be edited" {
			statusCode = fiber.StatusBadRequest
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	fmt.Printf("📝 [EditMessage] Message edited successfully\n")

	// ส่ง WebSocket notification สำหรับการแก้ไขข้อความ
	if h.notificationService != nil {
		editEventData := fiber.Map{
			"message_id":      message.ID.String(),
			"conversation_id": message.ConversationID.String(),
			"new_content":     message.Content,
			"edited_at":       message.UpdatedAt.Format(time.RFC3339),
		}
		h.notificationService.NotifyMessageEdited(message.ConversationID, editEventData)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Message updated successfully",
		"data":    message,
	})
}

// DeleteMessage จัดการคำขอลบข้อความ
func (h *MessageHandler) DeleteMessage(c *fiber.Ctx) error {
	// ดึง User ID จาก context
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	messageID, err := utils.ParseUUIDParam(c, "messageId")
	if err != nil {
		return err // error response ถูกจัดการในฟังก์ชันแล้ว
	}

	// เรียกใช้ service
	err = h.messageService.DeleteMessage(messageID, userID)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		// ตรวจสอบประเภทข้อผิดพลาด
		if err.Error() == "message not found" {
			statusCode = fiber.StatusNotFound
		} else if err.Error() == "only message owner or conversation admin can delete messages" {
			statusCode = fiber.StatusForbidden
		} else if err.Error() == "message is already deleted" {
			statusCode = fiber.StatusBadRequest
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Message deleted successfully",
	})
}

// GetMessageEditHistory จัดการคำขอดูประวัติการแก้ไขข้อความ
func (h *MessageHandler) GetMessageEditHistory(c *fiber.Ctx) error {
	// ดึง User ID จาก context
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	messageID, err := utils.ParseUUIDParam(c, "messageId")
	if err != nil {
		return err // error response ถูกจัดการในฟังก์ชันแล้ว
	}

	// เรียกใช้ service
	history, err := h.messageService.GetMessageEditHistory(messageID, userID)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		// ตรวจสอบประเภทข้อผิดพลาด
		if err.Error() == "message not found" {
			statusCode = fiber.StatusNotFound
		} else if err.Error() == "you are not a member of this conversation" {
			statusCode = fiber.StatusForbidden
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Edit history retrieved successfully",
		"data":    history,
	})
}

// GetMessageDeleteHistory จัดการคำขอดูประวัติการลบข้อความ
func (h *MessageHandler) GetMessageDeleteHistory(c *fiber.Ctx) error {
	// ดึง User ID จาก context
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	messageID, err := utils.ParseUUIDParam(c, "messageId")
	if err != nil {
		return err // error response ถูกจัดการในฟังก์ชันแล้ว
	}

	// เรียกใช้ service
	history, err := h.messageService.GetMessageDeleteHistory(messageID, userID)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		// ตรวจสอบประเภทข้อผิดพลาด
		if err.Error() == "message not found" {
			statusCode = fiber.StatusNotFound
		} else if err.Error() == "only admins can view delete history" {
			statusCode = fiber.StatusForbidden
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Delete history retrieved successfully",
		"data":    history,
	})
}

// ReplyToMessage จัดการคำขอตอบกลับข้อความ
func (h *MessageHandler) ReplyToMessage(c *fiber.Ctx) error {
	// ดึง User ID จาก context
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	replyToID, err := utils.ParseUUIDParam(c, "messageId")
	if err != nil {
		return err // error response ถูกจัดการในฟังก์ชันแล้ว
	}

	// รับข้อมูลการตอบกลับจาก request body
	var input struct {
		MessageType       string      `json:"message_type"`
		Content           string      `json:"content"`
		MediaURL          string      `json:"media_url"`
		MediaThumbnailURL string      `json:"media_thumbnail_url"`
		Metadata          types.JSONB `json:"metadata"`
		SenderType        string      `json:"sender_type"` // เพิ่ม field นี้
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body: " + err.Error(),
		})
	}

	// เรียกใช้ service
	message, err := h.messageService.ReplyToMessage(
		replyToID,
		userID,
		input.MessageType,
		input.Content,
		input.MediaURL,
		input.MediaThumbnailURL,
		input.Metadata,
	)

	if err != nil {
		statusCode := fiber.StatusInternalServerError
		// ตรวจสอบประเภทข้อผิดพลาด
		if err.Error() == "message not found" {
			statusCode = fiber.StatusNotFound
		} else if err.Error() == "you are not a member of this conversation" {
			statusCode = fiber.StatusForbidden
		} else if err.Error() == "cannot reply to deleted message" || err.Error() == "invalid message type" {
			statusCode = fiber.StatusBadRequest
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	h.notificationService.NotifyNewMessage(message.ConversationID, message)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Reply sent successfully",
		"data":    message,
	})
}

// PinMessage ปักหมุดข้อความ
func (h *MessageHandler) PinMessage(c *fiber.Ctx) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	conversationID, err := utils.ParseUUIDParam(c, "conversationId")
	if err != nil {
		return err
	}

	messageID, err := utils.ParseUUIDParam(c, "messageId")
	if err != nil {
		return err
	}

	// ปักหมุดข้อความ
	if err := h.messageService.PinMessage(messageID, conversationID, userID); err != nil {
		statusCode := fiber.StatusInternalServerError
		if err.Error() == "message not found" {
			statusCode = fiber.StatusNotFound
		} else if err.Error() == "user is not a member of this conversation" ||
			err.Error() == "only owner/admin can pin messages in group conversations" {
			statusCode = fiber.StatusForbidden
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	// TODO: ส่ง WebSocket notification
	// h.notificationService.NotifyMessagePinned(conversationID, types.JSONB{
	// 	"message_id": messageID.String(),
	// 	"pinned_by":  userID.String(),
	// })

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Message pinned successfully",
	})
}

// UnpinMessage ยกเลิกการปักหมุดข้อความ
func (h *MessageHandler) UnpinMessage(c *fiber.Ctx) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	conversationID, err := utils.ParseUUIDParam(c, "conversationId")
	if err != nil {
		return err
	}

	messageID, err := utils.ParseUUIDParam(c, "messageId")
	if err != nil {
		return err
	}

	// ยกเลิกการปักหมุด
	if err := h.messageService.UnpinMessage(messageID, conversationID, userID); err != nil {
		statusCode := fiber.StatusInternalServerError
		if err.Error() == "message not found" {
			statusCode = fiber.StatusNotFound
		} else if err.Error() == "user is not a member of this conversation" ||
			err.Error() == "only owner/admin or the user who pinned can unpin messages" {
			statusCode = fiber.StatusForbidden
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	// TODO: ส่ง WebSocket notification
	// h.notificationService.NotifyMessageUnpinned(conversationID, types.JSONB{
	// 	"message_id": messageID.String(),
	// 	"unpinned_by": userID.String(),
	// })

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Message unpinned successfully",
	})
}

// GetPinnedMessages ดึงรายการข้อความที่ปักหมุดในการสนทนา
func (h *MessageHandler) GetPinnedMessages(c *fiber.Ctx) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	conversationID, err := utils.ParseUUIDParam(c, "conversationId")
	if err != nil {
		return err
	}

	limit := c.QueryInt("limit", 20)
	if limit > 100 {
		limit = 100
	}
	offset := c.QueryInt("offset", 0)

	// ดึงข้อความที่ปักหมุด
	messages, total, err := h.messageService.GetPinnedMessages(conversationID, userID, limit, offset)
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
			"messages": messages,
			"pagination": fiber.Map{
				"total":  total,
				"limit":  limit,
				"offset": offset,
			},
		},
	})
}

// SearchMessages ค้นหาข้อความ (CURSOR-BASED)
func (h *MessageHandler) SearchMessages(c *fiber.Ctx) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// รับ query parameter
	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Search query (q) is required",
		})
	}

	// รับ conversation_id (optional)
	var conversationID *uuid.UUID
	conversationIDStr := c.Query("conversation_id")
	if conversationIDStr != "" {
		id, err := uuid.Parse(conversationIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid conversation_id format",
			})
		}
		conversationID = &id
	}

	// Cursor pagination parameters
	limit := c.QueryInt("limit", 20)
	if limit > 100 {
		limit = 100
	}

	cursor := c.Query("cursor") // Message ID
	var cursorPtr *string
	if cursor != "" {
		cursorPtr = &cursor
	}

	direction := c.Query("direction", "before") // "before" | "after"
	if direction != "before" && direction != "after" {
		direction = "before"
	}

	// ค้นหาข้อความ
	messages, nextCursor, hasMore, err := h.messageService.SearchMessages(
		query,
		conversationID,
		userID,
		limit,
		cursorPtr,
		direction,
	)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if err.Error() == "user is not a member of this conversation" {
			statusCode = fiber.StatusForbidden
		} else if err.Error() == "invalid cursor" || err.Error() == "cursor message not found" {
			statusCode = fiber.StatusBadRequest
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"messages": messages,
			"query":    query,
			"cursor":   nextCursor,
			"has_more": hasMore,
		},
	})
}

// ForwardMessages ส่งต่อข้อความไปยังการสนทนาอื่น
func (h *MessageHandler) ForwardMessages(c *fiber.Ctx) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// Parse request body
	var input struct {
		MessageIDs            []uuid.UUID `json:"message_ids"`
		TargetConversationIDs []uuid.UUID `json:"target_conversation_ids"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body: " + err.Error(),
		})
	}

	// Validate input
	if len(input.MessageIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "message_ids cannot be empty",
		})
	}

	if len(input.TargetConversationIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "target_conversation_ids cannot be empty",
		})
	}

	// ส่งต่อข้อความ
	results, err := h.messageService.ForwardMessages(input.MessageIDs, input.TargetConversationIDs, userID)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if err.Error() == "user is not a member of the source conversation" ||
		   err.Error() == "user is not a member of the target conversation" {
			statusCode = fiber.StatusForbidden
		} else if err.Error() == "message not found" {
			statusCode = fiber.StatusNotFound
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	// ส่ง WebSocket notification สำหรับแต่ละข้อความที่ส่งต่อ
	for conversationID, messages := range results {
		for _, message := range messages {
			h.notificationService.NotifyNewMessage(conversationID, message)
		}
	}

	// นับจำนวนข้อความที่ส่งต่อสำเร็จ
	totalForwarded := 0
	for _, messages := range results {
		totalForwarded += len(messages)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Messages forwarded successfully",
		"data": fiber.Map{
			"forwarded_messages": results,
			"total_forwarded":    totalForwarded,
		},
	})
}

