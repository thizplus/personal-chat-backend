// interfaces/api/routes/user_friendship_routes.go
package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/thizplus/gofiber-chat-api/interfaces/api/handler"
	"github.com/thizplus/gofiber-chat-api/interfaces/api/middleware"
)

// SetupUserFriendshipRoutes กำหนดเส้นทาง API สำหรับระบบเพื่อน
func SetupUserFriendshipRoutes(router fiber.Router, userFriendshipHandler *handler.UserFriendshipHandler) {
	// สร้างกลุ่มเส้นทางเพื่อน
	friends := router.Group("/friends")
	friends.Use(middleware.Protected())

	// ดึงรายชื่อเพื่อนทั้งหมด
	friends.Get("/", userFriendshipHandler.GetFriends) // [success] 4.1 การดึงรายชื่อเพื่อน [Y]

	// ค้นหาผู้ใช้เพื่อเพิ่มเป็นเพื่อน
	friends.Get("/search", userFriendshipHandler.SearchUsers) // [success] 4.2 การค้นหาผู้ใช้เพื่อเพิ่มเป็นเพื่อน [Y] [*]

	// ดึงคำขอเป็นเพื่อนที่รอการตอบรับ
	friends.Get("/pending", userFriendshipHandler.GetPendingRequests) // [success]  4.4 การดึงคำขอเป็นเพื่อนที่รอการตอบรับ [Y]

	// ดึงคำขอเป็นเพื่อนที่ส่งไป
	friends.Get("/sent", userFriendshipHandler.GetSentRequests) // การดึงคำขอเป็นเพื่อนที่ส่งไป

	// ดึงรายชื่อผู้ใช้ที่ถูกบล็อก
	friends.Get("/blocked", userFriendshipHandler.GetBlockedUsers) // [success]  4.10 การดูรายชื่อผู้ใช้ที่ถูกบล็อก [Y]

	// ดึงรายชื่อผู้ใช้ที่บล็อกเรา
	friends.Get("/blocked-by", userFriendshipHandler.GetBlockedByUsers) // การดูรายชื่อผู้ใช้ที่บล็อกเรา

	// ตรวจสอบสถานะการบล็อก
	friends.Get("/block-status/:userId", userFriendshipHandler.GetBlockStatus) // ตรวจสอบ block status กับผู้ใช้คนใดคนหนึ่ง

	// ส่งคำขอเป็นเพื่อน
	friends.Post("/request/:friendId", userFriendshipHandler.SendFriendRequest) // [success] 4.3 การส่งคำขอเป็นเพื่อน [Y]

	// ตอบรับคำขอเป็นเพื่อน
	friends.Put("/accept/:requestId", userFriendshipHandler.AcceptFriendRequest) // [success] 4.5 การตอบรับคำขอเป็นเพื่อน [Y]

	// ปฏิเสธคำขอเป็นเพื่อน
	friends.Put("/reject/:requestId", userFriendshipHandler.RejectFriendRequest) // [success] 4.6 การปฏิเสธคำขอเป็นเพื่อน [Y]

	// ยกเลิกคำขอเป็นเพื่อนที่ส่งไป
	friends.Delete("/request/:requestId", userFriendshipHandler.CancelFriendRequest) // การยกเลิกคำขอเป็นเพื่อนที่ส่งไป

	// ลบเพื่อน
	friends.Delete("/:friendId", userFriendshipHandler.RemoveFriend) // [success] 4.7 การลบเพื่อน [Y]

	// บล็อกผู้ใช้
	friends.Post("/block/:userId", userFriendshipHandler.BlockUser) // [success]  4.8 การบล็อกผู้ใช้ [Y]

	// เลิกบล็อกผู้ใช้
	friends.Delete("/block/:userId", userFriendshipHandler.UnblockUser) // [success] 4.9 การเลิกบล็อกผู้ใช้ [Y]
}
