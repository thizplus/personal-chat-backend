// domain/service/notification_service.go
package service

import (
	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/dto"
)

// WebSocketNotifier interface สำหรับส่ง real-time notifications
type NotificationService interface {
	// Message notifications
	NotifyNewMessage(conversationID uuid.UUID, message interface{})
	NotifyMessageRead(conversationID uuid.UUID, message interface{})
	NotifyMessageReadAll(conversationID uuid.UUID, message interface{})
	NotifyMessageReadToSender(senderID uuid.UUID, message interface{})        // ส่ง message.read ไปยังผู้ส่งเท่านั้น
	NotifyMessageReadAllToUser(userID uuid.UUID, message interface{})          // ส่ง message.read_all ไปยัง user ที่อ่าน
	NotifyMessageDelivered(conversationID uuid.UUID, message interface{})
	NotifyMessageEdited(conversationID uuid.UUID, message interface{})
	NotifyMessageReply(conversationID uuid.UUID, message interface{})
	NotifyMessageDeleted(conversationID uuid.UUID, messageID uuid.UUID)
	NotifyMessageReaction(conversationID uuid.UUID, reaction interface{})

	// Conversation notifications
	NotifyConversationCreated(userIDs []uuid.UUID, conversation interface{}) error
	NotifyConversationUpdated(conversationID uuid.UUID, update interface{})
	NotifyConversationUpdatedToUser(userID uuid.UUID, update interface{}) // ส่ง conversation.update ไปยัง user คนใดคนหนึ่ง (personalized)
	NotifyConversationDeleted(conversationID uuid.UUID, memberIDs []uuid.UUID)
	NotifyUserAddedToConversation(conversationID uuid.UUID, userID uuid.UUID)
	NotifyUserRemovedFromConversation(userID, conversationID uuid.UUID)
	NotifyNewConversation(conversation interface{}) error

	// Member role notifications
	NotifyMemberRoleChanged(conversationID, userID uuid.UUID, oldRole, newRole string, changedByUserID uuid.UUID)
	NotifyOwnershipTransferred(conversationID, previousOwnerID, newOwnerID uuid.UUID)

	// Activity log notifications
	NotifyNewActivity(conversationID uuid.UUID, activity *dto.ActivityDTO)


	// Customer Profile notifications

	// Friend notifications
	NotifyFriendRequestReceived(request interface{}) error
	NotifyFriendRequestAccepted(friendship interface{}) error
	NotifyFriendRequestRejected(friendship interface{}) error
	NotifyFriendRemoved(userID, friendID uuid.UUID)

	// User notifications
	NotifyUserBlocked(blockerID, blockedID uuid.UUID)
	NotifyUserUnblocked(unblockerID, unblockedID uuid.UUID)

	// General notifications
	SendNotification(userIDs []uuid.UUID, notification interface{})
	SendAlert(userID uuid.UUID, alert interface{})
	NotifySystemMessage(userIDs []uuid.UUID, message interface{})
}
