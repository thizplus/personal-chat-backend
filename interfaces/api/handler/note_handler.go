// interfaces/api/handler/note_handler.go
package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/service"
	"github.com/thizplus/gofiber-chat-api/interfaces/api/middleware"
	"github.com/thizplus/gofiber-chat-api/pkg/utils"
)

type NoteHandler struct {
	noteService service.NoteService
}

func NewNoteHandler(noteService service.NoteService) *NoteHandler {
	return &NoteHandler{
		noteService: noteService,
	}
}

// CreateNote สร้างบันทึกใหม่
func (h *NoteHandler) CreateNote(c *fiber.Ctx) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	// Parse request body
	var input struct {
		ConversationID *string  `json:"conversation_id,omitempty"` // Optional conversation_id
		Title          string   `json:"title"`
		Content        string   `json:"content"`
		Tags           []string `json:"tags"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body: " + err.Error(),
		})
	}

	// แปลง conversation_id จาก string เป็น UUID
	var conversationIDPtr *uuid.UUID
	if input.ConversationID != nil && *input.ConversationID != "" {
		conversationUUID, err := utils.ParseUUID(*input.ConversationID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid conversation_id format",
			})
		}
		conversationIDPtr = &conversationUUID
	}

	// สร้างบันทึก
	note, err := h.noteService.CreateNote(userID, conversationIDPtr, input.Title, input.Content, input.Tags)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if err.Error() == "user is not a member of this conversation" {
			statusCode = fiber.StatusForbidden
		}
		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Note created successfully",
		"data":    note,
	})
}

// GetNote ดึงข้อมูลบันทึก
func (h *NoteHandler) GetNote(c *fiber.Ctx) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	noteID, err := utils.ParseUUIDParam(c, "id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid note ID: " + err.Error(),
		})
	}

	note, err := h.noteService.GetNote(noteID, userID)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if err.Error() == "note not found" {
			statusCode = fiber.StatusNotFound
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    note,
	})
}

// GetNotes ดึงรายการบันทึกของผู้ใช้
// รองรับ query parameters:
// - conversation_id: ดึงเฉพาะ notes ของ conversation นั้น
// - scope=global: ดึงเฉพาะ global notes (conversation_id IS NULL)
// - scope=all หรือไม่มี: ดึงทั้งหมด (default)
func (h *NoteHandler) GetNotes(c *fiber.Ctx) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	limit := c.QueryInt("limit", 20)
	if limit > 100 {
		limit = 100
	}
	offset := c.QueryInt("offset", 0)

	// ตรวจสอบ query parameters
	conversationIDStr := c.Query("conversation_id")
	scope := c.Query("scope") // "global", "all", หรือ ""

	var notes []*models.Note
	var total int64

	// กรณีที่ระบุ conversation_id
	if conversationIDStr != "" {
		conversationID, err := utils.ParseUUID(conversationIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid conversation_id format",
			})
		}

		notes, total, err = h.noteService.GetConversationNotes(userID, conversationID, limit, offset)
		if err != nil {
			statusCode := fiber.StatusInternalServerError
			if err.Error() == "user is not a member of this conversation" {
				statusCode = fiber.StatusForbidden
			}
			return c.Status(statusCode).JSON(fiber.Map{
				"success": false,
				"message": err.Error(),
			})
		}
	} else if scope == "global" {
		// กรณีที่ขอเฉพาะ global notes
		notes, total, err = h.noteService.GetGlobalNotes(userID, limit, offset)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": err.Error(),
			})
		}
	} else {
		// กรณี default: ดึงทั้งหมด
		notes, total, err = h.noteService.GetUserNotes(userID, limit, offset)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": err.Error(),
			})
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"notes": notes,
			"pagination": fiber.Map{
				"total":  total,
				"limit":  limit,
				"offset": offset,
			},
		},
	})
}

// UpdateNote อัปเดตบันทึก
func (h *NoteHandler) UpdateNote(c *fiber.Ctx) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	noteID, err := utils.ParseUUIDParam(c, "id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid note ID: " + err.Error(),
		})
	}

	// Parse request body
	var input struct {
		Title   string   `json:"title"`
		Content string   `json:"content"`
		Tags    []string `json:"tags"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body: " + err.Error(),
		})
	}

	note, err := h.noteService.UpdateNote(noteID, userID, input.Title, input.Content, input.Tags)
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if err.Error() == "note not found" {
			statusCode = fiber.StatusNotFound
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Note updated successfully",
		"data":    note,
	})
}

// DeleteNote ลบบันทึก
func (h *NoteHandler) DeleteNote(c *fiber.Ctx) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	noteID, err := utils.ParseUUIDParam(c, "id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid note ID: " + err.Error(),
		})
	}

	if err := h.noteService.DeleteNote(noteID, userID); err != nil {
		statusCode := fiber.StatusInternalServerError
		if err.Error() == "note not found" {
			statusCode = fiber.StatusNotFound
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Note deleted successfully",
	})
}

// GetPinnedNotes ดึงรายการบันทึกที่ปักหมุด
func (h *NoteHandler) GetPinnedNotes(c *fiber.Ctx) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	limit := c.QueryInt("limit", 20)
	if limit > 100 {
		limit = 100
	}
	offset := c.QueryInt("offset", 0)

	notes, total, err := h.noteService.GetPinnedNotes(userID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"notes": notes,
			"pagination": fiber.Map{
				"total":  total,
				"limit":  limit,
				"offset": offset,
			},
		},
	})
}

// SearchNotes ค้นหาบันทึก
func (h *NoteHandler) SearchNotes(c *fiber.Ctx) error {
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
			"message": "Search query (q) is required",
		})
	}

	limit := c.QueryInt("limit", 20)
	if limit > 100 {
		limit = 100
	}
	offset := c.QueryInt("offset", 0)

	notes, total, err := h.noteService.SearchNotes(userID, query, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"notes": notes,
			"query": query,
			"pagination": fiber.Map{
				"total":  total,
				"limit":  limit,
				"offset": offset,
			},
		},
	})
}

// GetNotesByTag ดึงบันทึกตาม tag
func (h *NoteHandler) GetNotesByTag(c *fiber.Ctx) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	tag := c.Query("tag")
	if tag == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Tag query parameter is required",
		})
	}

	limit := c.QueryInt("limit", 20)
	if limit > 100 {
		limit = 100
	}
	offset := c.QueryInt("offset", 0)

	notes, total, err := h.noteService.GetNotesByTag(userID, tag, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"notes": notes,
			"tag":   tag,
			"pagination": fiber.Map{
				"total":  total,
				"limit":  limit,
				"offset": offset,
			},
		},
	})
}

// PinNote ปักหมุดบันทึก
func (h *NoteHandler) PinNote(c *fiber.Ctx) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	noteID, err := utils.ParseUUIDParam(c, "id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid note ID: " + err.Error(),
		})
	}

	if err := h.noteService.PinNote(noteID, userID); err != nil {
		statusCode := fiber.StatusInternalServerError
		if err.Error() == "note not found" {
			statusCode = fiber.StatusNotFound
		} else if err.Error() == "note is already pinned" {
			statusCode = fiber.StatusBadRequest
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Note pinned successfully",
	})
}

// UnpinNote ยกเลิกการปักหมุดบันทึก
func (h *NoteHandler) UnpinNote(c *fiber.Ctx) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: " + err.Error(),
		})
	}

	noteID, err := utils.ParseUUIDParam(c, "id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid note ID: " + err.Error(),
		})
	}

	if err := h.noteService.UnpinNote(noteID, userID); err != nil {
		statusCode := fiber.StatusInternalServerError
		if err.Error() == "note not found" {
			statusCode = fiber.StatusNotFound
		} else if err.Error() == "note is not pinned" {
			statusCode = fiber.StatusBadRequest
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Note unpinned successfully",
	})
}
