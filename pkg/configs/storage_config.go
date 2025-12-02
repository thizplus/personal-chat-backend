// pkg/configs/storage_config.go
package configs

import (
	"fmt"
	"log"
	"os"

	"github.com/thizplus/gofiber-chat-api/domain/service"
	"github.com/thizplus/gofiber-chat-api/infrastructure/storage/cloudinary"
	"github.com/thizplus/gofiber-chat-api/infrastructure/storage/r2"
)

// SetupStorageService สร้าง FileStorageService ตาม environment
func SetupStorageService() (service.FileStorageService, error) {
	storageType := os.Getenv("STORAGE_TYPE")

	// Default to cloudinary if not specified
	if storageType == "" {
		storageType = "cloudinary"
	}

	log.Printf("Setting up storage service with type: %s", storageType)

	switch storageType {
	case "cloudinary":
		return cloudinary.NewCloudinaryStorage(&cloudinary.CloudinaryConfig{
			CloudName:    os.Getenv("CLOUDINARY_CLOUD_NAME"),
			APIKey:       os.Getenv("CLOUDINARY_API_KEY"),
			APISecret:    os.Getenv("CLOUDINARY_API_SECRET"),
			UploadFolder: os.Getenv("CLOUDINARY_UPLOAD_FOLDER"),
		})

	case "r2":
		return r2.NewR2Storage(&r2.R2Config{
			AccountID:       os.Getenv("R2_ACCOUNT_ID"),
			AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
			Bucket:          os.Getenv("R2_BUCKET"),
			PublicURL:       os.Getenv("R2_PUBLIC_URL"),
			Region:          os.Getenv("R2_REGION"),
		})

	// ในอนาคตอาจเพิ่ม case อื่นๆ เช่น "s3" หรือ "local"
	// case "s3":
	//     return s3.NewS3Storage(&s3.S3Config{
	//         ...
	//     })
	// case "local":
	//     return local.NewLocalStorage(&local.LocalConfig{
	//         ...
	//     })

	default:
		return nil, fmt.Errorf("unsupported storage type: %s (supported: cloudinary, r2)", storageType)
	}
}
