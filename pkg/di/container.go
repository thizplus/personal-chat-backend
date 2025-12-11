// pkg/di/container.go
package di

import (
	"log"

	"github.com/go-redis/redis/v8"
	"github.com/thizplus/gofiber-chat-api/application/serviceimpl"
	"github.com/thizplus/gofiber-chat-api/domain/port"
	"github.com/thizplus/gofiber-chat-api/domain/repository"
	"github.com/thizplus/gofiber-chat-api/domain/service"
	"github.com/thizplus/gofiber-chat-api/infrastructure/adapter"
	"github.com/thizplus/gofiber-chat-api/infrastructure/persistence/postgres"
	"github.com/thizplus/gofiber-chat-api/interfaces/api/handler"
	"github.com/thizplus/gofiber-chat-api/interfaces/websocket"
	"github.com/thizplus/gofiber-chat-api/pkg/scheduler"

	"gorm.io/gorm"
)

// Container เก็บ dependencies ทั้งหมดของแอปพลิเคชัน
type Container struct {
	// Repositories
	UserRepo                   repository.UserRepository
	RefreshTokenRepo           repository.RefreshTokenRepository
	TokenBlacklistRepo         repository.TokenBlacklistRepository
	UserFriendshipRepo         repository.UserFriendshipRepository
	ConversationRepo           repository.ConversationRepository
	ConversationMemberRepo     repository.ConversationMemberRepository
	MessageRepo                repository.MessageRepository
	MessageReadRepo            repository.MessageReadRepository
	MessageMentionRepo         repository.MessageMentionRepository
	StickerRepo                repository.StickerRepository
	FileUploadRepo             repository.FileUploadRepository
	GroupActivityRepo          repository.GroupActivityRepository
	ScheduledMessageRepo       repository.ScheduledMessageRepository
	NoteRepo                   repository.NoteRepository

	// WebSocket Components
	WebSocketHub  *websocket.Hub
	WebSocketPort port.WebSocketPort

	// Services
	StorageService                service.FileStorageService
	AuthService                   service.AuthService
	UserService                   service.UserService
	UserFriendshipService         service.UserFriendshipService
	ConversationService           service.ConversationService
	ConversationMemberService     service.ConversationMemberService
	MessageService                service.MessageService
	MessageReadService            service.MessageReadService
	StickerService                service.StickerService
	NotificationService           service.NotificationService
	PresenceService               service.PresenceService
	GroupActivityService          service.GroupActivityService
	ScheduledMessageService       service.ScheduledMessageService
	NoteService                   service.NoteService

	// Handlers
	AuthHandler                   *handler.AuthHandler
	UserHandler                   *handler.UserHandler
	FileHandler                   *handler.FileHandler
	UserFriendshipHandler         *handler.UserFriendshipHandler
	ConversationHandler           *handler.ConversationHandler
	ConversationMemberHandler     *handler.ConversationMemberHandler
	MessageHandler                *handler.MessageHandler
	MessageReadHandler            *handler.MessageReadHandler
	MentionHandler                *handler.MentionHandler
	StickerHandler                *handler.StickerHandler
	SearchHandler                 *handler.SearchHandler
	PresenceHandler               *handler.PresenceHandler
	ScheduledMessageHandler       *handler.ScheduledMessageHandler
	NoteHandler                   *handler.NoteHandler

	// Scheduler & Background Jobs
	RedisClient                    *redis.Client
	FileCleanupScheduler           *scheduler.FileCleanupScheduler
	ScheduledMessageProcessor      *scheduler.ScheduledMessageProcessor
}

// NewContainer สร้าง container ใหม่พร้อมกับ dependencies ทั้งหมด
func NewContainer(db *gorm.DB, storageService service.FileStorageService, redisClient *redis.Client) (*Container, error) {
	container := &Container{
		StorageService: storageService,
		RedisClient:    redisClient,
	}

	// สร้าง repositories
	container.UserRepo = postgres.NewUserRepository(db)
	container.RefreshTokenRepo = postgres.NewRefreshTokenRepository(db)
	container.TokenBlacklistRepo = postgres.NewTokenBlacklistRepository(db)
	container.UserFriendshipRepo = postgres.NewUserFriendshipRepository(db)
	container.ConversationRepo = postgres.NewConversationRepository(db)
	container.ConversationMemberRepo = postgres.NewConversationMemberRepository(db)
	container.MessageRepo = postgres.NewMessageRepository(db)
	container.MessageReadRepo = postgres.NewMessageReadRepository(db)
	container.MessageMentionRepo = postgres.NewMessageMentionRepository(db)
	container.StickerRepo = postgres.NewStickerRepository(db)
	container.FileUploadRepo = postgres.NewFileUploadRepository(db)
	container.GroupActivityRepo = postgres.NewGroupActivityRepository(db)
	container.ScheduledMessageRepo = postgres.NewScheduledMessageRepository(db)
	container.NoteRepo = postgres.NewNoteRepository(db)

	log.Println("เชื่อมต่อกับบริการจัดเก็บไฟล์สำเร็จ")

	// สร้าง basic services
	container.AuthService = serviceimpl.NewAuthService(
		container.UserRepo,
		container.RefreshTokenRepo,
		container.TokenBlacklistRepo,
	)

	container.UserService = serviceimpl.NewUserService(container.UserRepo)
	container.UserFriendshipService = serviceimpl.NewUserFriendshipService(
		container.UserFriendshipRepo,
		container.UserRepo,
	)
	container.ConversationService = serviceimpl.NewConversationService(
		container.ConversationRepo,
		container.UserRepo,
		container.MessageRepo,
		container.MessageMentionRepo,
	)
	container.ConversationMemberService = serviceimpl.NewConversationMemberService(
		container.ConversationRepo,
		container.UserRepo,
		container.MessageRepo,
	)

	container.MessageReadService = serviceimpl.NewMessageReadService(
		container.MessageRepo,
		container.MessageReadRepo,
		container.ConversationRepo,
	)


	container.StickerService = serviceimpl.NewStickerService(
		container.StickerRepo,
		container.StorageService,
	)

	container.PresenceService = serviceimpl.NewPresenceService(
		redisClient,
		container.UserRepo,
		container.UserFriendshipRepo,
	)

	// MessageService และ ScheduledMessageService จะถูกสร้างหลัง NotificationService (ย้ายไปด้านล่าง)

	container.NoteService = serviceimpl.NewNoteService(
		container.NoteRepo,
		container.ConversationMemberRepo,
	)

	// สร้าง WebSocket Hub ที่มีเฉพาะ services ที่จำเป็น
	container.WebSocketHub = websocket.NewHub(
		container.ConversationService,
		container.ConversationMemberService,
		container.UserFriendshipService,
		nil,                 // NotificationService จะถูกตั้งค่าภายหลัง
		container.UserRepo, // 🆕 เพิ่ม UserRepo สำหรับ typing user info
	)

	// สร้าง WebSocketAdapter
	container.WebSocketPort = adapter.NewWebSocketAdapter(container.WebSocketHub)

	// สร้าง NotificationService
	container.NotificationService = serviceimpl.NewNotificationService(
		container.WebSocketPort,
		container.UserRepo,
		container.MessageRepo,
		container.ConversationRepo,
	)

	// ตั้งค่า NotificationService ใน Hub
	container.WebSocketHub.SetNotificationService(container.NotificationService)

	// สร้าง GroupActivityService (ต้องสร้างหลัง NotificationService)
	container.GroupActivityService = serviceimpl.NewGroupActivityService(
		container.GroupActivityRepo,
		container.ConversationRepo,
		container.NotificationService,
	)

	// สร้าง MessageService (ต้องสร้างหลัง NotificationService)
	container.MessageService = serviceimpl.NewMessageService(
		container.MessageRepo,
		container.MessageReadRepo,
		container.ConversationRepo,
		container.UserRepo,
		container.NotificationService,
		container.MessageMentionRepo,
	)

	// สร้าง ScheduledMessageService (ต้องสร้างหลัง MessageService และ NotificationService)
	container.ScheduledMessageService = serviceimpl.NewScheduledMessageService(
		container.ScheduledMessageRepo,
		container.ConversationRepo,
		container.MessageService,
		container.NotificationService, // ✅ เพิ่มเพื่อส่ง WebSocket notification เมื่อส่งข้อความตั้งเวลา
	)

	// สร้าง handlers
	container.AuthHandler = handler.NewAuthHandler(container.AuthService)
	container.UserHandler = handler.NewUserHandler(container.UserService, container.AuthService, container.StorageService)
	container.FileHandler = handler.NewFileHandler(container.StorageService, container.FileUploadRepo)
	container.UserFriendshipHandler = handler.NewUserFriendshipHandler(container.UserFriendshipService, container.UserService, container.ConversationMemberService, container.NotificationService)
	container.ConversationHandler = handler.NewConversationHandler(container.ConversationService, container.NotificationService, container.MessageReadService, container.GroupActivityService, container.ConversationRepo, container.MessageService)
	container.ConversationMemberHandler = handler.NewConversationMemberHandler(container.ConversationMemberService, container.NotificationService, container.GroupActivityService)
	container.MessageHandler = handler.NewMessageHandler(container.MessageService, container.NotificationService, container.ConversationMemberService, container.ConversationRepo, container.UserFriendshipService)
	container.MessageReadHandler = handler.NewMessageReadHandler(container.MessageReadService, container.NotificationService, container.MessageRepo)
	container.MentionHandler = handler.NewMentionHandler(container.MessageMentionRepo)
	container.StickerHandler = handler.NewStickerHandler(container.StickerService)
	container.SearchHandler = handler.NewSearchHandler(container.UserService, container.UserFriendshipService)
	container.PresenceHandler = handler.NewPresenceHandler(container.PresenceService)
	container.ScheduledMessageHandler = handler.NewScheduledMessageHandler(container.ScheduledMessageService)
	container.NoteHandler = handler.NewNoteHandler(container.NoteService, container.WebSocketPort)

	// สร้าง background jobs
	container.FileCleanupScheduler = scheduler.NewFileCleanupScheduler(
		container.FileUploadRepo,
		container.StorageService,
	)

	container.ScheduledMessageProcessor = scheduler.NewScheduledMessageProcessor(
		container.ScheduledMessageService,
	)

	// เชื่อมต่อ processor กับ service สำหรับ precise timing
	// (ต้องทำหลังจากสร้างทั้งสองแล้ว)
	container.ScheduledMessageService.SetProcessor(container.ScheduledMessageProcessor)

	return container, nil
}
