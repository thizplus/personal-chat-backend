// pkg/scheduler/timer_manager.go
package scheduler

import (
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TimerCallback เป็น function ที่จะถูกเรียกเมื่อ timer ถึงเวลา
type TimerCallback func(messageID uuid.UUID)

// ScheduledTimer เก็บข้อมูล timer แต่ละตัว
type ScheduledTimer struct {
	MessageID   uuid.UUID
	ScheduledAt time.Time
	Timer       *time.Timer
}

// TimerManager จัดการ in-memory timers สำหรับ scheduled messages
type TimerManager struct {
	mu       sync.RWMutex
	timers   map[uuid.UUID]*ScheduledTimer
	callback TimerCallback
}

// NewTimerManager สร้าง TimerManager ใหม่
func NewTimerManager(callback TimerCallback) *TimerManager {
	return &TimerManager{
		timers:   make(map[uuid.UUID]*ScheduledTimer),
		callback: callback,
	}
}

// Schedule สร้าง timer สำหรับ scheduled message
func (tm *TimerManager) Schedule(messageID uuid.UUID, scheduledAt time.Time) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// ยกเลิก timer เดิมถ้ามี
	if existing, ok := tm.timers[messageID]; ok {
		existing.Timer.Stop()
		delete(tm.timers, messageID)
	}

	duration := time.Until(scheduledAt)

	// ถ้าเวลาผ่านไปแล้ว ส่งทันที
	if duration <= 0 {
		log.Printf("[TimerManager] Message %s is past due, executing immediately", messageID)
		go tm.callback(messageID)
		return
	}

	// สร้าง timer ใหม่
	timer := time.AfterFunc(duration, func() {
		log.Printf("[TimerManager] Timer fired for message %s", messageID)
		tm.callback(messageID)
		tm.remove(messageID)
	})

	tm.timers[messageID] = &ScheduledTimer{
		MessageID:   messageID,
		ScheduledAt: scheduledAt,
		Timer:       timer,
	}

	log.Printf("[TimerManager] Scheduled message %s for %s (in %v)", messageID, scheduledAt.Format(time.RFC3339), duration)
}

// Cancel ยกเลิก scheduled message
func (tm *TimerManager) Cancel(messageID uuid.UUID) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if timer, ok := tm.timers[messageID]; ok {
		timer.Timer.Stop()
		delete(tm.timers, messageID)
		log.Printf("[TimerManager] Cancelled timer for message %s", messageID)
		return true
	}
	return false
}

// remove ลบ timer ออกจาก map (internal use)
func (tm *TimerManager) remove(messageID uuid.UUID) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	delete(tm.timers, messageID)
}

// Reschedule เปลี่ยนเวลา
func (tm *TimerManager) Reschedule(messageID uuid.UUID, newTime time.Time) {
	tm.Cancel(messageID)
	tm.Schedule(messageID, newTime)
}

// Count จำนวน active timers
func (tm *TimerManager) Count() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return len(tm.timers)
}

// Has ตรวจสอบว่ามี timer สำหรับ messageID หรือไม่
func (tm *TimerManager) Has(messageID uuid.UUID) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	_, ok := tm.timers[messageID]
	return ok
}

// GetScheduledTime ดึงเวลาที่ schedule ไว้
func (tm *TimerManager) GetScheduledTime(messageID uuid.UUID) (time.Time, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	if timer, ok := tm.timers[messageID]; ok {
		return timer.ScheduledAt, true
	}
	return time.Time{}, false
}

// StopAll หยุด timers ทั้งหมด (เรียกตอน shutdown)
func (tm *TimerManager) StopAll() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for id, timer := range tm.timers {
		timer.Timer.Stop()
		delete(tm.timers, id)
	}
	log.Println("[TimerManager] All timers stopped")
}

// LoadPendingMessages โหลด pending messages และสร้าง timers
func (tm *TimerManager) LoadPendingMessages(messages []struct {
	ID          uuid.UUID
	ScheduledAt time.Time
}) int {
	count := 0
	for _, msg := range messages {
		tm.Schedule(msg.ID, msg.ScheduledAt)
		count++
	}
	return count
}
