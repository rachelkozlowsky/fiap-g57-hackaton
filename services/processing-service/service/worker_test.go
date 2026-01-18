package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"processing-service/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	amqp "github.com/rabbitmq/amqp091-go"
)

type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) GetVideoByID(id string) (*domain.Video, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Video), args.Error(1)
}

func (m *MockDatabase) UpdateVideo(video *domain.Video) error {
	args := m.Called(video)
	return args.Error(0)
}

func (m *MockDatabase) CreateProcessingJob(job *domain.ProcessingJob) error {
	args := m.Called(job)
	return args.Error(0)
}

func (m *MockDatabase) UpdateProcessingJob(job *domain.ProcessingJob) error {
	args := m.Called(job)
	return args.Error(0)
}

type MockMinIO struct {
	mock.Mock
}

func (m *MockMinIO) DownloadFile(objectName, destPath string) error {
	args := m.Called(objectName, destPath)
	return args.Error(0)
}

func (m *MockMinIO) UploadProcessedFile(reader io.Reader, filename string, size int64) (string, error) {
	args := m.Called(reader, filename, size)
	return args.String(0), args.Error(1)
}

type MockRabbitMQ struct {
	mock.Mock
}

func (m *MockRabbitMQ) PublishNotification(message domain.NotificationMessage) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockRabbitMQ) SubscribeVideoUpload() (<-chan amqp.Delivery, error) {
	args := m.Called()
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

func MockExecCommand(ctx context.Context, command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	
	for _, arg := range args {
		if strings.Contains(arg, "FAIL_FFMPEG") {
			cmd.Env = append(cmd.Env, "FAIL_FFMPEG=1")
		}
		if strings.Contains(arg, "EMPTY_FRAMES") {
			cmd.Env = append(cmd.Env, "EMPTY_FRAMES=1")
		}
	}
	
	return cmd
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	
	if os.Getenv("FAIL_FFMPEG") == "1" {
		os.Stderr.WriteString("ffmpeg error simulation")
		os.Exit(1)
	}

	if os.Getenv("EMPTY_FRAMES") == "1" {
		os.Exit(0)
	}

	args := os.Args
	for i, arg := range args {
		if arg == "--" && i+1 < len(args) && args[i+1] == "ffmpeg" {
			for j := i + 2; j < len(args); j++ {
				if filepath.Ext(args[j]) == ".png" || strings.Contains(args[j], "frames") {
					framesDir := args[j]
					if filepath.Ext(args[j]) == ".png" {
						framesDir = filepath.Dir(args[j])
					}
					os.MkdirAll(framesDir, 0755)
					os.WriteFile(filepath.Join(framesDir, "frame_0001.png"), []byte("dummy"), 0644)
					break
				}
			}
		}
	}
	
	os.Exit(0)
}

func TestProcessVideo_Success(t *testing.T) {
	origExec := execCommand
	execCommand = MockExecCommand
	defer func() { execCommand = origExec }()

	mockDB := new(MockDatabase)
	mockMinio := new(MockMinIO)
	mockRabbit := new(MockMockRabbitMQ)
	_ = mockRabbit
	mockRabbitReal := new(MockRabbitMQ)
	worker := NewWorker(1, mockDB, mockMinio, mockRabbitReal)

	videoID := "test-video-123"
	userID := "user-123"
	filename := "test.mp4"
	storagePath := "raw/test.mp4"

	video := &domain.Video{
		ID:     videoID,
		UserID: userID,
		Status: "queued",
	}

	msg := &domain.VideoProcessingMessage{
		VideoID:     videoID,
		UserID:      userID,
		Filename:    filename,
		StoragePath: storagePath,
	}

	mockDB.On("GetVideoByID", videoID).Return(video, nil)
	mockDB.On("UpdateVideo", mock.MatchedBy(func(v *domain.Video) bool { return v.Status == "processing" })).Return(nil).Once()
	mockDB.On("CreateProcessingJob", mock.Anything).Return(nil)
	mockMinio.On("DownloadFile", storagePath, mock.Anything).Return(nil)
	
	mockMinio.On("UploadProcessedFile", mock.Anything, mock.Anything, mock.Anything).Return("processed/test.zip", nil)
	mockRabbitReal.On("PublishNotification", mock.Anything).Return(nil)
	
	mockDB.On("UpdateVideo", mock.MatchedBy(func(v *domain.Video) bool { return v.Status == "completed" })).Return(nil).Once()
	mockDB.On("UpdateProcessingJob", mock.MatchedBy(func(j *domain.ProcessingJob) bool { return j.Status == "completed" })).Return(nil)

	err := worker.processVideo(context.Background(), msg)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

type MockMockRabbitMQ = MockRabbitMQ 

func TestProcessVideo_FFmpegError(t *testing.T) {
	origExec := execCommand
	execCommand = MockExecCommand
	defer func() { execCommand = origExec }()

	mockDB := new(MockDatabase)
	mockMinio := new(MockMinIO)
	mockRabbit := new(MockRabbitMQ)
	worker := NewWorker(1, mockDB, mockMinio, mockRabbit)

	video := &domain.Video{ID: "v1", Status: "queued"}
	msg := &domain.VideoProcessingMessage{VideoID: "v1", Filename: "FAIL_FFMPEG", StoragePath: "s"}

	mockDB.On("GetVideoByID", "v1").Return(video, nil)
	mockDB.On("UpdateVideo", mock.Anything).Return(nil)
	mockDB.On("CreateProcessingJob", mock.Anything).Return(nil)
	mockMinio.On("DownloadFile", "s", mock.Anything).Return(nil)
	
	mockRabbit.On("PublishNotification", mock.Anything).Return(nil)
	mockDB.On("UpdateProcessingJob", mock.Anything).Return(nil)
	mockDB.On("UpdateVideo", mock.MatchedBy(func(v *domain.Video) bool { return v.Status == "failed" })).Return(nil)

	err := worker.processVideo(context.Background(), msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ffmpeg failed")
	mockDB.AssertExpectations(t)
}

func TestProcessVideo_NoFramesError(t *testing.T) {
	origExec := execCommand
	execCommand = MockExecCommand
	defer func() { execCommand = origExec }()

	mockDB := new(MockDatabase)
	mockMinio := new(MockMinIO)
	mockRabbit := new(MockRabbitMQ)
	worker := NewWorker(1, mockDB, mockMinio, mockRabbit)

	video := &domain.Video{ID: "v1", Status: "queued"}
	msg := &domain.VideoProcessingMessage{VideoID: "v1", Filename: "EMPTY_FRAMES", StoragePath: "s"}

	mockDB.On("GetVideoByID", "v1").Return(video, nil)
	mockDB.On("UpdateVideo", mock.Anything).Return(nil)
	mockDB.On("CreateProcessingJob", mock.Anything).Return(nil)
	mockMinio.On("DownloadFile", "s", mock.Anything).Return(nil)
	
	mockRabbit.On("PublishNotification", mock.Anything).Return(nil)
	mockDB.On("UpdateProcessingJob", mock.Anything).Return(nil)
	mockDB.On("UpdateVideo", mock.MatchedBy(func(v *domain.Video) bool { return v.Status == "failed" })).Return(nil)

	err := worker.processVideo(context.Background(), msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no frames extracted")
	mockDB.AssertExpectations(t)
}

func TestStart_Success(t *testing.T) {
	origExec := execCommand
	execCommand = MockExecCommand
	defer func() { execCommand = origExec }()

	mockDB := new(MockDatabase)
	mockMinio := new(MockMinIO)
	mockRabbit := new(MockRabbitMQ)
	mockAck := new(MockAcknowledger)
	worker := NewWorker(1, mockDB, mockMinio, mockRabbit)

	msgs := make(chan amqp.Delivery, 1)
	
	data, _ := json.Marshal(domain.VideoProcessingMessage{
		VideoID: "v1", Filename: "test.mp4", StoragePath: "s",
	})

	delivery := amqp.Delivery{
		Body: data,
		Acknowledger: mockAck,
		DeliveryTag: 1,
	}
	msgs <- delivery
	close(msgs)

	mockRabbit.On("SubscribeVideoUpload").Return((<-chan amqp.Delivery)(msgs), nil)
	
	video := &domain.Video{ID: "v1", Status: "queued"}
	mockDB.On("GetVideoByID", "v1").Return(video, nil)
	mockDB.On("UpdateVideo", mock.Anything).Return(nil)
	mockDB.On("CreateProcessingJob", mock.Anything).Return(nil)
	mockMinio.On("DownloadFile", "s", mock.Anything).Return(nil)
	mockMinio.On("UploadProcessedFile", mock.Anything, mock.Anything, mock.Anything).Return("zip", nil)
	mockRabbit.On("PublishNotification", mock.Anything).Return(nil)
	mockDB.On("UpdateProcessingJob", mock.Anything).Return(nil)
	
	mockAck.On("Ack", uint64(1), false).Return(nil)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	worker.Start(ctx)

	mockRabbit.AssertExpectations(t)
	mockAck.AssertExpectations(t)
}

func TestStart_UnmarshalError(t *testing.T) {
	mockRabbit := new(MockRabbitMQ)
	mockAck := new(MockAcknowledger)
	worker := NewWorker(1, nil, nil, mockRabbit)

	msgs := make(chan amqp.Delivery, 1)
	delivery := amqp.Delivery{
		Body: []byte("invalid json"),
		Acknowledger: mockAck,
		DeliveryTag: 1,
	}
	msgs <- delivery
	close(msgs)

	mockRabbit.On("SubscribeVideoUpload").Return((<-chan amqp.Delivery)(msgs), nil)
	mockAck.On("Nack", uint64(1), false, false).Return(nil)

	worker.Start(context.Background())

	mockRabbit.AssertExpectations(t)
	mockAck.AssertExpectations(t)
}

func TestStart_ProcessError(t *testing.T) {
	mockDB := new(MockDatabase)
	mockRabbit := new(MockRabbitMQ)
	mockAck := new(MockAcknowledger)
	worker := NewWorker(1, mockDB, nil, mockRabbit)

	msgs := make(chan amqp.Delivery, 1)
	data, _ := json.Marshal(domain.VideoProcessingMessage{VideoID: "v1"})
	delivery := amqp.Delivery{
		Body: data,
		Acknowledger: mockAck,
		DeliveryTag: 1,
	}
	msgs <- delivery
	close(msgs)

	mockRabbit.On("SubscribeVideoUpload").Return((<-chan amqp.Delivery)(msgs), nil)
	mockDB.On("GetVideoByID", "v1").Return(nil, errors.New("db error"))
	mockAck.On("Nack", uint64(1), false, true).Return(nil)

	worker.Start(context.Background())

	mockRabbit.AssertExpectations(t)
	mockAck.AssertExpectations(t)
}

func TestUpdateJobFailed(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := NewWorker(1, mockDB, nil, nil)
	
	job := &domain.ProcessingJob{ID: "j1", Status: "running"}
	testErr := errors.New("test error")
	
	mockDB.On("UpdateProcessingJob", mock.MatchedBy(func(j *domain.ProcessingJob) bool {
		return j.Status == "failed" && *j.ErrorMessage == "test error"
	})).Return(nil)
	
	worker.updateJobFailed(job, testErr)
	mockDB.AssertExpectations(t)
}

func TestUpdateVideoFailed(t *testing.T) {
	mockDB := new(MockDatabase)
	mockRabbit := new(MockRabbitMQ)
	worker := NewWorker(1, mockDB, nil, mockRabbit)
	
	video := &domain.Video{ID: "v1", UserID: "u1", Status: "processing"}
	testErr := errors.New("test error")
	
	mockDB.On("UpdateVideo", mock.MatchedBy(func(v *domain.Video) bool {
		return v.Status == "failed" && *v.ErrorMessage == "test error"
	})).Return(nil)
	
	mockRabbit.On("PublishNotification", mock.Anything).Return(nil)
	
	worker.updateVideoFailed(video, testErr)
	mockDB.AssertExpectations(t)
}

func TestGenerateID(t *testing.T) {
	id := generateID()
	assert.Len(t, id, 36)
}

func TestTimePtr(t *testing.T) {
	now := time.Now()
	ptr := timePtr(now)
	assert.Equal(t, now, *ptr)
}

func TestStringPtr(t *testing.T) {
	s := "test"
	ptr := stringPtr(s)
	assert.Equal(t, s, *ptr)
}

func TestNewWorker(t *testing.T) {
	w := NewWorker(1, nil, nil, nil)
	assert.Equal(t, 1, w.ID)
}

func TestCreateZipFile_Success(t *testing.T) {
	worker := NewWorker(1, nil, nil, nil)
	tempDir, _ := os.MkdirTemp("", "zip-test")
	defer os.RemoveAll(tempDir)
	
	f1 := filepath.Join(tempDir, "1.txt")
	os.WriteFile(f1, []byte("test"), 0644)
	
	zipPath := filepath.Join(tempDir, "test.zip")
	err := worker.createZipFile([]string{f1}, zipPath)
	
	assert.NoError(t, err)
	_, err = os.Stat(zipPath)
	assert.NoError(t, err)
}

func TestCreateZipFile_Error(t *testing.T) {
	worker := NewWorker(1, nil, nil, nil)
	err := worker.createZipFile([]string{"nonexistent"}, "/invalid/path")
	assert.Error(t, err)
}

func TestAddFileToZip_Error(t *testing.T) {
	worker := NewWorker(1, nil, nil, nil)
	err := worker.addFileToZip(nil, "nonexistent")
	assert.Error(t, err)
}
