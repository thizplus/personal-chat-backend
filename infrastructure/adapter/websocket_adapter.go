// infrastructure/adapter/websocket_adapter.go
package adapter

import (
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/port"
	"github.com/thizplus/gofiber-chat-api/interfaces/websocket"
	"github.com/thizplus/gofiber-chat-api/pkg/utils"
)

// WebSocketAdapter ใช้งานร่วมกับ WebSocketHub และ implements interface WebSocketPort
type WebSocketAdapter struct {
	hub *websocket.Hub
}

// NewWebSocketAdapter สร้าง WebSocketAdapter ตัวใหม่
func NewWebSocketAdapter(hub *websocket.Hub) port.WebSocketPort {
	return &WebSocketAdapter{
		hub: hub,
	}
}

// BroadcastToConversation ส่งข้อความไปยังสมาชิกทั้งหมดในบทสนทนา
func (a *WebSocketAdapter) BroadcastToConversation(conversationID uuid.UUID, messageType string, data interface{}) {

	a.hub.BroadcastToConversation(conversationID, websocket.MessageType(messageType), data)
}

// BroadcastToUser ส่งข้อความไปยังผู้ใช้คนใดคนหนึ่ง
func (a *WebSocketAdapter) BroadcastToUser(userID uuid.UUID, messageType string, data interface{}) {
	a.hub.BroadcastToUser(userID, websocket.MessageType(messageType), data)
}

// ใน websocket_adapter.go
func (a *WebSocketAdapter) BroadcastToUsers(userIDs []uuid.UUID, messageType string, data interface{}) error {
	a.hub.BroadcastToUsers(userIDs, websocket.MessageType(messageType), data)
	return nil // คืนค่า nil เสมอ หรืออาจมีการตรวจสอบความผิดพลาดแล้วคืนค่า error
}

// BroadcastToBusiness ส่งข้อความไปยังธุรกิจหนึ่ง
func (a *WebSocketAdapter) BroadcastToBusiness(businessID uuid.UUID, messageType string, data interface{}) {
	a.hub.BroadcastToBusiness(businessID, websocket.MessageType(messageType), data)
}

// เมธอดพิเศษสำหรับส่งข้อความตามประเภท

// BroadcastNewMessage ส่งการแจ้งเตือนว่ามีข้อความใหม่
func (a *WebSocketAdapter) BroadcastNewMessage(conversationID uuid.UUID, message interface{}) {

	a.BroadcastToConversation(conversationID, "message.receive", message)
}

// BroadcastMessageRead ส่งการแจ้งเตือนว่าข้อความถูกอ่าน (เก่า - ไม่แนะนำให้ใช้ใน group chat)
func (a *WebSocketAdapter) BroadcastMessageRead(conversationID uuid.UUID, message interface{}) {
	a.BroadcastToConversation(conversationID, "message.read", message)
}

// BroadcastMessageReadAll ส่งการแจ้งเตือนว่าข้อความทั้งหมดถูกอ่าน (เก่า - ไม่แนะนำให้ใช้)
func (a *WebSocketAdapter) BroadcastMessageReadAll(conversationID uuid.UUID, message interface{}) {
	a.BroadcastToConversation(conversationID, "message.read_all", message)
}

// SendMessageReadToSender ส่ง message.read event ไปยังผู้ส่งข้อความเท่านั้น (สำหรับ group chat)
func (a *WebSocketAdapter) SendMessageReadToSender(senderID uuid.UUID, message interface{}) {
	a.BroadcastToUser(senderID, "message.read", message)
}

// SendMessageReadAllToUser ส่ง message.read_all event ไปยัง user ที่อ่าน (สำหรับ multi-device sync)
func (a *WebSocketAdapter) SendMessageReadAllToUser(userID uuid.UUID, message interface{}) {
	a.BroadcastToUser(userID, "message.read_all", message)
}

// BroadcastMessageDelivered ส่งการแจ้งเตือนว่าข้อความถูกส่งสำเร็จ
func (a *WebSocketAdapter) BroadcastMessageDelivered(conversationID uuid.UUID, message interface{}) {
	a.BroadcastToConversation(conversationID, "message.delivered", message)
}

// BroadcastMessageEdited ส่งการแจ้งเตือนว่าข้อความถูกแก้ไข
func (a *WebSocketAdapter) BroadcastMessageEdited(conversationID uuid.UUID, message interface{}) {
	a.BroadcastToConversation(conversationID, "message.updated", message)
}

// BroadcastMessageReply ส่งการแจ้งเตือนว่าข้อความถูกตอบกลับ
func (a *WebSocketAdapter) BroadcastMessageReply(conversationID uuid.UUID, message interface{}) {
	a.BroadcastToConversation(conversationID, "message.reply", message)
}

// BroadcastMessageDeleted ส่งการแจ้งเตือนว่าข้อความถูกลบ
func (a *WebSocketAdapter) BroadcastMessageDeleted(conversationID uuid.UUID, messageID uuid.UUID) {
	data := map[string]interface{}{
		"message_id": messageID,
		"deleted_at": utils.Now(),
	}
	a.BroadcastToConversation(conversationID, "message.delete", data)
}

// BroadcastMessageReaction ส่งการแจ้งเตือนว่าเกิดการแสดงปฏิกิริยาในข้อความ
func (a *WebSocketAdapter) BroadcastMessageReaction(conversationID uuid.UUID, reaction interface{}) {
	a.BroadcastToConversation(conversationID, "message.reaction", reaction)
}

// BroadcastConversationCreated ส่งการแจ้งเตือนว่ามีการสร้างบทสนทนาใหม่
func (a *WebSocketAdapter) BroadcastConversationCreated(userIDs []uuid.UUID, conversation interface{}) error {
	return a.BroadcastToUsers(userIDs, "conversation.create", conversation)
}

// BroadcastConversationUpdated ส่งการแจ้งเตือนว่ามีการอัปเดตบทสนทนา
func (a *WebSocketAdapter) BroadcastConversationUpdated(conversationID uuid.UUID, update interface{}) {
	a.BroadcastToConversation(conversationID, "conversation.update", update)
}

// BroadcastConversationDeleted ส่งการแจ้งเตือนว่าบทสนทนาถูกลบ
func (a *WebSocketAdapter) BroadcastConversationDeleted(conversationID uuid.UUID, memberIDs []uuid.UUID) {
	data := map[string]interface{}{
		"conversation_id": conversationID,
		"deleted_at":      utils.Now(),
	}
	a.BroadcastToUsers(memberIDs, "conversation.deleted", data)
}

// BroadcastUserAddedToConversation ส่งการแจ้งเตือนว่ามีผู้ใช้ถูกเพิ่มในบทสนทนา
func (a *WebSocketAdapter) BroadcastUserAddedToConversation(conversationID uuid.UUID, userID uuid.UUID) {
	// แจ้งสมาชิกในบทสนทนา
	data := map[string]interface{}{
		"conversation_id": conversationID,
		"user_id":         userID,
		"added_at":        utils.Now(),
	}
	a.BroadcastToConversation(conversationID, "conversation.user_added", data)

	// แจ้งผู้ใช้ที่ถูกเพิ่ม
	userNotification := map[string]interface{}{
		"conversation_id": conversationID,
		"message":         "คุณถูกเพิ่มในบทสนทนา",
	}
	a.BroadcastToUser(userID, "conversation.create", userNotification)
}

// BroadcastUserRemovedFromConversation ส่งการแจ้งเตือนว่าผู้ใช้ถูกลบออกจากบทสนทนา
func (a *WebSocketAdapter) BroadcastUserRemovedFromConversation(userID uuid.UUID, conversationID uuid.UUID) {
	data := map[string]interface{}{
		"conversation_id": conversationID,
		"user_id":         userID,
		"removed_at":      utils.Now(),
	}

	// 1. แจ้งสมาชิกอื่นๆในห้องว่ามีคนถูกลบออก (คนที่ยังอยู่ในห้องจะได้รับ)
	a.BroadcastToConversation(conversationID, "conversation.user_removed", data)

	// 2. แจ้งผู้ใช้ที่ถูกลบออกด้วย (เผื่อยัง subscribe อยู่)
	a.BroadcastToUser(userID, "conversation.user_removed", data)
}

// BroadcastNewConversation ส่งการแจ้งเตือนบทสนทนาใหม่
func (a *WebSocketAdapter) BroadcastNewConversation(userID uuid.UUID, conversation interface{}) error {
	a.BroadcastToUser(userID, "conversation.create", conversation)
	return nil
}

// BroadcastBusinessBroadcast ส่งการแจ้งเตือนประกาศจากธุรกิจ
func (a *WebSocketAdapter) BroadcastBusinessBroadcast(userIDs []uuid.UUID, broadcast interface{}) {
	a.BroadcastToUsers(userIDs, "business.broadcast", broadcast)
}

// BroadcastBusinessNewFollower ส่งการแจ้งเตือนว่ามีผู้ติดตามธุรกิจใหม่
func (a *WebSocketAdapter) BroadcastBusinessNewFollower(businessID uuid.UUID, followerID uuid.UUID) {
	data := map[string]interface{}{
		"follower_id": followerID,
		"timestamp":   utils.Now(),
	}
	a.BroadcastToBusiness(businessID, "business.new_follower", data)
}

// BroadcastBusinessWelcomeMessage ส่งข้อความต้อนรับจากธุรกิจ
func (a *WebSocketAdapter) BroadcastBusinessWelcomeMessage(userID uuid.UUID, businessID uuid.UUID, message interface{}) {
	data := map[string]interface{}{
		"business_id": businessID,
		"message":     message,
		"timestamp":   utils.Now(),
	}
	a.BroadcastToUser(userID, "business.welcome", data)
}

// BroadcastBusinessFollowStatusChanged ส่งการแจ้งเตือนเมื่อสถานะการติดตามธุรกิจเปลี่ยน
func (a *WebSocketAdapter) BroadcastBusinessFollowStatusChanged(businessID uuid.UUID, userID uuid.UUID, isFollowing bool) {
	status := "unfollowed"
	if isFollowing {
		status = "followed"
	}

	// แจ้งธุรกิจ
	businessData := map[string]interface{}{
		"user_id":      userID,
		"business_id":  businessID,
		"status":       status,
		"is_following": isFollowing,
		"timestamp":    utils.Now(),
	}
	a.BroadcastToBusiness(businessID, "business.follow_status_changed", businessData)

	// แจ้งผู้ใช้
	userData := map[string]interface{}{
		"business_id":  businessID,
		"status":       status,
		"is_following": isFollowing,
		"timestamp":    utils.Now(),
	}
	a.BroadcastToUser(userID, "user.follow_status_changed", userData)
}

// BroadcastBusinessStatusChanged ส่งการแจ้งเตือนเมื่อสถานะธุรกิจเปลี่ยน
func (a *WebSocketAdapter) BroadcastBusinessStatusChanged(businessID uuid.UUID, status string) {
	data := map[string]interface{}{
		"business_id": businessID,
		"status":      status,
		"timestamp":   utils.Now(),
	}
	a.BroadcastToBusiness(businessID, "business.status", data)
}

// BroadcastFriendRequestReceived ส่งการแจ้งเตือนว่าได้รับคำขอเป็นเพื่อน
func (a *WebSocketAdapter) BroadcastFriendRequestReceived(userID uuid.UUID, request interface{}) error {
	a.BroadcastToUser(userID, "friend_request.received", request)
	return nil
}

// BroadcastFriendRequestAccepted ส่งการแจ้งเตือนว่าคำขอเป็นเพื่อนถูกยอมรับ
func (a *WebSocketAdapter) BroadcastFriendRequestAccepted(userID uuid.UUID, friendship interface{}) error {
	a.BroadcastToUser(userID, "friend_request.accepted", friendship)
	return nil
}

// BroadcastFriendRequestRejected ส่งการแจ้งเตือนว่าคำขอเป็นเพื่อนถูกปฏิเสธ
func (a *WebSocketAdapter) BroadcastFriendRequestRejected(userID uuid.UUID, friendship interface{}) error {
	a.BroadcastToUser(userID, "friend_request.rejected", friendship)
	return nil
}

// BroadcastFriendRemoved ส่งการแจ้งเตือนว่าเพื่อนถูกลบ
func (a *WebSocketAdapter) BroadcastFriendRemoved(userID uuid.UUID, friendID uuid.UUID) {
	now := time.Now()

	// ส่งถึง friendID ให้รู้ว่า userID ลบเขา
	dataForFriend := map[string]interface{}{
		"user_id":    userID.String(),
		"removed_at": now.Format(time.RFC3339),
	}
	a.BroadcastToUser(friendID, "friend.removed", dataForFriend)

	// ส่งถึง userID ให้รู้ว่าเขาลบ friendID
	dataForUser := map[string]interface{}{
		"user_id":    friendID.String(),
		"removed_at": now.Format(time.RFC3339),
	}
	a.BroadcastToUser(userID, "friend.removed", dataForUser)
}

// BroadcastUserBlocked ส่งการแจ้งเตือนว่าผู้ใช้ถูกบล็อก
func (a *WebSocketAdapter) BroadcastUserBlocked(blockerID uuid.UUID, blockedID uuid.UUID) {
	now := utils.Now()

	// ส่ง user.blocked event ไปยัง blocker (คนที่ทำการ block)
	blockerData := map[string]interface{}{
		"blocker_id":      blockerID.String(),
		"blocked_user_id": blockedID.String(),
		"blocked_at":      now,
	}
	a.BroadcastToUser(blockerID, "user.blocked", blockerData)

	// ส่ง user.blocked_by event ไปยัง blocked user (คนที่ถูก block)
	blockedData := map[string]interface{}{
		"blocker_id":      blockerID.String(),
		"blocked_user_id": blockedID.String(),
		"blocked_at":      now,
	}
	a.BroadcastToUser(blockedID, "user.blocked_by", blockedData)
}

// BroadcastUserUnblocked ส่งการแจ้งเตือนว่าผู้ใช้ถูกปลดบล็อก
func (a *WebSocketAdapter) BroadcastUserUnblocked(unblockerID uuid.UUID, unblockedID uuid.UUID) {
	now := utils.Now()

	// ส่ง user.unblocked event ไปยัง unblocker (คนที่ทำการ unblock)
	unblockerData := map[string]interface{}{
		"unblocker_id":      unblockerID.String(),
		"unblocked_user_id": unblockedID.String(),
		"unblocked_at":      now,
	}
	a.BroadcastToUser(unblockerID, "user.unblocked", unblockerData)

	// ส่ง user.unblocked_by event ไปยัง unblocked user (คนที่ถูก unblock)
	unblockedData := map[string]interface{}{
		"unblocker_id":      unblockerID.String(),
		"unblocked_user_id": unblockedID.String(),
		"unblocked_at":      now,
	}
	a.BroadcastToUser(unblockedID, "user.unblocked_by", unblockedData)
}

// BroadcastNotification ส่งการแจ้งเตือนทั่วไป
func (a *WebSocketAdapter) BroadcastNotification(userIDs []uuid.UUID, notification interface{}) {
	a.BroadcastToUsers(userIDs, "notification", notification)
}

// BroadcastAlert ส่งการแจ้งเตือนแบบเร่งด่วน
func (a *WebSocketAdapter) BroadcastAlert(userID uuid.UUID, alert interface{}) {
	a.BroadcastToUser(userID, "alert", alert)
}

// BroadcastSystemMessage ส่งข้อความจากระบบ
func (a *WebSocketAdapter) BroadcastSystemMessage(userIDs []uuid.UUID, message interface{}) {
	a.BroadcastToUsers(userIDs, "system.message", message)
}

// =========== Member Role Notifications ===========

// BroadcastMemberRoleChanged ส่งการแจ้งเตือนการเปลี่ยน role ของสมาชิก
func (a *WebSocketAdapter) BroadcastMemberRoleChanged(conversationID uuid.UUID, data interface{}) {
	a.BroadcastToConversation(conversationID, "conversation.member_role_changed", data)
}

// BroadcastOwnershipTransferred ส่งการแจ้งเตือนการโอนความเป็นเจ้าของ
func (a *WebSocketAdapter) BroadcastOwnershipTransferred(conversationID uuid.UUID, data interface{}) {
	a.BroadcastToConversation(conversationID, "conversation.ownership_transferred", data)
}

// BroadcastNewActivity ส่งการแจ้งเตือน activity ใหม่ในกลุ่ม
func (a *WebSocketAdapter) BroadcastNewActivity(conversationID uuid.UUID, activity interface{}) {
	a.BroadcastToConversation(conversationID, "conversation.activity.new", activity)
}

// =========== Customer Profile Notifications ===========

// BroadcastProfileUpdate ส่งการแจ้งเตือนการอัพเดทโปรไฟล์ลูกค้า
func (a *WebSocketAdapter) BroadcastProfileUpdate(businessID uuid.UUID, userID uuid.UUID, profile interface{}) {
	a.BroadcastToBusiness(businessID, "profile.update", profile)
}

func (a *WebSocketAdapter) BroadcastProfileUpdateTags(businessID uuid.UUID, userID uuid.UUID, payload interface{}) {
	// ส่งข้อมูลไปยังทุกคนในธุรกิจ
	a.BroadcastToBusiness(businessID, "profile.tag_update", payload)
}

// =========== Note Notifications ===========

// BroadcastNoteCreated ส่งการแจ้งเตือน note ใหม่ไปยังสมาชิกใน conversation
func (a *WebSocketAdapter) BroadcastNoteCreated(conversationID uuid.UUID, note interface{}) {
	a.BroadcastToConversation(conversationID, "note.create", note)
}

// BroadcastNoteUpdated ส่งการแจ้งเตือน note ถูกอัปเดตไปยังสมาชิกใน conversation
func (a *WebSocketAdapter) BroadcastNoteUpdated(conversationID uuid.UUID, note interface{}) {
	a.BroadcastToConversation(conversationID, "note.update", note)
}

// BroadcastNoteDeleted ส่งการแจ้งเตือน note ถูกลบไปยังสมาชิกใน conversation
func (a *WebSocketAdapter) BroadcastNoteDeleted(conversationID uuid.UUID, noteID uuid.UUID, userID uuid.UUID) {
	a.BroadcastToConversation(conversationID, "note.delete", map[string]interface{}{
		"note_id":         noteID.String(),
		"conversation_id": conversationID.String(),
		"deleted_by":      userID.String(),
	})
}

// =========== Pinned Message Notifications ===========

// BroadcastMessagePinned ส่งการแจ้งเตือน message ถูกปักหมุด (public) ไปยังสมาชิกใน conversation
func (a *WebSocketAdapter) BroadcastMessagePinned(conversationID uuid.UUID, pinnedMessage interface{}) {
	a.BroadcastToConversation(conversationID, "message.pinned", pinnedMessage)
}

// BroadcastMessageUnpinned ส่งการแจ้งเตือน message ถูกยกเลิกปักหมุด (public) ไปยังสมาชิกใน conversation
func (a *WebSocketAdapter) BroadcastMessageUnpinned(conversationID uuid.UUID, messageID uuid.UUID, userID uuid.UUID) {
	a.BroadcastToConversation(conversationID, "message.unpinned", map[string]interface{}{
		"message_id":      messageID.String(),
		"conversation_id": conversationID.String(),
		"unpinned_by":     userID.String(),
		"unpinned_at":     utils.Now(),
	})
}
