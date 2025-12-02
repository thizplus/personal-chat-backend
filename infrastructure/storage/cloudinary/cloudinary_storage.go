// infrastructure/storage/cloudinary/cloudinary_storage.go
package cloudinary

import (
	"context"
	"fmt"
	"mime/multipart"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/thizplus/gofiber-chat-api/domain/service"
)

// cloudinaryStorage จัดการการเก็บไฟล์ด้วย Cloudinary
type cloudinaryStorage struct {
	cld    *cloudinary.Cloudinary
	ctx    context.Context
	config *CloudinaryConfig
}

// NewCloudinaryStorage สร้าง FileStorageService ที่ใช้ Cloudinary
func NewCloudinaryStorage(config *CloudinaryConfig) (service.FileStorageService, error) {
	// สร้าง context
	ctx := context.Background()

	// สร้าง Cloudinary client
	cld, err := cloudinary.NewFromParams(config.CloudName, config.APIKey, config.APISecret)
	if err != nil {
		return nil, err
	}

	return &cloudinaryStorage{
		cld:    cld,
		ctx:    ctx,
		config: config,
	}, nil
}

// ใช้ฟังก์ชันช่วยสร้าง pointer to bool
func boolPtr(b bool) *bool {
	return &b
}

// UploadImage อัปโหลดรูปภาพไปยัง Cloudinary
func (c *cloudinaryStorage) UploadImage(file *multipart.FileHeader, folder string) (*service.FileUploadResult, error) {
	// เปิดไฟล์
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	// กำหนดตัวเลือกในการอัปโหลด
	uploadParams := uploader.UploadParams{
		Folder:         folder,
		UseFilename:    boolPtr(true),
		UniqueFilename: boolPtr(true),
		ResourceType:   "image",
		Transformation: "q_auto:good",
	}

	// อัปโหลดไปยัง Cloudinary
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	result, err := c.cld.Upload.Upload(ctx, src, uploadParams)
	if err != nil {
		return nil, err
	}

	// แปลงผลลัพธ์เป็น domain model
	return &service.FileUploadResult{
		URL:          result.SecureURL,
		Path:         result.PublicID, // ใช้ PublicID เป็น Path สำหรับ Cloudinary
		PublicID:     result.PublicID,
		ResourceType: result.ResourceType,
		Format:       result.Format,
		Size:         int(result.Bytes),
		Width:        result.Width,
		Height:       result.Height,
		Metadata:     map[string]string{},
	}, nil
}

// UploadFile อัปโหลดไฟล์ทั่วไปไปยัง Cloudinary
func (c *cloudinaryStorage) UploadFile(file *multipart.FileHeader, folder string) (*service.FileUploadResult, error) {
	// เปิดไฟล์
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	// กำหนดตัวเลือกในการอัปโหลด
	uploadParams := uploader.UploadParams{
		Folder:         folder,
		UseFilename:    boolPtr(true),
		UniqueFilename: boolPtr(true),
		ResourceType:   "auto",
	}

	// อัปโหลดไปยัง Cloudinary
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	result, err := c.cld.Upload.Upload(ctx, src, uploadParams)
	if err != nil {
		return nil, err
	}

	// แปลงผลลัพธ์เป็น domain model
	return &service.FileUploadResult{
		URL:          result.SecureURL,
		Path:         result.PublicID, // ใช้ PublicID เป็น Path สำหรับ Cloudinary
		PublicID:     result.PublicID,
		ResourceType: result.ResourceType,
		Format:       result.Format,
		Size:         int(result.Bytes),
		Width:        result.Width,
		Height:       result.Height,
		Metadata:     map[string]string{},
	}, nil
}

// DeleteFile ลบไฟล์จาก Cloudinary
func (c *cloudinaryStorage) DeleteFile(path string) error {
	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	// path สำหรับ Cloudinary คือ PublicID
	_, err := c.cld.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID: path,
	})
	if err != nil {
		return err
	}

	return nil
}

// GetPublicURL สร้าง public URL สำหรับไฟล์
func (c *cloudinaryStorage) GetPublicURL(path string) string {
	// สำหรับ Cloudinary, path คือ PublicID
	// สร้าง URL โดยใช้ cloud name
	return "https://res.cloudinary.com/" + c.config.CloudName + "/image/upload/" + path
}

// GeneratePresignedUploadURL สร้าง presigned URL สำหรับให้ client upload ตรง
// หมายเหตุ: Cloudinary ไม่รองรับ presigned URL แบบเดียวกับ S3/R2
// แต่รองรับ unsigned upload หรือ signed upload parameters
func (c *cloudinaryStorage) GeneratePresignedUploadURL(path string, contentType string, expiry time.Duration) (*service.PresignedURLResult, error) {
	// Cloudinary ใช้ signed parameters แทน presigned URL
	// สำหรับตอนนี้ return error เพราะต้อง implement แยก
	return nil, fmt.Errorf("presigned upload URL not implemented for Cloudinary")
}

// GeneratePresignedDownloadURL สร้าง presigned URL สำหรับ download ไฟล์
func (c *cloudinaryStorage) GeneratePresignedDownloadURL(path string, expiry time.Duration) (string, error) {
	// Cloudinary files are public by default
	// ถ้าต้องการ private access ต้องใช้ signed URLs
	return c.GetPublicURL(path), nil
}
