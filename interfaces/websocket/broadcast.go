// interfaces/websocket/broadcast.go
package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
)

// broadcastMessage sends a message to specified clients
func (h *Hub) broadcastMessage(msg *BroadcastMessage) {
	data, err := json.Marshal(WSResponse{
		Type:      msg.Type,
		Data:      msg.Data,
		Timestamp: time.Now(),
		Success:   true,
	})
	if err != nil {
		return
	}

	// Broadcast to specific users
	if len(msg.UserIDs) > 0 {
		for _, userID := range msg.UserIDs {
			h.sendToUser(userID, data, msg.ExcludeID)
		}
	}


	// Broadcast to conversation
	if msg.ConvID != nil {
		h.broadcastToConversation(*msg.ConvID, msg.Type, msg.Data, msg.ExcludeID)
	}
}

// sendToUser sends a message to all connections of a user
func (h *Hub) sendToUser(userID uuid.UUID, data []byte, excludeID *uuid.UUID) {
	h.userConnectionsMux.RLock()
	clientIDs := h.userConnections[userID]
	h.userConnectionsMux.RUnlock()

	for _, clientID := range clientIDs {
		if excludeID != nil && clientID == *excludeID {
			continue
		}

		h.clientsMux.RLock()
		client, ok := h.clients[clientID]
		h.clientsMux.RUnlock()

		if ok {
			select {
			case client.Send <- data:
			default:
				// Client's send channel is full, close it
				go func() {
					h.unregister <- client
				}()
			}
		}
	}
}


// broadcastToConversation sends a message to all members of a conversation
func (h *Hub) broadcastToConversation(convID uuid.UUID, msgType MessageType, data interface{}, excludeID *uuid.UUID) {
	// Get conversation subscribers
	h.conversationSubsMux.RLock()
	subscriberIDs := h.conversationSubs[convID]
	h.conversationSubsMux.RUnlock()

	response, err := json.Marshal(WSResponse{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
		Success:   true,
	})
	if err != nil {
		return
	}

	// Send to each subscriber
	for _, clientID := range subscriberIDs {
		if excludeID != nil && clientID == *excludeID {
			continue
		}

		h.clientsMux.RLock()
		client, ok := h.clients[clientID]
		h.clientsMux.RUnlock()

		if ok {
			select {
			case client.Send <- response:
			default:
				go func() {
					h.unregister <- client
				}()
			}
		}
	}
}

// sendToClient sends a message to a specific client
func (h *Hub) sendToClient(client *Client, response WSResponse) {
	// Recover from panic if channel is closed
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in sendToClient for client %s: %v", client.ID, r)
		}
	}()

	data, err := json.Marshal(response)
	if err != nil {
		return
	}

	// Check if client still exists in hub
	h.clientsMux.RLock()
	_, exists := h.clients[client.ID]
	h.clientsMux.RUnlock()

	if !exists {
		log.Printf("Client %s no longer exists, skipping send", client.ID)
		return
	}

	select {
	case client.Send <- data:
		// Successfully sent
	default:
		// Client's send channel is full or closed
		log.Printf("Failed to send to client %s (channel full or closed)", client.ID)
		go func() {
			h.unregister <- client
		}()
	}
}

// removeClientFromSlice removes a client ID from a slice
func (h *Hub) removeClientFromSlice(slice *[]uuid.UUID, clientID uuid.UUID) {
	for i, id := range *slice {
		if id == clientID {
			*slice = append((*slice)[:i], (*slice)[i+1:]...)
			break
		}
	}
}

// removeClientFromAllConversations removes client from all conversation subscriptions
func (h *Hub) removeClientFromAllConversations(clientID uuid.UUID) {
	h.conversationSubsMux.Lock()
	defer h.conversationSubsMux.Unlock()

	for convID, subscribers := range h.conversationSubs {
		// Create a copy to work with
		updatedSubscribers := make([]uuid.UUID, len(subscribers))
		copy(updatedSubscribers, subscribers)

		// Remove client from the copy
		h.removeClientFromSlice(&updatedSubscribers, clientID)

		// Update or delete the entry
		if len(updatedSubscribers) == 0 {
			delete(h.conversationSubs, convID)
		} else {
			h.conversationSubs[convID] = updatedSubscribers
		}
	}
}

// เพิ่มฟังก์ชันใหม่ใน hub.go
func (h *Hub) removeClientFromAllUserStatusSubscriptions(clientID uuid.UUID) {
	h.userStatusSubsMux.Lock()
	defer h.userStatusSubsMux.Unlock()

	// วนลูปทุก user status subscriptions
	for userID, subscribers := range h.userStatusSubs {
		// สร้าง slice ใหม่ที่ไม่มี clientID
		updatedSubscribers := make([]uuid.UUID, 0, len(subscribers))
		for _, subID := range subscribers {
			if subID != clientID {
				updatedSubscribers = append(updatedSubscribers, subID)
			}
		}

		// อัปเดตหรือลบ entry
		if len(updatedSubscribers) == 0 {
			delete(h.userStatusSubs, userID)
		} else {
			h.userStatusSubs[userID] = updatedSubscribers
		}
	}
}

// NotifyBroadcast ส่งข้อความผ่าน broadcast channel
func (h *Hub) NotifyBroadcast(msg *BroadcastMessage) {
	if h == nil || msg == nil {
		return
	}

	select {
	case h.broadcast <- msg:
		log.Printf("Message type %s queued to broadcast channel", msg.Type)
	default:
		log.Printf("Broadcast channel full, dropping message type %s", msg.Type)
	}
}
func (h *Hub) BroadcastToConversation(conversationID uuid.UUID, msgType MessageType, data interface{}) {

	h.NotifyBroadcast(&BroadcastMessage{
		Type:   msgType,
		Data:   data,
		ConvID: &conversationID,
	})
}

// BroadcastToUsers ส่งข้อความไปยังผู้ใช้หลายคน
func (h *Hub) BroadcastToUsers(userIDs []uuid.UUID, msgType MessageType, data interface{}) {
	h.NotifyBroadcast(&BroadcastMessage{
		Type:    msgType,
		Data:    data,
		UserIDs: userIDs,
	})
}

// BroadcastToBusiness ส่งข้อความไปยังธุรกิจ
func (h *Hub) BroadcastToBusiness(businessID uuid.UUID, msgType MessageType, data interface{}) {
	h.NotifyBroadcast(&BroadcastMessage{
		Type:       msgType,
		Data:       data,
		BusinessID: &businessID,
	})
}

// BroadcastToUser ส่งข้อความไปยังผู้ใช้คนเดียว
func (h *Hub) BroadcastToUser(userID uuid.UUID, msgType MessageType, data interface{}) {
	h.NotifyBroadcast(&BroadcastMessage{
		Type:    msgType,
		Data:    data,
		UserIDs: []uuid.UUID{userID},
	})
}
