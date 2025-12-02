// interfaces/websocket/handlers.go
package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// registerHandlers registers all message handlers
func (h *Hub) registerHandlers() {
	// Message handlers
	h.handlers[string(TypeMessageSend)] = &MessageSendHandler{hub: h}
	h.handlers[string(TypeMessageEdit)] = &MessageEditHandler{hub: h}
	h.handlers[string(TypeMessageDelete)] = &MessageDeleteHandler{hub: h}
	h.handlers[string(TypeMessageRead)] = &MessageReadHandler{hub: h}
	h.handlers[string(TypeMessageTyping)] = &MessageTypingHandler{hub: h}
	h.handlers[string(TypeTypingStart)] = &TypingStartHandler{hub: h}  // 🆕 เพิ่ม
	h.handlers[string(TypeTypingStop)] = &TypingStopHandler{hub: h}    // 🆕 เพิ่ม

	// Conversation handlers
	h.handlers[string(TypeConversationJoin)] = &ConversationJoinHandler{hub: h}
	h.handlers[string(TypeConversationLeave)] = &ConversationLeaveHandler{hub: h}
	h.handlers[string(TypeConversationCreate)] = &ConversationCreateHandler{hub: h}
	h.handlers[string(TypeConversationActive)] = &ConversationActiveHandler{hub: h}
	h.handlers[string(TypeConversationsLoad)] = &ConversationsLoadHandler{hub: h}

	// เพิ่ม handlers สำหรับ user status
	h.handlers[string(TypeUserStatusSubscribe)] = &SubscribeUserStatusHandler{hub: h}
	h.handlers[string(TypeUserStatusUnsubscribe)] = &UnsubscribeUserStatusHandler{hub: h}

	// Status handlers
	h.handlers[string(TypePing)] = &PingHandler{hub: h}
}

// MessageSendHandler handles sending messages
type MessageSendHandler struct {
	hub *Hub
}

type MessageSendData struct {
	ConversationID uuid.UUID              `json:"conversation_id"`
	Content        string                 `json:"content"`
	MessageType    string                 `json:"message_type"`
	MediaURL       string                 `json:"media_url,omitempty"`
	ThumbnailURL   string                 `json:"thumbnail_url,omitempty"`
	ReplyToID      *uuid.UUID             `json:"reply_to_id,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`

	// For sticker
	StickerID    *uuid.UUID `json:"sticker_id,omitempty"`
	StickerSetID *uuid.UUID `json:"sticker_set_id,omitempty"`

	// For file
	FileName string `json:"file_name,omitempty"`
	FileSize int64  `json:"file_size,omitempty"`
	FileType string `json:"file_type,omitempty"`
}

func (h *MessageSendHandler) Handle(ctx context.Context, client *Client, data json.RawMessage) error {
	var msgData MessageSendData
	if err := json.Unmarshal(data, &msgData); err != nil {
		return fmt.Errorf("invalid message data: %w", err)
	}

	// Check if the conversation service is available
	if h.hub.conversationService == nil {
		return fmt.Errorf("conversation service unavailable")
	}

	// Check membership
	isMember, err := h.hub.conversationService.CheckMembership(msgData.ConversationID, client.UserID)
	if err != nil || !isMember {
		return fmt.Errorf("user is not a member of this conversation")
	}

	// Check block status before sending message
	if h.hub.conversationMemberService != nil && h.hub.userFriendshipService != nil {
		members, _, err := h.hub.conversationMemberService.GetMembers(client.UserID, msgData.ConversationID, 1, 1000)
		if err == nil {
			for _, member := range members {
				memberUserID, parseErr := uuid.Parse(member.UserID)
				if parseErr != nil {
					continue
				}

				if memberUserID == client.UserID {
					continue // Skip self
				}

				isBlocked, isBlockedBy, blockErr := h.hub.userFriendshipService.CheckBlockStatus(client.UserID, memberUserID)
				if blockErr == nil && (isBlocked || isBlockedBy) {
					return fmt.Errorf("cannot send message: user is blocked")
				}
			}
		}
	}

	// Prepare message data for notification
	messageData := map[string]interface{}{
		"conversation_id": msgData.ConversationID,
		"sender_id":       client.UserID,
		"content":         msgData.Content,
		"message_type":    msgData.MessageType,
		"media_url":       msgData.MediaURL,
		"thumbnail_url":   msgData.ThumbnailURL,
		"metadata":        msgData.Metadata,
		"created_at":      time.Now(),
	}

	if msgData.ReplyToID != nil {
		messageData["reply_to_id"] = msgData.ReplyToID
	}

	if client.BusinessID != nil {
		messageData["business_id"] = client.BusinessID
	}

	// Add file info if applicable
	if msgData.MessageType == "file" {
		messageData["file_name"] = msgData.FileName
		messageData["file_size"] = msgData.FileSize
		messageData["file_type"] = msgData.FileType
	}

	// Add sticker info if applicable
	if msgData.MessageType == "sticker" && msgData.StickerID != nil && msgData.StickerSetID != nil {
		messageData["sticker_id"] = msgData.StickerID
		messageData["sticker_set_id"] = msgData.StickerSetID
	}

	// Broadcast to conversation members
	h.hub.BroadcastToConversation(msgData.ConversationID, TypeMessageReceive, messageData)

	return nil
}

func (h *MessageSendHandler) ValidateData(data json.RawMessage) error {
	var msgData MessageSendData
	return json.Unmarshal(data, &msgData)
}

// Global typing cache for auto-stop mechanism
var (
	typingCache      = sync.Map{} // key: "conv_id:user_id" -> *TypingStatus
	lastTypingUpdate = sync.Map{} // key: "conv_id:user_id" -> time.Time (for rate limiting)
)

// TypingStatus tracks typing state with auto-stop timer
type TypingStatus struct {
	ConversationID uuid.UUID
	UserID         uuid.UUID
	IsTyping       bool
	StartTime      time.Time
	StopTimer      *time.Timer
}

// MessageTypingHandler handles typing indicators
type MessageTypingHandler struct {
	hub *Hub
}

type TypingData struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	IsTyping       bool      `json:"is_typing"`
}

func (h *MessageTypingHandler) Handle(ctx context.Context, client *Client, data json.RawMessage) error {
	var typingData TypingData
	if err := json.Unmarshal(data, &typingData); err != nil {
		return err
	}

	// Rate limiting: Max 1 event per second (only for typing start)
	key := fmt.Sprintf("%s:%s", typingData.ConversationID.String(), client.UserID.String())
	if typingData.IsTyping {
		if lastTime, exists := lastTypingUpdate.Load(key); exists {
			if time.Since(lastTime.(time.Time)) < 1*time.Second {
				// Ignore - rate limited
				return nil
			}
		}
		lastTypingUpdate.Store(key, time.Now())
	}

	// Process typing with auto-stop logic
	h.processTypingWithAutoStop(client.UserID, typingData.ConversationID, typingData.IsTyping)

	return nil
}

// processTypingWithAutoStop handles typing events with auto-stop timer
func (h *MessageTypingHandler) processTypingWithAutoStop(userID, conversationID uuid.UUID, isTyping bool) {
	key := fmt.Sprintf("%s:%s", conversationID.String(), userID.String())

	if isTyping {
		// Cancel existing timer if any
		if val, exists := typingCache.Load(key); exists {
			status := val.(*TypingStatus)
			if status.StopTimer != nil {
				status.StopTimer.Stop()
			}
		}

		// Create auto-stop timer (5 seconds)
		timer := time.AfterFunc(5*time.Second, func() {
			h.autoStopTyping(userID, conversationID)
		})

		// Store in cache
		typingCache.Store(key, &TypingStatus{
			ConversationID: conversationID,
			UserID:         userID,
			IsTyping:       true,
			StartTime:      time.Now(),
			StopTimer:      timer,
		})

		// Broadcast typing start
		h.broadcastTyping(userID, conversationID, true)
	} else {
		// Manual stop - clear cache and timer
		if val, exists := typingCache.Load(key); exists {
			status := val.(*TypingStatus)
			if status.StopTimer != nil {
				status.StopTimer.Stop()
			}
			typingCache.Delete(key)
		}

		// Broadcast typing stop
		h.broadcastTyping(userID, conversationID, false)
	}
}

// autoStopTyping is called by timer after 5 seconds
func (h *MessageTypingHandler) autoStopTyping(userID, conversationID uuid.UUID) {
	key := fmt.Sprintf("%s:%s", conversationID.String(), userID.String())

	// Remove from cache
	if val, exists := typingCache.Load(key); exists {
		status := val.(*TypingStatus)
		if status.StopTimer != nil {
			status.StopTimer.Stop()
		}
		typingCache.Delete(key)
	}

	// Broadcast typing stop
	h.broadcastTyping(userID, conversationID, false)

	log.Printf("Auto-stopped typing for user %s in conversation %s", userID, conversationID)
}

// broadcastTyping sends typing event to conversation members with user info
func (h *MessageTypingHandler) broadcastTyping(userID, conversationID uuid.UUID, isTyping bool) {
	// Query user info for username and display_name
	var username, displayName string

	if h.hub.userRepo != nil {
		user, err := h.hub.userRepo.FindByID(userID)
		if err != nil {
			log.Printf("Error fetching user info for typing event: %v", err)
			username = userID.String() // Fallback to user ID
			displayName = ""
		} else {
			username = user.Username
			if user.DisplayName != "" {
				displayName = user.DisplayName
			} else {
				displayName = user.Username // Fallback to username
			}
		}
	} else {
		// Fallback if userRepo not available
		username = userID.String()
		displayName = ""
	}

	typingInfo := map[string]interface{}{
		"user_id":         userID.String(),
		"username":        username,          // 🆕 เพิ่ม
		"display_name":    displayName,       // 🆕 เพิ่ม
		"conversation_id": conversationID.String(),
		"is_typing":       isTyping,
	}

	// ส่งทั้ง event แบบเก่าและใหม่
	h.hub.BroadcastToConversation(conversationID, TypeMessageTyping, typingInfo) // เก่า
	h.hub.BroadcastToConversation(conversationID, "user_typing", typingInfo)     // ใหม่ (ตาม spec)
}

func (h *MessageTypingHandler) ValidateData(data json.RawMessage) error {
	var typingData TypingData
	return json.Unmarshal(data, &typingData)
}

// TypingStartHandler handles typing_start events (new spec)
type TypingStartHandler struct {
	hub *Hub
}

func (h *TypingStartHandler) Handle(ctx context.Context, client *Client, data json.RawMessage) error {
	var req struct {
		ConversationID uuid.UUID `json:"conversation_id"`
	}

	if err := json.Unmarshal(data, &req); err != nil {
		return err
	}

	// Reuse existing typing logic with isTyping=true
	typingHandler := &MessageTypingHandler{hub: h.hub}
	typingHandler.processTypingWithAutoStop(client.UserID, req.ConversationID, true)

	return nil
}

func (h *TypingStartHandler) ValidateData(data json.RawMessage) error {
	var req struct {
		ConversationID uuid.UUID `json:"conversation_id"`
	}
	return json.Unmarshal(data, &req)
}

// TypingStopHandler handles typing_stop events (new spec)
type TypingStopHandler struct {
	hub *Hub
}

func (h *TypingStopHandler) Handle(ctx context.Context, client *Client, data json.RawMessage) error {
	var req struct {
		ConversationID uuid.UUID `json:"conversation_id"`
	}

	if err := json.Unmarshal(data, &req); err != nil {
		return err
	}

	// Reuse existing typing logic with isTyping=false
	typingHandler := &MessageTypingHandler{hub: h.hub}
	typingHandler.processTypingWithAutoStop(client.UserID, req.ConversationID, false)

	return nil
}

func (h *TypingStopHandler) ValidateData(data json.RawMessage) error {
	var req struct {
		ConversationID uuid.UUID `json:"conversation_id"`
	}
	return json.Unmarshal(data, &req)
}

// StartTypingCacheCleanup starts a background routine to cleanup stale typing cache
// Call this once on application startup
func StartTypingCacheCleanup() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		log.Println("Typing cache cleanup routine started")

		for range ticker.C {
			now := time.Now()
			cleanedCount := 0

			typingCache.Range(func(key, value interface{}) bool {
				status := value.(*TypingStatus)
				// Remove stale entries (older than 10 seconds)
				if now.Sub(status.StartTime) > 10*time.Second {
					if status.StopTimer != nil {
						status.StopTimer.Stop()
					}
					typingCache.Delete(key)
					cleanedCount++
					log.Printf("Cleaned up stale typing cache: %v", key)
				}
				return true
			})

			if cleanedCount > 0 {
				log.Printf("Typing cache cleanup: removed %d stale entries", cleanedCount)
			}
		}
	}()
}

// MessageReadHandler handles message read status
type MessageReadHandler struct {
	hub *Hub
}

type MessageReadData struct {
	MessageID      uuid.UUID `json:"message_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
}

func (h *MessageReadHandler) Handle(ctx context.Context, client *Client, data json.RawMessage) error {
	var readData MessageReadData
	if err := json.Unmarshal(data, &readData); err != nil {
		return err
	}

	// Broadcast read status to conversation members
	readInfo := map[string]interface{}{
		"message_id": readData.MessageID,
		"user_id":    client.UserID,
		"read_at":    time.Now(),
	}

	h.hub.BroadcastToConversation(readData.ConversationID, TypeMessageRead, readInfo)

	return nil
}

func (h *MessageReadHandler) ValidateData(data json.RawMessage) error {
	var readData MessageReadData
	return json.Unmarshal(data, &readData)
}

// MessageEditHandler handles message editing
type MessageEditHandler struct {
	hub *Hub
}

type MessageEditData struct {
	MessageID      uuid.UUID `json:"message_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	NewContent     string    `json:"new_content"`
}

func (h *MessageEditHandler) Handle(ctx context.Context, client *Client, data json.RawMessage) error {
	var editData MessageEditData
	if err := json.Unmarshal(data, &editData); err != nil {
		return err
	}

	// Broadcast edited message to conversation members
	editInfo := map[string]interface{}{
		"message_id":      editData.MessageID.String(),
		"conversation_id": editData.ConversationID.String(),
		"new_content":     editData.NewContent,
		"edited_at":       time.Now().Format(time.RFC3339),
	}

	h.hub.BroadcastToConversation(editData.ConversationID, TypeMessageEdit, editInfo)

	return nil
}

func (h *MessageEditHandler) ValidateData(data json.RawMessage) error {
	var editData MessageEditData
	return json.Unmarshal(data, &editData)
}

// MessageDeleteHandler handles message deletion
type MessageDeleteHandler struct {
	hub *Hub
}

type MessageDeleteData struct {
	MessageID      uuid.UUID `json:"message_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
}

func (h *MessageDeleteHandler) Handle(ctx context.Context, client *Client, data json.RawMessage) error {
	var deleteData MessageDeleteData
	if err := json.Unmarshal(data, &deleteData); err != nil {
		return err
	}

	// Broadcast deletion to conversation members
	deleteInfo := map[string]interface{}{
		"message_id": deleteData.MessageID,
		"deleted_by": client.UserID,
		"deleted_at": time.Now(),
	}

	h.hub.BroadcastToConversation(deleteData.ConversationID, TypeMessageDelete, deleteInfo)

	return nil
}

func (h *MessageDeleteHandler) ValidateData(data json.RawMessage) error {
	var deleteData MessageDeleteData
	return json.Unmarshal(data, &deleteData)
}

// ConversationsLoadHandler loads user's conversations
type ConversationsLoadHandler struct {
	hub *Hub
}

type ConversationsLoadData struct {
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
	Order  string `json:"order,omitempty"`
}

func (h *ConversationsLoadHandler) Handle(ctx context.Context, client *Client, data json.RawMessage) error {
	var loadData ConversationsLoadData
	if err := json.Unmarshal(data, &loadData); err != nil {
		return err
	}

	// Set default values if not specified
	if loadData.Limit <= 0 {
		loadData.Limit = 20
	}
	if loadData.Order == "" {
		loadData.Order = "last_message_at DESC"
	}

	// Check if the conversation service is available
	if h.hub.conversationService == nil {
		return fmt.Errorf("conversation service unavailable")
	}

	// Get user's conversations
	conversations, _, err := h.hub.conversationService.GetUserConversations(
		client.UserID, loadData.Limit, loadData.Offset, loadData.Order, false,
	)
	if err != nil {
		log.Printf("Error loading conversations for user %s: %v", client.UserID, err)
		return fmt.Errorf("error loading conversations: %w", err)
	}

	// Subscribe to first few conversations (lazy loading)
	subscribeCount := min(5, len(conversations))
	h.hub.conversationSubsMux.Lock()
	for i, conv := range conversations {
		if i < subscribeCount {
			h.hub.conversationSubs[conv.ID] = append(
				h.hub.conversationSubs[conv.ID], client.ID,
			)
		}
	}
	h.hub.conversationSubsMux.Unlock()

	// Prepare conversation list for response
	// Prepare conversation list for response
	conversationList := make([]map[string]interface{}, len(conversations))
	for i, conv := range conversations {
		// สร้าง base fields ที่จำเป็นต้องมีเสมอ
		conversationList[i] = map[string]interface{}{
			"id":            conv.ID,
			"title":         conv.Title,
			"type":          conv.Type,
			"created_at":    conv.CreatedAt,
			"updated_at":    conv.UpdatedAt,
			"is_active":     conv.IsActive,
			"member_count":  conv.MemberCount,
			"is_subscribed": i < subscribeCount,
		}

		// เพิ่ม fields ที่อาจเป็น nil หรือค่าว่าง แต่ควรรวมเสมอตาม DTO
		if conv.LastMessageAt != nil {
			conversationList[i]["last_message_at"] = conv.LastMessageAt
		}

		if conv.LastMessageText != "" { // แก้จาก == เป็น !=
			conversationList[i]["last_message_text"] = conv.LastMessageText
		}

		if conv.IconURL != "" {
			conversationList[i]["icon_url"] = conv.IconURL
		}

		if conv.CreatorID != nil {
			conversationList[i]["creator_id"] = conv.CreatorID
		}

		if conv.BusinessID != nil {
			conversationList[i]["business_id"] = conv.BusinessID
		}

		// เพิ่ม fields ตามที่มีค่า
		if conv.UnreadCount > 0 {
			conversationList[i]["unread_count"] = conv.UnreadCount
		} else {
			conversationList[i]["unread_count"] = 0 // เพิ่มค่า default ให้สอดคล้องกับ DTO
		}

		// fields ที่เป็น boolean ควรส่งเสมอเพื่อความสอดคล้องกับ DTO
		conversationList[i]["is_pinned"] = conv.IsPinned
		conversationList[i]["is_muted"] = conv.IsMuted

		// เพิ่ม fields ที่เป็น object ถ้ามีค่า
		if conv.Metadata != nil {
			conversationList[i]["metadata"] = conv.Metadata
		}

		if conv.ContactInfo != nil {
			conversationList[i]["contact_info"] = conv.ContactInfo
		}

		if conv.BusinessInfo != nil {
			conversationList[i]["business_info"] = conv.BusinessInfo
		}

	}

	// Send conversation list to client
	h.hub.sendToClient(client, WSResponse{
		Type:      "conversation.list",
		Data:      conversationList,
		Timestamp: time.Now(),
		Success:   true,
	})

	log.Printf("User %s loaded %d conversations, subscribed to %d",
		client.UserID, len(conversations), subscribeCount)

	return nil
}

func (h *ConversationsLoadHandler) ValidateData(data json.RawMessage) error {
	var loadData ConversationsLoadData
	return json.Unmarshal(data, &loadData)
}

// ConversationJoinHandler handles joining conversations
type ConversationJoinHandler struct {
	hub *Hub
}

type ConversationJoinData struct {
	ConversationID uuid.UUID `json:"conversation_id"`
}

func (h *ConversationJoinHandler) Handle(ctx context.Context, client *Client, data json.RawMessage) error {
	var joinData ConversationJoinData
	if err := json.Unmarshal(data, &joinData); err != nil {
		return err
	}

	log.Printf("Join request received for conversation: %s by user: %s",
		joinData.ConversationID.String(), client.UserID.String())

	// Check if the conversation service is available
	if h.hub.conversationService == nil {
		return fmt.Errorf("conversation service unavailable")
	}

	// Check membership
	isMember, err := h.hub.conversationService.CheckMembership(client.UserID, joinData.ConversationID)
	if err != nil {
		log.Printf("Error checking membership: %v", err)
		return fmt.Errorf("error checking conversation membership: %w", err)
	}

	if !isMember {
		log.Printf("User %s is not a member of conversation %s",
			client.UserID.String(), joinData.ConversationID.String())
		return fmt.Errorf("user is not a member of this conversation")
	}

	// Set the active conversation
	h.hub.clientsMux.Lock()
	client.ActiveConversationID = &joinData.ConversationID
	h.hub.clientsMux.Unlock()

	// Subscribe to conversation
	h.hub.conversationSubsMux.Lock()
	// Check if already subscribed
	alreadySubscribed := false
	for _, clientID := range h.hub.conversationSubs[joinData.ConversationID] {
		if clientID == client.ID {
			alreadySubscribed = true
			break
		}
	}
	// Add to subscription if not already subscribed
	if !alreadySubscribed {
		h.hub.conversationSubs[joinData.ConversationID] = append(
			h.hub.conversationSubs[joinData.ConversationID], client.ID,
		)
	}
	h.hub.conversationSubsMux.Unlock()

	// Notify other members that this user is active in the conversation
	activeInfo := map[string]interface{}{
		"user_id":         client.UserID,
		"conversation_id": joinData.ConversationID,
		"active":          true,
		"timestamp":       time.Now(),
	}
	h.hub.BroadcastToConversation(joinData.ConversationID, "conversation.user_active", activeInfo)

	// Send joined confirmation to client
	h.hub.sendToClient(client, WSResponse{
		Type: "conversation.joined",
		Data: map[string]interface{}{
			"conversation_id": joinData.ConversationID,
			"success":         true,
		},
		Timestamp: time.Now(),
		Success:   true,
	})

	log.Printf("User %s successfully joined conversation %s",
		client.UserID.String(), joinData.ConversationID.String())

	return nil
}

func (h *ConversationJoinHandler) ValidateData(data json.RawMessage) error {
	var joinData ConversationJoinData
	return json.Unmarshal(data, &joinData)
}

// ConversationLeaveHandler handles leaving conversations
type ConversationLeaveHandler struct {
	hub *Hub
}

type ConversationLeaveData struct {
	ConversationID uuid.UUID `json:"conversation_id"`
}

func (h *ConversationLeaveHandler) Handle(ctx context.Context, client *Client, data json.RawMessage) error {
	var leaveData ConversationLeaveData
	if err := json.Unmarshal(data, &leaveData); err != nil {
		return err
	}

	// Check if this is the active conversation
	h.hub.clientsMux.RLock()
	isActiveConversation := client.ActiveConversationID != nil && *client.ActiveConversationID == leaveData.ConversationID
	h.hub.clientsMux.RUnlock()

	// If active, set active conversation to nil
	if isActiveConversation {
		h.hub.clientsMux.Lock()
		client.ActiveConversationID = nil
		h.hub.clientsMux.Unlock()

		// Notify other members that this user is no longer active
		inactiveInfo := map[string]interface{}{
			"user_id":         client.UserID,
			"conversation_id": leaveData.ConversationID,
			"active":          false,
			"timestamp":       time.Now(),
		}
		h.hub.BroadcastToConversation(leaveData.ConversationID, "conversation.user_active", inactiveInfo)
	}

	// Unsubscribe from conversation
	h.hub.conversationSubsMux.Lock()
	if subscribers, exists := h.hub.conversationSubs[leaveData.ConversationID]; exists {
		// Create a copy to work with
		updatedSubscribers := make([]uuid.UUID, len(subscribers))
		copy(updatedSubscribers, subscribers)

		// Remove client from the copy
		h.hub.removeClientFromSlice(&updatedSubscribers, client.ID)

		// Update or delete the entry
		if len(updatedSubscribers) == 0 {
			delete(h.hub.conversationSubs, leaveData.ConversationID)
		} else {
			h.hub.conversationSubs[leaveData.ConversationID] = updatedSubscribers
		}
	}
	h.hub.conversationSubsMux.Unlock()

	// Send confirmation to client
	h.hub.sendToClient(client, WSResponse{
		Type: "conversation.left",
		Data: map[string]interface{}{
			"conversation_id": leaveData.ConversationID,
			"success":         true,
		},
		Timestamp: time.Now(),
		Success:   true,
	})

	return nil
}

func (h *ConversationLeaveHandler) ValidateData(data json.RawMessage) error {
	var leaveData ConversationLeaveData
	return json.Unmarshal(data, &leaveData)
}

// ConversationActiveHandler handles setting the active conversation
type ConversationActiveHandler struct {
	hub *Hub
}

type ConversationActiveData struct {
	ConversationID *uuid.UUID `json:"conversation_id"` // Can be null to indicate no active conversation
}

func (h *ConversationActiveHandler) Handle(ctx context.Context, client *Client, data json.RawMessage) error {
	var activeData ConversationActiveData
	if err := json.Unmarshal(data, &activeData); err != nil {
		return err
	}

	// Get current active conversation
	h.hub.clientsMux.RLock()
	oldActiveConvID := client.ActiveConversationID
	h.hub.clientsMux.RUnlock()

	// If leaving an active conversation, notify others
	if oldActiveConvID != nil && (activeData.ConversationID == nil || *oldActiveConvID != *activeData.ConversationID) {
		inactiveInfo := map[string]interface{}{
			"user_id":         client.UserID,
			"conversation_id": *oldActiveConvID,
			"active":          false,
			"timestamp":       time.Now(),
		}
		h.hub.BroadcastToConversation(*oldActiveConvID, "conversation.user_active", inactiveInfo)
	}

	// Update active conversation
	h.hub.clientsMux.Lock()
	client.ActiveConversationID = activeData.ConversationID
	h.hub.clientsMux.Unlock()

	// If setting a new active conversation
	if activeData.ConversationID != nil {
		// Check if conversation service is available
		if h.hub.conversationService != nil {
			// Check membership
			isMember, err := h.hub.conversationService.CheckMembership(client.UserID, *activeData.ConversationID)
			if err != nil || !isMember {
				return fmt.Errorf("user is not a member of this conversation")
			}
		}

		// Check if already subscribed
		h.hub.conversationSubsMux.RLock()
		isSubscribed := false
		for _, clientID := range h.hub.conversationSubs[*activeData.ConversationID] {
			if clientID == client.ID {
				isSubscribed = true
				break
			}
		}
		h.hub.conversationSubsMux.RUnlock()

		// Subscribe if not already subscribed
		if !isSubscribed {
			h.hub.conversationSubsMux.Lock()
			h.hub.conversationSubs[*activeData.ConversationID] = append(
				h.hub.conversationSubs[*activeData.ConversationID], client.ID,
			)
			h.hub.conversationSubsMux.Unlock()
		}

		// Notify other members that this user is active
		activeInfo := map[string]interface{}{
			"user_id":         client.UserID,
			"conversation_id": *activeData.ConversationID,
			"active":          true,
			"timestamp":       time.Now(),
		}
		h.hub.BroadcastToConversation(*activeData.ConversationID, "conversation.user_active", activeInfo)
	}

	// Send confirmation to client
	h.hub.sendToClient(client, WSResponse{
		Type: "conversation.active_set",
		Data: map[string]interface{}{
			"conversation_id": activeData.ConversationID,
			"success":         true,
		},
		Timestamp: time.Now(),
		Success:   true,
	})

	return nil
}

func (h *ConversationActiveHandler) ValidateData(data json.RawMessage) error {
	var activeData ConversationActiveData
	return json.Unmarshal(data, &activeData)
}

// ConversationCreateHandler handles creating new conversations
type ConversationCreateHandler struct {
	hub *Hub
}

type ConversationCreateData struct {
	Type       string      `json:"type"` // direct, group, business
	Title      string      `json:"title,omitempty"`
	IconURL    string      `json:"icon_url,omitempty"`
	MemberIDs  []uuid.UUID `json:"member_ids,omitempty"`
	FriendID   *uuid.UUID  `json:"friend_id,omitempty"`
	BusinessID *uuid.UUID  `json:"business_id,omitempty"`
}

func (h *ConversationCreateHandler) Handle(ctx context.Context, client *Client, data json.RawMessage) error {
	var createData ConversationCreateData
	if err := json.Unmarshal(data, &createData); err != nil {
		return err
	}

	// Check if conversation service is available
	if h.hub.conversationService == nil {
		return fmt.Errorf("conversation service unavailable")
	}

	// Validate required fields
	switch createData.Type {
	case "direct":
		if createData.FriendID == nil {
			return fmt.Errorf("friend_id is required for direct conversation")
		}
	case "group":
		if len(createData.MemberIDs) == 0 {
			return fmt.Errorf("member_ids is required for group conversation")
		}
	case "business":
		if createData.BusinessID == nil {
			return fmt.Errorf("business_id is required for business conversation")
		}
	default:
		return fmt.Errorf("invalid conversation type: %s", createData.Type)
	}

	// Create conversation
	var conversation interface{}
	var err error

	switch createData.Type {
	case "direct":
		conversation, err = h.hub.conversationService.CreateDirectConversation(
			client.UserID, *createData.FriendID,
		)
	case "group":
		conversation, err = h.hub.conversationService.CreateGroupConversation(
			client.UserID, createData.Title, createData.IconURL, createData.MemberIDs,
		)
	}

	if err != nil {
		return fmt.Errorf("failed to create conversation: %w", err)
	}

	// Get conversation ID
	var conversationID uuid.UUID
	conversationMap, ok := conversation.(map[string]interface{})
	if ok {
		convIDStr, ok := conversationMap["id"].(string)
		if ok {
			conversationID, _ = uuid.Parse(convIDStr)
		}
	}

	// Add client to subscription
	if conversationID != uuid.Nil {
		h.hub.conversationSubsMux.Lock()
		h.hub.conversationSubs[conversationID] = append(
			h.hub.conversationSubs[conversationID], client.ID,
		)
		h.hub.conversationSubsMux.Unlock()

		// Set as active conversation
		h.hub.clientsMux.Lock()
		client.ActiveConversationID = &conversationID
		h.hub.clientsMux.Unlock()
	}

	// Send response to creator
	h.hub.sendToClient(client, WSResponse{
		Type:      TypeConversationCreate,
		Data:      conversation,
		Timestamp: time.Now(),
		Success:   true,
		RequestID: client.ID.String(),
	})

	// Notify other members if it's a group conversation
	if createData.Type == "group" && len(createData.MemberIDs) > 0 {
		h.hub.BroadcastToUsers(createData.MemberIDs, TypeConversationCreate, conversation)
	}

	return nil
}

func (h *ConversationCreateHandler) ValidateData(data json.RawMessage) error {
	var createData ConversationCreateData
	return json.Unmarshal(data, &createData)
}

// PingHandler handles ping messages
type PingHandler struct {
	hub *Hub
}

func (h *PingHandler) Handle(ctx context.Context, client *Client, data json.RawMessage) error {
	client.IsAlive = true
	client.LastPingTime = time.Now()

	// Send pong response
	h.hub.sendToClient(client, WSResponse{
		Type:      TypePong,
		Data:      map[string]interface{}{"message": "pong"},
		Timestamp: time.Now(),
		Success:   true,
	})

	return nil
}

func (h *PingHandler) ValidateData(data json.RawMessage) error {
	return nil
}

// UserStatusHandler handles user status
// SubscribeUserStatusHandler จัดการการลงทะเบียนติดตามสถานะผู้ใช้
type SubscribeUserStatusHandler struct {
	hub *Hub
}

// ปรับปรุง SubscribeUserStatusHandler ใน handlers.go
func (h *SubscribeUserStatusHandler) Handle(ctx context.Context, client *Client, data json.RawMessage) error {
	var req struct {
		UserID   string `json:"user_id"`
		ClientID string `json:"client_id,omitempty"`
	}

	if err := json.Unmarshal(data, &req); err != nil {
		return err
	}

	targetUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		return err
	}

	// ใช้ client ID ปัจจุบันเสมอ
	h.hub.subscribeToUserStatus(client.ID, targetUserID)

	// สำเร็จ ส่งผลลัพธ์กลับ
	h.hub.sendToClient(client, WSResponse{
		Type: "user.status.subscribed",
		Data: map[string]interface{}{
			"user_id": targetUserID.String(),
			"success": true,
		},
		Timestamp: time.Now(),
		Success:   true,
		RequestID: uuid.New().String(), // แก้จาก crypto.NewUUID() เป็น uuid.New()
	})

	return nil
}

func (h *SubscribeUserStatusHandler) ValidateData(data json.RawMessage) error {
	var req struct {
		UserID string `json:"user_id"`
	}
	return json.Unmarshal(data, &req)
}

// UnsubscribeUserStatusHandler จัดการการยกเลิกการติดตามสถานะผู้ใช้
type UnsubscribeUserStatusHandler struct {
	hub *Hub
}

// ปรับปรุง UnsubscribeUserStatusHandler ใน handlers.go
func (h *UnsubscribeUserStatusHandler) Handle(ctx context.Context, client *Client, data json.RawMessage) error {
	var req struct {
		UserID string `json:"user_id"`
	}

	if err := json.Unmarshal(data, &req); err != nil {
		return err
	}

	targetUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		return err
	}

	// ยกเลิกการติดตาม
	h.hub.unsubscribeFromUserStatus(client.ID, targetUserID)

	// สำเร็จ ส่งผลลัพธ์กลับ
	h.hub.sendToClient(client, WSResponse{
		Type: "user.status.unsubscribed",
		Data: map[string]interface{}{
			"user_id": targetUserID.String(),
			"success": true,
		},
		Timestamp: time.Now(),
		Success:   true,
		RequestID: uuid.New().String(), // แก้จาก crypto.NewUUID() เป็น uuid.New()
	})

	return nil
}

func (h *UnsubscribeUserStatusHandler) ValidateData(data json.RawMessage) error {
	var req struct {
		UserID string `json:"user_id"`
	}
	return json.Unmarshal(data, &req)
}
