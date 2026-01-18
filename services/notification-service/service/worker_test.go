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

type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) CreateNotification(notification *domain.Notification) error {
	args := m.Called(notification)
	return args.Error(0)
}

func (m *MockDatabase) UpdateNotification(notification *domain.Notification) error {
	args := m.Called(notification)
	return args.Error(0)
}

func (m *MockDatabase) GetVideoByID(id string) (*domain.Video, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Video), args.Error(1)
}

func (m *MockDatabase) GetUserByID(id string) (*domain.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockDatabase) Ping() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDatabase) Close() error {
	args := m.Called()
	return args.Error(0)
}

type MockSMTPClient struct {
	mock.Mock
}

func (m *MockSMTPClient) SendEmail(to, subject, body string) error {
	args := m.Called(to, subject, body)
	return args.Error(0)
}

type MockRabbitMQ struct {
	mock.Mock
}

func (m *MockRabbitMQ) SubscribeNotification() (<-chan amqp.Delivery, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(<-chan amqp.Delivery), args.Error(1)
}

type MockAcknowledger struct {
	mock.Mock
}

func (m *MockAcknowledger) Ack(tag uint64, multiple bool) error {
	args := m.Called(tag, multiple)
	return args.Error(0)
}

func (m *MockAcknowledger) Nack(tag uint64, multiple bool, requeue bool) error {
	args := m.Called(tag, multiple, requeue)
	return args.Error(0)
}

func (m *MockAcknowledger) Reject(tag uint64, requeue bool) error {
	args := m.Called(tag, requeue)
	return args.Error(0)
}

func TestNewNotificationWorker(t *testing.T) {
	mockDB := new(MockDatabase)
	mockSMTP := new(MockSMTPClient)
	
	worker := NewNotificationWorker(1, mockDB, nil, mockSMTP)
	
	assert.NotNil(t, worker)
	assert.Equal(t, 1, worker.ID)
	assert.Equal(t, mockDB, worker.db)
	assert.Equal(t, mockSMTP, worker.smtp)
}

func TestNewNotificationWorker_MultipleWorkers(t *testing.T) {
	mockDB := new(MockDatabase)
	mockSMTP := new(MockSMTPClient)
	
	worker1 := NewNotificationWorker(1, mockDB, nil, mockSMTP)
	worker2 := NewNotificationWorker(2, mockDB, nil, mockSMTP)
	worker3 := NewNotificationWorker(3, mockDB, nil, mockSMTP)
	
	assert.Equal(t, 1, worker1.ID)
	assert.Equal(t, 2, worker2.ID)
	assert.Equal(t, 3, worker3.ID)
}

func TestSendNotification_CustomType(t *testing.T) {
	mockDB := new(MockDatabase)
	mockSMTP := new(MockSMTPClient)
	
	user := &domain.User{
		ID:    "user1",
		Email: "test@example.com",
		Name:  "Test User",
	}
	
	video := &domain.Video{
		ID:           "video1",
		UserID:       "user1",
		OriginalName: "test.mp4",
		Status:       "completed",
	}
	
	message := &rabbitmq.NotificationMessage{
		VideoID: "video1",
		UserID:  "user1",
		Type:    "custom_notification",
		Subject: "Custom Notification",
		Message: "This is a plain text message",
	}
	
	mockDB.On("GetUserByID", "user1").Return(user, nil)
	mockDB.On("GetVideoByID", "video1").Return(video, nil)
	mockDB.On("CreateNotification", mock.AnythingOfType("*domain.Notification")).Return(nil)
	mockDB.On("UpdateNotification", mock.AnythingOfType("*domain.Notification")).Return(nil)
	mockSMTP.On("SendEmail", "test@example.com", "Custom Notification", "This is a plain text message").Return(nil)
	
	worker := NewNotificationWorker(1, mockDB, nil, mockSMTP)
	err := worker.sendNotification(context.Background(), message)
	
	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
	mockSMTP.AssertExpectations(t)
}

func TestSendNotification_UserNotFound(t *testing.T) {
	mockDB := new(MockDatabase)
	
	message := &rabbitmq.NotificationMessage{
		VideoID: "video1",
		UserID:  "user999",
		Type:    "custom",
		Subject: "Test",
	}
	
	mockDB.On("GetUserByID", "user999").Return(nil, errors.New("user not found"))
	
	worker := NewNotificationWorker(1, mockDB, nil, nil)
	err := worker.sendNotification(context.Background(), message)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get user")
	mockDB.AssertExpectations(t)
}

func TestSendNotification_VideoNotFound(t *testing.T) {
	mockDB := new(MockDatabase)
	
	user := &domain.User{
		ID:    "user1",
		Email: "test@example.com",
		Name:  "Test User",
	}
	
	message := &rabbitmq.NotificationMessage{
		VideoID: "video999",
		UserID:  "user1",
		Type:    "custom",
		Subject: "Test",
	}
	
	mockDB.On("GetUserByID", "user1").Return(user, nil)
	mockDB.On("GetVideoByID", "video999").Return(nil, errors.New("video not found"))
	
	worker := NewNotificationWorker(1, mockDB, nil, nil)
	err := worker.sendNotification(context.Background(), message)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get video")
	mockDB.AssertExpectations(t)
}

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
			result := formatBytes(tt.bytes)
			assert.Equal(t, tt.expected, result)
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
		{"2 hours 15 minutes", 135 * time.Minute, "2 hours 15 minutes"},
		{"10 hours", 10 * time.Hour, "10 hours 0 minutes"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStart_Success(t *testing.T) {
	mockDB := new(MockDatabase)
	mockRabbit := new(MockRabbitMQ)
	mockSMTP := new(MockSMTPClient)
	mockAck := new(MockAcknowledger)

	worker := NewNotificationWorker(1, mockDB, mockRabbit, mockSMTP)
	ctx, cancel := context.WithCancel(context.Background())

	msgs := make(chan amqp.Delivery, 1)
	mockRabbit.On("SubscribeNotification").Return((<-chan amqp.Delivery)(msgs), nil)

	user := &domain.User{ID: "u1", Email: "test@example.com", Name: "User"}
	video := &domain.Video{ID: "v1", OriginalName: "video.mp4"}
	msg := rabbitmq.NotificationMessage{UserID: "u1", VideoID: "v1", Type: "custom", Subject: "Sub", Message: "Msg"}
	body, _ := json.Marshal(msg)

	delivery := amqp.Delivery{
		Body:         body,
		Acknowledger: mockAck,
		DeliveryTag:  1,
	}

	mockDB.On("GetUserByID", "u1").Return(user, nil)
	mockDB.On("GetVideoByID", "v1").Return(video, nil)
	mockDB.On("CreateNotification", mock.Anything).Return(nil)
	mockDB.On("UpdateNotification", mock.Anything).Return(nil)
	mockSMTP.On("SendEmail", "test@example.com", "Sub", "Msg").Return(nil)
	mockAck.On("Ack", uint64(1), false).Return(nil)

	go func() {
		time.Sleep(100 * time.Millisecond)
		msgs <- delivery
		time.Sleep(100 * time.Millisecond)
		cancel()
		close(msgs)
	}()

	worker.Start(ctx)

	mockRabbit.AssertExpectations(t)
	mockDB.AssertExpectations(t)
	mockSMTP.AssertExpectations(t)
	mockAck.AssertExpectations(t)
}

func TestStart_UnmarshalError(t *testing.T) {
	mockRabbit := new(MockRabbitMQ)
	mockAck := new(MockAcknowledger)

	worker := NewNotificationWorker(1, nil, mockRabbit, nil)
	ctx, cancel := context.WithCancel(context.Background())

	msgs := make(chan amqp.Delivery, 1)
	mockRabbit.On("SubscribeNotification").Return((<-chan amqp.Delivery)(msgs), nil)

	delivery := amqp.Delivery{
		Body:         []byte("invalid json"),
		Acknowledger: mockAck,
		DeliveryTag:  1,
	}

	mockAck.On("Nack", uint64(1), false, false).Return(nil)

	go func() {
		time.Sleep(50 * time.Millisecond)
		msgs <- delivery
		time.Sleep(50 * time.Millisecond)
		cancel()
		close(msgs)
	}()

	worker.Start(ctx)

	mockAck.AssertExpectations(t)
}

func TestSendNotification_VideoCompleted_Success(t *testing.T) {
	mockDB := new(MockDatabase)
	mockSMTP := new(MockSMTPClient)
	
	user := &domain.User{ID: "u1", Email: "test@example.com", Name: "User"}
	frameCount := 100
	zipSize := int64(1024 * 1024)
	video := &domain.Video{
		ID: "v1", OriginalName: "video.mp4", 
		FrameCount: &frameCount, ZipSizeBytes: &zipSize,
		ProcessingStartedAt: timePtr(time.Now().Add(-1 * time.Minute)),
		ProcessingCompletedAt: timePtr(time.Now()),
	}
	
	message := &rabbitmq.NotificationMessage{
		UserID: "u1", VideoID: "v1", Type: "video_completed", Subject: "Done",
	}

	os.MkdirAll("templates", 0755)
	os.WriteFile("templates/video_completed.html", []byte("Completed: {{.VideoName}}"), 0644)
	defer os.RemoveAll("templates")

	mockDB.On("GetUserByID", "u1").Return(user, nil)
	mockDB.On("GetVideoByID", "v1").Return(video, nil)
	mockDB.On("CreateNotification", mock.Anything).Return(nil)
	mockDB.On("UpdateNotification", mock.Anything).Return(nil)
	mockSMTP.On("SendEmail", "test@example.com", "Done", "Completed: video.mp4").Return(nil)

	worker := NewNotificationWorker(1, mockDB, nil, mockSMTP)
	err := worker.sendNotification(context.Background(), message)
	
	assert.NoError(t, err)
}

func TestSendNotification_VideoFailed_Success(t *testing.T) {
	mockDB := new(MockDatabase)
	mockSMTP := new(MockSMTPClient)
	
	user := &domain.User{ID: "u1", Email: "test@example.com", Name: "User"}
	video := &domain.Video{
		ID: "v1", OriginalName: "video.mp4", 
		ErrorMessage: stringPtr("FFmpeg error"),
	}
	
	message := &rabbitmq.NotificationMessage{
		UserID: "u1", VideoID: "v1", Type: "video_failed", Subject: "Failed",
	}

	os.MkdirAll("templates", 0755)
	os.WriteFile("templates/video_failed.html", []byte("Failed: {{.ErrorMessage}}"), 0644)
	defer os.RemoveAll("templates")

	mockDB.On("GetUserByID", "u1").Return(user, nil)
	mockDB.On("GetVideoByID", "v1").Return(video, nil)
	mockDB.On("CreateNotification", mock.Anything).Return(nil)
	mockDB.On("UpdateNotification", mock.Anything).Return(nil)
	mockSMTP.On("SendEmail", "test@example.com", "Failed", "Failed: FFmpeg error").Return(nil)

	worker := NewNotificationWorker(1, mockDB, nil, mockSMTP)
	err := worker.sendNotification(context.Background(), message)
	
	assert.NoError(t, err)
}
