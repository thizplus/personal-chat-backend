// domain/port/websocket_port.go
package port

import "github.com/google/uuid"

// WebSocketPort เป็น interface สำหรับส่งข้อมูลผ่าน WebSocket
type WebSocketPort interface {
	// Core WebSocket methods
	BroadcastToUser(userID uuid.UUID, messageType string, data interface{}) // ส่งข้อความไปยังผู้ใช้คนใดคนหนึ่ง

	// Message notifications
	BroadcastNewMessage(conversationID uuid.UUID, message interface{})
	BroadcastMessageRead(conversationID uuid.UUID, message interface{})
	BroadcastMessageReadAll(conversationID uuid.UUID, message interface{})
	SendMessageReadToSender(senderID uuid.UUID, message interface{})        // ส่ง message.read ไปยังผู้ส่งข้อความเท่านั้น
	SendMessageReadAllToUser(userID uuid.UUID, message interface{})          // ส่ง message.read_all ไปยัง user ที่อ่าน (multi-device sync)
	BroadcastMessageDelivered(conversationID uuid.UUID, message interface{})
	BroadcastMessageEdited(conversationID uuid.UUID, message interface{})
	BroadcastMessageReply(conversationID uuid.UUID, message interface{})
	BroadcastMessageDeleted(conversationID uuid.UUID, messageID uuid.UUID)
	BroadcastMessageReaction(conversationID uuid.UUID, reaction interface{})

	// Conversation notifications
	BroadcastConversationCreated(userIDs []uuid.UUID, conversation interface{}) error
	BroadcastConversationUpdated(conversationID uuid.UUID, update interface{})
	BroadcastConversationDeleted(conversationID uuid.UUID, memberIDs []uuid.UUID)
	BroadcastUserAddedToConversation(conversationID uuid.UUID, userID uuid.UUID)
	BroadcastUserRemovedFromConversation(userID, conversationID uuid.UUID)
	BroadcastNewConversation(userID uuid.UUID, conversation interface{}) error

	// Member role notifications
	BroadcastMemberRoleChanged(conversationID uuid.UUID, data interface{})
	BroadcastOwnershipTransferred(conversationID uuid.UUID, data interface{})

	// Activity log notifications
	BroadcastNewActivity(conversationID uuid.UUID, activity interface{})

	// Business notifications
	BroadcastBusinessBroadcast(userIDs []uuid.UUID, broadcast interface{})
	BroadcastBusinessNewFollower(businessID, followerID uuid.UUID)
	BroadcastBusinessWelcomeMessage(userID, businessID uuid.UUID, message interface{})
	BroadcastBusinessFollowStatusChanged(businessID, userID uuid.UUID, isFollowing bool)
	BroadcastBusinessStatusChanged(businessID uuid.UUID, status string)

	// Customer Profile notifications
	BroadcastProfileUpdate(businessID, userID uuid.UUID, profile interface{})
	BroadcastProfileUpdateTags(businessID uuid.UUID, userID uuid.UUID, payload interface{})

	// Friend notifications
	BroadcastFriendRequestReceived(userID uuid.UUID, request interface{}) error
	BroadcastFriendRequestAccepted(userID uuid.UUID, friendship interface{}) error
	BroadcastFriendRequestRejected(userID uuid.UUID, friendship interface{}) error
	BroadcastFriendRemoved(userID, friendID uuid.UUID)

	// User notifications
	BroadcastUserBlocked(blockerID, blockedID uuid.UUID)
	BroadcastUserUnblocked(unblockerID, unblockedID uuid.UUID)

	// General notifications
	BroadcastNotification(userIDs []uuid.UUID, notification interface{})
	BroadcastAlert(userID uuid.UUID, alert interface{})
	BroadcastSystemMessage(userIDs []uuid.UUID, message interface{})

	// Note notifications (broadcast to conversation members for shared notes)
	BroadcastNoteCreated(conversationID uuid.UUID, note interface{})
	BroadcastNoteUpdated(conversationID uuid.UUID, note interface{})
	BroadcastNoteDeleted(conversationID uuid.UUID, noteID uuid.UUID, userID uuid.UUID)

	// Pinned message notifications (for public pins)
	BroadcastMessagePinned(conversationID uuid.UUID, pinnedMessage interface{})
	BroadcastMessageUnpinned(conversationID uuid.UUID, messageID uuid.UUID, userID uuid.UUID)
}
