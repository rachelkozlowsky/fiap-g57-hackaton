package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"notification-service/domain"
	"notification-service/infra/rabbitmq"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ─── Mocks ────────────────────────────────────────────────────────────────────

type MockDatabase struct{ mock.Mock }

func (m *MockDatabase) CreateNotification(notification *domain.Notification) error {
	return m.Called(notification).Error(0)
}
func (m *MockDatabase) UpdateNotification(notification *domain.Notification) error {
	return m.Called(notification).Error(0)
}
func (m *MockDatabase) Ping() error  { return m.Called().Error(0) }
func (m *MockDatabase) Close() error { return m.Called().Error(0) }

type MockAuthClient struct{ mock.Mock }

func (m *MockAuthClient) GetUserByID(id string) (*domain.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

type MockVideoClient struct{ mock.Mock }

func (m *MockVideoClient) GetVideoByID(id string) (*domain.Video, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Video), args.Error(1)
}

type MockSMTPClient struct{ mock.Mock }

func (m *MockSMTPClient) SendEmail(to, subject, body string) error {
	return m.Called(to, subject, body).Error(0)
}

type MockRabbitMQ struct{ mock.Mock }

func (m *MockRabbitMQ) SubscribeNotification() (<-chan amqp.Delivery, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(<-chan amqp.Delivery), args.Error(1)
}

type MockAcknowledger struct{ mock.Mock }

func (m *MockAcknowledger) Ack(tag uint64, multiple bool) error {
	return m.Called(tag, multiple).Error(0)
}
func (m *MockAcknowledger) Nack(tag uint64, multiple bool, requeue bool) error {
	return m.Called(tag, multiple, requeue).Error(0)
}
func (m *MockAcknowledger) Reject(tag uint64, requeue bool) error {
	return m.Called(tag, requeue).Error(0)
}

// helper — builds worker with all 6 args, accepting nil for unused interfaces
func newTestWorker(id int, db *MockDatabase, mq *MockRabbitMQ, smtp *MockSMTPClient, auth *MockAuthClient, video *MockVideoClient) *NotificationWorker {
	var dbI domain.DatabaseInterface
	var mqI domain.RabbitMQInterface
	var smtpI domain.SMTPInterface
	var authI domain.AuthServiceClient
	var videoI domain.VideoServiceClient
	if db != nil {
		dbI = db
	}
	if mq != nil {
		mqI = mq
	}
	if smtp != nil {
		smtpI = smtp
	}
	if auth != nil {
		authI = auth
	}
	if video != nil {
		videoI = video
	}
	return NewNotificationWorker(id, dbI, mqI, smtpI, authI, videoI)
}

// ─── NewNotificationWorker ────────────────────────────────────────────────────

func TestNewNotificationWorker(t *testing.T) {
	w := newTestWorker(1, new(MockDatabase), nil, new(MockSMTPClient), nil, nil)
	assert.NotNil(t, w)
	assert.Equal(t, 1, w.ID)
}

func TestNewNotificationWorker_MultipleWorkers(t *testing.T) {
	for id := 1; id <= 3; id++ {
		w := newTestWorker(id, nil, nil, nil, nil, nil)
		assert.Equal(t, id, w.ID)
	}
}

// ─── sendNotification ─────────────────────────────────────────────────────────

func TestSendNotification_CustomType(t *testing.T) {
	db := new(MockDatabase)
	smtp := new(MockSMTPClient)
	auth := new(MockAuthClient)
	video := new(MockVideoClient)

	user := &domain.User{ID: "user1", Email: "test@example.com", Name: "Test User"}
	vid := &domain.Video{ID: "video1", UserID: "user1", OriginalName: "test.mp4", Status: "completed"}
	message := &rabbitmq.NotificationMessage{
		VideoID: "video1", UserID: "user1",
		Type: "custom_notification", Subject: "Custom Notification",
		Message: "This is a plain text message",
	}

	auth.On("GetUserByID", "user1").Return(user, nil)
	video.On("GetVideoByID", "video1").Return(vid, nil)
	db.On("CreateNotification", mock.AnythingOfType("*domain.Notification")).Return(nil)
	db.On("UpdateNotification", mock.AnythingOfType("*domain.Notification")).Return(nil)
	smtp.On("SendEmail", "test@example.com", "Custom Notification", "This is a plain text message").Return(nil)

	w := newTestWorker(1, db, nil, smtp, auth, video)
	err := w.sendNotification(context.Background(), message)

	assert.NoError(t, err)
	db.AssertExpectations(t)
	smtp.AssertExpectations(t)
}

func TestSendNotification_UserNotFound(t *testing.T) {
	auth := new(MockAuthClient)
	auth.On("GetUserByID", "user999").Return(nil, errors.New("user not found"))

	w := newTestWorker(1, nil, nil, nil, auth, nil)
	err := w.sendNotification(context.Background(), &rabbitmq.NotificationMessage{
		VideoID: "video1", UserID: "user999", Type: "custom",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get user")
	auth.AssertExpectations(t)
}

func TestSendNotification_VideoNotFound(t *testing.T) {
	auth := new(MockAuthClient)
	video := new(MockVideoClient)

	auth.On("GetUserByID", "user1").Return(&domain.User{ID: "user1", Email: "u@e.com"}, nil)
	video.On("GetVideoByID", "video999").Return(nil, errors.New("video not found"))

	w := newTestWorker(1, nil, nil, nil, auth, video)
	err := w.sendNotification(context.Background(), &rabbitmq.NotificationMessage{
		VideoID: "video999", UserID: "user1", Type: "custom",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get video")
	video.AssertExpectations(t)
}

func TestSendNotification_CreateNotificationError(t *testing.T) {
	db := new(MockDatabase)
	auth := new(MockAuthClient)
	video := new(MockVideoClient)

	auth.On("GetUserByID", "u1").Return(&domain.User{ID: "u1", Email: "u@e.com"}, nil)
	video.On("GetVideoByID", "v1").Return(&domain.Video{ID: "v1"}, nil)
	db.On("CreateNotification", mock.Anything).Return(errors.New("db error"))

	w := newTestWorker(1, db, nil, nil, auth, video)
	err := w.sendNotification(context.Background(), &rabbitmq.NotificationMessage{
		UserID: "u1", VideoID: "v1", Type: "custom",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create notification")
}

func TestSendNotification_SMTPError(t *testing.T) {
	db := new(MockDatabase)
	smtp := new(MockSMTPClient)
	auth := new(MockAuthClient)
	video := new(MockVideoClient)

	auth.On("GetUserByID", "u1").Return(&domain.User{ID: "u1", Email: "u@e.com", Name: "U"}, nil)
	video.On("GetVideoByID", "v1").Return(&domain.Video{ID: "v1", OriginalName: "v.mp4"}, nil)
	db.On("CreateNotification", mock.Anything).Return(nil)
	db.On("UpdateNotification", mock.Anything).Return(nil)
	smtp.On("SendEmail", "u@e.com", "Sub", "Msg").Return(errors.New("smtp error"))

	w := newTestWorker(1, db, nil, smtp, auth, video)
	err := w.sendNotification(context.Background(), &rabbitmq.NotificationMessage{
		UserID: "u1", VideoID: "v1", Type: "custom", Subject: "Sub", Message: "Msg",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send email")
}

func TestSendNotification_VideoCompleted_Success(t *testing.T) {
	db := new(MockDatabase)
	smtp := new(MockSMTPClient)
	auth := new(MockAuthClient)
	video := new(MockVideoClient)

	frameCount := 100
	zipSize := int64(1024 * 1024)
	user := &domain.User{ID: "u1", Email: "test@example.com", Name: "User"}
	vid := &domain.Video{
		ID: "v1", OriginalName: "video.mp4",
		FrameCount: &frameCount, ZipSizeBytes: &zipSize,
		ProcessingStartedAt:   timePtr(time.Now().Add(-1 * time.Minute)),
		ProcessingCompletedAt: timePtr(time.Now()),
	}

	os.MkdirAll("templates", 0755)
	os.WriteFile("templates/video_completed.html", []byte("Completed: {{.VideoName}}"), 0644)
	defer os.RemoveAll("templates")

	auth.On("GetUserByID", "u1").Return(user, nil)
	video.On("GetVideoByID", "v1").Return(vid, nil)
	db.On("CreateNotification", mock.Anything).Return(nil)
	db.On("UpdateNotification", mock.Anything).Return(nil)
	smtp.On("SendEmail", "test@example.com", "Done", "Completed: video.mp4").Return(nil)

	w := newTestWorker(1, db, nil, smtp, auth, video)
	err := w.sendNotification(context.Background(), &rabbitmq.NotificationMessage{
		UserID: "u1", VideoID: "v1", Type: "video_completed", Subject: "Done",
	})

	assert.NoError(t, err)
}

func TestSendNotification_VideoCompleted_NoOptionalFields(t *testing.T) {
	db := new(MockDatabase)
	smtp := new(MockSMTPClient)
	auth := new(MockAuthClient)
	video := new(MockVideoClient)

	user := &domain.User{ID: "u1", Email: "u@e.com", Name: "User"}
	vid := &domain.Video{ID: "v1", OriginalName: "v.mp4"} // nil FrameCount, ZipSizeBytes, etc.

	os.MkdirAll("templates", 0755)
	os.WriteFile("templates/video_completed.html", []byte("Done: {{.VideoName}}"), 0644)
	defer os.RemoveAll("templates")

	auth.On("GetUserByID", "u1").Return(user, nil)
	video.On("GetVideoByID", "v1").Return(vid, nil)
	db.On("CreateNotification", mock.Anything).Return(nil)
	db.On("UpdateNotification", mock.Anything).Return(nil)
	smtp.On("SendEmail", "u@e.com", "Done", "Done: v.mp4").Return(nil)

	w := newTestWorker(1, db, nil, smtp, auth, video)
	err := w.sendNotification(context.Background(), &rabbitmq.NotificationMessage{
		UserID: "u1", VideoID: "v1", Type: "video_completed", Subject: "Done",
	})

	assert.NoError(t, err)
}

func TestSendNotification_VideoCompleted_TemplateNotFound(t *testing.T) {
	db := new(MockDatabase)
	auth := new(MockAuthClient)
	video := new(MockVideoClient)

	auth.On("GetUserByID", "u1").Return(&domain.User{ID: "u1", Email: "u@e.com", Name: "U"}, nil)
	video.On("GetVideoByID", "v1").Return(&domain.Video{ID: "v1", OriginalName: "v.mp4"}, nil)
	db.On("CreateNotification", mock.Anything).Return(nil)
	db.On("UpdateNotification", mock.Anything).Return(nil)

	os.RemoveAll("templates")

	w := newTestWorker(1, db, nil, nil, auth, video)
	err := w.sendNotification(context.Background(), &rabbitmq.NotificationMessage{
		UserID: "u1", VideoID: "v1", Type: "video_completed", Subject: "Done",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to render template")
}

func TestSendNotification_VideoFailed_Success(t *testing.T) {
	db := new(MockDatabase)
	smtp := new(MockSMTPClient)
	auth := new(MockAuthClient)
	video := new(MockVideoClient)

	errMsg := "FFmpeg error"
	user := &domain.User{ID: "u1", Email: "test@example.com", Name: "User"}
	vid := &domain.Video{ID: "v1", OriginalName: "video.mp4", ErrorMessage: &errMsg}

	os.MkdirAll("templates", 0755)
	os.WriteFile("templates/video_failed.html", []byte("Failed: {{.ErrorMessage}}"), 0644)
	defer os.RemoveAll("templates")

	auth.On("GetUserByID", "u1").Return(user, nil)
	video.On("GetVideoByID", "v1").Return(vid, nil)
	db.On("CreateNotification", mock.Anything).Return(nil)
	db.On("UpdateNotification", mock.Anything).Return(nil)
	smtp.On("SendEmail", "test@example.com", "Failed", "Failed: FFmpeg error").Return(nil)

	w := newTestWorker(1, db, nil, smtp, auth, video)
	err := w.sendNotification(context.Background(), &rabbitmq.NotificationMessage{
		UserID: "u1", VideoID: "v1", Type: "video_failed", Subject: "Failed",
	})

	assert.NoError(t, err)
}

func TestSendNotification_VideoFailed_NoErrorMessage(t *testing.T) {
	db := new(MockDatabase)
	smtp := new(MockSMTPClient)
	auth := new(MockAuthClient)
	video := new(MockVideoClient)

	user := &domain.User{ID: "u1", Email: "u@e.com", Name: "User"}
	vid := &domain.Video{ID: "v1", OriginalName: "v.mp4", ErrorMessage: nil}

	os.MkdirAll("templates", 0755)
	os.WriteFile("templates/video_failed.html", []byte("Error: {{.ErrorMessage}}"), 0644)
	defer os.RemoveAll("templates")

	auth.On("GetUserByID", "u1").Return(user, nil)
	video.On("GetVideoByID", "v1").Return(vid, nil)
	db.On("CreateNotification", mock.Anything).Return(nil)
	db.On("UpdateNotification", mock.Anything).Return(nil)
	smtp.On("SendEmail", "u@e.com", "F", "Error: Unknown error occurred").Return(nil)

	w := newTestWorker(1, db, nil, smtp, auth, video)
	err := w.sendNotification(context.Background(), &rabbitmq.NotificationMessage{
		UserID: "u1", VideoID: "v1", Type: "video_failed", Subject: "F",
	})

	assert.NoError(t, err)
}

func TestSendNotification_VideoFailed_TemplateNotFound(t *testing.T) {
	db := new(MockDatabase)
	auth := new(MockAuthClient)
	video := new(MockVideoClient)

	auth.On("GetUserByID", "u1").Return(&domain.User{ID: "u1", Email: "u@e.com", Name: "U"}, nil)
	video.On("GetVideoByID", "v1").Return(&domain.Video{ID: "v1", OriginalName: "v.mp4"}, nil)
	db.On("CreateNotification", mock.Anything).Return(nil)
	db.On("UpdateNotification", mock.Anything).Return(nil)

	os.RemoveAll("templates")

	w := newTestWorker(1, db, nil, nil, auth, video)
	err := w.sendNotification(context.Background(), &rabbitmq.NotificationMessage{
		UserID: "u1", VideoID: "v1", Type: "video_failed", Subject: "Failed",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to render template")
}

// ─── Start ────────────────────────────────────────────────────────────────────

func TestStart_SubscribeError(t *testing.T) {
	mq := new(MockRabbitMQ)
	mq.On("SubscribeNotification").Return(nil, errors.New("subscribe failed"))

	w := newTestWorker(1, nil, mq, nil, nil, nil)
	w.Start(context.Background()) // should return immediately

	mq.AssertExpectations(t)
}

func TestStart_Success(t *testing.T) {
	db := new(MockDatabase)
	mq := new(MockRabbitMQ)
	smtp := new(MockSMTPClient)
	auth := new(MockAuthClient)
	video := new(MockVideoClient)
	ack := new(MockAcknowledger)

	msgs := make(chan amqp.Delivery, 1)
	mq.On("SubscribeNotification").Return((<-chan amqp.Delivery)(msgs), nil)

	user := &domain.User{ID: "u1", Email: "test@example.com", Name: "User"}
	vid := &domain.Video{ID: "v1", OriginalName: "video.mp4"}
	msg := rabbitmq.NotificationMessage{UserID: "u1", VideoID: "v1", Type: "custom", Subject: "Sub", Message: "Msg"}
	body, _ := json.Marshal(msg)

	auth.On("GetUserByID", "u1").Return(user, nil)
	video.On("GetVideoByID", "v1").Return(vid, nil)
	db.On("CreateNotification", mock.Anything).Return(nil)
	db.On("UpdateNotification", mock.Anything).Return(nil)
	smtp.On("SendEmail", "test@example.com", "Sub", "Msg").Return(nil)
	ack.On("Ack", uint64(1), false).Return(nil)

	ctx, cancel := context.WithCancel(context.Background())
	w := newTestWorker(1, db, mq, smtp, auth, video)

	go func() {
		time.Sleep(50 * time.Millisecond)
		msgs <- amqp.Delivery{Body: body, Acknowledger: ack, DeliveryTag: 1}
		time.Sleep(150 * time.Millisecond)
		cancel()
		close(msgs)
	}()

	w.Start(ctx)

	mq.AssertExpectations(t)
	db.AssertExpectations(t)
	smtp.AssertExpectations(t)
	ack.AssertExpectations(t)
}

func TestStart_UnmarshalError(t *testing.T) {
	mq := new(MockRabbitMQ)
	ack := new(MockAcknowledger)

	msgs := make(chan amqp.Delivery, 1)
	mq.On("SubscribeNotification").Return((<-chan amqp.Delivery)(msgs), nil)
	ack.On("Nack", uint64(1), false, false).Return(nil)

	ctx, cancel := context.WithCancel(context.Background())
	w := newTestWorker(1, nil, mq, nil, nil, nil)

	go func() {
		time.Sleep(50 * time.Millisecond)
		msgs <- amqp.Delivery{Body: []byte("invalid json"), Acknowledger: ack, DeliveryTag: 1}
		time.Sleep(50 * time.Millisecond)
		cancel()
		close(msgs)
	}()

	w.Start(ctx)

	ack.AssertExpectations(t)
}

func TestStart_SendError_NotFound_Acked(t *testing.T) {
	// 404-type error → Ack (drop the message, don't retry)
	mq := new(MockRabbitMQ)
	auth := new(MockAuthClient)
	ack := new(MockAcknowledger)

	auth.On("GetUserByID", "u1").Return(nil, errors.New("unexpected status code 404: not found"))

	msg := rabbitmq.NotificationMessage{UserID: "u1", VideoID: "v1", Type: "custom"}
	body, _ := json.Marshal(msg)

	msgs := make(chan amqp.Delivery, 1)
	mq.On("SubscribeNotification").Return((<-chan amqp.Delivery)(msgs), nil)
	ack.On("Ack", uint64(1), false).Return(nil)

	ctx, cancel := context.WithCancel(context.Background())
	w := newTestWorker(1, nil, mq, nil, auth, nil)

	go func() {
		time.Sleep(50 * time.Millisecond)
		msgs <- amqp.Delivery{Body: body, Acknowledger: ack, DeliveryTag: 1}
		time.Sleep(150 * time.Millisecond)
		cancel()
		close(msgs)
	}()

	w.Start(ctx)

	ack.AssertExpectations(t)
}

func TestStart_SendError_Transient_Nacked(t *testing.T) {
	// Non-404 transient error → Nack with requeue=true
	mq := new(MockRabbitMQ)
	auth := new(MockAuthClient)
	ack := new(MockAcknowledger)

	auth.On("GetUserByID", "u1").Return(nil, errors.New("connection timeout"))

	msg := rabbitmq.NotificationMessage{UserID: "u1", VideoID: "v1", Type: "custom"}
	body, _ := json.Marshal(msg)

	msgs := make(chan amqp.Delivery, 1)
	mq.On("SubscribeNotification").Return((<-chan amqp.Delivery)(msgs), nil)
	ack.On("Nack", uint64(1), false, true).Return(nil)

	ctx, cancel := context.WithCancel(context.Background())
	w := newTestWorker(1, nil, mq, nil, auth, nil)

	go func() {
		time.Sleep(50 * time.Millisecond)
		msgs <- amqp.Delivery{Body: body, Acknowledger: ack, DeliveryTag: 1}
		time.Sleep(150 * time.Millisecond)
		cancel()
		close(msgs)
	}()

	w.Start(ctx)

	ack.AssertExpectations(t)
}

// ─── helper functions ─────────────────────────────────────────────────────────

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"Zero bytes", 0, "0 B"},
		{"Less than KB", 512, "512 B"},
		{"Exactly 1 KB", 1024, "1.0 KB"},
		{"1.5 MB", 1024 * 1024 * 3 / 2, "1.5 MB"},
		{"1 GB", 1024 * 1024 * 1024, "1.0 GB"},
		{"10 GB", 1024 * 1024 * 1024 * 10, "10.0 GB"},
		{"Large file", 1024 * 1024 * 1024 * 1024, "1.0 TB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatBytes(tt.bytes))
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"0 seconds", 0, "0 seconds"},
		{"30 seconds", 30 * time.Second, "30 seconds"},
		{"59 seconds", 59 * time.Second, "59 seconds"},
		{"1 minute", 60 * time.Second, "1 minutes 0 seconds"},
		{"2 minutes 30 seconds", 150 * time.Second, "2 minutes 30 seconds"},
		{"59 minutes", 59 * time.Minute, "59 minutes 0 seconds"},
		{"1 hour", 60 * time.Minute, "1 hours 0 minutes"},
		{"1 hour 30 minutes", 90 * time.Minute, "1 hours 30 minutes"},
		{"10 hours", 10 * time.Hour, "10 hours 0 minutes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatDuration(tt.duration))
		})
	}
}

func TestIsNotFoundErr(t *testing.T) {
	assert.False(t, isNotFoundErr(nil))
	assert.True(t, isNotFoundErr(errors.New("unexpected status code 404: not found")))
	assert.True(t, isNotFoundErr(errors.New("404")))
	assert.False(t, isNotFoundErr(errors.New("500 internal server error")))
	assert.False(t, isNotFoundErr(errors.New("connection refused")))
}

func TestStringPtr(t *testing.T) {
	p := stringPtr("hello")
	assert.NotNil(t, p)
	assert.Equal(t, "hello", *p)
}

func TestTimePtr(t *testing.T) {
	now := time.Now()
	p := timePtr(now)
	assert.NotNil(t, p)
	assert.Equal(t, now, *p)
}

func TestGenerateID_NotEmpty(t *testing.T) {
	id := generateID()
	assert.NotEmpty(t, id)
	assert.Len(t, id, 36) // UUID v4 format
}

func TestGenerateID_Unique(t *testing.T) {
	ids := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		ids[generateID()] = true
	}
	assert.Len(t, ids, 100)
}
