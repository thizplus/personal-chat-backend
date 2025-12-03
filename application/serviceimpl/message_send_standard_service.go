// application/serviceimpl/message_send_standard.go
package serviceimpl

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/types"
)

// SendTextMessage ส่งข้อความประเภทข้อความ (text)
func (s *messageService) SendTextMessage(conversationID, userID uuid.UUID, content string, metadata map[string]interface{}) (*models.Message, error) {

	// ตรวจสอบว่าผู้ใช้เป็นสมาชิกของการสนทนา
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return nil, fmt.Errorf("error checking conversation membership: %w", err)
	}

	if !isMember {
		return nil, fmt.Errorf("user is not a member of this conversation")
	}

	// ตรวจสอบเนื้อหาข้อความ
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("message content cannot be empty")
	}

	// ดึงข้อมูลการสนทนา (เพื่อตรวจสอบประเภทการสนทนา)
	if err != nil {
		return nil, fmt.Errorf("error fetching conversation: %w", err)
	}

	// Extract links จากข้อความและเพิ่มลงใน metadata
	links := s.extractLinks(content)
	if len(links) > 0 {
		if metadata == nil {
			metadata = make(map[string]interface{})
		}
		metadata["links"] = links
	}

	// Extract mentions จาก metadata (ถ้ามี)
	var mentions interface{}
	if metadata != nil {
		if m, ok := metadata["mentions"]; ok {
			mentions = m
			// ลบ mentions ออกจาก metadata เพราะจะเก็บใน field แยก
			delete(metadata, "mentions")
		}
	}

	// แปลง mentions ให้เป็น JSONB ถ้ามี
	var mentionsJSON types.JSONB
	if mentions != nil {
		if mentionsArray, ok := mentions.([]interface{}); ok {
			// Wrap array in map for JSONB storage
			mentionsJSON = types.JSONB{"data": mentionsArray}
		} else if mentionsMap, ok := mentions.(map[string]interface{}); ok {
			mentionsJSON = types.JSONB(mentionsMap)
		}
	}

	// สร้าง message
	now := time.Now()
	message := &models.Message{
		ID:             uuid.New(),
		ConversationID: conversationID,
		SenderID:       &userID,
		SenderType:     "user",
		MessageType:    "text",
		Content:        content,
		Metadata:       s.convertMetadataToJSON(metadata),
		Mentions:       mentionsJSON,
		CreatedAt:      now,
		UpdatedAt:      now,
		IsDeleted:      false,
	}


	// บันทึกข้อความลงในฐานข้อมูล
	if err := s.messageRepo.Create(message); err != nil {
		return nil, fmt.Errorf("error creating message: %w", err)
	}

	// สร้างบันทึกการอ่านสำหรับผู้ส่ง
	messageRead := &models.MessageRead{
		ID:        uuid.New(),
		MessageID: message.ID,
		UserID:    userID,
		ReadAt:    now,
	}

	if err := s.messageReadRepo.CreateRead(messageRead); err != nil {
		fmt.Printf("Error creating read record: %v, messageID: %s, userID: %s", err, message.ID.String(), userID)
	}

	// อัปเดต last_read_at สำหรับผู้ส่ง
	if err := s.conversationRepo.UpdateMemberLastRead(conversationID, userID, now); err != nil {
		fmt.Printf("Error updating last read time: %v, conversationID: %s, userID: %s", err, conversationID, userID)
	}

	// อัปเดตข้อความล่าสุดของการสนทนา
	if err := s.messageRepo.UpdateConversationLastMessage(conversationID, content, now, message.ID); err != nil {
		fmt.Printf("Error updating conversation last message: %v, conversationID: %s", err, conversationID)
	}

	// ส่งการแจ้งเตือนสำหรับผู้ใช้ที่ถูก mention
	if mentions != nil {
		s.notifyMentionedUsers(message, mentions, userID)
	}

	// ส่ง WebSocket event แจ้งการอัปเดต conversation พร้อม mention data
	s.notifyConversationUpdated(conversationID, content, now, message.ID)

	return message, nil
}

// SendStickerMessage ส่งข้อความประเภทสติกเกอร์
func (s *messageService) SendStickerMessage(conversationID, userID, stickerID, stickerSetID uuid.UUID, mediaURL, thumbnailURL string, metadata map[string]interface{}) (*models.Message, error) {

	// ตรวจสอบว่าผู้ใช้เป็นสมาชิกของการสนทนา
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return nil, fmt.Errorf("error checking conversation membership: %w", err)
	}

	if !isMember {
		return nil, fmt.Errorf("user is not a member of this conversation")
	}

	// ตรวจสอบ URL สติกเกอร์
	if mediaURL == "" {
		return nil, fmt.Errorf("sticker URL is required")
	}

	// สร้าง metadata สำหรับสติกเกอร์
	stickerMetadata := make(map[string]interface{})
	if metadata != nil {
		for k, v := range metadata {
			stickerMetadata[k] = v
		}
	}

	// เพิ่มข้อมูลสติกเกอร์ลงใน metadata
	if stickerID != uuid.Nil {
		stickerMetadata["sticker_id"] = stickerID
	}

	if stickerSetID != uuid.Nil {
		stickerMetadata["sticker_set_id"] = stickerSetID
	}

	// ดึงข้อมูลการสนทนา (เพื่อตรวจสอบประเภทการสนทนา)
	if err != nil {
		return nil, fmt.Errorf("error fetching conversation: %w", err)
	}

	// สร้าง message
	now := time.Now()
	message := &models.Message{
		ID:                uuid.New(),
		ConversationID:    conversationID,
		SenderID:          &userID,
		SenderType:        "user",
		MessageType:       "sticker",
		MediaURL:          mediaURL,
		MediaThumbnailURL: thumbnailURL,
		Metadata:          s.convertMetadataToJSON(stickerMetadata),
		CreatedAt:         now,
		UpdatedAt:         now,
		IsDeleted:         false,
	}


	// บันทึกข้อความลงในฐานข้อมูล
	if err := s.messageRepo.Create(message); err != nil {
		return nil, fmt.Errorf("error creating message: %w", err)
	}

	// สร้างบันทึกการอ่านสำหรับผู้ส่ง
	messageRead := &models.MessageRead{
		ID:        uuid.New(),
		MessageID: message.ID,
		UserID:    userID,
		ReadAt:    now,
	}

	if err := s.messageReadRepo.CreateRead(messageRead); err != nil {
		fmt.Printf("Error creating read record: %v, messageID: %s, userID: %s", err, message.ID.String(), userID)
	}

	// อัปเดต last_read_at สำหรับผู้ส่ง
	if err := s.conversationRepo.UpdateMemberLastRead(conversationID, userID, now); err != nil {
		fmt.Printf("Error updating last read time: %v, conversationID: %s, userID: %s", err, conversationID, userID)
	}

	// อัปเดตข้อความล่าสุดของการสนทนา
	if err := s.messageRepo.UpdateConversationLastMessage(conversationID, "[Sticker]", now, message.ID); err != nil {
		fmt.Printf("Error updating conversation last message: %v, conversationID: %s", err, conversationID)
	}

	// ส่ง WebSocket event แจ้งการอัปเดต conversation พร้อม mention data
	s.notifyConversationUpdated(conversationID, "[Sticker]", now, message.ID)

	return message, nil
}

// SendImageMessage ส่งข้อความประเภทรูปภาพ
func (s *messageService) SendImageMessage(conversationID, userID uuid.UUID, mediaURL, thumbnailURL, caption string, metadata map[string]interface{}) (*models.Message, error) {

	// ตรวจสอบว่าผู้ใช้เป็นสมาชิกของการสนทนา
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return nil, fmt.Errorf("error checking conversation membership: %w", err)
	}

	if !isMember {
		return nil, fmt.Errorf("user is not a member of this conversation")
	}

	// ตรวจสอบ URL รูปภาพ
	if mediaURL == "" {
		return nil, fmt.Errorf("image URL is required")
	}

	// ดึงข้อมูลการสนทนา (เพื่อตรวจสอบประเภทการสนทนา)
	if err != nil {
		return nil, fmt.Errorf("error fetching conversation: %w", err)
	}

	// สร้าง message
	now := time.Now()
	message := &models.Message{
		ID:                uuid.New(),
		ConversationID:    conversationID,
		SenderID:          &userID,
		SenderType:        "user",
		MessageType:       "image",
		Content:           caption,
		MediaURL:          mediaURL,
		MediaThumbnailURL: thumbnailURL,
		Metadata:          s.convertMetadataToJSON(metadata),
		CreatedAt:         now,
		UpdatedAt:         now,
		IsDeleted:         false,
	}


	// บันทึกข้อความลงในฐานข้อมูล
	if err := s.messageRepo.Create(message); err != nil {
		return nil, fmt.Errorf("error creating message: %w", err)
	}

	// สร้างบันทึกการอ่านสำหรับผู้ส่ง
	messageRead := &models.MessageRead{
		ID:        uuid.New(),
		MessageID: message.ID,
		UserID:    userID,
		ReadAt:    now,
	}

	if err := s.messageReadRepo.CreateRead(messageRead); err != nil {
		fmt.Printf("Error creating read record: %v, messageID: %s, userID: %s", err, message.ID.String(), userID)
	}

	// อัปเดต last_read_at สำหรับผู้ส่ง
	if err := s.conversationRepo.UpdateMemberLastRead(conversationID, userID, now); err != nil {
		fmt.Printf("Error updating last read time: %v, conversationID: %s, userID: %s", err, conversationID, userID)
	}

	// อัปเดตข้อความล่าสุดของการสนทนา
	lastMsgText := "[Image]"
	if caption != "" {
		lastMsgText = "[Image] " + caption
	}

	if err := s.messageRepo.UpdateConversationLastMessage(conversationID, lastMsgText, now, message.ID); err != nil {
		fmt.Printf("Error updating conversation last message: %v, conversationID: %s", err, conversationID)
	}

	// ส่ง WebSocket event แจ้งการอัปเดต conversation พร้อม mention data
	s.notifyConversationUpdated(conversationID, lastMsgText, now, message.ID)

	return message, nil
}

// SendFileMessage ส่งข้อความประเภทไฟล์
func (s *messageService) SendFileMessage(conversationID, userID uuid.UUID, mediaURL, fileName string, fileSize int64, fileType string, metadata map[string]interface{}) (*models.Message, error) {

	// ตรวจสอบว่าผู้ใช้เป็นสมาชิกของการสนทนา
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return nil, fmt.Errorf("error checking conversation membership: %w", err)
	}

	if !isMember {
		return nil, fmt.Errorf("user is not a member of this conversation")
	}

	// ตรวจสอบ URL ไฟล์
	if mediaURL == "" {
		return nil, fmt.Errorf("file URL is required")
	}

	// สร้าง metadata สำหรับไฟล์
	fileMetadata := make(map[string]interface{})
	if metadata != nil {
		for k, v := range metadata {
			fileMetadata[k] = v
		}
	}

	// เพิ่มข้อมูลไฟล์ลงใน metadata
	if fileName != "" {
		fileMetadata["file_name"] = fileName
	}

	if fileSize > 0 {
		fileMetadata["file_size"] = fileSize
	}

	if fileType != "" {
		fileMetadata["file_type"] = fileType
	}

	// ดึงข้อมูลการสนทนา (เพื่อตรวจสอบประเภทการสนทนา)
	if err != nil {
		return nil, fmt.Errorf("error fetching conversation: %w", err)
	}

	// สร้าง message
	now := time.Now()
	message := &models.Message{
		ID:             uuid.New(),
		ConversationID: conversationID,
		SenderID:       &userID,
		SenderType:     "user",
		MessageType:    "file",
		Content:        fileName,
		MediaURL:       mediaURL,
		Metadata:       s.convertMetadataToJSON(fileMetadata),
		CreatedAt:      now,
		UpdatedAt:      now,
		IsDeleted:      false,
	}


	// บันทึกข้อความลงในฐานข้อมูล
	if err := s.messageRepo.Create(message); err != nil {
		return nil, fmt.Errorf("error creating message: %w", err)
	}

	// สร้างบันทึกการอ่านสำหรับผู้ส่ง
	messageRead := &models.MessageRead{
		ID:        uuid.New(),
		MessageID: message.ID,
		UserID:    userID,
		ReadAt:    now,
	}

	if err := s.messageReadRepo.CreateRead(messageRead); err != nil {
		fmt.Printf("Error creating read record: %v, messageID: %s, userID: %s", err, message.ID.String(), userID)
	}

	// อัปเดต last_read_at สำหรับผู้ส่ง
	if err := s.conversationRepo.UpdateMemberLastRead(conversationID, userID, now); err != nil {
		fmt.Printf("Error updating last read time: %v, conversationID: %s, userID: %s", err, conversationID, userID)
	}

	// อัปเดตข้อความล่าสุดของการสนทนา
	lastMsgText := "[File]"
	if fileName != "" {
		lastMsgText = "[File] " + fileName
	}

	if err := s.messageRepo.UpdateConversationLastMessage(conversationID, lastMsgText, now, message.ID); err != nil {
		fmt.Printf("Error updating conversation last message: %v, conversationID: %s", err, conversationID)
	}

	// ส่ง WebSocket event แจ้งการอัปเดต conversation พร้อม mention data
	s.notifyConversationUpdated(conversationID, lastMsgText, now, message.ID)

	return message, nil
}

// SendBulkMessages ส่งหลายไฟล์ในรูปแบบอัลบั้ม (Album Message)
// ส่งกลับ 1 message ที่มี type "album" พร้อม album_files array
func (s *messageService) SendBulkMessages(conversationID, userID uuid.UUID, caption string, items []map[string]interface{}) (*models.Message, error) {
	// ตรวจสอบว่าผู้ใช้เป็นสมาชิกของการสนทนา
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return nil, fmt.Errorf("error checking conversation membership: %w", err)
	}

	if !isMember {
		return nil, fmt.Errorf("user is not a member of this conversation")
	}

	// ตรวจสอบจำนวนไฟล์ (สูงสุด 10 ไฟล์)
	if len(items) == 0 {
		return nil, fmt.Errorf("at least one file is required")
	}
	if len(items) > 10 {
		return nil, fmt.Errorf("maximum 10 files per album")
	}

	now := time.Now()

	// สร้าง album_files array
	albumFiles := make([]map[string]interface{}, 0, len(items))
	var tempID string

	for i, item := range items {
		// ดึงข้อมูลจาก item
		fileType, _ := item["message_type"].(string)  // "image", "video", "file"
		mediaURL, _ := item["media_url"].(string)
		mediaThumbnailURL, _ := item["media_thumbnail_url"].(string)
		fileName, _ := item["file_name"].(string)

		// Validate required fields
		if fileType == "" || mediaURL == "" {
			return nil, fmt.Errorf("message_type and media_url are required for all items")
		}

		// เก็บ temp_id จาก item แรก
		if i == 0 {
			tempID, _ = item["temp_id"].(string)
		}

		// สร้าง album file object
		albumFile := map[string]interface{}{
			"id":                   uuid.New().String(),
			"file_type":            fileType,
			"media_url":            mediaURL,
			"media_thumbnail_url":  mediaThumbnailURL,
			"position":             i,
		}

		// เพิ่มข้อมูลไฟล์ (สำหรับ type "file")
		if fileName != "" {
			albumFile["file_name"] = fileName
		}
		if fileSize, ok := item["file_size"].(float64); ok {
			albumFile["file_size"] = int64(fileSize)
		} else if fileSize, ok := item["file_size"].(int64); ok {
			albumFile["file_size"] = fileSize
		}
		if fileTypeStr, ok := item["file_type"].(string); ok {
			albumFile["file_type"] = fileTypeStr
		}

		// เพิ่ม duration สำหรับ video
		if duration, ok := item["duration"].(float64); ok {
			albumFile["duration"] = int(duration)
		} else if duration, ok := item["duration"].(int); ok {
			albumFile["duration"] = duration
		}

		// เพิ่ม width/height ถ้ามี
		if width, ok := item["width"].(float64); ok {
			albumFile["width"] = int(width)
		} else if width, ok := item["width"].(int); ok {
			albumFile["width"] = width
		}
		if height, ok := item["height"].(float64); ok {
			albumFile["height"] = int(height)
		} else if height, ok := item["height"].(int); ok {
			albumFile["height"] = height
		}

		albumFiles = append(albumFiles, albumFile)
	}

	// สร้าง metadata
	metadata := types.JSONB{
		"album_total": len(items),
	}
	if tempID != "" {
		metadata["tempId"] = tempID
	}

	// สร้าง 1 message ที่มี type "album"
	message := &models.Message{
		ID:             uuid.New(),
		ConversationID: conversationID,
		SenderID:       &userID,
		SenderType:     "user",
		MessageType:    "album",  // ใช้ type "album"
		Content:        caption,  // caption จาก item แรก (ถ้ามี)
		AlbumFiles:     albumFiles,  // array ของไฟล์ทั้งหมด
		Metadata:       metadata,
		CreatedAt:      now,
		UpdatedAt:      now,
		IsDeleted:      false,
	}

	// บันทึก message ลงในฐานข้อมูล
	if err := s.messageRepo.Create(message); err != nil {
		return nil, fmt.Errorf("error creating album message: %w", err)
	}

	// สร้างบันทึกการอ่านสำหรับผู้ส่ง
	messageRead := &models.MessageRead{
		ID:        uuid.New(),
		MessageID: message.ID,
		UserID:    userID,
		ReadAt:    now,
	}

	if err := s.messageReadRepo.CreateRead(messageRead); err != nil {
		fmt.Printf("Error creating read record: %v, messageID: %s, userID: %s", err, message.ID.String(), userID)
	}

	// อัปเดต last_read_at สำหรับผู้ส่ง
	if err := s.conversationRepo.UpdateMemberLastRead(conversationID, userID, now); err != nil {
		fmt.Printf("Error updating last read time: %v, conversationID: %s, userID: %s", err, conversationID, userID)
	}

	// อัปเดตข้อความล่าสุดของการสนทนา
	lastMsgText := fmt.Sprintf("[Album: %d files]", len(items))
	if caption != "" {
		lastMsgText = caption
	}

	if err := s.messageRepo.UpdateConversationLastMessage(conversationID, lastMsgText, now, message.ID); err != nil {
		fmt.Printf("Error updating conversation last message: %v, conversationID: %s", err, conversationID)
	}

	// ส่ง WebSocket event แจ้งการอัปเดต conversation พร้อม mention data
	s.notifyConversationUpdated(conversationID, lastMsgText, now, message.ID)

	return message, nil
}
