// pkg/app/app.go - แก้ไขโดยไม่แตะต้อง SetupRoutes
package app

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/thizplus/gofiber-chat-api/interfaces/api/middleware"
	"github.com/thizplus/gofiber-chat-api/interfaces/api/routes"
	"github.com/thizplus/gofiber-chat-api/interfaces/websocket"
	"github.com/thizplus/gofiber-chat-api/pkg/di"
	// เพิ่ม imports สำหรับ websocket
)

// SetupApp สร้างและตั้งค่า Fiber app
func SetupApp(container *di.Container) *fiber.App {
	// สร้าง Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError

			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}

			return c.Status(code).JSON(fiber.Map{
				"success": false,
				"message": err.Error(),
			})
		},
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	})

	// ใช้ middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} ${latency}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Requested-With",
		ExposeHeaders:    "Content-Length,Content-Type",
		AllowCredentials: false,
		MaxAge:           86400, // 24 ชั่วโมง
	}))
	app.Use(compress.New())

	// ตั้งค่าเส้นทาง API
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "ยินดีต้อนรับสู่ Line Official API",
			"status":  "online",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// กำหนดเส้นทางทั้งหมด (ไม่แก้ไข - ใช้แบบเดิม)
	routes.SetupRoutes(
		app,
		container.AuthHandler,
		container.FileHandler,
		container.UserFriendshipHandler,
		container.UserHandler,
		container.ConversationHandler,
		container.ConversationMemberHandler,
		container.MessageHandler,
		container.MessageReadHandler,
		container.MentionHandler,
		container.ScheduledMessageHandler,
		container.StickerHandler,
		container.NoteHandler,
		container.SearchHandler,
		container.PresenceHandler,
	)

	// เพิ่ม WebSocket routes แยกต่างหาก (หลังจาก SetupRoutes)
	authMiddleware := middleware.NewAuthMiddleware(container.AuthService)
	websocket.RegisterWebSocketRoutes(app, container.WebSocketHub, authMiddleware.Protected())

	return app
}
