// domain/service/note_service.go
package service

import (
	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
)

// NoteService เป็น interface ที่กำหนดฟังก์ชันของ Note Service
type NoteService interface {
	// CRUD operations
	CreateNote(userID uuid.UUID, conversationID *uuid.UUID, title, content string, tags []string) (*models.Note, error)
	GetNote(id, userID uuid.UUID) (*models.Note, error)
	UpdateNote(id, userID uuid.UUID, title, content string, tags []string) (*models.Note, error)
	DeleteNote(id, userID uuid.UUID) error

	// Query operations
	GetUserNotes(userID uuid.UUID, limit, offset int) ([]*models.Note, int64, error)
	GetPinnedNotes(userID uuid.UUID, limit, offset int) ([]*models.Note, int64, error)
	SearchNotes(userID uuid.UUID, query string, limit, offset int) ([]*models.Note, int64, error)
	GetNotesByTag(userID uuid.UUID, tag string, limit, offset int) ([]*models.Note, int64, error)

	// Conversation-scoped query operations
	GetConversationNotes(userID, conversationID uuid.UUID, limit, offset int) ([]*models.Note, int64, error)
	GetGlobalNotes(userID uuid.UUID, limit, offset int) ([]*models.Note, int64, error)

	// Pin operations
	PinNote(id, userID uuid.UUID) error
	UnpinNote(id, userID uuid.UUID) error
}
