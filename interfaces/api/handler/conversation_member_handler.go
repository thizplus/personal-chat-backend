// interfaces/api/handler/conversation_member_handler.go
package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/dto"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/service"
)

// ConversationMemberHandler จัดการคำขอสำหรับการจัดการสมาชิกในการสนทนา
type ConversationMemberHandler struct {
	memberService        service.ConversationMemberService
	notificationService  service.NotificationService
	groupActivityService service.GroupActivityService
}

// NewConversationMemberHandler สร้าง handler ใหม่
func NewConversationMemberHandler(
	memberService service.ConversationMemberService,
	notificationService service.NotificationService,
	groupActivityService service.GroupActivityService,
) *ConversationMemberHandler {
	return &ConversationMemberHandler{
		memberService:        memberService,
		notificationService:  notificationService,
		groupActivityService: groupActivityService,
	}
}

// AddConversationMember เพิ่มสมาชิกในการสนทนา
func (h *ConversationMemberHandler) AddConversationMember(c *fiber.Ctx) error {
	// 1. ดึงข้อมูลผู้ใช้จาก context
	userID, err := uuid.Parse(c.Locals("userID").(string))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// 2. ดึง conversation ID จาก parameter
	conversationIDStr := c.Params("conversationId")
	if conversationIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Conversation ID is required",
		})
	}

	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid conversation ID",
		})
	}

	// 3. รับข้อมูลผู้ใช้ที่ต้องการเพิ่ม
	var input dto.AddMemberRequest
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request data: " + err.Error(),
		})
	}

	// 4. ตรวจสอบ user ID
	if input.UserID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "User ID is required",
		})
	}

	newMemberID, err := uuid.Parse(input.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid user ID",
		})
	}

	// 5. เรียกใช้ service
	memberDTO, err := h.memberService.AddMember(userID, conversationID, newMemberID)
	if err != nil {
		// จัดการรหัสสถานะตามข้อผิดพลาด
		statusCode := fiber.StatusInternalServerError
		switch err.Error() {
		case "user is already a member of this conversation":
			statusCode = fiber.StatusConflict
		case "user to add not found":
			statusCode = fiber.StatusNotFound
		case "only admins can add members", "you are not a member of this conversation":
			statusCode = fiber.StatusForbidden
		case "cannot add members to direct conversation":
			statusCode = fiber.StatusBadRequest
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	// ส่ง WebSocket notification แจ้งว่ามีสมาชิกใหม่ถูกเพิ่มเข้ากลุ่ม
	h.notificationService.NotifyUserAddedToConversation(conversationID, newMemberID)

	// บันทึก activity log
	h.groupActivityService.LogMemberAdded(conversationID, userID, newMemberID)

	// 6. ส่งผลลัพธ์กลับ
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Member added successfully",
		"data":    memberDTO,
	})
}

// BulkAddConversationMembers เพิ่มสมาชิกหลายคนในการสนทนา
func (h *ConversationMemberHandler) BulkAddConversationMembers(c *fiber.Ctx) error {
	// 1. ดึงข้อมูลผู้ใช้จาก context
	userID, err := uuid.Parse(c.Locals("userID").(string))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// 2. ดึง conversation ID จาก parameter
	conversationIDStr := c.Params("conversationId")
	if conversationIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Conversation ID is required",
		})
	}

	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid conversation ID",
		})
	}

	// 3. รับข้อมูลผู้ใช้ที่ต้องการเพิ่ม
	var input dto.BulkAddMembersRequest
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request data: " + err.Error(),
		})
	}

	// 4. ตรวจสอบ user IDs
	if len(input.UserIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "At least one user ID is required",
		})
	}

	// แปลง string IDs เป็น UUID
	var memberIDs []uuid.UUID
	for _, userIDStr := range input.UserIDs {
		memberID, err := uuid.Parse(userIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid user ID format: " + userIDStr,
			})
		}
		memberIDs = append(memberIDs, memberID)
	}

	// 5. เรียกใช้ service
	addedMembers, failedMembers, err := h.memberService.BulkAddMembers(userID, conversationID, memberIDs)
	if err != nil {
		// จัดการรหัสสถานะตามข้อผิดพลาด
		statusCode := fiber.StatusInternalServerError
		switch err.Error() {
		case "only admins can add members", "you are not a member of this conversation":
			statusCode = fiber.StatusForbidden
		case "cannot add members to direct conversation":
			statusCode = fiber.StatusBadRequest
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	// 6. สร้าง response
	failedResponse := []fiber.Map{}
	for _, failed := range failedMembers {
		failedResponse = append(failedResponse, fiber.Map{
			"user_id": failed.UserID.String(),
			"reason":  failed.Reason,
		})
	}

	message := "Members added successfully"
	if len(addedMembers) == 0 {
		message = "No members were added"
	} else if len(failedMembers) > 0 {
		message = "Some members were added successfully, some failed"
	}

	// ส่ง WebSocket notification สำหรับแต่ละคนที่ถูกเพิ่มสำเร็จ
	for _, member := range addedMembers {
		memberUUID, _ := uuid.Parse(member.UserID)
		h.notificationService.NotifyUserAddedToConversation(conversationID, memberUUID)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": message,
		"data": fiber.Map{
			"added_members":  addedMembers,
			"failed_members": failedResponse,
			"total_added":    len(addedMembers),
			"total_failed":   len(failedMembers),
		},
	})
}

// GetConversationMembers ดึงรายการสมาชิกในการสนทนา
func (h *ConversationMemberHandler) GetConversationMembers(c *fiber.Ctx) error {
	// 1. ดึงข้อมูลผู้ใช้จาก context
	userID, err := uuid.Parse(c.Locals("userID").(string))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// 2. ดึง conversation ID จาก parameter
	conversationIDStr := c.Params("conversationId")
	if conversationIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Conversation ID is required",
		})
	}

	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid conversation ID",
		})
	}

	// 3. ดึงค่า query parameters
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	// 4. เรียกใช้ service
	members, total, err := h.memberService.GetMembers(userID, conversationID, page, limit)
	if err != nil {
		// จัดการรหัสสถานะตามข้อผิดพลาด
		statusCode := fiber.StatusInternalServerError
		if err.Error() == "you are not a member of this conversation" {
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
		"message": "Members retrieved successfully",
		"data": fiber.Map{
			"members":     members,
			"total":       total,
			"page":        page,
			"limit":       limit,
			"total_pages": (total + limit - 1) / limit,
		},
	})
}

// RemoveConversationMember ลบสมาชิกออกจากการสนทนา
func (h *ConversationMemberHandler) RemoveConversationMember(c *fiber.Ctx) error {
	// 1. ดึงข้อมูลผู้ใช้จาก context
	userID, err := uuid.Parse(c.Locals("userID").(string))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// 2. ดึง conversation ID จาก parameter
	conversationIDStr := c.Params("conversationId")
	if conversationIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Conversation ID is required",
		})
	}

	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid conversation ID",
		})
	}

	// 3. ดึง user ID ที่ต้องการลบ
	targetUserIDStr := c.Params("userId")
	if targetUserIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "User ID is required",
		})
	}

	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid user ID",
		})
	}

	// 4. เรียกใช้ service
	err = h.memberService.RemoveMember(userID, conversationID, targetUserID)
	if err != nil {
		// จัดการรหัสสถานะตามข้อผิดพลาด
		statusCode := fiber.StatusInternalServerError
		switch err.Error() {
		case "user is not a member of this conversation":
			statusCode = fiber.StatusNotFound
		case "only admins can remove other members", "you are not a member of this conversation":
			statusCode = fiber.StatusForbidden
		case "cannot remove members from direct conversation", "cannot remove the last admin from the conversation":
			statusCode = fiber.StatusBadRequest
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	// ส่ง WebSocket notification แจ้งว่าสมาชิกถูกลบออกจากกลุ่ม
	h.notificationService.NotifyUserRemovedFromConversation(targetUserID, conversationID)

	// บันทึก activity log
	h.groupActivityService.LogMemberRemoved(conversationID, userID, targetUserID)

	// 5. ส่งผลลัพธ์กลับ
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Member removed successfully",
	})
}

// ToggleMemberAdmin เปลี่ยนสถานะแอดมินของสมาชิก
func (h *ConversationMemberHandler) ToggleMemberAdmin(c *fiber.Ctx) error {
	// 1. ดึงข้อมูลผู้ใช้จาก context
	userID, err := uuid.Parse(c.Locals("userID").(string))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// 2. ดึง conversation ID จาก parameter
	conversationIDStr := c.Params("conversationId")
	if conversationIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Conversation ID is required",
		})
	}

	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid conversation ID",
		})
	}

	// 3. ดึง user ID ที่ต้องการเปลี่ยนสถานะ
	targetUserIDStr := c.Params("userId")
	if targetUserIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "User ID is required",
		})
	}

	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid user ID",
		})
	}

	// 4. รับข้อมูลการเปลี่ยนแปลงสถานะ
	var input dto.ToggleAdminRequest
	if err := c.BodyParser(&input); err != nil {
		// ใช้ค่าเริ่มต้นคือ true ถ้าไม่มีข้อมูลส่งมา (toggle)
		input.IsAdmin = true
	}

	// 5. เรียกใช้ service
	isAdmin, err := h.memberService.ToggleAdminStatus(userID, conversationID, targetUserID, input.IsAdmin)
	if err != nil {
		// จัดการรหัสสถานะตามข้อผิดพลาด
		statusCode := fiber.StatusInternalServerError
		switch err.Error() {
		case "user is not a member of this conversation":
			statusCode = fiber.StatusNotFound
		case "only admins can change admin status":
			statusCode = fiber.StatusForbidden
		case "cannot change admin status in direct conversation", "cannot remove admin status from the last admin":
			statusCode = fiber.StatusBadRequest
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	// 6. ส่งผลลัพธ์กลับ
	return c.JSON(fiber.Map{
		"success":  true,
		"message":  "Admin status updated successfully",
		"is_admin": isAdmin,
	})
}

// ChangeRole เปลี่ยน role ของสมาชิก (owner, admin, member)
func (h *ConversationMemberHandler) ChangeRole(c *fiber.Ctx) error {
	// 1. ดึงข้อมูลผู้ใช้จาก context
	userID, err := uuid.Parse(c.Locals("userID").(string))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized",
		})
	}

	// 2. ดึง conversation ID และ target user ID
	conversationID, err := uuid.Parse(c.Params("conversationId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid conversation ID",
		})
	}

	targetUserID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid user ID",
		})
	}

	// 3. รับข้อมูล role ใหม่
	var input struct {
		Role string `json:"role"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	// 4. Validate role
	validRoles := []string{"owner", "admin", "member"}
	isValidRole := false
	for _, validRole := range validRoles {
		if input.Role == validRole {
			isValidRole = true
			break
		}
	}
	if !isValidRole {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid role. Valid roles: owner, admin, member",
		})
	}

	// 5. ตรวจสอบสิทธิ์ของผู้ที่ต้องการเปลี่ยน role
	hasPermission, err := h.memberService.HasPermission(conversationID, userID, service.PermissionChangeRole)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to check permissions: " + err.Error(),
		})
	}
	if !hasPermission {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": "Only the owner can change member roles",
		})
	}

	// 6. ดึงข้อมูลสมาชิกเป้าหมาย
	targetMember, err := h.memberService.GetMember(conversationID, targetUserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Target user is not a member of this conversation",
		})
	}

	// 7. ไม่สามารถเปลี่ยน role ของ owner ได้
	if targetMember.Role == "owner" && input.Role != "owner" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": "Cannot change the owner's role. Transfer ownership first.",
		})
	}

	// 8. เก็บ old role สำหรับ notification
	oldRole := string(targetMember.Role)

	// 9. เปลี่ยน role
	newRole := models.MemberRole(input.Role)
	updatedMember, err := h.memberService.ChangeRole(conversationID, targetUserID, newRole)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to change member role: " + err.Error(),
		})
	}

	// 10. ส่ง WebSocket notification
	h.notificationService.NotifyMemberRoleChanged(conversationID, targetUserID, oldRole, string(newRole), userID)

	// บันทึก activity log
	h.groupActivityService.LogMemberRoleChanged(conversationID, userID, targetUserID, oldRole, string(newRole))

	// 11. ส่งผลลัพธ์กลับ
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Member role updated successfully",
		"data": fiber.Map{
			"conversation_id": conversationID,
			"user_id":         targetUserID,
			"old_role":        oldRole,
			"new_role":        updatedMember.Role,
		},
	})
}
