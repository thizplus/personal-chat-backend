// pkg/scheduler/file_cleanup.go
package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/thizplus/gofiber-chat-api/domain/repository"
	"github.com/thizplus/gofiber-chat-api/domain/service"
)

// FileCleanupScheduler ทำงาน cleanup ไฟล์ที่ค้างและหมดอายุ
type FileCleanupScheduler struct {
	fileUploadRepo repository.FileUploadRepository
	storageService service.FileStorageService
	interval       time.Duration
	maxAge         time.Duration
}

// NewFileCleanupScheduler สร้าง scheduler ใหม่
func NewFileCleanupScheduler(
	fileUploadRepo repository.FileUploadRepository,
	storageService service.FileStorageService,
) *FileCleanupScheduler {
	return &FileCleanupScheduler{
		fileUploadRepo: fileUploadRepo,
		storageService: storageService,
		interval:       1 * time.Hour,  // ทำงานทุก 1 ชั่วโมง
		maxAge:         24 * time.Hour, // ลบไฟล์ที่ค้างเกิน 24 ชั่วโมง
	}
}

// Start เริ่มการทำงานของ scheduler
func (s *FileCleanupScheduler) Start(ctx context.Context) {
	log.Println("File cleanup scheduler started")

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// รันทันทีครั้งแรก
	s.cleanup()

	for {
		select {
		case <-ctx.Done():
			log.Println("File cleanup scheduler stopped")
			return
		case <-ticker.C:
			s.cleanup()
		}
	}
}

// cleanup ทำการลบไฟล์ที่ค้างและหมดอายุ
func (s *FileCleanupScheduler) cleanup() {
	log.Println("Running file cleanup...")

	cutoff := time.Now().Add(-s.maxAge)

	// หา uploads ที่ค้างเกิน maxAge
	abandonedUploads, err := s.fileUploadRepo.FindPendingOlderThan(cutoff)
	if err != nil {
		log.Printf("Error finding abandoned uploads: %v", err)
		return
	}

	if len(abandonedUploads) == 0 {
		log.Println("No abandoned uploads found")
		return
	}

	log.Printf("Found %d abandoned uploads to clean up", len(abandonedUploads))

	// ลบแต่ละไฟล์
	cleanedCount := 0
	errorCount := 0

	for _, upload := range abandonedUploads {
		// ลบไฟล์จาก storage
		if err := s.storageService.DeleteFile(upload.Path); err != nil {
			log.Printf("Error deleting file %s: %v", upload.Path, err)
			errorCount++
			continue
		}

		// ลบ record จาก database
		if err := s.fileUploadRepo.Delete(upload.ID); err != nil {
			log.Printf("Error deleting upload record %s: %v", upload.ID, err)
			errorCount++
			continue
		}

		cleanedCount++
	}

	log.Printf("File cleanup completed: %d cleaned, %d errors", cleanedCount, errorCount)
}
