// pkg/scheduler/scheduled_message_processor.go
package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/service"
)

// ScheduledMessageProcessor ประมวลผลข้อความที่กำหนดเวลาส่ง
// ใช้ Hybrid Approach: In-Memory Timer + Fallback Poll
type ScheduledMessageProcessor struct {
	scheduledMessageService service.ScheduledMessageService
	timerManager            *TimerManager
	fallbackInterval        time.Duration // ตรวจสอบ fallback ทุก 5 นาที
}

// NewScheduledMessageProcessor สร้าง processor ใหม่
func NewScheduledMessageProcessor(
	scheduledMessageService service.ScheduledMessageService,
) *ScheduledMessageProcessor {
	processor := &ScheduledMessageProcessor{
		scheduledMessageService: scheduledMessageService,
		fallbackInterval:        5 * time.Minute,
	}

	// สร้าง TimerManager พร้อม callback
	processor.timerManager = NewTimerManager(processor.executeScheduledMessage)

	return processor
}

// Start เริ่มการทำงานของ processor
func (p *ScheduledMessageProcessor) Start(ctx context.Context) {
	log.Println("[ScheduledMessageProcessor] Starting with precise timing mode...")

	// 1. โหลด pending messages จาก DB และสร้าง timers
	p.loadPendingMessages()

	// 2. เริ่ม fallback processor (ทุก 5 นาที เป็น safety net)
	ticker := time.NewTicker(p.fallbackInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[ScheduledMessageProcessor] Stopping...")
			p.timerManager.StopAll()
			log.Println("[ScheduledMessageProcessor] Stopped")
			return
		case <-ticker.C:
			p.processFallback()
		}
	}
}

// loadPendingMessages โหลด pending messages ตอน startup
func (p *ScheduledMessageProcessor) loadPendingMessages() {
	log.Println("[ScheduledMessageProcessor] Loading pending messages from database...")

	// ดึง pending messages ทั้งหมด (ถึงเวลาแล้ว + ยังไม่ถึงเวลา)
	futureTime := time.Now().Add(24 * time.Hour) // ดึงล่วงหน้า 24 ชม.
	messages, err := p.scheduledMessageService.GetPendingMessagesForProcessor(futureTime, 1000)
	if err != nil {
		log.Printf("[ScheduledMessageProcessor] Error loading pending messages: %v", err)
		return
	}

	// สร้าง timers สำหรับแต่ละ message
	for _, msg := range messages {
		p.timerManager.Schedule(msg.ID, msg.ScheduledAt)
	}

	log.Printf("[ScheduledMessageProcessor] Loaded %d pending scheduled messages", len(messages))
}

// processFallback ตรวจสอบ messages ที่ตกค้าง (safety net)
func (p *ScheduledMessageProcessor) processFallback() {
	// หา messages ที่ควรส่งแล้วแต่ยัง pending อยู่ (อาจเกิดจาก server restart)
	messages, err := p.scheduledMessageService.GetPendingMessagesForProcessor(time.Now(), 100)
	if err != nil {
		log.Printf("[ScheduledMessageProcessor] Fallback error: %v", err)
		return
	}

	if len(messages) == 0 {
		return
	}

	log.Printf("[ScheduledMessageProcessor] Fallback found %d overdue messages", len(messages))

	for _, msg := range messages {
		// ถ้ายังไม่มี timer ให้สร้างใหม่ (จะส่งทันทีเพราะเวลาผ่านไปแล้ว)
		if !p.timerManager.Has(msg.ID) {
			p.timerManager.Schedule(msg.ID, msg.ScheduledAt)
		}
	}
}

// executeScheduledMessage ส่ง message (callback จาก timer)
func (p *ScheduledMessageProcessor) executeScheduledMessage(messageID uuid.UUID) {
	log.Printf("[ScheduledMessageProcessor] Executing scheduled message: %s", messageID)

	err := p.scheduledMessageService.ProcessSingleScheduledMessage(messageID)
	if err != nil {
		log.Printf("[ScheduledMessageProcessor] Failed to process message %s: %v", messageID, err)
	} else {
		log.Printf("[ScheduledMessageProcessor] Successfully sent message %s", messageID)
	}
}

// ScheduleMessage เรียกจาก service เมื่อสร้าง scheduled message ใหม่
func (p *ScheduledMessageProcessor) ScheduleMessage(messageID uuid.UUID, scheduledAt time.Time) {
	p.timerManager.Schedule(messageID, scheduledAt)
}

// CancelMessage เรียกเมื่อยกเลิก scheduled message
func (p *ScheduledMessageProcessor) CancelMessage(messageID uuid.UUID) {
	p.timerManager.Cancel(messageID)
}

// RescheduleMessage เรียกเมื่อเปลี่ยนเวลา
func (p *ScheduledMessageProcessor) RescheduleMessage(messageID uuid.UUID, newTime time.Time) {
	p.timerManager.Reschedule(messageID, newTime)
}

// GetActiveTimerCount ดึงจำนวน active timers
func (p *ScheduledMessageProcessor) GetActiveTimerCount() int {
	return p.timerManager.Count()
}
