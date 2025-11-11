// interfaces/api/middleware/business_admin.go
package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/thizplus/gofiber-chat-api/domain/service"
	"github.com/thizplus/gofiber-chat-api/pkg/utils"
)

// CheckBusinessAdmin middleware ตรวจสอบว่าผู้ใช้เป็นแอดมินของธุรกิจ
func CheckBusinessAdmin(businessAdminService service.BusinessAdminService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// ดึง userID จาก middleware ก่อนหน้า
		userID, err := GetUserUUID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "Unauthorized: " + err.Error(),
			})
		}

		// ดึง businessID จาก URL parameter
		businessID, err := utils.ParseUUIDParam(c, "businessId")
		if err != nil {
			return err
		}

		// ตรวจสอบสิทธิ์
		hasPermission, err := businessAdminService.CheckAdminPermission(userID, businessID, []string{})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Error checking permissions: " + err.Error(),
			})
		}

		if !hasPermission {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"message": "You don't have permission to access this business x3",
			})
		}

		// ดึงข้อมูลบทบาทของผู้ใช้
		admin, err := businessAdminService.GetAdminByUserAndBusinessID(userID, businessID)
		var userRole string = "member" // ค่าเริ่มต้น
		if err == nil && admin != nil {
			userRole = admin.Role
		}

		// เก็บข้อมูลใน context
		c.Locals("businessID", businessID)
		c.Locals("businessRole", userRole)
		c.Locals("businessUserID", userID)

		return c.Next()
	}
}

// CheckBusinessAdminWithRoles middleware ตรวจสอบบทบาทเฉพาะ
func CheckBusinessAdminWithRoles(businessAdminService service.BusinessAdminService, allowedRoles []string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// ดึง userID จาก middleware ก่อนหน้า
		userID, err := GetUserUUID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "Unauthorized: " + err.Error(),
			})
		}

		// ดึง businessID จาก URL parameter
		businessID, err := utils.ParseUUIDParam(c, "businessId")
		if err != nil {
			return err
		}

		// ตรวจสอบสิทธิ์พร้อมบทบาท
		hasPermission, err := businessAdminService.CheckAdminPermission(userID, businessID, allowedRoles)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Error checking permissions: " + err.Error(),
			})
		}

		if !hasPermission {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"message": "You don't have sufficient permissions for this action",
			})
		}

		// ดึงข้อมูลบทบาทของผู้ใช้
		admin, err := businessAdminService.GetAdminByUserAndBusinessID(userID, businessID)
		var userRole string = "member"
		if err == nil && admin != nil {
			userRole = admin.Role
		}

		// เก็บข้อมูลใน context
		c.Locals("businessID", businessID)
		c.Locals("businessRole", userRole)
		c.Locals("businessUserID", userID)

		return c.Next()
	}
}
