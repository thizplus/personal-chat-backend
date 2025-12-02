// domain/dto/media_dto.go
package dto

import (
	"time"

	"github.com/thizplus/gofiber-chat-api/domain/types"
)

// MediaSummaryDTO สรุปจำนวน media ในการสนทนา
type MediaSummaryDTO struct {
	ImageCount int64 `json:"image_count"`
	VideoCount int64 `json:"video_count"`
	FileCount  int64 `json:"file_count"`
	LinkCount  int64 `json:"link_count"`
	TotalMedia int64 `json:"total_media"`
}

// MediaItemDTO ข้อมูลรายละเอียดของแต่ละ media
type MediaItemDTO struct {
	MessageID        string      `json:"message_id"`
	MessageType      string      `json:"message_type"`
	Content          string      `json:"content,omitempty"`
	MediaURL         string      `json:"media_url,omitempty"`
	ThumbnailURL     string      `json:"thumbnail_url,omitempty"`
	FileName         string      `json:"file_name,omitempty"`
	FileSize         int64       `json:"file_size,omitempty"`
	Metadata         types.JSONB `json:"metadata,omitempty"`
	CreatedAt        time.Time   `json:"created_at"`
	IsAlbum          bool        `json:"is_album"` // true ถ้ามาจาก album message
}

// MediaListDTO รายการ media พร้อม pagination
type MediaListDTO struct {
	Data       []*MediaItemDTO `json:"data"`
	Pagination PaginationDTO   `json:"pagination"`
}

// PaginationDTO ข้อมูล pagination
type PaginationDTO struct {
	Total   int64 `json:"total"`
	Limit   int   `json:"limit"`
	Offset  int   `json:"offset"`
	HasMore bool  `json:"has_more"`
}
