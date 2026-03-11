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

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockDatabase struct{ mock.Mock }

func (m *MockDatabase) CreateProcessingJob(job *domain.ProcessingJob) error {
	return m.Called(job).Error(0)
}
func (m *MockDatabase) UpdateProcessingJob(job *domain.ProcessingJob) error {
	return m.Called(job).Error(0)
}

type MockVideoClient struct{ mock.Mock }

func (m *MockVideoClient) GetVideoByID(videoID string) (*domain.Video, error) {
	args := m.Called(videoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Video), args.Error(1)
}
func (m *MockVideoClient) UpdateVideoStatus(videoID, status, errorMessage string) error {
	return m.Called(videoID, status, errorMessage).Error(0)
}
func (m *MockVideoClient) CompleteVideo(videoID, zipPath string, zipSize int64, frameCount int) error {
	return m.Called(videoID, zipPath, zipSize, frameCount).Error(0)
}
func (m *MockVideoClient) FailVideo(videoID, errorMessage string) error {
	return m.Called(videoID, errorMessage).Error(0)
}

type MockMinIO struct{ mock.Mock }

func (m *MockMinIO) DownloadFile(objectName, destPath string) error {
	return m.Called(objectName, destPath).Error(0)
}
func (m *MockMinIO) UploadProcessedFile(reader io.Reader, filename string, size int64) (string, error) {
	args := m.Called(reader, filename, size)
	return args.String(0), args.Error(1)
}

type MockRabbitMQ struct{ mock.Mock }

func (m *MockRabbitMQ) PublishNotification(message domain.NotificationMessage) error {
	return m.Called(message).Error(0)
}
func (m *MockRabbitMQ) SubscribeVideoUpload() (<-chan amqp.Delivery, error) {
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

// helper — injects nil for unused interfaces
func newTestWorker(id int, db *MockDatabase, minio *MockMinIO, mq *MockRabbitMQ, vc *MockVideoClient) *Worker {
	var dbI domain.DatabaseInterface
	var minioI domain.MinIOInterface
	var mqI domain.RabbitMQInterface
	var vcI domain.VideoServiceClient
	if db != nil {
		dbI = db
	}
	if minio != nil {
		minioI = minio
	}
	if mq != nil {
		mqI = mq
	}
	if vc != nil {
		vcI = vc
	}
	return NewWorker(id, dbI, minioI, mqI, vcI)
}

// ─── ffmpeg helper process ────────────────────────────────────────────────────

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

// ─── NewWorker ────────────────────────────────────────────────────────────────

func TestNewWorker(t *testing.T) {
	w := newTestWorker(1, nil, nil, nil, nil)
	assert.Equal(t, 1, w.ID)
}

func TestNewWorker_MultipleIDs(t *testing.T) {
	for id := 1; id <= 5; id++ {
		w := newTestWorker(id, nil, nil, nil, nil)
		assert.Equal(t, id, w.ID)
	}
}

// ─── processVideo ─────────────────────────────────────────────────────────────

func TestProcessVideo_GetVideoError(t *testing.T) {
	db := new(MockDatabase)
	vc := new(MockVideoClient)

	vc.On("GetVideoByID", "v1").Return(nil, errors.New("not found"))

	w := newTestWorker(1, db, nil, nil, vc)
	err := w.processVideo(context.Background(), &domain.VideoProcessingMessage{VideoID: "v1"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get video")
}

func TestProcessVideo_DownloadError(t *testing.T) {
	db := new(MockDatabase)
	minio := new(MockMinIO)
	mq := new(MockRabbitMQ)
	vc := new(MockVideoClient)

	video := &domain.Video{ID: "v1", UserID: "u1", Status: "queued"}
	vc.On("GetVideoByID", "v1").Return(video, nil)
	vc.On("UpdateVideoStatus", "v1", "processing", "").Return(nil)
	db.On("CreateProcessingJob", mock.Anything).Return(nil)
	minio.On("DownloadFile", "s3/path", mock.Anything).Return(errors.New("download failed"))
	db.On("UpdateProcessingJob", mock.Anything).Return(nil)
	vc.On("FailVideo", "v1", mock.Anything).Return(nil)
	mq.On("PublishNotification", mock.Anything).Return(nil)

	w := newTestWorker(1, db, minio, mq, vc)
	err := w.processVideo(context.Background(), &domain.VideoProcessingMessage{
		VideoID: "v1", UserID: "u1", Filename: "v.mp4", StoragePath: "s3/path",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to download video")
}

func TestProcessVideo_FFmpegError(t *testing.T) {
	origExec := execCommand
	execCommand = MockExecCommand
	defer func() { execCommand = origExec }()

	db := new(MockDatabase)
	minio := new(MockMinIO)
	mq := new(MockRabbitMQ)
	vc := new(MockVideoClient)

	video := &domain.Video{ID: "v1", UserID: "u1", Status: "queued"}
	vc.On("GetVideoByID", "v1").Return(video, nil)
	vc.On("UpdateVideoStatus", "v1", "processing", "").Return(nil)
	db.On("CreateProcessingJob", mock.Anything).Return(nil)
	minio.On("DownloadFile", "s", mock.Anything).Return(nil)
	db.On("UpdateProcessingJob", mock.Anything).Return(nil)
	vc.On("FailVideo", "v1", mock.Anything).Return(nil)
	mq.On("PublishNotification", mock.Anything).Return(nil)

	w := newTestWorker(1, db, minio, mq, vc)
	err := w.processVideo(context.Background(), &domain.VideoProcessingMessage{
		VideoID: "v1", UserID: "u1", Filename: "FAIL_FFMPEG", StoragePath: "s",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ffmpeg failed")
	db.AssertExpectations(t)
}

func TestProcessVideo_NoFramesError(t *testing.T) {
	origExec := execCommand
	execCommand = MockExecCommand
	defer func() { execCommand = origExec }()

	db := new(MockDatabase)
	minio := new(MockMinIO)
	mq := new(MockRabbitMQ)
	vc := new(MockVideoClient)

	video := &domain.Video{ID: "v1", UserID: "u1", Status: "queued"}
	vc.On("GetVideoByID", "v1").Return(video, nil)
	vc.On("UpdateVideoStatus", "v1", "processing", "").Return(nil)
	db.On("CreateProcessingJob", mock.Anything).Return(nil)
	minio.On("DownloadFile", "s", mock.Anything).Return(nil)
	db.On("UpdateProcessingJob", mock.Anything).Return(nil)
	vc.On("FailVideo", "v1", mock.Anything).Return(nil)
	mq.On("PublishNotification", mock.Anything).Return(nil)

	w := newTestWorker(1, db, minio, mq, vc)
	err := w.processVideo(context.Background(), &domain.VideoProcessingMessage{
		VideoID: "v1", UserID: "u1", Filename: "EMPTY_FRAMES", StoragePath: "s",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no frames extracted")
}

func TestProcessVideo_UploadError(t *testing.T) {
	origExec := execCommand
	execCommand = MockExecCommand
	defer func() { execCommand = origExec }()

	db := new(MockDatabase)
	minio := new(MockMinIO)
	mq := new(MockRabbitMQ)
	vc := new(MockVideoClient)

	video := &domain.Video{ID: "v1", UserID: "u1", Status: "queued"}
	vc.On("GetVideoByID", "v1").Return(video, nil)
	vc.On("UpdateVideoStatus", "v1", "processing", "").Return(nil)
	db.On("CreateProcessingJob", mock.Anything).Return(nil)
	minio.On("DownloadFile", "s", mock.Anything).Return(nil)
	minio.On("UploadProcessedFile", mock.Anything, mock.Anything, mock.Anything).Return("", errors.New("upload failed"))
	db.On("UpdateProcessingJob", mock.Anything).Return(nil)
	vc.On("FailVideo", "v1", mock.Anything).Return(nil)
	mq.On("PublishNotification", mock.Anything).Return(nil)

	w := newTestWorker(1, db, minio, mq, vc)
	err := w.processVideo(context.Background(), &domain.VideoProcessingMessage{
		VideoID: "v1", UserID: "u1", Filename: "test.mp4", StoragePath: "s",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to upload zip")
}

func TestProcessVideo_Success(t *testing.T) {
	origExec := execCommand
	execCommand = MockExecCommand
	defer func() { execCommand = origExec }()

	db := new(MockDatabase)
	minio := new(MockMinIO)
	mq := new(MockRabbitMQ)
	vc := new(MockVideoClient)

	video := &domain.Video{ID: "v1", UserID: "u1", Status: "queued"}
	vc.On("GetVideoByID", "v1").Return(video, nil)
	vc.On("UpdateVideoStatus", "v1", "processing", "").Return(nil)
	db.On("CreateProcessingJob", mock.Anything).Return(nil)
	minio.On("DownloadFile", "s3/path", mock.Anything).Return(nil)
	minio.On("UploadProcessedFile", mock.Anything, mock.Anything, mock.Anything).Return("processed/frames.zip", nil)
	vc.On("CompleteVideo", "v1", "processed/frames.zip", mock.AnythingOfType("int64"), mock.AnythingOfType("int")).Return(nil)
	db.On("UpdateProcessingJob", mock.Anything).Return(nil)
	mq.On("PublishNotification", mock.Anything).Return(nil)

	w := newTestWorker(1, db, minio, mq, vc)
	err := w.processVideo(context.Background(), &domain.VideoProcessingMessage{
		VideoID: "v1", UserID: "u1", Filename: "test.mp4", StoragePath: "s3/path",
	})

	assert.NoError(t, err)
	db.AssertExpectations(t)
	vc.AssertExpectations(t)
	mq.AssertExpectations(t)
}

// ─── Start ────────────────────────────────────────────────────────────────────

func TestStart_UnmarshalError(t *testing.T) {
	mq := new(MockRabbitMQ)
	ack := new(MockAcknowledger)

	msgs := make(chan amqp.Delivery, 1)
	msgs <- amqp.Delivery{Body: []byte("invalid json"), Acknowledger: ack, DeliveryTag: 1}
	close(msgs)

	mq.On("SubscribeVideoUpload").Return((<-chan amqp.Delivery)(msgs), nil)
	ack.On("Nack", uint64(1), false, false).Return(nil)

	newTestWorker(1, nil, nil, mq, nil).Start(context.Background())

	mq.AssertExpectations(t)
	ack.AssertExpectations(t)
}

func TestStart_ProcessError_Nacked(t *testing.T) {
	db := new(MockDatabase)
	mq := new(MockRabbitMQ)
	ack := new(MockAcknowledger)
	vc := new(MockVideoClient)

	vc.On("GetVideoByID", "v1").Return(nil, errors.New("not found"))

	data, _ := json.Marshal(domain.VideoProcessingMessage{VideoID: "v1"})
	msgs := make(chan amqp.Delivery, 1)
	msgs <- amqp.Delivery{Body: data, Acknowledger: ack, DeliveryTag: 1}
	close(msgs)

	mq.On("SubscribeVideoUpload").Return((<-chan amqp.Delivery)(msgs), nil)
	ack.On("Nack", uint64(1), false, true).Return(nil)

	newTestWorker(1, db, nil, mq, vc).Start(context.Background())

	mq.AssertExpectations(t)
	ack.AssertExpectations(t)
}

func TestStart_ContextCancelled(t *testing.T) {
	mq := new(MockRabbitMQ)

	msgs := make(chan amqp.Delivery) // never receives
	mq.On("SubscribeVideoUpload").Return((<-chan amqp.Delivery)(msgs), nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	newTestWorker(1, nil, nil, mq, nil).Start(ctx)

	mq.AssertExpectations(t)
}

func TestStart_Success(t *testing.T) {
	origExec := execCommand
	execCommand = MockExecCommand
	defer func() { execCommand = origExec }()

	db := new(MockDatabase)
	minio := new(MockMinIO)
	mq := new(MockRabbitMQ)
	ack := new(MockAcknowledger)
	vc := new(MockVideoClient)

	video := &domain.Video{ID: "v1", UserID: "u1", Status: "queued"}
	vc.On("GetVideoByID", "v1").Return(video, nil)
	vc.On("UpdateVideoStatus", "v1", "processing", "").Return(nil)
	db.On("CreateProcessingJob", mock.Anything).Return(nil)
	minio.On("DownloadFile", "s", mock.Anything).Return(nil)
	minio.On("UploadProcessedFile", mock.Anything, mock.Anything, mock.Anything).Return("zip/path", nil)
	vc.On("CompleteVideo", "v1", "zip/path", mock.AnythingOfType("int64"), mock.AnythingOfType("int")).Return(nil)
	db.On("UpdateProcessingJob", mock.Anything).Return(nil)
	mq.On("PublishNotification", mock.Anything).Return(nil)
	ack.On("Ack", uint64(1), false).Return(nil)

	data, _ := json.Marshal(domain.VideoProcessingMessage{
		VideoID: "v1", UserID: "u1", Filename: "test.mp4", StoragePath: "s",
	})
	msgs := make(chan amqp.Delivery, 1)
	msgs <- amqp.Delivery{Body: data, Acknowledger: ack, DeliveryTag: 1}
	close(msgs)

	mq.On("SubscribeVideoUpload").Return((<-chan amqp.Delivery)(msgs), nil)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	newTestWorker(1, db, minio, mq, vc).Start(ctx)

	mq.AssertExpectations(t)
	ack.AssertExpectations(t)
	vc.AssertExpectations(t)
}

// ─── updateJobFailed ──────────────────────────────────────────────────────────

func TestUpdateJobFailed(t *testing.T) {
	db := new(MockDatabase)
	db.On("UpdateProcessingJob", mock.MatchedBy(func(j *domain.ProcessingJob) bool {
		return j.Status == "failed" && j.ErrorMessage != nil && *j.ErrorMessage == "test error"
	})).Return(nil)

	w := newTestWorker(1, db, nil, nil, nil)
	w.updateJobFailed(&domain.ProcessingJob{ID: "j1", Status: "running"}, errors.New("test error"))

	db.AssertExpectations(t)
}

// ─── updateVideoFailed ────────────────────────────────────────────────────────

func TestUpdateVideoFailed(t *testing.T) {
	mq := new(MockRabbitMQ)
	vc := new(MockVideoClient)

	vc.On("FailVideo", "v1", "test error").Return(nil)
	mq.On("PublishNotification", mock.Anything).Return(nil)

	w := newTestWorker(1, nil, nil, mq, vc)
	w.updateVideoFailed(&domain.Video{ID: "v1", UserID: "u1"}, errors.New("test error"))

	vc.AssertExpectations(t)
	mq.AssertExpectations(t)
}

func TestUpdateVideoFailed_FailVideoError(t *testing.T) {
	mq := new(MockRabbitMQ)
	vc := new(MockVideoClient)

	vc.On("FailVideo", "v1", "boom").Return(errors.New("http error"))
	mq.On("PublishNotification", mock.Anything).Return(nil)

	w := newTestWorker(1, nil, nil, mq, vc)
	w.updateVideoFailed(&domain.Video{ID: "v1", UserID: "u1"}, errors.New("boom"))

	vc.AssertExpectations(t)
	mq.AssertExpectations(t)
}

// ─── createZipFile / addFileToZip ─────────────────────────────────────────────

func TestCreateZipFile_Success(t *testing.T) {
	w := newTestWorker(1, nil, nil, nil, nil)
	tempDir, _ := os.MkdirTemp("", "zip-test")
	defer os.RemoveAll(tempDir)

	f1 := filepath.Join(tempDir, "1.txt")
	os.WriteFile(f1, []byte("test content"), 0644)

	zipPath := filepath.Join(tempDir, "out.zip")
	err := w.createZipFile([]string{f1}, zipPath)

	assert.NoError(t, err)
	_, statErr := os.Stat(zipPath)
	assert.NoError(t, statErr)
}

func TestCreateZipFile_InvalidDestination(t *testing.T) {
	w := newTestWorker(1, nil, nil, nil, nil)
	err := w.createZipFile([]string{"notexist.txt"}, "/no/such/dir/out.zip")
	assert.Error(t, err)
}

func TestCreateZipFile_MissingSourceFile(t *testing.T) {
	w := newTestWorker(1, nil, nil, nil, nil)
	tempDir, _ := os.MkdirTemp("", "zip-test")
	defer os.RemoveAll(tempDir)

	zipPath := filepath.Join(tempDir, "out.zip")
	err := w.createZipFile([]string{"/no/such/file.txt"}, zipPath)
	assert.Error(t, err)
}

func TestAddFileToZip_Error(t *testing.T) {
	w := newTestWorker(1, nil, nil, nil, nil)
	err := w.addFileToZip(nil, "nonexistent.png")
	assert.Error(t, err)
}

// ─── helper functions ─────────────────────────────────────────────────────────

func TestGenerateID(t *testing.T) {
	id := generateID()
	assert.Len(t, id, 36)
}

func TestGenerateID_Unique(t *testing.T) {
	ids := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		ids[generateID()] = true
	}
	assert.Len(t, ids, 100)
}

func TestTimePtr(t *testing.T) {
	now := time.Now()
	ptr := timePtr(now)
	assert.Equal(t, now, *ptr)
}

func TestStringPtr(t *testing.T) {
	s := "hello"
	ptr := stringPtr(s)
	assert.Equal(t, s, *ptr)
}
