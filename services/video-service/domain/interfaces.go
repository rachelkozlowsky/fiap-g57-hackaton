package domain

import (
	"io"
	"github.com/minio/minio-go/v7"
)

type DatabaseInterface interface {
	CreateVideo(video *Video) error
	GetVideoByID(id string) (*Video, error)
	GetVideosByUserID(userID, status string) ([]*Video, error)
	UpdateVideo(video *Video) error
	DeleteVideo(id string) error
	GetUserStats(userID string) (*UserStats, error)
	GetSystemStats() (*SystemStats, error)


    Ping() error
    Close() error
}

type MinIOInterface interface {
	UploadFile(reader io.Reader, filename string, size int64) (string, error)
	DeleteFile(objectName string) error
	GetFileStream(objectName string) (*minio.Object, error)
}

type RabbitMQInterface interface {
	PublishVideoUpload(message VideoProcessingMessage) error
    Ping() error
    Close() error
}

type AuthServiceClient interface {
	CreateAuditLog(req AuditLogRequest) error
}

