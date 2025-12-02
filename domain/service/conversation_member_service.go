// domain/service/conversation_member_service.go
package service

import (
	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/dto"
	"github.com/thizplus/gofiber-chat-api/domain/models"
)

// Permission represents permissions in a conversation
type Permission string

const (
	PermissionAddMember    Permission = "add_member"
	PermissionRemoveMember Permission = "remove_member"
	PermissionChangeRole   Permission = "change_role"
	PermissionUpdateInfo   Permission = "update_info"
	PermissionDeleteGroup  Permission = "delete_group"
)

// ConversationMemberService interface สำหรับจัดการสมาชิกในการสนทนา
type ConversationMemberService interface {
	// AddMember เพิ่มสมาชิกในการสนทนากลุ่ม
	AddMember(userID, conversationID, newMemberID uuid.UUID) (*dto.MemberDTO, error)

	// BulkAddMembers เพิ่มสมาชิกหลายคนพร้อมกันในการสนทนากลุ่ม
	BulkAddMembers(userID, conversationID uuid.UUID, newMemberIDs []uuid.UUID) (addedMembers []*dto.MemberDTO, failed []struct {
		UserID uuid.UUID
		Reason string
	}, err error)

	// GetMembers ดึงรายการสมาชิกในการสนทนา
	GetMembers(userID, conversationID uuid.UUID, page, limit int) ([]*dto.MemberDTO, int, error)

	// GetMember ดึงข้อมูลสมาชิกคนเดียว
	GetMember(conversationID, userID uuid.UUID) (*models.ConversationMember, error)

	// RemoveMember ลบสมาชิกออกจากการสนทนา
	RemoveMember(userID, conversationID, memberToRemoveID uuid.UUID) error

	// ToggleAdminStatus เปลี่ยนสถานะแอดมินของสมาชิก (Deprecated: use ChangeRole)
	ToggleAdminStatus(userID, conversationID, targetUserID uuid.UUID, isAdmin bool) (bool, error)

	// ChangeRole เปลี่ยน role ของสมาชิก
	ChangeRole(conversationID, userID uuid.UUID, newRole models.MemberRole) (*models.ConversationMember, error)

	// HasPermission ตรวจสอบว่าผู้ใช้มีสิทธิ์ทำอะไรใน conversation หรือไม่
	HasPermission(conversationID, userID uuid.UUID, permission Permission) (bool, error)

	//ค้นหาการสนทนาแบบ direct ระหว่างผู้ใช้สองคน
	FindDirectConversationBetweenUsers(userID, friendID uuid.UUID) (uuid.UUID, error)
}
