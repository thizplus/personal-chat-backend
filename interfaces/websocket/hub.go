// interfaces/websocket/hub.go
package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/repository"
	"github.com/thizplus/gofiber-chat-api/domain/service"
)

// Hub manages all WebSocket connections
type Hub struct {
	// Registered clients
	clients    map[uuid.UUID]*Client
	clientsMux sync.RWMutex

	// User connections mapping (userID -> clientIDs)
	userConnections    map[uuid.UUID][]uuid.UUID
	userConnectionsMux sync.RWMutex

	// Business connections mapping (businessID -> clientIDs)

	// Conversation subscriptions (conversationID -> clientIDs)
	conversationSubs    map[uuid.UUID][]uuid.UUID
	conversationSubsMux sync.RWMutex

	// เพิ่มส่วนนี้
	userStatusSubs    map[uuid.UUID][]uuid.UUID
	userStatusSubsMux sync.RWMutex

	// Message handlers
	handlers map[string]MessageHandler

	// Core services (เฉพาะที่จำเป็น)
	conversationService       service.ConversationService
	conversationMemberService service.ConversationMemberService
	userFriendshipService     service.UserFriendshipService
	notificationService       service.NotificationService
	presenceService           service.PresenceService
	userRepo                  repository.UserRepository // 🆕 เพิ่มสำหรับ typing user info

	// Channels
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage

	// Statistics
	startTime       time.Time
	totalMessages   int64
	messagesSentMux sync.RWMutex
}

// เพิ่ม Rate Limiter struct
type RateLimiter struct {
	rate       int
	interval   time.Duration
	tokens     int
	lastRefill time.Time
	mu         sync.Mutex
}

func NewRateLimiter(rate int, interval time.Duration) *RateLimiter {
	return &RateLimiter{
		rate:       rate,
		interval:   interval,
		tokens:     rate,
		lastRefill: time.Now(),
	}
}

func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if now.Sub(r.lastRefill) > r.interval {
		r.tokens = r.rate
		r.lastRefill = now
	}

	if r.tokens > 0 {
		r.tokens--
		return true
	}
	return false
}

// Client represents a WebSocket connection
type Client struct {
	ID                   uuid.UUID
	UserID               uuid.UUID
	BusinessID           *uuid.UUID // If connected as business
	ActiveConversationID *uuid.UUID // เพิ่ม field นี้
	Conn                 *websocket.Conn
	Send                 chan []byte
	Hub                  *Hub
	IsAlive              bool
	LastPingTime         time.Time
	RateLimiter          *RateLimiter
	messageCount         int
	lastReset            time.Time
}

// Message types
type MessageType string

const (
	// Connection management
	TypeConnect    MessageType = "connect"
	TypeDisconnect MessageType = "disconnect"
	TypePing       MessageType = "ping"
	TypePong       MessageType = "pong"

	// Chat messages
	TypeMessageSend      MessageType = "message.send"
	TypeMessageReceive   MessageType = "message.receive"
	TypeMessageEdit      MessageType = "message.updated"
	TypeMessageDelete    MessageType = "message.delete"
	TypeMessageRead      MessageType = "message.read"
	TypeMessageDelivered MessageType = "message.delivered"
	TypeMessageTyping    MessageType = "message.typing"
	TypeTypingStart      MessageType = "typing_start"    // 🆕 เพิ่ม
	TypeTypingStop       MessageType = "typing_stop"     // 🆕 เพิ่ม
	TypeUserTyping       MessageType = "user_typing"     // 🆕 เพิ่ม (broadcast)

	// Conversation events
	TypeConversationCreate MessageType = "conversation.create"
	TypeConversationUpdate MessageType = "conversation.update"
	TypeConversationActive MessageType = "conversation.active"
	TypeConversationsLoad  MessageType = "conversation.load"
	TypeConversationJoin   MessageType = "conversation.join"
	TypeConversationLeave  MessageType = "conversation.leave"

	// Business events

	// User subscribe status
	TypeUserStatusSubscribe   MessageType = "user.status.subscribe"
	TypeUserStatusUnsubscribe MessageType = "user.status.unsubscribe"

	// User status
	TypeUserOnline  MessageType = "user.online"
	TypeUserOffline MessageType = "user.offline"
	TypeUserStatus  MessageType = "user.status"

	// Friend events
	TypeFriendRequest MessageType = "friend.request"
	TypeFriendAccept  MessageType = "friend.accept"
	TypeFriendRemove  MessageType = "friend.remove"

	// Block events
	TypeUserBlocked     MessageType = "user.blocked"       // ส่งไปยัง blocker
	TypeUserBlockedBy   MessageType = "user.blocked_by"    // ส่งไปยังคนที่ถูก block
	TypeUserUnblocked   MessageType = "user.unblocked"     // ส่งไปยัง unblocker
	TypeUserUnblockedBy MessageType = "user.unblocked_by"  // ส่งไปยังคนที่ถูก unblock

	// Notifications
	TypeNotification MessageType = "notification"
	TypeAlert        MessageType = "alert"
)

// WebSocket message structure
type WSMessage struct {
	Type      MessageType     `json:"type"`
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
	RequestID string          `json:"request_id,omitempty"`
}

// Response message structure
type WSResponse struct {
	Type      MessageType `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
	Success   bool        `json:"success"`
	Error     string      `json:"error,omitempty"`
}

// BroadcastMessage for sending messages to multiple clients
type BroadcastMessage struct {
	Type       MessageType
	Data       interface{}
	UserIDs    []uuid.UUID
	BusinessID *uuid.UUID
	ConvID     *uuid.UUID
	ExcludeID  *uuid.UUID // Exclude specific client
}

// MessageHandler interface for handling different message types
type MessageHandler interface {
	Handle(ctx context.Context, client *Client, data json.RawMessage) error
	ValidateData(data json.RawMessage) error
}

// NewHub creates a new WebSocket hub
func NewHub(
	conversationService service.ConversationService,
	conversationMemberService service.ConversationMemberService,
	userFriendshipService service.UserFriendshipService,
	notificationService service.NotificationService,
	userRepo repository.UserRepository, // 🆕 เพิ่ม userRepo
) *Hub {
	hub := &Hub{
		clients:                   make(map[uuid.UUID]*Client),
		userConnections:           make(map[uuid.UUID][]uuid.UUID),
		conversationSubs:          make(map[uuid.UUID][]uuid.UUID),
		userStatusSubs:            make(map[uuid.UUID][]uuid.UUID),
		handlers:                  make(map[string]MessageHandler),
		conversationService:       conversationService,
		conversationMemberService: conversationMemberService,
		userFriendshipService:     userFriendshipService,
		notificationService:       notificationService,
		userRepo:                  userRepo, // 🆕 เพิ่ม userRepo
		register:                  make(chan *Client),
		unregister:                make(chan *Client),
		broadcast:                 make(chan *BroadcastMessage, 1000), // Buffer size
		startTime:                 time.Now(),
		totalMessages:        0,
	}

	// Register handlers
	hub.registerHandlers()

	// Log services status
	log.Printf("WebSocket Hub initialized with services:")
	log.Printf("- ConversationService: %v", conversationService != nil)
	log.Printf("- NotificationService: %v", notificationService != nil)

	return hub
}

// Run starts the hub
func (h *Hub) Run(ctx context.Context) {
	log.Println("=== WebSocket Hub Run Started ===")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("WebSocket Hub: Context cancelled, shutting down")
			return

		case client := <-h.register:
			log.Printf("WebSocket Hub: Registering new client %s", client.ID)
			h.registerClient(client)

		case client := <-h.unregister:
			log.Printf("WebSocket Hub: Unregistering client %s", client.ID)
			h.unregisterClient(client)

		case message := <-h.broadcast:
			log.Printf("WebSocket Hub: Broadcasting message type: %s", message.Type)
			h.broadcastMessage(message)

		case <-ticker.C:
			log.Println("WebSocket Hub: Checking alive clients")
			h.checkAliveClients()
		}
	}
}

// GetStats returns WebSocket statistics
func (h *Hub) GetStats() map[string]interface{} {
	h.clientsMux.RLock()
	totalClients := len(h.clients)
	h.clientsMux.RUnlock()

	h.userConnectionsMux.RLock()
	totalUsers := len(h.userConnections)
	h.userConnectionsMux.RUnlock()


	h.conversationSubsMux.RLock()
	totalConversations := len(h.conversationSubs)
	h.conversationSubsMux.RUnlock()

	h.messagesSentMux.RLock()
	messages := h.totalMessages
	h.messagesSentMux.RUnlock()

	// Calculate connection distribution
	connDistribution := h.getConnectionDistribution()

	return map[string]interface{}{
		"total_connections":       totalClients,
		"unique_users":            totalUsers,
		"active_conversations":    totalConversations,
		"total_messages":          messages,
		"uptime":                  time.Since(h.startTime).String(),
		"started_at":              h.startTime,
		"connection_distribution": connDistribution,
	}
}

// getConnectionDistribution returns how many connections each user has
func (h *Hub) getConnectionDistribution() map[string]int {
	h.userConnectionsMux.RLock()
	defer h.userConnectionsMux.RUnlock()

	distribution := map[string]int{
		"single_connection":    0,
		"multiple_connections": 0,
	}

	for _, connections := range h.userConnections {
		if len(connections) == 1 {
			distribution["single_connection"]++
		} else if len(connections) > 1 {
			distribution["multiple_connections"]++
		}
	}

	return distribution
}

// IncrementMessageCount increments total message count (thread-safe)
func (h *Hub) IncrementMessageCount() {
	h.messagesSentMux.Lock()
	h.totalMessages++
	h.messagesSentMux.Unlock()
}

// registerClient registers a new client
func (h *Hub) registerClient(client *Client) {
	h.clientsMux.Lock()
	h.clients[client.ID] = client
	h.clientsMux.Unlock()

	// Add to user connections
	h.userConnectionsMux.Lock()
	isFirstConnection := len(h.userConnections[client.UserID]) == 0
	h.userConnections[client.UserID] = append(h.userConnections[client.UserID], client.ID)
	h.userConnectionsMux.Unlock()

	// Load conversations based on connection type
	if h.conversationService != nil {
		// Regular user connection - load user conversations
		go h.loadUserConversations(client)
	} else {
		log.Println("Warning: ConversationService is nil, skipping conversation loading")
	}

	// แจ้งเตือนสถานะออนไลน์ให้กับผู้ที่ subscribe
	if isFirstConnection {
		// Update presence in Redis and Database
		if h.presenceService != nil {
			if err := h.presenceService.SetUserOnline(client.UserID); err != nil {
				log.Printf("Error setting user online: %v", err)
			}
		}

		now := time.Now()
		statusData := map[string]interface{}{
			"user_id":   client.UserID.String(),
			"status":    "online",           // 🆕 เพิ่ม status field
			"online":    true,               // ✅ เก็บไว้ (backward compatible)
			"timestamp": now.Format(time.RFC3339),
		}

		// 1. แจ้งไปยังผู้ใช้ทุกคนที่ subscribe สถานะของผู้ใช้นี้
		h.userStatusSubsMux.RLock()
		subscriberIDs := h.userStatusSubs[client.UserID]
		h.userStatusSubsMux.RUnlock()

		for _, subClientID := range subscriberIDs {
			h.clientsMux.RLock()
			subClient, ok := h.clients[subClientID]
			h.clientsMux.RUnlock()

			if ok && subClientID != client.ID {
				log.Printf("Notifying client %s that user %s is online", subClientID, client.UserID)

				// ส่ง event แบบเก่า (backward compatible)
				h.sendToClient(subClient, WSResponse{
					Type:      TypeUserOnline,  // "user.online"
					Data:      statusData,
					Timestamp: now,
					Success:   true,
				})

				// ส่ง event แบบใหม่ (ตาม spec)
				h.sendToClient(subClient, WSResponse{
					Type:      TypeUserStatus,  // "user.status"
					Data:      statusData,
					Timestamp: now,
					Success:   true,
				})
			}
		}

		// 2. แจ้งสถานะของตัวเองกลับไปที่ client เพื่อให้รู้ว่าเชื่อมต่อสำเร็จ
		h.sendToClient(client, WSResponse{
			Type:      TypeUserOnline,
			Data:      statusData,
			Timestamp: now,
			Success:   true,
		})
	}

	// 3. เพิ่มส่วนนี้: ส่งสถานะของผู้ใช้อื่นที่กำลังออนไลน์อยู่ให้กับ client ใหม่
	// ทำเฉพาะเมื่อ client นี้ subscribe ผู้ใช้คนอื่นๆ
	go func() {
		// รอให้ client ได้ subscribe ผู้ใช้อื่นก่อน (รอการเรียก loadUserConversations)
		time.Sleep(1 * time.Second)

		// ดึงรายการผู้ใช้ทั้งหมดที่ออนไลน์
		h.userConnectionsMux.RLock()
		onlineUsers := make([]uuid.UUID, 0, len(h.userConnections))
		for userID, connections := range h.userConnections {
			if len(connections) > 0 && userID != client.UserID {
				onlineUsers = append(onlineUsers, userID)
			}
		}
		h.userConnectionsMux.RUnlock()

		// ส่งสถานะของผู้ใช้ที่ออนไลน์ทั้งหมดให้กับ client
		for _, onlineUserID := range onlineUsers {
			// ตรวจสอบว่า client นี้ subscribe ผู้ใช้คนนี้หรือไม่
			h.userStatusSubsMux.RLock()
			isSubscribed := false
			for _, subClientID := range h.userStatusSubs[onlineUserID] {
				if subClientID == client.ID {
					isSubscribed = true
					break
				}
			}
			h.userStatusSubsMux.RUnlock()

			// ถ้า subscribe ไว้ ส่งสถานะออนไลน์ไปให้
			if isSubscribed {
				log.Printf("Sending online status of user %s to new client %s", onlineUserID, client.ID)
				h.sendToClient(client, WSResponse{
					Type: TypeUserOnline,
					Data: map[string]interface{}{
						"user_id":   onlineUserID,
						"online":    true,
						"timestamp": time.Now(),
					},
					Timestamp: time.Now(),
					Success:   true,
				})
			}
		}
	}()

	// Send welcome message
	h.sendToClient(client, WSResponse{
		Type: TypeConnect,
		Data: map[string]interface{}{
			"message":   "Connected successfully",
			"client_id": client.ID.String(), // ส่ง client_id ไปด้วย
		},
		Timestamp: time.Now(),
		Success:   true,
	})
}

// ปรับปรุง unregisterClient ใน hub.go
func (h *Hub) unregisterClient(client *Client) {
	h.clientsMux.Lock()
	if _, ok := h.clients[client.ID]; ok {
		delete(h.clients, client.ID)
		close(client.Send)
	}
	h.clientsMux.Unlock()

	// Remove from user connections and check if this is the last connection
	h.userConnectionsMux.Lock()
	isLastConnection := false
	if connections, exists := h.userConnections[client.UserID]; exists {
		h.removeClientFromSlice(&connections, client.ID)
		if len(connections) == 0 {
			delete(h.userConnections, client.UserID)
			isLastConnection = true
		} else {
			h.userConnections[client.UserID] = connections
		}
	}
	h.userConnectionsMux.Unlock()


	// Remove from conversation subscriptions
	h.removeClientFromAllConversations(client.ID)

	// เก็บ userID ไว้ก่อนลบออกจาก subscriptions
	userID := client.UserID

	// Remove from all user status subscriptions
	h.removeClientFromAllUserStatusSubscriptions(client.ID)

	// แจ้งเตือนสถานะออฟไลน์
	if isLastConnection {
		// Update presence in Redis and Database
		if h.presenceService != nil {
			if err := h.presenceService.SetUserOffline(userID); err != nil {
				log.Printf("Error setting user offline: %v", err)
			}
		}

		now := time.Now()
		statusData := map[string]interface{}{
			"user_id":   userID.String(),
			"status":    "offline",          // 🆕 เพิ่ม status field
			"online":    false,              // ✅ เก็บไว้ (backward compatible)
			"last_seen": now.Format(time.RFC3339),  // 🆕 เพิ่ม last_seen
			"timestamp": now.Format(time.RFC3339),
		}

		// แจ้งไปยังผู้ใช้ทุกคนที่ subscribe สถานะของผู้ใช้นี้
		h.userStatusSubsMux.RLock()
		subscriberIDs := h.userStatusSubs[userID]
		h.userStatusSubsMux.RUnlock()

		for _, subClientID := range subscriberIDs {
			h.clientsMux.RLock()
			subClient, ok := h.clients[subClientID]
			h.clientsMux.RUnlock()

			if ok {
				log.Printf("Notifying client %s that user %s is offline", subClientID, userID)

				// ส่ง event แบบเก่า (backward compatible)
				h.sendToClient(subClient, WSResponse{
					Type:      TypeUserOffline,  // "user.offline"
					Data:      statusData,
					Timestamp: now,
					Success:   true,
				})

				// ส่ง event แบบใหม่ (ตาม spec)
				h.sendToClient(subClient, WSResponse{
					Type:      TypeUserStatus,  // "user.status"
					Data:      statusData,
					Timestamp: now,
					Success:   true,
				})
			}
		}
	}
}

// loadUserConversations loads and subscribes to user's conversations
func (h *Hub) loadUserConversations(client *Client) {
	// Check if service is available
	if h.conversationService == nil {
		log.Println("Error: ConversationService is nil in loadUserConversations")
		return
	}

	// Get user's conversations - ใช้พารามิเตอร์ที่ถูกต้อง
	conversations, _, err := h.conversationService.GetUserConversations(
		client.UserID, 100, 0, "", false, // เปลี่ยนจาก "last_message_at DESC" เป็น ""
	)
	if err != nil {
		log.Printf("Error loading conversations for user %s: %v", client.UserID, err)
		return
	}

	// Check if client still exists before subscribing
	h.clientsMux.RLock()
	_, exists := h.clients[client.ID]
	h.clientsMux.RUnlock()

	if !exists {
		log.Printf("Client %s disconnected before conversations loaded, skipping", client.ID)
		return
	}

	// Subscribe to each conversation
	h.conversationSubsMux.Lock()
	for _, conv := range conversations {
		h.conversationSubs[conv.ID] = append(h.conversationSubs[conv.ID], client.ID)
	}
	h.conversationSubsMux.Unlock()

	// Check again if client still exists before sending
	h.clientsMux.RLock()
	_, exists = h.clients[client.ID]
	h.clientsMux.RUnlock()

	if !exists {
		log.Printf("Client %s disconnected before sending conversations, skipping", client.ID)
		return
	}

	// สร้างรายการการสนทนาเพื่อส่งให้ client
	conversationList := make([]map[string]interface{}, len(conversations))
	for i, conv := range conversations {
		conversationList[i] = map[string]interface{}{
			"id":              conv.ID,
			"title":           conv.Title,
			"type":            conv.Type,
			"last_message_at": conv.LastMessageAt,
			"unread_count":    conv.UnreadCount, // ถ้ามี
			"is_subscribed":   i < 5,            // บอกว่า subscribe แล้วหรือยัง
		}
	}

	// ส่งรายการการสนทนาให้ client
	h.sendToClient(client, WSResponse{
		Type:      "conversation.list",
		Data:      conversationList,
		Timestamp: time.Now(),
		Success:   true,
	})

	log.Printf("User %s loaded %d conversations, subscribed to %d",
		client.UserID, len(conversations), min(5, len(conversations)))
}


// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// checkAliveClients checks and removes dead connections
func (h *Hub) checkAliveClients() {
	h.clientsMux.RLock()
	clientsCopy := make([]*Client, 0, len(h.clients))
	for _, client := range h.clients {
		clientsCopy = append(clientsCopy, client)
	}
	h.clientsMux.RUnlock()

	for _, client := range clientsCopy {
		if time.Since(client.LastPingTime) > 90*time.Second {
			h.unregister <- client
		}
	}
}

// GetAllConnections returns detailed information about all active WebSocket connections
func (h *Hub) GetAllConnections() map[string]interface{} {
	h.clientsMux.RLock()
	h.userConnectionsMux.RLock()
	h.conversationSubsMux.RLock()
	h.userStatusSubsMux.RLock() // เพิ่ม lock สำหรับ userStatusSubs
	defer h.clientsMux.RUnlock()
	defer h.userConnectionsMux.RUnlock()
	defer h.conversationSubsMux.RUnlock()
	defer h.userStatusSubsMux.RUnlock() // เพิ่ม unlock

	// สร้างข้อมูลรายละเอียดของผู้ใช้ที่เชื่อมต่อ
	userConnections := make(map[string][]string)
	for userID, clientIDs := range h.userConnections {
		clientIDsStr := make([]string, len(clientIDs))
		for i, cid := range clientIDs {
			clientIDsStr[i] = cid.String()
		}
		userConnections[userID.String()] = clientIDsStr
	}

	// สร้างข้อมูลรายละเอียดของการสนทนาที่มีการ subscribe
	conversationSubscriptions := make(map[string][]string)
	for convID, clientIDs := range h.conversationSubs {
		clientIDsStr := make([]string, len(clientIDs))
		for i, cid := range clientIDs {
			clientIDsStr[i] = cid.String()
		}
		conversationSubscriptions[convID.String()] = clientIDsStr
	}

	// สร้างข้อมูลรายละเอียดของแต่ละ client
	clientDetails := make(map[string]map[string]interface{})
	for clientID, client := range h.clients {
		clientDetails[clientID.String()] = map[string]interface{}{
			"user_id":        client.UserID.String(),
			"business_id":    client.BusinessID,
			"is_alive":       client.IsAlive,
			"last_ping_time": client.LastPingTime,
		}
	}

	// เพิ่มส่วนนี้: สร้างข้อมูลรายละเอียดของการ subscribe สถานะผู้ใช้
	userStatusSubscriptions := make(map[string][]string)
	for userID, clientIDs := range h.userStatusSubs {
		clientIDsStr := make([]string, len(clientIDs))
		for i, cid := range clientIDs {
			clientIDsStr[i] = cid.String()
		}
		userStatusSubscriptions[userID.String()] = clientIDsStr
	}

	return map[string]interface{}{
		"total_clients":                  len(h.clients),
		"clients":                        clientDetails,
		"user_connections":               userConnections,
		"conversation_subscriptions":     conversationSubscriptions,
		"user_status_subscriptions":      userStatusSubscriptions, // เพิ่มส่วนนี้
		"total_users_connected":          len(h.userConnections),
		"total_subscribed_conversations": len(h.conversationSubs),
		"total_status_subscriptions":     len(h.userStatusSubs), // เพิ่มส่วนนี้
		"timestamp":                      time.Now(),
	}
}

// GetConversationSubscribers returns details about subscribers of a specific conversation
func (h *Hub) GetConversationSubscribers(conversationID uuid.UUID) map[string]interface{} {
	h.conversationSubsMux.RLock()
	defer h.conversationSubsMux.RUnlock()

	subscriberIDs := h.conversationSubs[conversationID]
	subscriberCount := len(subscriberIDs)

	// สร้างข้อมูลรายละเอียดของผู้ subscribe
	subscribers := make([]map[string]interface{}, 0, subscriberCount)

	h.clientsMux.RLock()
	defer h.clientsMux.RUnlock()

	for _, clientID := range subscriberIDs {
		client, exists := h.clients[clientID]
		if exists {
			subscribers = append(subscribers, map[string]interface{}{
				"client_id":      clientID.String(),
				"user_id":        client.UserID.String(),
				"is_alive":       client.IsAlive,
				"last_ping_time": client.LastPingTime,
			})
		}
	}

	return map[string]interface{}{
		"conversation_id":  conversationID.String(),
		"subscriber_count": subscriberCount,
		"subscribers":      subscribers,
		"timestamp":        time.Now(),
	}
}

// GetUserStatusSubscribers returns details about subscribers of a specific user's status
func (h *Hub) GetUserStatusSubscribers(userID uuid.UUID) map[string]interface{} {
	h.userStatusSubsMux.RLock()
	defer h.userStatusSubsMux.RUnlock()

	subscriberIDs := h.userStatusSubs[userID]
	subscriberCount := len(subscriberIDs)

	// สร้างข้อมูลรายละเอียดของผู้ subscribe
	subscribers := make([]map[string]interface{}, 0, subscriberCount)

	h.clientsMux.RLock()
	defer h.clientsMux.RUnlock()

	for _, clientID := range subscriberIDs {
		client, exists := h.clients[clientID]
		if exists {
			subscribers = append(subscribers, map[string]interface{}{
				"client_id":      clientID.String(),
				"user_id":        client.UserID.String(),
				"is_alive":       client.IsAlive,
				"last_ping_time": client.LastPingTime,
			})
		}
	}

	// ตรวจสอบสถานะออนไลน์ของผู้ใช้
	isOnline := h.isUserOnline(userID)

	return map[string]interface{}{
		"user_id":          userID.String(),
		"is_online":        isOnline,
		"subscriber_count": subscriberCount,
		"subscribers":      subscribers,
		"timestamp":        time.Now(),
	}
}

func (h *Hub) SetNotificationService(notificationService service.NotificationService) {
	h.notificationService = notificationService
	log.Println("NotificationService has been set in WebSocket Hub")
}

func (h *Hub) SetPresenceService(presenceService service.PresenceService) {
	h.presenceService = presenceService
	log.Println("PresenceService has been set in WebSocket Hub")
}

// ปรับปรุง subscribeToUserStatus ใน hub.go
func (h *Hub) subscribeToUserStatus(clientID, targetUserID uuid.UUID) {
	h.userStatusSubsMux.Lock()

	// สร้าง slice ใหม่ถ้ายังไม่มี
	if _, exists := h.userStatusSubs[targetUserID]; !exists {
		h.userStatusSubs[targetUserID] = []uuid.UUID{}
	}

	// ตรวจสอบว่า clientID มีอยู่ในรายการแล้วหรือไม่
	alreadySubscribed := false
	for _, id := range h.userStatusSubs[targetUserID] {
		if id == clientID {
			alreadySubscribed = true
			break
		}
	}

	// เพิ่ม clientID ใหม่ถ้ายังไม่มี
	if !alreadySubscribed {
		h.userStatusSubs[targetUserID] = append(h.userStatusSubs[targetUserID], clientID)
		log.Printf("Client %s subscribed to status of user %s", clientID, targetUserID)
	}

	h.userStatusSubsMux.Unlock()

	// ส่งสถานะปัจจุบันกลับไปทันที
	h.clientsMux.RLock()
	client, ok := h.clients[clientID]
	h.clientsMux.RUnlock()

	if ok {
		// ตรวจสอบว่าผู้ใช้เป้าหมายออนไลน์อยู่หรือไม่
		isOnline := h.isUserOnline(targetUserID)

		statusType := TypeUserOffline
		if isOnline {
			statusType = TypeUserOnline
		}

		log.Printf("Sending current status of user %s (online=%v) to client %s",
			targetUserID, isOnline, clientID)

		h.sendToClient(client, WSResponse{
			Type: statusType,
			Data: map[string]interface{}{
				"user_id":   targetUserID,
				"online":    isOnline,
				"timestamp": time.Now(),
			},
			Timestamp: time.Now(),
			Success:   true,
		})
	}
}

// ยกเลิกการ subscribe สถานะผู้ใช้
func (h *Hub) unsubscribeFromUserStatus(clientID, targetUserID uuid.UUID) {
	h.userStatusSubsMux.Lock()
	defer h.userStatusSubsMux.Unlock()

	subs := h.userStatusSubs[targetUserID]
	for i, id := range subs {
		if id == clientID {
			// ลบ clientID ออกจาก slice
			h.userStatusSubs[targetUserID] = append(subs[:i], subs[i+1:]...)
			break
		}
	}

	// ถ้าไม่มี subscribers เหลือ ลบ key ทิ้ง
	if len(h.userStatusSubs[targetUserID]) == 0 {
		delete(h.userStatusSubs, targetUserID)
	}
}

// ปรับปรุง isUserOnline ใน hub.go - เพิ่ม logging
func (h *Hub) isUserOnline(userID uuid.UUID) bool {
	h.userConnectionsMux.RLock()
	defer h.userConnectionsMux.RUnlock()

	connections, exists := h.userConnections[userID]
	isOnline := exists && len(connections) > 0
	log.Printf("Checking if user %s is online: %v (connections: %d)",
		userID, isOnline, len(connections))
	return isOnline
}
