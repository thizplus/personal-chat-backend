// application/serviceimpl/note_service.go
package serviceimpl

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/repository"
	"github.com/thizplus/gofiber-chat-api/domain/service"
	"github.com/thizplus/gofiber-chat-api/domain/types"
)

type noteService struct {
	noteRepo              repository.NoteRepository
	conversationMemberRepo repository.ConversationMemberRepository
}

// NewNoteService สร้าง instance ใหม่ของ NoteService
func NewNoteService(noteRepo repository.NoteRepository, conversationMemberRepo repository.ConversationMemberRepository) service.NoteService {
	return &noteService{
		noteRepo:              noteRepo,
		conversationMemberRepo: conversationMemberRepo,
	}
}

// CreateNote สร้างบันทึกใหม่
func (s *noteService) CreateNote(userID uuid.UUID, conversationID *uuid.UUID, title, content string, tags []string, visibility models.NoteVisibility) (*models.Note, error) {
	// ถ้ามี conversation_id ให้ตรวจสอบว่าผู้ใช้เป็นสมาชิกของ conversation นั้น
	if conversationID != nil {
		member, err := s.conversationMemberRepo.GetByConversationAndUserID(*conversationID, userID)
		if err != nil {
			return nil, err
		}
		if member == nil {
			return nil, errors.New("user is not a member of this conversation")
		}
	}

	// แปลง tags เป็น JSONB
	var tagsJSON types.JSONB
	if tags != nil && len(tags) > 0 {
		tagsArray := make([]interface{}, len(tags))
		for i, tag := range tags {
			tagsArray[i] = tag
		}
		tagsJSON = types.JSONB{"data": tagsArray}
	} else {
		tagsJSON = types.JSONB{}
	}

	// กำหนด visibility (shared ใช้ได้เฉพาะ conversation notes)
	if visibility == "" {
		visibility = models.NoteVisibilityPrivate
	}
	if conversationID == nil && visibility == models.NoteVisibilityShared {
		visibility = models.NoteVisibilityPrivate // ไม่มี conversation → ต้องเป็น private
	}

	note := &models.Note{
		ID:             uuid.New(),
		UserID:         userID,
		ConversationID: conversationID,
		Title:          title,
		Content:        content,
		Tags:           tagsJSON,
		IsPinned:       false,
		Visibility:     visibility,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.noteRepo.Create(note); err != nil {
		return nil, err
	}

	return note, nil
}

// GetNote ดึงข้อมูลบันทึก
func (s *noteService) GetNote(id, userID uuid.UUID) (*models.Note, error) {
	note, err := s.noteRepo.GetByID(id, userID)
	if err != nil {
		return nil, err
	}
	if note == nil {
		return nil, errors.New("note not found")
	}

	return note, nil
}

// UpdateNote อัปเดตบันทึก
func (s *noteService) UpdateNote(id, userID uuid.UUID, title, content string, tags []string, visibility *models.NoteVisibility) (*models.Note, error) {
	// ดึงบันทึกเดิม
	note, err := s.noteRepo.GetByID(id, userID)
	if err != nil {
		return nil, err
	}
	if note == nil {
		return nil, errors.New("note not found")
	}

	// อัปเดตข้อมูล
	note.Title = title
	note.Content = content

	// แปลง tags เป็น JSONB
	if tags != nil {
		if len(tags) > 0 {
			tagsArray := make([]interface{}, len(tags))
			for i, tag := range tags {
				tagsArray[i] = tag
			}
			note.Tags = types.JSONB{"data": tagsArray}
		} else {
			note.Tags = types.JSONB{}
		}
	}

	// อัปเดต visibility (ถ้ามีการส่งมา)
	if visibility != nil {
		// shared ใช้ได้เฉพาะ conversation notes
		if note.ConversationID == nil && *visibility == models.NoteVisibilityShared {
			// ไม่มี conversation → ไม่เปลี่ยนเป็น shared
		} else {
			note.Visibility = *visibility
		}
	}

	if err := s.noteRepo.Update(note); err != nil {
		return nil, err
	}

	return note, nil
}

// DeleteNote ลบบันทึก
func (s *noteService) DeleteNote(id, userID uuid.UUID) error {
	// ตรวจสอบว่าบันทึกมีอยู่และเป็นของผู้ใช้
	note, err := s.noteRepo.GetByID(id, userID)
	if err != nil {
		return err
	}
	if note == nil {
		return errors.New("note not found")
	}

	return s.noteRepo.Delete(id, userID)
}

// GetUserNotes ดึงรายการบันทึกของผู้ใช้
func (s *noteService) GetUserNotes(userID uuid.UUID, limit, offset int) ([]*models.Note, int64, error) {
	return s.noteRepo.FindByUserID(userID, limit, offset)
}

// GetPinnedNotes ดึงรายการบันทึกที่ปักหมุด
func (s *noteService) GetPinnedNotes(userID uuid.UUID, limit, offset int) ([]*models.Note, int64, error) {
	return s.noteRepo.FindPinnedByUserID(userID, limit, offset)
}

// SearchNotes ค้นหาบันทึก
func (s *noteService) SearchNotes(userID uuid.UUID, query string, limit, offset int) ([]*models.Note, int64, error) {
	if query == "" {
		return nil, 0, errors.New("search query cannot be empty")
	}

	return s.noteRepo.SearchNotes(userID, query, limit, offset)
}

// GetNotesByTag ดึงบันทึกตาม tag
func (s *noteService) GetNotesByTag(userID uuid.UUID, tag string, limit, offset int) ([]*models.Note, int64, error) {
	if tag == "" {
		return nil, 0, errors.New("tag cannot be empty")
	}

	return s.noteRepo.FindByTag(userID, tag, limit, offset)
}

// PinNote ปักหมุดบันทึก
func (s *noteService) PinNote(id, userID uuid.UUID) error {
	// ตรวจสอบว่าบันทึกมีอยู่และเป็นของผู้ใช้
	note, err := s.noteRepo.GetByID(id, userID)
	if err != nil {
		return err
	}
	if note == nil {
		return errors.New("note not found")
	}

	if note.IsPinned {
		return errors.New("note is already pinned")
	}

	return s.noteRepo.PinNote(id, userID)
}

// UnpinNote ยกเลิกการปักหมุดบันทึก
func (s *noteService) UnpinNote(id, userID uuid.UUID) error {
	// ตรวจสอบว่าบันทึกมีอยู่และเป็นของผู้ใช้
	note, err := s.noteRepo.GetByID(id, userID)
	if err != nil {
		return err
	}
	if note == nil {
		return errors.New("note not found")
	}

	if !note.IsPinned {
		return errors.New("note is not pinned")
	}

	return s.noteRepo.UnpinNote(id, userID)
}

// GetConversationNotes ดึงบันทึกเฉพาะ conversation
func (s *noteService) GetConversationNotes(userID, conversationID uuid.UUID, limit, offset int) ([]*models.Note, int64, error) {
	// ตรวจสอบว่าผู้ใช้เป็นสมาชิกของ conversation
	member, err := s.conversationMemberRepo.GetByConversationAndUserID(conversationID, userID)
	if err != nil {
		return nil, 0, err
	}
	if member == nil {
		return nil, 0, errors.New("user is not a member of this conversation")
	}

	return s.noteRepo.FindByConversationID(userID, conversationID, limit, offset)
}

// GetGlobalNotes ดึงบันทึกส่วนตัว (global) ที่ไม่ผูกกับ conversation ใด ๆ
func (s *noteService) GetGlobalNotes(userID uuid.UUID, limit, offset int) ([]*models.Note, int64, error) {
	return s.noteRepo.FindGlobalNotes(userID, limit, offset)
}
