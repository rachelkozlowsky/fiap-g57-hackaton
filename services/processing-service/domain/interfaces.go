package domain

import (
	"io"
	amqp "github.com/rabbitmq/amqp091-go"
)

type DatabaseInterface interface {
	CreateProcessingJob(job *ProcessingJob) error
	UpdateProcessingJob(job *ProcessingJob) error
}

type MinIOInterface interface {
	DownloadFile(objectName, destPath string) error
	UploadProcessedFile(reader io.Reader, filename string, size int64) (string, error)
}

type RabbitMQInterface interface {
	PublishNotification(message NotificationMessage) error
	SubscribeVideoUpload() (<-chan amqp.Delivery, error)
}

type VideoServiceClient interface {
	GetVideoByID(videoID string) (*Video, error)
	UpdateVideoStatus(videoID, status string, errorMessage string) error
	CompleteVideo(videoID, zipPath string, zipSize int64, frameCount int) error
	FailVideo(videoID, errorMessage string) error
}
