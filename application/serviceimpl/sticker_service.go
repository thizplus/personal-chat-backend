// application/serviceimpl/sticker_service.go
package serviceimpl

import (
	"errors"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/repository"
	"github.com/thizplus/gofiber-chat-api/domain/service"
)

type stickerService struct {
	stickerRepo    repository.StickerRepository
	storageService service.FileStorageService
}

// NewStickerService สร้าง service สำหรับจัดการสติกเกอร์
func NewStickerService(stickerRepo repository.StickerRepository, storageService service.FileStorageService) service.StickerService {
	return &stickerService{
		stickerRepo:    stickerRepo,
		storageService: storageService,
	}
}

// CreateStickerSet สร้างชุดสติกเกอร์ใหม่
func (s *stickerService) CreateStickerSet(name, description, author string, isOfficial, isDefault bool) (*models.StickerSet, error) {
	stickerSet := &models.StickerSet{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		Author:      author,
		CreatedAt:   time.Now(),
		IsOfficial:  isOfficial,
		IsDefault:   isDefault,
		SortOrder:   0,
	}

	if err := s.stickerRepo.CreateStickerSet(stickerSet); err != nil {
		return nil, err
	}

	return stickerSet, nil
}

// GetStickerSetByID ดึงข้อมูลชุดสติกเกอร์ตาม ID
func (s *stickerService) GetStickerSetByID(id uuid.UUID) (*models.StickerSet, error) {
	return s.stickerRepo.GetStickerSetByID(id)
}

// GetAllStickerSets ดึงข้อมูลชุดสติกเกอร์ทั้งหมด
func (s *stickerService) GetAllStickerSets(limit, offset int) ([]*models.StickerSet, int64, error) {
	return s.stickerRepo.GetAllStickerSets(limit, offset)
}

// GetDefaultStickerSets ดึงข้อมูลชุดสติกเกอร์เริ่มต้น
func (s *stickerService) GetDefaultStickerSets() ([]*models.StickerSet, error) {
	return s.stickerRepo.GetDefaultStickerSets()
}

// UpdateStickerSet อัปเดตข้อมูลชุดสติกเกอร์
func (s *stickerService) UpdateStickerSet(id uuid.UUID, name, description, author string, isOfficial, isDefault bool) (*models.StickerSet, error) {
	stickerSet, err := s.stickerRepo.GetStickerSetByID(id)
	if err != nil {
		return nil, err
	}

	// อัปเดตข้อมูล
	stickerSet.Name = name
	stickerSet.Description = description
	stickerSet.Author = author
	stickerSet.IsOfficial = isOfficial
	stickerSet.IsDefault = isDefault

	if err := s.stickerRepo.UpdateStickerSet(stickerSet); err != nil {
		return nil, err
	}

	return stickerSet, nil
}

// DeleteStickerSet ลบชุดสติกเกอร์
func (s *stickerService) DeleteStickerSet(id uuid.UUID) error {
	return s.stickerRepo.DeleteStickerSet(id)
}

// UploadStickerSetCover อัปโหลดรูปปกชุดสติกเกอร์
func (s *stickerService) UploadStickerSetCover(id uuid.UUID, file *multipart.FileHeader) (*models.StickerSet, error) {
	// ตรวจสอบว่าชุดสติกเกอร์มีอยู่จริง
	stickerSet, err := s.stickerRepo.GetStickerSetByID(id)
	if err != nil {
		return nil, err
	}

	// ตรวจสอบขนาดไฟล์ (เช่น ไม่เกิน 100MB)
	maxSize := 100 * 1024 * 1024 // 100MB
	if file.Size > int64(maxSize) {
		return nil, errors.New("file too large (max 100MB)")
	}

	// ตรวจสอบประเภทไฟล์
	ext := strings.ToLower(filepath.Ext(file.Filename))
	validExt := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
	}

	if !validExt[ext] {
		return nil, errors.New("invalid file type. Only JPG, JPEG and PNG are allowed")
	}

	// อัปโหลดไฟล์
	result, err := s.storageService.UploadImage(file, "sticker_set_covers")
	if err != nil {
		return nil, err
	}

	// อัปเดต URL ในข้อมูลชุดสติกเกอร์
	stickerSet.CoverImageURL = result.URL
	if err := s.stickerRepo.UpdateStickerSet(stickerSet); err != nil {
		return nil, err
	}

	return stickerSet, nil
}

// AddStickerToSet เพิ่มสติกเกอร์ใหม่ลงในชุด
func (s *stickerService) AddStickerToSet(setID uuid.UUID, name string, file *multipart.FileHeader, isAnimated bool, sortOrder int) (*models.Sticker, error) {
	// ตรวจสอบว่าชุดสติกเกอร์มีอยู่จริง
	_, err := s.stickerRepo.GetStickerSetByID(setID)
	if err != nil {
		return nil, err
	}

	// ตรวจสอบขนาดไฟล์ (เช่น ไม่เกิน 100MB)
	maxSize := 100 * 1024 * 1024 // 100MB
	if file.Size > int64(maxSize) {
		return nil, errors.New("file too large (max 100MB)")
	}

	// ตรวจสอบประเภทไฟล์
	ext := strings.ToLower(filepath.Ext(file.Filename))
	validExt := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  isAnimated, // อนุญาต GIF เฉพาะเมื่อเป็นสติกเกอร์แบบเคลื่อนไหว
		".webp": true,
	}

	if !validExt[ext] {
		return nil, errors.New("invalid file type")
	}

	// อัปโหลดไฟล์
	result, err := s.storageService.UploadImage(file, "stickers")
	if err != nil {
		return nil, err
	}

	// สร้าง thumbnail (ในกรณีนี้ใช้ URL เดียวกันกับรูปหลัก, ในระบบจริงอาจต้องสร้าง thumbnail แยก)
	thumbnailURL := result.URL

	// สร้างข้อมูลสติกเกอร์ใหม่
	sticker := &models.Sticker{
		ID:           uuid.New(),
		StickerSetID: setID,
		Name:         name,
		StickerURL:   result.URL,
		ThumbnailURL: thumbnailURL,
		CreatedAt:    time.Now(),
		IsAnimated:   isAnimated,
		SortOrder:    sortOrder,
	}

	// บันทึกลงฐานข้อมูล
	if err := s.stickerRepo.CreateSticker(sticker); err != nil {
		return nil, err
	}

	return sticker, nil
}

// GetStickerByID ดึงข้อมูลสติกเกอร์ตาม ID
func (s *stickerService) GetStickerByID(id uuid.UUID) (*models.Sticker, error) {
	return s.stickerRepo.GetStickerByID(id)
}

// GetStickersBySetID ดึงข้อมูลสติกเกอร์ทั้งหมดในชุด
func (s *stickerService) GetStickersBySetID(setID uuid.UUID) ([]*models.Sticker, error) {
	return s.stickerRepo.GetStickersBySetID(setID)
}

// UpdateSticker อัปเดตข้อมูลสติกเกอร์
func (s *stickerService) UpdateSticker(id uuid.UUID, name string, sortOrder int) (*models.Sticker, error) {
	sticker, err := s.stickerRepo.GetStickerByID(id)
	if err != nil {
		return nil, err
	}

	// อัปเดตข้อมูล
	sticker.Name = name
	sticker.SortOrder = sortOrder

	if err := s.stickerRepo.UpdateSticker(sticker); err != nil {
		return nil, err
	}

	return sticker, nil
}

// DeleteSticker ลบสติกเกอร์
func (s *stickerService) DeleteSticker(id uuid.UUID) error {
	return s.stickerRepo.DeleteSticker(id)
}

// AddStickerSetToUser เพิ่มชุดสติกเกอร์ให้ผู้ใช้
func (s *stickerService) AddStickerSetToUser(userID, stickerSetID uuid.UUID) error {
	userStickerSet := &models.UserStickerSet{
		ID:           uuid.New(),
		UserID:       userID,
		StickerSetID: stickerSetID,
		PurchasedAt:  time.Now(),
		IsFavorite:   false,
	}

	return s.stickerRepo.AddStickerSetToUser(userStickerSet)
}

// GetUserStickerSets ดึงชุดสติกเกอร์ของผู้ใช้
func (s *stickerService) GetUserStickerSets(userID uuid.UUID) ([]*models.StickerSet, error) {
	return s.stickerRepo.GetUserStickerSets(userID)
}

// SetStickerSetAsFavorite ตั้งค่าชุดสติกเกอร์เป็นรายการโปรด
func (s *stickerService) SetStickerSetAsFavorite(userID, stickerSetID uuid.UUID, isFavorite bool) error {
	return s.stickerRepo.SetStickerSetAsFavorite(userID, stickerSetID, isFavorite)
}

// RemoveStickerSetFromUser ลบชุดสติกเกอร์ออกจากผู้ใช้
func (s *stickerService) RemoveStickerSetFromUser(userID, stickerSetID uuid.UUID) error {
	return s.stickerRepo.RemoveStickerSetFromUser(userID, stickerSetID)
}

// RecordStickerUsage บันทึกการใช้งานสติกเกอร์
func (s *stickerService) RecordStickerUsage(userID, stickerID uuid.UUID) error {
	// บันทึกการใช้งานสติกเกอร์ล่าสุด
	userRecentSticker := &models.UserRecentSticker{
		ID:        uuid.New(),
		UserID:    userID,
		StickerID: stickerID,
		UsedAt:    time.Now(),
	}

	return s.stickerRepo.AddRecentSticker(userRecentSticker)
}

// GetUserRecentStickers ดึงสติกเกอร์ที่ใช้ล่าสุดของผู้ใช้
func (s *stickerService) GetUserRecentStickers(userID uuid.UUID, limit int) ([]*models.Sticker, error) {
	return s.stickerRepo.GetUserRecentStickers(userID, limit)
}

// GetUserFavoriteStickers ดึงสติกเกอร์โปรดของผู้ใช้
func (s *stickerService) GetUserFavoriteStickers(userID uuid.UUID) ([]*models.Sticker, error) {
	return s.stickerRepo.GetUserFavoriteStickers(userID)
}
