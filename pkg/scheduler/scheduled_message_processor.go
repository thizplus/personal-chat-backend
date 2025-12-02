// pkg/scheduler/scheduled_message_processor.go
package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/thizplus/gofiber-chat-api/domain/service"
)

// ScheduledMessageProcessor ประมวลผลข้อความที่กำหนดเวลาส่ง
type ScheduledMessageProcessor struct {
	scheduledMessageService service.ScheduledMessageService
	interval                time.Duration
}

// NewScheduledMessageProcessor สร้าง processor ใหม่
func NewScheduledMessageProcessor(
	scheduledMessageService service.ScheduledMessageService,
) *ScheduledMessageProcessor {
	return &ScheduledMessageProcessor{
		scheduledMessageService: scheduledMessageService,
		interval:                1 * time.Minute, // ตรวจสอบทุก 1 นาที
	}
}

// Start เริ่มการทำงานของ processor
func (p *ScheduledMessageProcessor) Start(ctx context.Context) {
	log.Println("Scheduled message processor started")

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	// รันทันทีครั้งแรก
	p.process()

	for {
		select {
		case <-ctx.Done():
			log.Println("Scheduled message processor stopped")
			return
		case <-ticker.C:
			p.process()
		}
	}
}

// process ประมวลผลข้อความที่ถึงเวลาส่ง
func (p *ScheduledMessageProcessor) process() {
	if err := p.scheduledMessageService.ProcessScheduledMessages(); err != nil {
		log.Printf("Error processing scheduled messages: %v", err)
	}
}
