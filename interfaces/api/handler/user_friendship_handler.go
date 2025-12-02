// interfaces/api/handler/user_friendship_handler.go
package handler

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/service"
	"github.com/thizplus/gofiber-chat-api/domain/types"
	"github.com/thizplus/gofiber-chat-api/interfaces/api/middleware"
	"github.com/thizplus/gofiber-chat-api/pkg/utils"
)

type UserFriendshipHandler struct {
	userFriendshipService     service.UserFriendshipService
	userService               service.UserService
	conversationMemberService service.ConversationMemberService
	notificationService       service.NotificationService
}

func NewUserFriendshipHandler(
	userFriendshipService service.UserFriendshipService,
	userService service.UserService,
	conversationMemberService service.ConversationMemberService,
	notificationService service.NotificationService,

) *UserFriendshipHandler {
	return &UserFriendshipHandler{
		userFriendshipService:     userFriendshipService,
		userService:               userService,
		conversationMemberService: conversationMemberService,
		notificationService:       notificationService,
	}
}

// GetFriends ดึงรายชื่อเพื่อนทั้งหมด
func (h *UserFriendshipHandler) GetFriends(c *fiber.Ctx) error {
	// ดึง User ID จาก token
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// ดึงรายชื่อเพื่อน
	friends, err := h.userFriendshipService.GetFriends(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to fetch friends",
			"error":   err.Error(),
		})
	}

	// สร้างข้อมูลสำหรับส่งกลับ
	var responseData []types.JSONB
	for _, friend := range friends {
		// ตรวจสอบว่ามีการสนทนาส่วนตัวกับเพื่อนคนนี้หรือไม่
		// หมายเหตุ: ในอนาคตควรย้ายไปอยู่ในส่วนของ ConversationService
		conversationID, err := h.conversationMemberService.FindDirectConversationBetweenUsers(userID, friend.ID)

		// ตรวจสอบสถานะความสัมพันธ์
		status, friendshipID, _ := h.userFriendshipService.GetFriendshipStatus(userID, friend.ID)

		friendData := types.JSONB{
			"id":                friend.ID.String(),
			"username":          friend.Username,
			"display_name":      friend.DisplayName,
			"profile_image_url": friend.ProfileImageURL,
			"bio":               friend.Bio,
			"status":            friend.Status,
			"last_active_at":    friend.LastActiveAt,
			"friendship_id":     friendshipID,
			"friendship_status": status,
		}

		// เพิ่ม conversation_id ถ้ามี
		if err == nil && conversationID != uuid.Nil {
			friendData["conversation_id"] = conversationID.String()
		}

		responseData = append(responseData, friendData)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responseData,
	})
}

// SearchUsers ค้นหาผู้ใช้
// ปรับปรุงเมธอด SearchUsers ใน user_friendship_handler.go
func (h *UserFriendshipHandler) SearchUsers(c *fiber.Ctx) error {
	// ดึง User ID จาก token
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Search query is required",
		})
	}

	// ดึงพารามิเตอร์ exact_match จาก query
	exactMatch := false
	if c.Query("exact_match") == "true" {
		exactMatch = true
	}

	// ค้นหาผู้ใช้
	var users []*models.User
	var searchErr error

	if exactMatch {
		// ค้นหาแบบตรงกับทั้งหมด
		users, _, searchErr = h.userService.SearchUsersExact(query, 20, 0)
	} else {
		// ค้นหาแบบเดิม (บางส่วน)
		users, _, searchErr = h.userService.SearchUsers(query, 20, 0)
	}

	if searchErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to search users",
			"error":   searchErr.Error(),
		})
	}

	// กรองตัวเองออก
	var filteredUsers []*models.User
	for _, user := range users {
		if user.ID != userID {
			filteredUsers = append(filteredUsers, user)
		}
	}

	// สร้างข้อมูลสำหรับส่งกลับ
	var responseData []types.JSONB
	for _, user := range filteredUsers {
		// ตรวจสอบความสัมพันธ์
		status, friendshipID, _ := h.userFriendshipService.GetFriendshipStatus(userID, user.ID)

		userData := types.JSONB{
			"id":                user.ID.String(),
			"username":          user.Username,
			"display_name":      user.DisplayName,
			"profile_image_url": user.ProfileImageURL,
			"bio":               user.Bio,
			"friendship_status": status,
			"friendship_id":     friendshipID,
		}

		responseData = append(responseData, userData)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responseData,
	})
}

// SendFriendRequest ส่งคำขอเป็นเพื่อน
func (h *UserFriendshipHandler) SendFriendRequest(c *fiber.Ctx) error {
	// ดึง User ID จาก token
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	friendID, err := utils.ParseUUIDParam(c, "friendId")
	if err != nil {
		return err // error response ถูกจัดการในฟังก์ชันแล้ว
	}

	// ส่งคำขอเป็นเพื่อน
	friendship, err := h.userFriendshipService.SendFriendRequest(userID, friendID)
	if err != nil {
		if strings.Contains(err.Error(), "friend request already exists") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Friend request already exists",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to send friend request",
			"error":   err.Error(),
		})
	}

	err = h.notificationService.NotifyFriendRequestReceived(friendship)
	if err != nil {
		// บันทึก log แต่ไม่ส่ง error กลับไป
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Friend request sent successfully",
		"data": types.JSONB{
			"id":           friendship.ID.String(),
			"user_id":      friendship.UserID.String(),
			"friend_id":    friendship.FriendID.String(),
			"status":       friendship.Status,
			"requested_at": friendship.RequestedAt,
			"updated_at":   friendship.UpdatedAt,
		},
	})
}

// GetPendingRequests ดึงคำขอเป็นเพื่อนที่รอการตอบรับ
func (h *UserFriendshipHandler) GetPendingRequests(c *fiber.Ctx) error {
	// ดึง User ID จาก token
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// ดึงคำขอเป็นเพื่อนที่รอการตอบรับ
	pendingRequests, err := h.userFriendshipService.GetPendingRequests(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to fetch pending requests",
			"error":   err.Error(),
		})
	}

	// สร้างข้อมูลสำหรับส่งกลับ
	var responseData []types.JSONB
	for _, request := range pendingRequests {
		// ดึงข้อมูลผู้ขอเป็นเพื่อน
		requester, err := h.userService.GetUserByID(request.UserID)
		if err != nil {
			continue
		}

		requestData := types.JSONB{
			"request_id":        request.ID.String(),
			"user_id":           requester.ID.String(),
			"username":          requester.Username,
			"display_name":      requester.DisplayName,
			"profile_image_url": requester.ProfileImageURL,
			"requested_at":      request.RequestedAt,
		}

		responseData = append(responseData, requestData)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responseData,
	})
}

// GetSentRequests ดึงคำขอเป็นเพื่อนที่ส่งไป
func (h *UserFriendshipHandler) GetSentRequests(c *fiber.Ctx) error {
	// ดึง User ID จาก token
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// ดึงคำขอเป็นเพื่อนที่ส่งไป
	sentRequests, err := h.userFriendshipService.GetSentRequests(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to fetch sent requests",
			"error":   err.Error(),
		})
	}

	// สร้างข้อมูลสำหรับส่งกลับ
	var responseData []types.JSONB
	for _, request := range sentRequests {
		// ดึงข้อมูลผู้รับคำขอ
		receiver, err := h.userService.GetUserByID(request.FriendID)
		if err != nil {
			continue
		}

		requestData := types.JSONB{
			"request_id":        request.ID.String(),
			"user_id":           receiver.ID.String(),
			"username":          receiver.Username,
			"display_name":      receiver.DisplayName,
			"profile_image_url": receiver.ProfileImageURL,
			"requested_at":      request.RequestedAt,
		}

		responseData = append(responseData, requestData)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responseData,
	})
}

// AcceptFriendRequest ยอมรับคำขอเป็นเพื่อน
func (h *UserFriendshipHandler) AcceptFriendRequest(c *fiber.Ctx) error {
	// ดึง User ID จาก token
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	requestID, err := utils.ParseUUIDParam(c, "requestId")
	if err != nil {
		return err // error response ถูกจัดการในฟังก์ชันแล้ว
	}

	// ยอมรับคำขอเป็นเพื่อน
	friendship, err := h.userFriendshipService.AcceptFriendRequest(requestID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "friend request not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"message": "Friend request not found or already processed",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to accept friend request",
			"error":   err.Error(),
		})
	}

	err = h.notificationService.NotifyFriendRequestAccepted(friendship)
	if err != nil {
		// บันทึก log แต่ไม่ส่ง error กลับไป
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Friend request accepted",
		"data": types.JSONB{
			"id":           friendship.ID.String(),
			"user_id":      friendship.UserID.String(),
			"friend_id":    friendship.FriendID.String(),
			"status":       friendship.Status,
			"requested_at": friendship.RequestedAt,
			"updated_at":   friendship.UpdatedAt,
		},
	})
}

// RejectFriendRequest ปฏิเสธคำขอเป็นเพื่อน
func (h *UserFriendshipHandler) RejectFriendRequest(c *fiber.Ctx) error {
	// ดึง User ID จาก token
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	requestID, err := utils.ParseUUIDParam(c, "requestId")
	if err != nil {
		return err // error response ถูกจัดการในฟังก์ชันแล้ว
	}

	// ปฏิเสธคำขอเป็นเพื่อน
	friendship, err := h.userFriendshipService.RejectFriendRequest(requestID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "friend request not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"message": "Friend request not found or already processed",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to reject friend request",
			"error":   err.Error(),
		})
	}

	// ส่ง WebSocket notification
	err = h.notificationService.NotifyFriendRequestRejected(friendship)
	if err != nil {
		log.Printf("Failed to send friend request rejected notification: %v", err)
		// ไม่ return error เพราะการส่ง notification ล้มเหลวไม่ควรทำให้ API ล้มเหลว
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Friend request rejected",
		"data": types.JSONB{
			"id":           friendship.ID.String(),
			"user_id":      friendship.UserID.String(),
			"friend_id":    friendship.FriendID.String(),
			"status":       friendship.Status,
			"requested_at": friendship.RequestedAt,
			"updated_at":   friendship.UpdatedAt,
		},
	})
}

// CancelFriendRequest ยกเลิกคำขอเป็นเพื่อนที่ส่งไป
func (h *UserFriendshipHandler) CancelFriendRequest(c *fiber.Ctx) error {
	// ดึง User ID จาก token
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	requestID, err := utils.ParseUUIDParam(c, "requestId")
	if err != nil {
		return err // error response ถูกจัดการในฟังก์ชันแล้ว
	}

	// ยกเลิกคำขอเป็นเพื่อน
	err = h.userFriendshipService.CancelFriendRequest(requestID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "friend request not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"message": "Friend request not found",
			})
		}
		if strings.Contains(err.Error(), "you can only cancel your own friend requests") {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"message": "You can only cancel your own friend requests",
			})
		}
		if strings.Contains(err.Error(), "can only cancel pending friend requests") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Can only cancel pending friend requests",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to cancel friend request",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Friend request cancelled successfully",
	})
}

// RemoveFriend ลบเพื่อน
func (h *UserFriendshipHandler) RemoveFriend(c *fiber.Ctx) error {
	// ดึง User ID จาก token
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	friendID, err := utils.ParseUUIDParam(c, "friendId")
	if err != nil {
		return err // error response ถูกจัดการในฟังก์ชันแล้ว
	}

	// ลบเพื่อน
	err = h.userFriendshipService.RemoveFriend(userID, friendID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to remove friend",
			"error":   err.Error(),
		})
	}

	// ส่ง WebSocket notification แจ้งทั้งสองฝ่าย
	h.notificationService.NotifyFriendRemoved(userID, friendID)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Friend removed successfully",
	})
}

// BlockUser บล็อกผู้ใช้
func (h *UserFriendshipHandler) BlockUser(c *fiber.Ctx) error {
	// ดึง User ID จาก token
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	targetID, err := utils.ParseUUIDParam(c, "userId")
	if err != nil {
		return err // error response ถูกจัดการในฟังก์ชันแล้ว
	}

	// บล็อกผู้ใช้
	err = h.userFriendshipService.BlockUser(userID, targetID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to block user",
			"error":   err.Error(),
		})
	}

	// ส่ง WebSocket notification แจ้งผู้ถูกบล็อก
	h.notificationService.NotifyUserBlocked(userID, targetID)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User blocked successfully",
	})
}

// UnblockUser เลิกบล็อกผู้ใช้
func (h *UserFriendshipHandler) UnblockUser(c *fiber.Ctx) error {
	// ดึง User ID จาก token
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	targetID, err := utils.ParseUUIDParam(c, "userId")
	if err != nil {
		return err // error response ถูกจัดการในฟังก์ชันแล้ว
	}

	// เลิกบล็อกผู้ใช้
	err = h.userFriendshipService.UnblockUser(userID, targetID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to unblock user",
			"error":   err.Error(),
		})
	}

	// ส่ง WebSocket notification แจ้งผู้ถูกปลดบล็อก
	h.notificationService.NotifyUserUnblocked(userID, targetID)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User unblocked successfully",
	})
}

// GetBlockedUsers ดึงรายชื่อผู้ใช้ที่ถูกบล็อก
func (h *UserFriendshipHandler) GetBlockedUsers(c *fiber.Ctx) error {
	// ดึง User ID จาก token
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// ดึงรายชื่อผู้ใช้ที่ถูกบล็อก
	blockedUsers, err := h.userFriendshipService.GetBlockedUsers(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to fetch blocked users",
			"error":   err.Error(),
		})
	}

	// สร้างข้อมูลสำหรับส่งกลับ
	var responseData []types.JSONB
	for _, user := range blockedUsers {
		userData := types.JSONB{
			"id":                user.ID.String(),
			"username":          user.Username,
			"display_name":      user.DisplayName,
			"profile_image_url": user.ProfileImageURL,
		}

		responseData = append(responseData, userData)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responseData,
	})
}

// GetBlockedByUsers ดึงรายชื่อผู้ใช้ที่บล็อกเรา
func (h *UserFriendshipHandler) GetBlockedByUsers(c *fiber.Ctx) error {
	// ดึง User ID จาก token
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// ดึงรายชื่อผู้ใช้ที่บล็อกเรา
	blockedByUsers, err := h.userFriendshipService.GetBlockedByUsers(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to fetch users who blocked you",
			"error":   err.Error(),
		})
	}

	// สร้างข้อมูลสำหรับส่งกลับ
	var responseData []types.JSONB
	for _, user := range blockedByUsers {
		userData := types.JSONB{
			"id":                user.ID.String(),
			"username":          user.Username,
			"display_name":      user.DisplayName,
			"profile_image_url": user.ProfileImageURL,
		}

		responseData = append(responseData, userData)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "รายการผู้ใช้ที่บล็อกคุณ",
		"data":    responseData,
	})
}

// GetBlockStatus ตรวจสอบสถานะการบล็อกกับผู้ใช้คนใดคนหนึ่ง
func (h *UserFriendshipHandler) GetBlockStatus(c *fiber.Ctx) error {
	// ดึง User ID จาก token
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// ดึง target user ID จาก URL parameter
	targetUserID, err := utils.ParseUUIDParam(c, "userId")
	if err != nil {
		return err // error response ถูกจัดการในฟังก์ชันแล้ว
	}

	// ตรวจสอบ block status
	isBlocked, isBlockedBy, err := h.userFriendshipService.CheckBlockStatus(userID, targetUserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to check block status",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": types.JSONB{
			"user_id":       targetUserID.String(),
			"is_blocked":    isBlocked,    // เราบล็อคคนนี้หรือไม่
			"is_blocked_by": isBlockedBy,  // คนนี้บล็อคเราหรือไม่
		},
	})
}
