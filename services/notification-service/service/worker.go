package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"time"
	"notification-service/domain"
	"notification-service/infra/rabbitmq"
	"github.com/google/uuid"
)

type NotificationWorker struct {
	ID          int
	db          domain.DatabaseInterface
	rabbitmq    domain.RabbitMQInterface
	smtp        domain.SMTPInterface
	authClient  domain.AuthServiceClient
	videoClient domain.VideoServiceClient
}

type EmailData struct {
	UserName       string
	VideoID        string
	VideoName      string
	FrameCount     int
	ZipSize        string
	DownloadURL    string
	ErrorMessage   string
	ProcessingTime string
}

func NewNotificationWorker(id int, db domain.DatabaseInterface, rabbitmq domain.RabbitMQInterface, smtp domain.SMTPInterface, authClient domain.AuthServiceClient, videoClient domain.VideoServiceClient) *NotificationWorker {
	return &NotificationWorker{
		ID:          id,
		db:          db,
		rabbitmq:    rabbitmq,
		smtp:        smtp,
		authClient:  authClient,
		videoClient: videoClient,
	}
}

func (w *NotificationWorker) Start(ctx context.Context) {
	log.Printf("Notification Worker %d started", w.ID)

	msgs, err := w.rabbitmq.SubscribeNotification()
	if err != nil {
		log.Printf("Worker %d: Failed to subscribe to notification queue: %v", w.ID, err)
		return
	}

	for msg := range msgs {
		select {
		case <-ctx.Done():
			log.Printf("Notification Worker %d stopping...", w.ID)
			return
		default:
			var message rabbitmq.NotificationMessage
			if err := json.Unmarshal(msg.Body, &message); err != nil {
				log.Printf("Worker %d: Error parsing message: %v", w.ID, err)
				msg.Nack(false, false)
				continue
			}

			log.Printf("Worker %d: Sending notification for video %s", w.ID, message.VideoID)
			err = w.sendNotification(ctx, &message)

			if err != nil {
				if isNotFoundErr(err) {
					log.Printf("Worker %d: Dropping message for missing data (404): %v", w.ID, err)
					msg.Ack(false)
				} else {
					log.Printf("Worker %d: Error sending notification: %v", w.ID, err)
					msg.Nack(false, true)
				}
			} else {
				log.Printf("Worker %d: Notification sent successfully", w.ID)
				msg.Ack(false)
			}
		}
	}
}

func (w *NotificationWorker) sendNotification(ctx context.Context, message *rabbitmq.NotificationMessage) error {
	user, err := w.authClient.GetUserByID(message.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user from Auth Service: %w", err)
	}

	video, err := w.videoClient.GetVideoByID(message.VideoID)
	if err != nil {
		return fmt.Errorf("failed to get video from Video Service: %w", err)
	}

	notification := &domain.Notification{
		ID:        generateID(),
		UserID:    message.UserID,
		VideoID:   &message.VideoID,
		Type:      "email",
		Status:    "pending",
		Subject:   message.Subject,
		Message:   message.Message,
		Recipient: user.Email,
		CreatedAt: time.Now(),
	}

	if err := w.db.CreateNotification(notification); err != nil {
		return fmt.Errorf("failed to create notification record: %w", err)
	}

	emailData := EmailData{
		UserName:  user.Name,
		VideoID:   video.ID,
		VideoName: video.OriginalName,
	}

	var htmlBody string
	

	switch message.Type {
	case "video_completed":
		if video.FrameCount != nil {
			emailData.FrameCount = *video.FrameCount
		}
		if video.ZipSizeBytes != nil {
			emailData.ZipSize = formatBytes(*video.ZipSizeBytes)
		}
		emailData.DownloadURL = fmt.Sprintf("http://localhost:8080/api/v1/videos/%s/download", video.ID)
		
		if video.ProcessingStartedAt != nil && video.ProcessingCompletedAt != nil {
			duration := video.ProcessingCompletedAt.Sub(*video.ProcessingStartedAt)
			emailData.ProcessingTime = formatDuration(duration)
		}

		htmlBody, err = w.renderTemplate("video_completed.html", emailData)

	case "video_failed":
		if video.ErrorMessage != nil {
			emailData.ErrorMessage = *video.ErrorMessage
		} else {
			emailData.ErrorMessage = "Unknown error occurred"
		}

		htmlBody, err = w.renderTemplate("video_failed.html", emailData)

	default:
		htmlBody = message.Message
	}

	if err != nil {
		notification.Status = "failed"
		notification.ErrorMessage = stringPtr(err.Error())
		w.db.UpdateNotification(notification)
		return fmt.Errorf("failed to render template: %w", err)
	}

	err = w.smtp.SendEmail(user.Email, message.Subject, htmlBody)
	if err != nil {
		notification.Status = "failed"
		notification.ErrorMessage = stringPtr(err.Error())
		notification.RetryCount++
		w.db.UpdateNotification(notification)
		return fmt.Errorf("failed to send email: %w", err)
	}

	notification.Status = "sent"
	notification.SentAt = timePtr(time.Now())
	w.db.UpdateNotification(notification)

	log.Printf("Worker %d: Email sent to %s for video %s", w.ID, user.Email, video.ID)
	return nil
}

func (w *NotificationWorker) renderTemplate(templateName string, data EmailData) (string, error) {
	tmpl, err := template.ParseFiles(fmt.Sprintf("templates/%s", templateName))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatDuration(d time.Duration) string {
	seconds := int(d.Seconds())
	if seconds < 60 {
		return fmt.Sprintf("%d seconds", seconds)
	}
	minutes := seconds / 60
	seconds = seconds % 60
	if minutes < 60 {
		return fmt.Sprintf("%d minutes %d seconds", minutes, seconds)
	}
	hours := minutes / 60
	minutes = minutes % 60
	return fmt.Sprintf("%d hours %d minutes", hours, minutes)
}

func generateID() string {
	return uuid.New().String()
}

func stringPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	return bytes.Contains([]byte(err.Error()), []byte("404"))
}
