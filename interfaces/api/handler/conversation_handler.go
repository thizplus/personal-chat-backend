// interfaces/api/handler/conversation_handler.go
package handler

import (
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/dto"
	"github.com/thizplus/gofiber-chat-api/domain/repository"
	"github.com/thizplus/gofiber-chat-api/domain/service"
	"github.com/thizplus/gofiber-chat-api/domain/types"
	"github.com/thizplus/gofiber-chat-api/interfaces/api/middleware"
	"github.com/thizplus/gofiber-chat-api/pkg/utils"
)

// ConversationHandler จัดการ HTTP requests เกี่ยวกับการสนทนา
type ConversationHandler struct {
	conversationService  service.ConversationService
	notificationService  service.NotificationService
	messageReadService   service.MessageReadService
	groupActivityService service.GroupActivityService
	conversationRepo     repository.ConversationRepository
	messageService       service.MessageService
}

// NewConversationHandler สร้าง handler ใหม่
func NewConversationHandler(
	conversationService service.ConversationService,
	notificationService service.NotificationService,
	messageReadService service.MessageReadService,
	groupActivityService service.GroupActivityService,
	conversationRepo repository.ConversationRepository,
	messageService service.MessageService,
) *ConversationHandler {
	return &ConversationHandler{
		conversationService:  conversationService,
		notificationService:  notificationService,
		messageReadService:   messageReadService,
		groupActivityService: groupActivityService,
		conversationRepo:     conversationRepo,
		messageService:       messageService,
	}
}

// Create สร้างการสนทนาใหม่
func (h *ConversationHandler) Create(c *fiber.Ctx) error {
	// ดึง User ID จาก middleware
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// รับข้อมูลการสนทนาจาก request body
	var input types.JSONB
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request data: " + err.Error(),
		})
	}

	// ตรวจสอบและกำหนดค่า type
	conversationType, ok := input["type"].(string)
	if !ok || (conversationType != "direct" && conversationType != "group" && conversationType != "business") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid conversation type, must be 'direct', 'group', or 'business'",
		})
	}

	// แยกจัดการตามประเภทการสนทนา
	switch conversationType {
	case "direct":
		return h.createDirectConversation(c, userID, input)
	case "group":
		return h.createGroupConversation(c, userID, input)
	default:
		// ไม่ควรเข้าเงื่อนไขนี้เนื่องจากมีการตรวจสอบข้างต้นแล้ว
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid conversation type",
		})
	}
}

// createDirectConversation สร้างการสนทนาแบบ direct (ระหว่างผู้ใช้สองคน)
func (h *ConversationHandler) createDirectConversation(c *fiber.Ctx, userID uuid.UUID, input types.JSONB) error {
	// ตรวจสอบ member_ids
	memberIDs, ok := input["member_ids"].([]interface{})
	if !ok || len(memberIDs) != 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Direct conversation requires exactly one other user ID",
		})
	}

	otherUserIDStr, ok := memberIDs[0].(string)
	if !ok || otherUserIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid member ID",
		})
	}

	// แปลง string เป็น UUID
	otherUserID, err := uuid.Parse(otherUserIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid member ID format",
		})
	}

	// เรียกใช้ service
	conversation, err := h.conversationService.CreateDirectConversation(userID, otherUserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	// รวมรายการผู้ใช้ที่เกี่ยวข้องกับการสนทนานี้
	allMembers := []uuid.UUID{userID, otherUserID}

	// เรียกใช้ NotifyConversationCreated แทน NotifyNewConversation
	err = h.notificationService.NotifyConversationCreated(allMembers, conversation)

	if err != nil {
		// บันทึก log แต่ไม่ส่ง error กลับไป
		log.Printf("Error sending conversation notification: %v", err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success":      true,
		"message":      "Conversation created successfully",
		"conversation": conversation,
	})
}

// createGroupConversation สร้างการสนทนาแบบกลุ่ม
func (h *ConversationHandler) createGroupConversation(c *fiber.Ctx, userID uuid.UUID, input types.JSONB) error {
	// ตรวจสอบชื่อกลุ่ม
	title, ok := input["title"].(string)
	if !ok || title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Group conversation requires a title",
		})
	}

	// รูปกลุ่ม (ไม่บังคับ)
	iconURL := ""
	if iconURLValue, ok := input["icon_url"].(string); ok {
		iconURL = iconURLValue
	}

	// แปลง member_ids เป็น []uuid.UUID
	var memberIDs []uuid.UUID
	if memberIDsValue, ok := input["member_ids"].([]interface{}); ok {
		for _, memberID := range memberIDsValue {
			if idStr, ok := memberID.(string); ok && idStr != "" {
				if id, err := uuid.Parse(idStr); err == nil {
					memberIDs = append(memberIDs, id)
				}
			}
		}
	}

	// เรียกใช้ service
	conversation, err := h.conversationService.CreateGroupConversation(userID, title, iconURL, memberIDs)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	// รวมรายการผู้ใช้ทั้งหมดที่เกี่ยวข้อง (creator + members)
	allMembers := append([]uuid.UUID{userID}, memberIDs...)

	// ส่ง WebSocket notification แจ้งสมาชิกทุกคนในกลุ่ม
	err = h.notificationService.NotifyConversationCreated(allMembers, conversation)
	if err != nil {
		log.Printf("Failed to send group conversation created notification: %v", err)
		// ไม่ return error เพราะการส่ง notification ล้มเหลวไม่ควรทำให้ API ล้มเหลว
	}

	// บันทึก activity log
	conversationID, _ := uuid.Parse(conversation.ID.String())
	h.groupActivityService.LogGroupCreated(conversationID, userID)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success":      true,
		"message":      "Group conversation created successfully",
		"conversation": conversation,
	})
}


// GetUserConversations ดึงรายการการสนทนาทั้งหมดของผู้ใช้
func (h *ConversationHandler) GetUserConversations(c *fiber.Ctx) error {
	// ดึง User ID จาก middleware
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// ดึงพารามิเตอร์การแบ่งหน้าและการโหลดเพิ่มเติม
	limit := 20
	offset := 0

	if limitStr := c.Query("limit"); limitStr != "" {
		limitVal := c.QueryInt("limit", 20) // ใช้ค่าเริ่มต้น 20
		if limitVal > 0 {
			if limitVal > 50 {
				limitVal = 50 // จำกัดสูงสุดที่ 50
			}
			limit = limitVal
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		offsetVal := c.QueryInt("offset", 0) // ใช้ค่าเริ่มต้น 0
		if offsetVal >= 0 {
			offset = offsetVal
		}
	}

	// พารามิเตอร์สำหรับการเลื่อนหน้า (Infinity Scroll)
	beforeTime := c.Query("before_time") // เวลาของการสนทนาเก่าสุดที่มี (เพื่อโหลดเก่ากว่า)
	afterTime := c.Query("after_time")   // เวลาของการสนทนาใหม่สุดที่มี (เพื่อโหลดใหม่กว่า)
	beforeID := c.Query("before_id")     // ID ของการสนทนาเก่าสุดที่มี (ทางเลือกเพิ่มเติม)
	afterID := c.Query("after_id")       // ID ของการสนทนาใหม่สุดที่มี (ทางเลือกเพิ่มเติม)

	// ตัวกรองเพิ่มเติม
	conversationType := c.Query("type")    // กรองตามประเภท: direct, group, business
	pinned := c.QueryBool("pinned", false) // กรองเฉพาะที่ปักหมุด

	// เรียกใช้ service เพื่อดึงรายการการสนทนา
	var conversations []*dto.ConversationDTO
	var total int
	var hasMore bool

	// จัดการตามโหมดการดึงข้อมูล
	if beforeTime != "" {
		// โหมดโหลดการสนทนาที่เก่ากว่า (โดยใช้เวลา)
		conversations, total, err = h.conversationService.GetConversationsBeforeTime(
			userID, beforeTime, limit, conversationType, pinned)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Error retrieving conversations before time: " + err.Error(),
			})
		}

		hasMore = len(conversations) >= limit
	} else if afterTime != "" {
		// โหมดโหลดการสนทนาที่ใหม่กว่า (โดยใช้เวลา)
		conversations, total, err = h.conversationService.GetConversationsAfterTime(
			userID, afterTime, limit, conversationType, pinned)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Error retrieving conversations after time: " + err.Error(),
			})
		}

		hasMore = len(conversations) >= limit
	} else if beforeID != "" {
		// โหมดโหลดการสนทนาที่เก่ากว่า (โดยใช้ ID)
		beforeUUID, err := uuid.Parse(beforeID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid before_id format",
			})
		}

		conversations, total, err = h.conversationService.GetConversationsBeforeID(
			userID, beforeUUID, limit, conversationType, pinned)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Error retrieving conversations before ID: " + err.Error(),
			})
		}

		hasMore = len(conversations) >= limit
	} else if afterID != "" {
		// โหมดโหลดการสนทนาที่ใหม่กว่า (โดยใช้ ID)
		afterUUID, err := uuid.Parse(afterID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid after_id format",
			})
		}

		conversations, total, err = h.conversationService.GetConversationsAfterID(
			userID, afterUUID, limit, conversationType, pinned)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Error retrieving conversations after ID: " + err.Error(),
			})
		}

		hasMore = len(conversations) >= limit
	} else {
		// โหมดโหลดการสนทนาล่าสุด (เริ่มต้น)
		conversations, total, err = h.conversationService.GetUserConversations(
			userID, limit, offset, conversationType, pinned)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Error retrieving conversations: " + err.Error(),
			})
		}

		hasMore = offset+len(conversations) < total
	}

	// ส่งคืนข้อมูลในรูปแบบที่สอดคล้องกับโค้ดเก่า
	if c.Query("format") == "legacy" || c.Query("format") == "old" {
		// รูปแบบเก่า (สำหรับความเข้ากันได้กับระบบเดิม)
		return c.JSON(fiber.Map{
			"success":       true,
			"message":       "Conversations retrieved successfully",
			"conversations": conversations,
			"pagination": fiber.Map{
				"total":  total,
				"limit":  limit,
				"offset": offset,
			},
		})
	}

	// รูปแบบใหม่ (เพิ่ม has_more สำหรับ Infinity Scroll)
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Conversations retrieved successfully",
		"data": fiber.Map{
			"conversations": conversations,
			"has_more":      hasMore,
			"pagination": fiber.Map{
				"total":  total,
				"limit":  limit,
				"offset": offset,
			},
		},
	})
}

// GetConversationMessages ดึงข้อความทั้งหมดในการสนทนา
func (h *ConversationHandler) GetConversationMessages(c *fiber.Ctx) error {
	// ดึง User ID จาก middleware
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// ดึงและตรวจสอบ conversation ID
	conversationID, err := utils.ParseUUIDParam(c, "conversationId")
	if err != nil {
		return err
	}

	// ตรวจสอบว่าผู้ใช้เป็นสมาชิกของการสนทนานี้
	isMember, err := h.conversationService.CheckMembership(userID, conversationID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Error checking membership: " + err.Error(),
		})
	}

	if !isMember {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": "You are not a member of this conversation",
		})
	}

	// ดึงพารามิเตอร์
	limit := c.QueryInt("limit", 20)
	if limit > 50 {
		limit = 50 // จำกัดสูงสุดที่ 50
	}

	// ตรวจสอบโหมดการดึงข้อมูล
	beforeID := c.Query("before") // ID ข้อความเก่าสุดที่มี (เพื่อโหลดข้อความเก่ากว่า)
	afterID := c.Query("after")   // ID ข้อความใหม่สุดที่มี (เพื่อโหลดข้อความใหม่กว่า)
	targetID := c.Query("target") // ID ข้อความเป้าหมาย (เพื่อไปยังข้อความเฉพาะ)

	// ตัวแปรสำหรับเก็บผลลัพธ์
	var messages []*dto.MessageDTO
	var hasMore bool
	var hasMoreBefore, hasMoreAfter bool = false, false
	var total int64

	if targetID != "" {
		// โหมด Jump to Message
		beforeCount := c.QueryInt("before_count", 10)
		afterCount := c.QueryInt("after_count", 10)

		// เรียกใช้ service เพื่อดึงข้อความรอบๆ เป้าหมาย
		var err error
		messages, hasMoreBefore, hasMoreAfter, err = h.conversationService.GetMessageContext(
			conversationID, userID, targetID, beforeCount, afterCount)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Error retrieving target message: " + err.Error(),
			})
		}

		// ส่งคืนข้อมูลในรูปแบบสำหรับ jump to message
		return c.JSON(fiber.Map{
			"success": true,
			"message": "Messages retrieved successfully",
			"data": fiber.Map{
				"messages":        messages,
				"target_id":       targetID,
				"has_more_before": hasMoreBefore,
				"has_more_after":  hasMoreAfter,
			},
		})
	} else if beforeID != "" {
		// โหมดโหลดข้อความเก่ากว่า (เลื่อนขึ้น - scroll up)
		var err error
		messages, total, err = h.conversationService.GetMessagesBeforeID(
			conversationID, userID, beforeID, limit)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Error retrieving messages: " + err.Error(),
			})
		}

		hasMore = len(messages) >= limit
	} else if afterID != "" {
		// โหมดโหลดข้อความใหม่กว่า (เลื่อนลง - scroll down)
		var err error
		messages, total, err = h.conversationService.GetMessagesAfterID(
			conversationID, userID, afterID, limit)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Error retrieving messages: " + err.Error(),
			})
		}

		hasMore = len(messages) >= limit
	} else {
		// โหมดโหลดข้อความล่าสุด (เริ่มต้น)
		offset := c.QueryInt("offset", 0)

		var err error
		messages, total, err = h.conversationService.GetConversationMessages(
			conversationID, userID, limit, offset)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Error retrieving messages: " + err.Error(),
			})
		}

		hasMore = int64(offset+len(messages)) < total
	}

	// ส่งคืนข้อมูลในรูปแบบทั่วไป
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Messages retrieved successfully",
		"data": fiber.Map{
			"messages": messages,
			"has_more": hasMore,
			"total":    total,
		},
	})
}

// TogglePinConversation เปลี่ยนสถานะปักหมุดของการสนทนา
func (h *ConversationHandler) TogglePinConversation(c *fiber.Ctx) error {
	// ดึง User ID จาก middleware
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// ดึงและตรวจสอบ conversation ID
	conversationID, err := utils.ParseUUIDParam(c, "conversationId")
	if err != nil {
		return err
	}

	// รับข้อมูลการปักหมุดจาก request body
	var input struct {
		IsPinned bool `json:"is_pinned"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request data: " + err.Error(),
		})
	}

	// เรียกใช้ service
	err = h.conversationService.SetPinStatus(conversationID, userID, input.IsPinned)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if err.Error() == "you are not a member of this conversation" {
			statusCode = fiber.StatusForbidden
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Conversation pin status updated successfully",
		"data": fiber.Map{
			"is_pinned": input.IsPinned,
		},
	})
}

// ToggleMuteConversation เปลี่ยนสถานะการปิดเสียงของการสนทนา
func (h *ConversationHandler) ToggleMuteConversation(c *fiber.Ctx) error {
	// ดึง User ID จาก middleware
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// ดึงและตรวจสอบ conversation ID
	conversationID, err := utils.ParseUUIDParam(c, "conversationId")
	if err != nil {
		return err
	}

	// รับข้อมูลการปิดเสียงจาก request body
	var input struct {
		IsMuted bool `json:"is_muted"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request data: " + err.Error(),
		})
	}

	// เรียกใช้ service
	err = h.conversationService.SetMuteStatus(conversationID, userID, input.IsMuted)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if err.Error() == "you are not a member of this conversation" {
			statusCode = fiber.StatusForbidden
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Conversation mute status updated successfully",
		"data": fiber.Map{
			"is_muted": input.IsMuted,
		},
	})
}

// UpdateConversation อัปเดตข้อมูลการสนทนา (ชื่อ, icon)
func (h *ConversationHandler) UpdateConversation(c *fiber.Ctx) error {
	// ดึงข้อมูลผู้ใช้จาก context
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// ดึง conversation ID จาก parameter
	conversationID, err := utils.ParseUUIDParam(c, "conversationId")
	if err != nil {
		return err
	}

	// ตรวจสอบสิทธิ์ - ตรวจสอบว่าผู้ใช้เป็นสมาชิกของการสนทนาหรือไม่
	isMember, err := h.conversationService.CheckMembership(userID, conversationID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Error checking membership: " + err.Error(),
		})
	}

	if !isMember {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": "You are not a member of this conversation",
		})
	}

	// ดึงข้อมูลเดิมก่อน update (สำหรับ activity log)
	oldConversation, err := h.conversationRepo.GetByID(conversationID)
	if err != nil {
		// ถ้าดึงไม่ได้ ให้ทำต่อไปได้ แต่จะไม่มี activity log
		oldConversation = nil
	}

	// รับข้อมูลที่ต้องการอัปเดต
	var input struct {
		Title   string `json:"title"`
		IconURL string `json:"icon_url"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request data: " + err.Error(),
		})
	}

	// สร้าง types.JSONB โดยตรง (สำคัญ!)
	updateData := types.JSONB{}

	if input.Title != "" {
		updateData["title"] = input.Title
	}

	if input.IconURL != "" {
		updateData["icon_url"] = input.IconURL
	}

	if len(updateData) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "No changes to update",
		})
	}

	// เพิ่มเวลาอัปเดต
	updateData["updated_at"] = time.Now()

	// ก่อนเรียก UpdateConversation
	fmt.Printf("UpdateData before call: %+v\n", updateData)

	// ส่ง types.JSONB ไปยัง service
	err = h.conversationService.UpdateConversation(conversationID, updateData)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to update conversation: " + err.Error(),
		})
	}

	// ส่ง WebSocket notification แจ้งสมาชิกทุกคนในกลุ่ม
	notificationData := types.JSONB{
		"conversation_id": conversationID.String(),
	}
	if title, ok := updateData["title"]; ok {
		notificationData["title"] = title
	}
	if iconURL, ok := updateData["icon_url"]; ok {
		notificationData["icon_url"] = iconURL
	}
	h.notificationService.NotifyConversationUpdated(conversationID, notificationData)

	// บันทึก activity log
	if oldConversation != nil {
		if input.Title != "" && input.Title != oldConversation.Title {
			h.groupActivityService.LogGroupNameChanged(conversationID, userID, oldConversation.Title, input.Title)
		}
		if input.IconURL != "" && input.IconURL != oldConversation.IconURL {
			h.groupActivityService.LogGroupIconChanged(conversationID, userID, oldConversation.IconURL, input.IconURL)
		}
	}

	// ส่งผลลัพธ์กลับ
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Conversation updated successfully",
	})
}

// GetMediaSummary ดึงสรุปจำนวน media และ link ในการสนทนา
// GET /conversations/:conversationId/media/summary
func (h *ConversationHandler) GetMediaSummary(c *fiber.Ctx) error {
	// ดึง userID จาก context
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// ดึง conversationID จาก path parameter
	conversationIDStr := c.Params("conversationId")
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid conversation ID",
		})
	}

	// เรียก service
	summary, err := h.conversationService.GetConversationMediaSummary(conversationID, userID)
	if err != nil {
		if err.Error() == "user is not a member of this conversation" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    summary,
	})
}

// GetMediaByType ดึงรายละเอียด media ตามประเภทพร้อม pagination
// GET /conversations/:conversationId/media?type=image&limit=20&offset=0
func (h *ConversationHandler) GetMediaByType(c *fiber.Ctx) error {
	// ดึง userID จาก context
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// ดึง conversationID จาก path parameter
	conversationIDStr := c.Params("conversationId")
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid conversation ID",
		})
	}

	// ดึง query parameters
	mediaType := c.Query("type", "image")
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	// เรียก service
	result, err := h.conversationService.GetConversationMediaByType(conversationID, userID, mediaType, limit, offset)
	if err != nil {
		if err.Error() == "user is not a member of this conversation" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result.Data,
		"pagination": result.Pagination,
	})
}

// GetMessageContext ดึงข้อความเป้าหมายพร้อมข้อความก่อนหน้าและถัดไป (Jump to Message)
// GET /conversations/:conversationId/messages/context?targetId=xxx&before=10&after=10
func (h *ConversationHandler) GetMessageContext(c *fiber.Ctx) error {
	// ดึง userID จาก context
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// ดึง conversationID จาก path parameter
	conversationIDStr := c.Params("conversationId")
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid conversation ID",
		})
	}

	// ดึง query parameters
	targetID := c.Query("targetId")
	if targetID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "targetId is required",
		})
	}

	beforeCount := c.QueryInt("before", 10)
	afterCount := c.QueryInt("after", 10)

	// เรียก service
	messages, hasBefore, hasAfter, err := h.conversationService.GetMessageContext(
		conversationID, userID, targetID, beforeCount, afterCount,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success":    true,
		"data":       messages,
		"has_before": hasBefore,
		"has_after":  hasAfter,
	})
}

// HideConversation ซ่อน/แสดง conversation
func (h *ConversationHandler) HideConversation(c *fiber.Ctx) error {
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

	var input dto.ConversationHideRequest
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request data: " + err.Error(),
		})
	}

	err = h.conversationService.SetHiddenStatus(conversationID, userID, input.IsHidden)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	var hiddenAt *time.Time
	if input.IsHidden {
		now := time.Now()
		hiddenAt = &now
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Conversation hidden status updated successfully",
		"data": fiber.Map{
			"is_hidden": input.IsHidden,
			"hidden_at": hiddenAt,
		},
	})
}

// DeleteConversation ลบ conversation (smart delete)
func (h *ConversationHandler) DeleteConversation(c *fiber.Ctx) error {
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

	action, err := h.conversationService.DeleteConversation(conversationID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	var message string
	if action == "hidden" {
		message = "Conversation hidden successfully"
	} else {
		message = "Left conversation successfully"
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": message,
		"data": fiber.Map{
			"conversation_id": conversationID.String(),
			"action":          action,
			"message":         message,
		},
	})
}

// MarkConversationAsRead ทำเครื่องหมายข้อความในการสนทนาว่าอ่านแล้ว
func (h *ConversationHandler) MarkConversationAsRead(c *fiber.Ctx) error {
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

	var input struct {
		LastReadMessageID string `json:"last_read_message_id"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request data: " + err.Error(),
		})
	}

	// แปลง lastReadMessageID เป็น UUID
	lastReadMessageID, err := uuid.Parse(input.LastReadMessageID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid message ID format",
		})
	}

	// Mark conversation as read
	unreadCount, err := h.messageReadService.MarkConversationAsRead(conversationID, userID, lastReadMessageID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to mark conversation as read: " + err.Error(),
		})
	}

	// Send WebSocket event
	readData := fiber.Map{
		"conversation_id":       conversationID.String(),
		"user_id":              userID.String(),
		"last_read_message_id": lastReadMessageID.String(),
		"read_at":              time.Now().Format(time.RFC3339),
	}
	h.notificationService.NotifyMessageRead(conversationID, readData)

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"conversation_id":       conversationID.String(),
			"last_read_message_id": lastReadMessageID.String(),
			"unread_count":         unreadCount,
		},
	})
}

// GetUnreadCounts ดึงจำนวนข้อความที่ยังไม่ได้อ่านในทุกการสนทนา
func (h *ConversationHandler) GetUnreadCounts(c *fiber.Ctx) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// Get unread counts
	unreadCounts, totalUnread, err := h.messageReadService.GetUnreadCounts(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to get unread counts: " + err.Error(),
		})
	}

	// แปลงเป็น array สำหรับ JSON response
	conversations := make([]fiber.Map, 0)
	for conversationID, count := range unreadCounts {
		conversations = append(conversations, fiber.Map{
			"conversation_id": conversationID.String(),
			"unread_count":    count,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"conversations": conversations,
			"total_unread":  totalUnread,
		},
	})
}

// TransferOwnership โอนความเป็นเจ้าของกลุ่มให้สมาชิกคนอื่น
// POST /conversations/:conversationId/transfer-ownership
func (h *ConversationHandler) TransferOwnership(c *fiber.Ctx) error {
	// 1. ดึงข้อมูลผู้ใช้จาก context (current owner)
	currentOwnerID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// 2. ดึง conversation ID จาก parameter
	conversationID, err := utils.ParseUUIDParam(c, "conversationId")
	if err != nil {
		return err
	}

	// 3. รับข้อมูล new owner ID จาก request body
	var input struct {
		NewOwnerID string `json:"new_owner_id"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request data: " + err.Error(),
		})
	}

	// 4. Validate new owner ID
	if input.NewOwnerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "new_owner_id is required",
		})
	}

	newOwnerID, err := uuid.Parse(input.NewOwnerID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid new_owner_id format",
		})
	}

	// 5. เรียก service เพื่อโอนความเป็นเจ้าของ
	err = h.conversationService.TransferOwnership(conversationID, currentOwnerID, newOwnerID)
	if err != nil {
		// กำหนด status code ตามประเภทของ error
		statusCode := fiber.StatusInternalServerError
		errorMessage := err.Error()

		switch errorMessage {
		case "only the owner can transfer ownership":
			statusCode = fiber.StatusForbidden
		case "current owner is not a member of this conversation",
			"new owner is not a member of this conversation":
			statusCode = fiber.StatusNotFound
		case "cannot transfer ownership to yourself",
			"ownership transfer is only available for group conversations":
			statusCode = fiber.StatusBadRequest
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": errorMessage,
		})
	}

	// 6. ส่ง WebSocket notification แจ้งสมาชิกทุกคนในกลุ่ม
	h.notificationService.NotifyOwnershipTransferred(conversationID, currentOwnerID, newOwnerID)

	// บันทึก activity log
	h.groupActivityService.LogOwnershipTransferred(conversationID, currentOwnerID, newOwnerID)

	// 7. ส่งผลลัพธ์กลับ
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Ownership transferred successfully",
		"data": fiber.Map{
			"conversation_id":    conversationID.String(),
			"previous_owner_id": currentOwnerID.String(),
			"new_owner_id":      newOwnerID.String(),
		},
	})
}

// GetActivities ดึง activity log ของ conversation
// GET /conversations/:conversationId/activities
func (h *ConversationHandler) GetActivities(c *fiber.Ctx) error {
	// 1. ดึงข้อมูลผู้ใช้จาก context
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// 2. ดึง conversation ID จาก parameter
	conversationID, err := utils.ParseUUIDParam(c, "conversationId")
	if err != nil {
		return err
	}

	// 3. ดึง query parameters สำหรับ pagination และ filter
	limit := c.QueryInt("limit", 20)
	if limit > 100 {
		limit = 100 // จำกัดสูงสุดที่ 100
	}
	offset := c.QueryInt("offset", 0)
	activityType := c.Query("type", "") // ถ้าระบุ type จะ filter ตาม type นั้น

	// 4. เรียกใช้ service พร้อม type filter
	activities, total, err := h.groupActivityService.GetActivities(conversationID, userID, limit, offset, activityType)
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

	// 5. ส่งผลลัพธ์กลับ
	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"activities": activities,
			"pagination": fiber.Map{
				"total":  total,
				"limit":  limit,
				"offset": offset,
			},
		},
	})
}

// GetMessagesByDate ดึงข้อความตามวันที่กำหนด (Jump to Date)
func (h *ConversationHandler) GetMessagesByDate(c *fiber.Ctx) error {
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

	// รับ date จาก query parameter
	dateStr := c.Query("date")
	if dateStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "date query parameter is required (format: YYYY-MM-DD)",
		})
	}

	limit := c.QueryInt("limit", 50)
	if limit > 100 {
		limit = 100
	}

	// ดึงข้อความตามวันที่
	messages, total, hasMoreBefore, hasMoreAfter, err := h.messageService.GetMessagesByDate(conversationID, userID, dateStr, limit)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if err.Error() == "user is not a member of this conversation" {
			statusCode = fiber.StatusForbidden
		} else if err.Error() == "invalid date format, use YYYY-MM-DD" {
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
			"messages":        messages,
			"date":            dateStr,
			"total":           total,
			"has_more_before": hasMoreBefore,
			"has_more_after":  hasMoreAfter,
		},
	})
}

//for business conversation

