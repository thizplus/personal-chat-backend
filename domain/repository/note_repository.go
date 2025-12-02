// domain/repository/note_repository.go
package repository

import (
	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
)

type NoteRepository interface {
	// CRUD operations
	Create(note *models.Note) error
	GetByID(id, userID uuid.UUID) (*models.Note, error)
	Update(note *models.Note) error
	Delete(id, userID uuid.UUID) error

	// Query operations
	FindByUserID(userID uuid.UUID, limit, offset int) ([]*models.Note, int64, error)
	FindPinnedByUserID(userID uuid.UUID, limit, offset int) ([]*models.Note, int64, error)
	SearchNotes(userID uuid.UUID, query string, limit, offset int) ([]*models.Note, int64, error)
	FindByTag(userID uuid.UUID, tag string, limit, offset int) ([]*models.Note, int64, error)

	// Conversation-scoped query operations
	FindByConversationID(userID, conversationID uuid.UUID, limit, offset int) ([]*models.Note, int64, error)
	FindGlobalNotes(userID uuid.UUID, limit, offset int) ([]*models.Note, int64, error) // conversation_id IS NULL

	// Pin operations
	PinNote(id, userID uuid.UUID) error
	UnpinNote(id, userID uuid.UUID) error
}
