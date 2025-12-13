// application/serviceimpl/user_friendship_service.go
package serviceimpl

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/thizplus/gofiber-chat-api/domain/models"
	"github.com/thizplus/gofiber-chat-api/domain/repository"
	"github.com/thizplus/gofiber-chat-api/domain/service"
)

type userFriendshipService struct {
	userFriendshipRepo repository.UserFriendshipRepository
	userRepo           repository.UserRepository
}

func NewUserFriendshipService(
	userFriendshipRepo repository.UserFriendshipRepository,
	userRepo repository.UserRepository,

) service.UserFriendshipService {
	return &userFriendshipService{
		userFriendshipRepo: userFriendshipRepo,
		userRepo:           userRepo,
	}
}

// SendFriendRequest ส่งคำขอเป็นเพื่อน
func (s *userFriendshipService) SendFriendRequest(userID, friendID uuid.UUID) (*models.UserFriendship, error) {
	return s.SendFriendRequestWithMessage(userID, friendID, nil)
}

// SendFriendRequestWithMessage ส่งคำขอเป็นเพื่อนพร้อมข้อความ (Message Request feature)
func (s *userFriendshipService) SendFriendRequestWithMessage(userID, friendID uuid.UUID, initialMessage *string) (*models.UserFriendship, error) {
	// ตรวจสอบว่าไม่ได้ส่งคำขอเป็นเพื่อนกับตัวเอง
	if userID == friendID {
		return nil, errors.New("cannot send friend request to yourself")
	}

	// ตรวจสอบว่ามีผู้ใช้ที่ต้องการเป็นเพื่อนอยู่ในระบบ
	_, err := s.userRepo.FindByID(friendID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// ตรวจสอบความสัมพันธ์ที่มีอยู่
	friendships, err := s.userFriendshipRepo.FindByUserIDOrFriendID(userID, friendID)
	if err == nil && len(friendships) > 0 {
		var rejectedFriendship *models.UserFriendship
		var activeFriendship *models.UserFriendship

		for _, friendship := range friendships {
			if friendship.Status == "rejected" {
				rejectedFriendship = friendship
			} else if friendship.Status == "pending" || friendship.Status == "accepted" {
				activeFriendship = friendship
			}
		}

		// ถ้าพบความสัมพันธ์ที่ active ให้ส่งข้อความว่ามีคำขออยู่แล้ว
		if activeFriendship != nil {
			return nil, errors.New("friend request already exists")
		}

		// ถ้าพบความสัมพันธ์ที่ถูกปฏิเสธ ให้อัพเดต record เดิม
		if rejectedFriendship != nil {
			// อัพเดตเป็น pending และอัพเดตเวลา
			now := time.Now()
			rejectedFriendship.Status = "pending"
			rejectedFriendship.RequestedAt = now
			rejectedFriendship.UpdatedAt = now

			// เพิ่ม initial message ถ้ามี
			if initialMessage != nil && *initialMessage != "" {
				rejectedFriendship.InitialMessage = initialMessage
				rejectedFriendship.InitialMessageAt = &now
			}

			// ถ้าทิศทางความสัมพันธ์เดิมเป็น เพื่อน -> ผู้ใช้ปัจจุบัน ให้สลับทิศทาง
			if rejectedFriendship.UserID != userID {
				rejectedFriendship.UserID = userID
				rejectedFriendship.FriendID = friendID
			}

			err := s.userFriendshipRepo.Update(rejectedFriendship)
			if err != nil {
				return nil, err
			}

			return rejectedFriendship, nil
		}
	}

	// สร้างคำขอเป็นเพื่อนใหม่
	now := time.Now()

	friendship := &models.UserFriendship{
		ID:          uuid.New(),
		UserID:      userID,
		FriendID:    friendID,
		Status:      "pending",
		RequestedAt: now,
		UpdatedAt:   now,
	}

	// เพิ่ม initial message ถ้ามี
	if initialMessage != nil && *initialMessage != "" {
		friendship.InitialMessage = initialMessage
		friendship.InitialMessageAt = &now
	}

	err = s.userFriendshipRepo.Create(friendship)
	if err != nil {
		return nil, err
	}

	return friendship, nil
}

// AcceptFriendRequest ยอมรับคำขอเป็นเพื่อน
// AcceptFriendRequest ยอมรับคำขอเป็นเพื่อน
func (s *userFriendshipService) AcceptFriendRequest(requestID, userID uuid.UUID) (*models.UserFriendship, error) {
	// ดึงข้อมูลคำขอเป็นเพื่อน
	friendship, err := s.userFriendshipRepo.FindByID(requestID)
	if err != nil {
		return nil, errors.New("friend request not found")
	}

	// ตรวจสอบว่าเป็นคำขอที่ส่งถึงผู้ใช้นี้และสถานะเป็น pending
	if friendship.FriendID != userID || friendship.Status != "pending" {
		return nil, errors.New("friend request not found or already processed")
	}

	// อัพเดตสถานะ
	err = s.userFriendshipRepo.UpdateStatus(requestID, "accepted")
	if err != nil {
		return nil, err
	}

	// ดึงข้อมูลที่อัพเดตแล้ว
	updatedFriendship, err := s.userFriendshipRepo.FindByID(requestID)
	if err != nil {
		return nil, err
	}

	return updatedFriendship, nil
}

// RejectFriendRequest ปฏิเสธคำขอเป็นเพื่อน
func (s *userFriendshipService) RejectFriendRequest(requestID, userID uuid.UUID) (*models.UserFriendship, error) {
	// ดึงข้อมูลคำขอเป็นเพื่อน
	friendship, err := s.userFriendshipRepo.FindByID(requestID)
	if err != nil {
		return nil, errors.New("friend request not found")
	}

	// ตรวจสอบว่าเป็นคำขอที่ส่งถึงผู้ใช้นี้และสถานะเป็น pending
	if friendship.FriendID != userID || friendship.Status != "pending" {
		return nil, errors.New("friend request not found or already processed")
	}

	// อัพเดตสถานะ
	err = s.userFriendshipRepo.UpdateStatus(requestID, "rejected")
	if err != nil {
		return nil, err
	}

	// ดึงข้อมูลที่อัพเดตแล้ว
	updatedFriendship, err := s.userFriendshipRepo.FindByID(requestID)
	if err != nil {
		return nil, err
	}

	return updatedFriendship, nil
}

// RemoveFriend ลบเพื่อน
func (s *userFriendshipService) RemoveFriend(userID, friendID uuid.UUID) error {
	// ลบความสัมพันธ์แบบเพื่อนที่ยอมรับแล้ว
	return s.userFriendshipRepo.DeleteByUserIDAndFriendID(userID, friendID)
}

// GetFriends ดึงรายชื่อเพื่อนทั้งหมด
func (s *userFriendshipService) GetFriends(userID uuid.UUID) ([]*models.User, error) {
	// ดึงความสัมพันธ์แบบเพื่อนที่ยอมรับแล้ว
	friendships, err := s.userFriendshipRepo.FindAcceptedFriendships(userID)
	if err != nil {
		return nil, err
	}

	friends := make([]*models.User, 0)
	for _, friendship := range friendships {
		var friendID uuid.UUID
		if friendship.UserID == userID {
			friendID = friendship.FriendID
		} else {
			friendID = friendship.UserID
		}

		friend, err := s.userRepo.FindByID(friendID)
		if err != nil {
			continue
		}

		friends = append(friends, friend)
	}

	return friends, nil
}

// GetPendingRequests ดึงคำขอเป็นเพื่อนที่รอการตอบรับ
func (s *userFriendshipService) GetPendingRequests(userID uuid.UUID) ([]*models.UserFriendship, error) {
	// ดึงคำขอเป็นเพื่อนที่รอการตอบรับจากผู้ใช้นี้
	return s.userFriendshipRepo.FindPendingRequestsByFriendID(userID)
}

// GetSentRequests ดึงคำขอเป็นเพื่อนที่ส่งไป
func (s *userFriendshipService) GetSentRequests(userID uuid.UUID) ([]*models.UserFriendship, error) {
	// ดึงคำขอเป็นเพื่อนที่ส่งโดยผู้ใช้นี้และยังเป็น pending อยู่
	return s.userFriendshipRepo.FindPendingRequestsByUserID(userID)
}

// CancelFriendRequest ยกเลิกคำขอเป็นเพื่อนที่ส่งไป
func (s *userFriendshipService) CancelFriendRequest(requestID, userID uuid.UUID) error {
	// ดึงข้อมูล friendship
	friendship, err := s.userFriendshipRepo.FindByID(requestID)
	if err != nil {
		return errors.New("friend request not found")
	}

	// ตรวจสอบว่าเป็นคำขอที่ส่งโดยผู้ใช้นี้และยังเป็น pending อยู่
	if friendship.UserID != userID {
		return errors.New("you can only cancel your own friend requests")
	}

	if friendship.Status != "pending" {
		return errors.New("can only cancel pending friend requests")
	}

	// ลบคำขอ
	return s.userFriendshipRepo.Delete(requestID)
}

// BlockUser บล็อกผู้ใช้
func (s *userFriendshipService) BlockUser(userID, targetID uuid.UUID) error {
	// ตรวจสอบว่ามีความสัมพันธ์อยู่แล้วหรือไม่
	friendships, err := s.userFriendshipRepo.FindByUserIDOrFriendID(userID, targetID)

	if err == nil && len(friendships) > 0 {
		// มีความสัมพันธ์อยู่แล้ว ให้ลบความสัมพันธ์เดิมก่อน
		err = s.userFriendshipRepo.DeleteByUserIDAndFriendID(userID, targetID)
		if err != nil {
			return err
		}
	}

	// สร้างความสัมพันธ์แบบบล็อก

	now := time.Now()

	friendship := &models.UserFriendship{
		ID:          uuid.New(),
		UserID:      userID,
		FriendID:    targetID,
		Status:      "blocked",
		RequestedAt: now,
		UpdatedAt:   now,
	}

	return s.userFriendshipRepo.Create(friendship)
}

// UnblockUser เลิกบล็อกผู้ใช้
func (s *userFriendshipService) UnblockUser(userID, targetID uuid.UUID) error {
	// ลบความสัมพันธ์แบบบล็อก
	friendships, err := s.userFriendshipRepo.FindByUserIDAndFriendID(userID, targetID)
	if err != nil {
		return err
	}

	if friendships.Status != "blocked" {
		return errors.New("user is not blocked")
	}

	return s.userFriendshipRepo.Delete(friendships.ID)
}

// GetBlockedUsers ดึงรายชื่อผู้ใช้ที่ถูกบล็อก
func (s *userFriendshipService) GetBlockedUsers(userID uuid.UUID) ([]*models.User, error) {
	// ดึงความสัมพันธ์แบบบล็อก
	blockedFriendships, err := s.userFriendshipRepo.FindBlockedUsers(userID)
	if err != nil {
		return nil, err
	}

	blockedUsers := make([]*models.User, 0)
	for _, friendship := range blockedFriendships {
		blockedUser, err := s.userRepo.FindByID(friendship.FriendID)
		if err != nil {
			continue
		}

		blockedUsers = append(blockedUsers, blockedUser)
	}

	return blockedUsers, nil
}

// GetBlockedByUsers ดึงรายชื่อผู้ใช้ที่บล็อกเรา
func (s *userFriendshipService) GetBlockedByUsers(userID uuid.UUID) ([]*models.User, error) {
	// ดึงความสัมพันธ์ที่เราถูกบล็อก (เราคือ friend_id)
	blockedByFriendships, err := s.userFriendshipRepo.FindBlockedByUsers(userID)
	if err != nil {
		return nil, err
	}

	blockedByUsers := make([]*models.User, 0)
	for _, friendship := range blockedByFriendships {
		// ดึงข้อมูลของคนที่บล็อกเรา (UserID ไม่ใช่ FriendID)
		blockerUser, err := s.userRepo.FindByID(friendship.UserID)
		if err != nil {
			continue
		}

		blockedByUsers = append(blockedByUsers, blockerUser)
	}

	return blockedByUsers, nil
}

// GetFriendshipStatus ตรวจสอบความสัมพันธ์ระหว่างผู้ใช้สองคน
func (s *userFriendshipService) GetFriendshipStatus(userID, otherUserID uuid.UUID) (string, uuid.UUID, error) {
	friendships, err := s.userFriendshipRepo.FindByUserIDOrFriendID(userID, otherUserID)
	if err != nil || len(friendships) == 0 {
		return "none", uuid.Nil, nil // ใช้ uuid.Nil แทนสตริงว่าง
	}

	return friendships[0].Status, friendships[0].ID, nil // ส่งคืน UUID โดยตรง ไม่ต้องแปลงเป็นสตริง
}

// IsFriend ตรวจสอบว่าผู้ใช้สองคนเป็นเพื่อนกันหรือไม่
func (s *userFriendshipService) IsFriend(userID, otherUserID uuid.UUID) (bool, error) {
	friendships, err := s.userFriendshipRepo.FindByUserIDOrFriendID(userID, otherUserID)
	if err != nil || len(friendships) == 0 {
		return false, nil
	}

	return friendships[0].Status == "accepted", nil
}

// HasPendingRequest ตรวจสอบว่ามีคำขอเป็นเพื่อนที่รอการตอบรับอยู่หรือไม่
func (s *userFriendshipService) HasPendingRequest(userID, otherUserID uuid.UUID) (bool, string, error) {
	friendships, err := s.userFriendshipRepo.FindByUserIDOrFriendID(userID, otherUserID)
	if err != nil || len(friendships) == 0 {
		return false, "", nil
	}

	if friendships[0].Status != "pending" {
		return false, "", nil
	}

	// ตรวจสอบทิศทางของคำขอ
	direction := ""
	if friendships[0].UserID == userID {
		direction = "sent"
	} else {
		direction = "received"
	}

	return true, direction, nil
}

// IsBlocked ตรวจสอบว่า userID บล็อค targetID หรือไม่
func (s *userFriendshipService) IsBlocked(userID, targetID uuid.UUID) (bool, error) {
	friendship, err := s.userFriendshipRepo.FindByUserIDAndFriendID(userID, targetID)
	if err != nil {
		return false, nil // ถ้าไม่เจอ record แสดงว่าไม่ได้บล็อค
	}

	return friendship.Status == "blocked", nil
}

// IsBlockedBy ตรวจสอบว่า userID ถูก targetID บล็อคหรือไม่
func (s *userFriendshipService) IsBlockedBy(userID, targetID uuid.UUID) (bool, error) {
	// ตรวจสอบว่า targetID บล็อค userID หรือไม่
	friendship, err := s.userFriendshipRepo.FindByUserIDAndFriendID(targetID, userID)
	if err != nil {
		return false, nil // ถ้าไม่เจอ record แสดงว่าไม่ได้ถูกบล็อค
	}

	return friendship.Status == "blocked", nil
}

// CheckBlockStatus ตรวจสอบ block status แบบ bidirectional
func (s *userFriendshipService) CheckBlockStatus(user1ID, user2ID uuid.UUID) (isBlocked bool, isBlockedBy bool, err error) {
	// ตรวจสอบว่า user1 บล็อค user2 หรือไม่
	isBlocked, err = s.IsBlocked(user1ID, user2ID)
	if err != nil {
		return false, false, err
	}

	// ตรวจสอบว่า user1 ถูก user2 บล็อคหรือไม่
	isBlockedBy, err = s.IsBlockedBy(user1ID, user2ID)
	if err != nil {
		return false, false, err
	}

	return isBlocked, isBlockedBy, nil
}
