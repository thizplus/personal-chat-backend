// application/serviceimpl/conversation_member_service.go
package serviceimpl

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/dto"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/repository"
	"github.com/thizplus/gofiber-chat-api/domain/service"
)

type conversationMemberService struct {
	conversationRepo repository.ConversationRepository
	userRepo         repository.UserRepository
	messageRepo      repository.MessageRepository
}

// NewConversationMemberService สร้าง service ใหม่
func NewConversationMemberService(
	conversationRepo repository.ConversationRepository,
	userRepo repository.UserRepository,
	messageRepo repository.MessageRepository,
) service.ConversationMemberService {
	return &conversationMemberService{
		conversationRepo: conversationRepo,
		userRepo:         userRepo,
		messageRepo:      messageRepo,
	}
}

// AddMember เพิ่มสมาชิกในการสนทนากลุ่ม
func (s *conversationMemberService) AddMember(userID, conversationID, newMemberID uuid.UUID) (*dto.MemberDTO, error) {
	// 1. ตรวจสอบว่าผู้ใช้เป็นสมาชิกและเป็นแอดมินหรือไม่
	member, err := s.conversationRepo.GetMember(conversationID, userID)
	if err != nil {
		return nil, errors.New("error checking membership: " + err.Error())
	}
	if member == nil {
		return nil, errors.New("you are not a member of this conversation")
	}
	if !member.IsAdmin {
		return nil, errors.New("only admins can add members")
	}

	// 2. ตรวจสอบประเภทการสนทนาว่าเป็นกลุ่มหรือไม่
	conversation, err := s.conversationRepo.GetByID(conversationID)
	if err != nil {
		return nil, errors.New("error fetching conversation: " + err.Error())
	}
	if conversation.Type == "direct" {
		return nil, errors.New("cannot add members to direct conversation")
	}

	// 3. ตรวจสอบว่าผู้ใช้ที่จะเพิ่มมีอยู่จริงหรือไม่
	user, err := s.userRepo.FindByID(newMemberID)
	if err != nil || user == nil {
		return nil, errors.New("user to add not found")
	}

	// 4. ตรวจสอบว่าผู้ใช้เป็นสมาชิกอยู่แล้วหรือไม่
	isMember, err := s.conversationRepo.IsMember(conversationID, newMemberID)
	if err != nil {
		return nil, errors.New("error checking existing membership: " + err.Error())
	}
	if isMember {
		return nil, errors.New("user is already a member of this conversation")
	}

	// 5. เพิ่มสมาชิกใหม่
	now := time.Now()
	newMember := &models.ConversationMember{
		ID:             uuid.New(),
		ConversationID: conversationID,
		UserID:         newMemberID,
		IsAdmin:        false,
		JoinedAt:       now,
	}

	if err := s.conversationRepo.AddMember(newMember); err != nil {
		return nil, errors.New("error adding member: " + err.Error())
	}

	// 6. สร้างข้อความระบบ
	adderName, _ := s.getUserName(userID)
	newMemberName, _ := s.getUserName(newMemberID)
	systemMessage := adderName + " added " + newMemberName + " to the group"

	s.createSystemMessage(conversationID, systemMessage)
	s.conversationRepo.UpdateLastMessage(conversationID, systemMessage, now)

	// 7. สร้าง DTO เพื่อส่งกลับ
	memberDTO := &dto.MemberDTO{
		ID:             newMember.ID.String(),
		UserID:         newMember.UserID.String(),
		Username:       user.Username,
		DisplayName:    user.DisplayName,
		ProfilePicture: user.ProfileImageURL,
		Role:           "member",
		JoinedAt:       newMember.JoinedAt,
		IsOnline:       false, // ต้องมี logic การตรวจสอบว่า online หรือไม่
	}

	return memberDTO, nil
}

// BulkAddMembers เพิ่มสมาชิกหลายคนพร้อมกันในการสนทนากลุ่ม
func (s *conversationMemberService) BulkAddMembers(userID, conversationID uuid.UUID, newMemberIDs []uuid.UUID) (addedMembers []*dto.MemberDTO, failed []struct {
	UserID uuid.UUID
	Reason string
}, err error) {
	// 1. ตรวจสอบว่าผู้ใช้เป็นสมาชิกและเป็นแอดมินหรือไม่
	member, err := s.conversationRepo.GetMember(conversationID, userID)
	if err != nil {
		return nil, nil, errors.New("error checking membership: " + err.Error())
	}
	if member == nil {
		return nil, nil, errors.New("you are not a member of this conversation")
	}
	if !member.IsAdmin {
		return nil, nil, errors.New("only admins can add members")
	}

	// 2. ตรวจสอบประเภทการสนทนาว่าเป็นกลุ่มหรือไม่
	conversation, err := s.conversationRepo.GetByID(conversationID)
	if err != nil {
		return nil, nil, errors.New("error fetching conversation: " + err.Error())
	}
	if conversation.Type == "direct" {
		return nil, nil, errors.New("cannot add members to direct conversation")
	}

	// 3. เพิ่มสมาชิกทีละคน
	addedMembers = []*dto.MemberDTO{}
	failed = []struct {
		UserID uuid.UUID
		Reason string
	}{}

	now := time.Now()
	adderName, _ := s.getUserName(userID)
	addedNames := []string{}

	for _, newMemberID := range newMemberIDs {
		// ตรวจสอบว่าผู้ใช้ที่จะเพิ่มมีอยู่จริงหรือไม่
		user, err := s.userRepo.FindByID(newMemberID)
		if err != nil || user == nil {
			failed = append(failed, struct {
				UserID uuid.UUID
				Reason string
			}{UserID: newMemberID, Reason: "user not found"})
			continue
		}

		// ตรวจสอบว่าผู้ใช้เป็นสมาชิกอยู่แล้วหรือไม่
		isMember, err := s.conversationRepo.IsMember(conversationID, newMemberID)
		if err != nil {
			failed = append(failed, struct {
				UserID uuid.UUID
				Reason string
			}{UserID: newMemberID, Reason: "error checking membership"})
			continue
		}
		if isMember {
			failed = append(failed, struct {
				UserID uuid.UUID
				Reason string
			}{UserID: newMemberID, Reason: "already a member"})
			continue
		}

		// เพิ่มสมาชิกใหม่
		newMember := &models.ConversationMember{
			ID:             uuid.New(),
			ConversationID: conversationID,
			UserID:         newMemberID,
			IsAdmin:        false,
			JoinedAt:       now,
		}

		if err := s.conversationRepo.AddMember(newMember); err != nil {
			failed = append(failed, struct {
				UserID uuid.UUID
				Reason string
			}{UserID: newMemberID, Reason: "error adding to database"})
			continue
		}

		// สร้าง DTO
		memberDTO := &dto.MemberDTO{
			ID:             newMember.ID.String(),
			UserID:         newMember.UserID.String(),
			Username:       user.Username,
			DisplayName:    user.DisplayName,
			ProfilePicture: user.ProfileImageURL,
			Role:           "member",
			JoinedAt:       newMember.JoinedAt,
			IsOnline:       false,
		}

		addedMembers = append(addedMembers, memberDTO)
		addedNames = append(addedNames, user.DisplayName)
	}

	// 4. สร้างข้อความระบบถ้ามีคนถูกเพิ่มสำเร็จ
	if len(addedMembers) > 0 {
		var systemMessage string
		if len(addedMembers) == 1 {
			systemMessage = adderName + " added " + addedNames[0] + " to the group"
		} else {
			systemMessage = adderName + " added " + fmt.Sprintf("%d members", len(addedMembers)) + " to the group"
		}

		s.createSystemMessage(conversationID, systemMessage)
		s.conversationRepo.UpdateLastMessage(conversationID, systemMessage, now)
	}

	return addedMembers, failed, nil
}

// GetMembers ดึงรายการสมาชิกในการสนทนา
func (s *conversationMemberService) GetMembers(userID, conversationID uuid.UUID, page, limit int) ([]*dto.MemberDTO, int, error) {
	// 1. ตรวจสอบว่าผู้ใช้เป็นสมาชิกของการสนทนานี้หรือไม่
	isMember, err := s.conversationRepo.IsMember(conversationID, userID)
	if err != nil {
		return nil, 0, errors.New("error checking membership: " + err.Error())
	}
	if !isMember {
		return nil, 0, errors.New("you are not a member of this conversation")
	}

	// 2. ดึงรายการสมาชิกทั้งหมด
	members, err := s.conversationRepo.GetMembers(conversationID)
	if err != nil {
		return nil, 0, errors.New("error fetching members: " + err.Error())
	}

	// 3. จัดการ pagination
	start := (page - 1) * limit
	end := start + limit
	if start >= len(members) {
		return []*dto.MemberDTO{}, len(members), nil
	}
	if end > len(members) {
		end = len(members)
	}

	paginatedMembers := members[start:end]

	// 4. แปลงเป็น DTOs
	memberDTOs := make([]*dto.MemberDTO, 0, len(paginatedMembers))
	for _, member := range paginatedMembers {
		// ดึงข้อมูลผู้ใช้
		user, err := s.userRepo.FindByID(member.UserID)
		if err != nil || user == nil {
			continue // ข้ามถ้าไม่พบผู้ใช้
		}

		// กำหนดบทบาท
		role := "member"
		if member.IsAdmin {
			role = "admin"
		}

		// สร้าง DTO
		memberDTO := &dto.MemberDTO{
			ID:             member.ID.String(),
			UserID:         member.UserID.String(),
			Username:       user.Username,
			DisplayName:    user.DisplayName,
			ProfilePicture: user.ProfileImageURL,
			Role:           role,
			JoinedAt:       member.JoinedAt,
			IsOnline:       false, // ต้องมี logic การตรวจสอบว่า online หรือไม่
		}

		memberDTOs = append(memberDTOs, memberDTO)
	}

	return memberDTOs, len(members), nil
}

// RemoveMember ลบสมาชิกออกจากการสนทนา
func (s *conversationMemberService) RemoveMember(userID, conversationID, memberToRemoveID uuid.UUID) error {
	// 1. ตรวจสอบประเภทการสนทนา
	conversation, err := s.conversationRepo.GetByID(conversationID)
	if err != nil {
		return errors.New("error fetching conversation: " + err.Error())
	}
	if conversation.Type == "direct" {
		return errors.New("cannot remove members from direct conversation")
	}

	// 2. กำหนดสิทธิ์การลบสมาชิก
	if userID == memberToRemoveID {
		// ผู้ใช้ต้องการออกด้วยตัวเอง - ต้องเป็นสมาชิก
		isMember, err := s.conversationRepo.IsMember(conversationID, userID)
		if err != nil {
			return errors.New("error checking membership: " + err.Error())
		}
		if !isMember {
			return errors.New("you are not a member of this conversation")
		}
	} else {
		// ผู้ใช้ต้องการลบผู้อื่น - ต้องเป็นแอดมิน
		member, err := s.conversationRepo.GetMember(conversationID, userID)
		if err != nil {
			return errors.New("error checking admin status: " + err.Error())
		}
		if member == nil || !member.IsAdmin {
			return errors.New("only admins can remove other members")
		}
	}

	// 3. ตรวจสอบว่าเป้าหมายเป็นสมาชิกอยู่จริง
	targetMember, err := s.conversationRepo.GetMember(conversationID, memberToRemoveID)
	if err != nil || targetMember == nil {
		return errors.New("user is not a member of this conversation")
	}

	// 4. ตรวจสอบกรณีลบแอดมินคนสุดท้าย
	if targetMember.IsAdmin && userID != memberToRemoveID {
		// นับจำนวนแอดมินในการสนทนา
		members, err := s.conversationRepo.GetMembers(conversationID)
		if err != nil {
			return errors.New("error checking admin count: " + err.Error())
		}

		adminCount := 0
		for _, m := range members {
			if m.IsAdmin {
				adminCount++
			}
		}

		if adminCount <= 1 {
			return errors.New("cannot remove the last admin from the conversation")
		}
	}

	// 5. ลบสมาชิก
	if err := s.conversationRepo.RemoveMember(conversationID, memberToRemoveID); err != nil {
		return errors.New("error removing member: " + err.Error())
	}

	// 6. สร้างข้อความระบบ
	now := time.Now()
	var systemMessage string
	if userID == memberToRemoveID {
		userName, _ := s.getUserName(userID)
		systemMessage = userName + " left the group"
	} else {
		removerName, _ := s.getUserName(userID)
		removedName, _ := s.getUserName(memberToRemoveID)
		systemMessage = removerName + " removed " + removedName + " from the group"
	}

	s.createSystemMessage(conversationID, systemMessage)
	s.conversationRepo.UpdateLastMessage(conversationID, systemMessage, now)

	return nil
}

// ToggleAdminStatus เปลี่ยนสถานะแอดมินของสมาชิก
func (s *conversationMemberService) ToggleAdminStatus(userID, conversationID, targetUserID uuid.UUID, isAdmin bool) (bool, error) {
	// 1. ตรวจสอบว่าผู้ใช้เป็นแอดมิน
	member, err := s.conversationRepo.GetMember(conversationID, userID)
	if err != nil {
		return false, errors.New("error checking admin status: " + err.Error())
	}
	if member == nil || !member.IsAdmin {
		return false, errors.New("only admins can change admin status")
	}

	// 2. ตรวจสอบประเภทการสนทนา
	conversation, err := s.conversationRepo.GetByID(conversationID)
	if err != nil {
		return false, errors.New("error fetching conversation: " + err.Error())
	}
	if conversation.Type == "direct" {
		return false, errors.New("cannot change admin status in direct conversation")
	}

	// 3. ตรวจสอบว่าเป้าหมายเป็นสมาชิกอยู่จริง
	targetMember, err := s.conversationRepo.GetMember(conversationID, targetUserID)
	if err != nil || targetMember == nil {
		return false, errors.New("user is not a member of this conversation")
	}

	// 4. ตรวจสอบกรณีลบสิทธิ์แอดมินคนสุดท้าย
	if targetMember.IsAdmin && !isAdmin {
		// นับจำนวนแอดมินในการสนทนา
		members, err := s.conversationRepo.GetMembers(conversationID)
		if err != nil {
			return false, errors.New("error checking admin count: " + err.Error())
		}

		adminCount := 0
		for _, m := range members {
			if m.IsAdmin {
				adminCount++
			}
		}

		if adminCount <= 1 {
			return false, errors.New("cannot remove admin status from the last admin")
		}
	}

	// 5. อัพเดทสถานะแอดมิน
	if err := s.conversationRepo.UpdateMemberAdmin(conversationID, targetUserID, isAdmin); err != nil {
		return false, errors.New("error updating admin status: " + err.Error())
	}

	// 6. สร้างข้อความระบบ
	now := time.Now()
	actionUserName, _ := s.getUserName(userID)
	targetName, _ := s.getUserName(targetUserID)
	var systemMessage string
	if isAdmin {
		systemMessage = actionUserName + " made " + targetName + " an admin"
	} else {
		systemMessage = actionUserName + " removed admin status from " + targetName
	}

	s.createSystemMessage(conversationID, systemMessage)
	s.conversationRepo.UpdateLastMessage(conversationID, systemMessage, now)

	return isAdmin, nil
}

// Helper functions

// getUserName ดึงชื่อผู้ใช้ (ชื่อแสดงหรือชื่อผู้ใช้)
func (s *conversationMemberService) getUserName(userID uuid.UUID) (string, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return "", err
	}

	if user.DisplayName != "" {
		return user.DisplayName, nil
	}
	return user.Username, nil
}

// createSystemMessage สร้างข้อความระบบในการสนทนา
func (s *conversationMemberService) createSystemMessage(conversationID uuid.UUID, content string) error {
	systemMessage := &models.Message{
		ID:             uuid.New(),
		ConversationID: conversationID,
		MessageType:    "system",
		Content:        content,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return s.messageRepo.Create(systemMessage)
}

// FindDirectConversationBetweenUsers ค้นหาการสนทนาแบบ direct ระหว่างผู้ใช้สองคน
func (s *conversationMemberService) FindDirectConversationBetweenUsers(userID, friendID uuid.UUID) (uuid.UUID, error) {
	// ดึงการสนทนาทั้งหมดที่ผู้ใช้ userID เป็นสมาชิก
	userMemberships, err := s.conversationRepo.GetUserMemberships(userID)
	if err != nil {
		return uuid.Nil, err
	}

	// สร้าง map ของ conversation IDs ที่ผู้ใช้เป็นสมาชิก
	userConversationIDs := make(map[uuid.UUID]bool)
	for _, membership := range userMemberships {
		userConversationIDs[membership.ConversationID] = true
	}

	// ดึงการสนทนาทั้งหมดที่ friendID เป็นสมาชิก
	friendMemberships, err := s.conversationRepo.GetUserMemberships(friendID)
	if err != nil {
		return uuid.Nil, err
	}

	// ตรวจสอบการสนทนาที่ทั้งสองคนเป็นสมาชิก
	for _, membership := range friendMemberships {
		if userConversationIDs[membership.ConversationID] {
			// ดึงข้อมูลการสนทนา
			conversation, err := s.conversationRepo.GetByID(membership.ConversationID)
			if err != nil {
				continue
			}

			// ตรวจสอบว่าเป็นการสนทนาแบบ direct หรือไม่
			if conversation.Type == "direct" {
				// ตรวจสอบว่ามีสมาชิกเพียง 2 คน
				members, err := s.conversationRepo.GetMembers(conversation.ID)
				if err != nil || len(members) != 2 {
					continue
				}

				return conversation.ID, nil
			}
		}
	}

	return uuid.Nil, nil // ไม่พบการสนทนา direct ระหว่างผู้ใช้สองคน
}

// GetMember ดึงข้อมูลสมาชิกคนเดียว
func (s *conversationMemberService) GetMember(conversationID, userID uuid.UUID) (*models.ConversationMember, error) {
	member, err := s.conversationRepo.GetMember(conversationID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get member: %w", err)
	}
	if member == nil {
		return nil, errors.New("member not found")
	}
	return member, nil
}

// ChangeRole เปลี่ยน role ของสมาชิก
func (s *conversationMemberService) ChangeRole(conversationID, userID uuid.UUID, newRole models.MemberRole) (*models.ConversationMember, error) {
	// ดึงข้อมูลสมาชิก
	member, err := s.conversationRepo.GetMember(conversationID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get member: %w", err)
	}
	if member == nil {
		return nil, errors.New("member not found")
	}

	// อัปเดต role
	member.Role = newRole

	// Sync is_admin field for backward compatibility
	if newRole == models.RoleAdmin || newRole == models.RoleOwner {
		member.IsAdmin = true
	} else {
		member.IsAdmin = false
	}

	// บันทึกการเปลี่ยนแปลง
	if err := s.conversationRepo.UpdateMember(member); err != nil {
		return nil, fmt.Errorf("failed to update member role: %w", err)
	}

	return member, nil
}

// HasPermission ตรวจสอบว่าผู้ใช้มีสิทธิ์ทำอะไรใน conversation หรือไม่
func (s *conversationMemberService) HasPermission(conversationID, userID uuid.UUID, permission service.Permission) (bool, error) {
	member, err := s.conversationRepo.GetMember(conversationID, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get member: %w", err)
	}
	if member == nil {
		return false, errors.New("user is not a member of this conversation")
	}

	switch permission {
	case service.PermissionAddMember:
		// Owner และ Admin เท่านั้น
		return member.Role == models.RoleOwner || member.Role == models.RoleAdmin, nil

	case service.PermissionRemoveMember:
		// Owner และ Admin เท่านั้น
		return member.Role == models.RoleOwner || member.Role == models.RoleAdmin, nil

	case service.PermissionChangeRole:
		// Owner เท่านั้น
		return member.Role == models.RoleOwner, nil

	case service.PermissionUpdateInfo:
		// Owner และ Admin เท่านั้น
		return member.Role == models.RoleOwner || member.Role == models.RoleAdmin, nil

	case service.PermissionDeleteGroup:
		// Owner เท่านั้น
		return member.Role == models.RoleOwner, nil

	default:
		return false, errors.New("unknown permission")
	}
}
