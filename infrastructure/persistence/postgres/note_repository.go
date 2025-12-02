// infrastructure/persistence/postgres/note_repository.go
package postgres

import (
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/repository"
	"gorm.io/gorm"
)

type noteRepository struct {
	db *gorm.DB
}

// NewNoteRepository สร้าง instance ใหม่ของ NoteRepository
func NewNoteRepository(db *gorm.DB) repository.NoteRepository {
	return &noteRepository{db: db}
}

// Create สร้างบันทึกใหม่
func (r *noteRepository) Create(note *models.Note) error {
	return r.db.Create(note).Error
}

// GetByID ดึงข้อมูลบันทึกตาม ID และตรวจสอบเจ้าของ
func (r *noteRepository) GetByID(id, userID uuid.UUID) (*models.Note, error) {
	var note models.Note
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&note).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &note, nil
}

// Update อัปเดตข้อมูลบันทึก
func (r *noteRepository) Update(note *models.Note) error {
	note.UpdatedAt = time.Now()
	return r.db.Save(note).Error
}

// Delete ลบบันทึก
func (r *noteRepository) Delete(id, userID uuid.UUID) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Note{}).Error
}

// FindByUserID ดึงรายการบันทึกของผู้ใช้
func (r *noteRepository) FindByUserID(userID uuid.UUID, limit, offset int) ([]*models.Note, int64, error) {
	var notes []*models.Note
	var total int64

	// นับจำนวนทั้งหมด
	if err := r.db.Model(&models.Note{}).
		Where("user_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// ดึงข้อมูล โดยเรียง pinned ก่อน แล้วตาม updated_at
	err := r.db.Where("user_id = ?", userID).
		Order("is_pinned DESC, updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notes).Error

	if err != nil {
		return nil, 0, err
	}

	return notes, total, nil
}

// FindPinnedByUserID ดึงรายการบันทึกที่ปักหมุด
func (r *noteRepository) FindPinnedByUserID(userID uuid.UUID, limit, offset int) ([]*models.Note, int64, error) {
	var notes []*models.Note
	var total int64

	// นับจำนวนทั้งหมด
	if err := r.db.Model(&models.Note{}).
		Where("user_id = ? AND is_pinned = ?", userID, true).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// ดึงข้อมูล
	err := r.db.Where("user_id = ? AND is_pinned = ?", userID, true).
		Order("updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notes).Error

	if err != nil {
		return nil, 0, err
	}

	return notes, total, nil
}

// SearchNotes ค้นหาบันทึกด้วย full-text search
func (r *noteRepository) SearchNotes(userID uuid.UUID, searchQuery string, limit, offset int) ([]*models.Note, int64, error) {
	var notes []*models.Note
	var total int64

	baseQuery := r.db.Model(&models.Note{}).
		Where("user_id = ?", userID).
		Where("content_tsvector @@ plainto_tsquery('english', ?)", searchQuery)

	// นับจำนวนทั้งหมด
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// ดึงข้อมูล
	err := baseQuery.
		Order("updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notes).Error

	if err != nil {
		return nil, 0, err
	}

	return notes, total, nil
}

// FindByTag ค้นหาบันทึกตาม tag
func (r *noteRepository) FindByTag(userID uuid.UUID, tag string, limit, offset int) ([]*models.Note, int64, error) {
	var notes []*models.Note
	var total int64

	// ใช้ JSONB operator เพื่อค้นหา tag
	baseQuery := r.db.Model(&models.Note{}).
		Where("user_id = ?", userID).
		Where("tags @> ?", `["`+tag+`"]`)

	// นับจำนวนทั้งหมด
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// ดึงข้อมูล
	err := baseQuery.
		Order("is_pinned DESC, updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notes).Error

	if err != nil {
		return nil, 0, err
	}

	return notes, total, nil
}

// FindByConversationID ดึงบันทึกที่เฉพาะเจาะจงกับ conversation
func (r *noteRepository) FindByConversationID(userID, conversationID uuid.UUID, limit, offset int) ([]*models.Note, int64, error) {
	var notes []*models.Note
	var total int64

	// นับจำนวนทั้งหมด
	if err := r.db.Model(&models.Note{}).
		Where("user_id = ? AND conversation_id = ?", userID, conversationID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// ดึงข้อมูล โดยเรียง pinned ก่อน แล้วตาม updated_at
	err := r.db.Where("user_id = ? AND conversation_id = ?", userID, conversationID).
		Order("is_pinned DESC, updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notes).Error

	if err != nil {
		return nil, 0, err
	}

	return notes, total, nil
}

// FindGlobalNotes ดึงบันทึกส่วนตัว (global) ที่ไม่ได้ผูกกับ conversation ใด ๆ
func (r *noteRepository) FindGlobalNotes(userID uuid.UUID, limit, offset int) ([]*models.Note, int64, error) {
	var notes []*models.Note
	var total int64

	// นับจำนวนทั้งหมด
	if err := r.db.Model(&models.Note{}).
		Where("user_id = ? AND conversation_id IS NULL", userID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// ดึงข้อมูล โดยเรียง pinned ก่อน แล้วตาม updated_at
	err := r.db.Where("user_id = ? AND conversation_id IS NULL", userID).
		Order("is_pinned DESC, updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notes).Error

	if err != nil {
		return nil, 0, err
	}

	return notes, total, nil
}

// PinNote ปักหมุดบันทึก
func (r *noteRepository) PinNote(id, userID uuid.UUID) error {
	return r.db.Model(&models.Note{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{
			"is_pinned":  true,
			"updated_at": time.Now(),
		}).Error
}

// UnpinNote ยกเลิกการปักหมุดบันทึก
func (r *noteRepository) UnpinNote(id, userID uuid.UUID) error {
	return r.db.Model(&models.Note{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{
			"is_pinned":  false,
			"updated_at": time.Now(),
		}).Error
}
