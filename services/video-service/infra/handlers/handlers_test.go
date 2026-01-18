package handlers

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"video-service/domain"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) CreateVideo(video *domain.Video) error {
	args := m.Called(video)
	return args.Error(0)
}

func (m *MockDatabase) GetVideoByID(id string) (*domain.Video, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Video), args.Error(1)
}

func (m *MockDatabase) GetVideosByUserID(userID, status string) ([]*domain.Video, error) {
	args := m.Called(userID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Video), args.Error(1)
}

func (m *MockDatabase) UpdateVideo(video *domain.Video) error {
	args := m.Called(video)
	return args.Error(0)
}

func (m *MockDatabase) DeleteVideo(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockDatabase) CreateAuditLog(log *domain.AuditLog) error {
	args := m.Called(log)
	return args.Error(0)
}

func (m *MockDatabase) Ping() error {
    args := m.Called()
    return args.Error(0)
}

func (m *MockDatabase) Close() error {
    args := m.Called()
    return args.Error(0)
}

type MockMinIO struct {
	mock.Mock
}

func (m *MockMinIO) UploadFile(reader io.Reader, filename string, size int64) (string, error) {
	args := m.Called(reader, filename, size)
	return args.String(0), args.Error(1)
}

func (m *MockMinIO) DeleteFile(objectName string) error {
	args := m.Called(objectName)
	return args.Error(0)
}

func (m *MockMinIO) GetFileStream(objectName string) (*minio.Object, error) {
	args := m.Called(objectName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*minio.Object), args.Error(1)
}

type MockRabbitMQ struct {
	mock.Mock
}

func (m *MockRabbitMQ) PublishVideoUpload(message domain.VideoProcessingMessage) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockRabbitMQ) Ping() error {
    args := m.Called()
    return args.Error(0)
}

func (m *MockRabbitMQ) Close() error {
    args := m.Called()
    return args.Error(0)
}

func TestUpload_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	mockMinio := new(MockMinIO)
	mockRabbit := new(MockRabbitMQ)
	handler := NewVideoHandler(mockDB, mockMinio, mockRabbit)

	r := gin.Default()
	r.POST("/upload", func(c *gin.Context) {
		c.Set("user_id", "user123")
		handler.Upload(c)
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("video", "test.mp4")
	part.Write([]byte("fake video content"))
	writer.Close()

	mockMinio.On("UploadFile", mock.Anything, mock.Anything, mock.Anything).Return("path/test.mp4", nil)
	mockDB.On("CreateVideo", mock.Anything).Return(nil)
	mockRabbit.On("PublishVideoUpload", mock.Anything).Return(nil)
	mockDB.On("UpdateVideo", mock.Anything).Return(nil)
	mockDB.On("CreateAuditLog", mock.Anything).Return(nil)

	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpload_InvalidFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewVideoHandler(nil, nil, nil)

	r := gin.Default()
	r.POST("/upload", handler.Upload)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("video", "test.txt")
	part.Write([]byte("not a video"))
	writer.Close()

	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid video format")
}

func TestGetVideo_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	handler := NewVideoHandler(mockDB, nil, nil)

	r := gin.Default()
	r.GET("/videos/:id", func(c *gin.Context) {
		c.Set("user_id", "user123")
		handler.GetVideo(c)
	})

	video := &domain.Video{ID: "v1", UserID: "user123", Status: "completed", ZipPath: StringPtr("z.zip")}
	mockDB.On("GetVideoByID", "v1").Return(video, nil)

	req, _ := http.NewRequest("GET", "/videos/v1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestList_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	handler := NewVideoHandler(mockDB, nil, nil)

	r := gin.Default()
	r.GET("/videos", func(c *gin.Context) {
		c.Set("user_id", "user123")
		handler.List(c)
	})

	videos := []*domain.Video{{ID: "v1", UserID: "user123"}}
	mockDB.On("GetVideosByUserID", "user123", "").Return(videos, nil)

	req, _ := http.NewRequest("GET", "/videos", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteVideo_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	mockMinio := new(MockMinIO)
	handler := NewVideoHandler(mockDB, mockMinio, nil)

	r := gin.Default()
	r.DELETE("/videos/:id", func(c *gin.Context) {
		c.Set("user_id", "user123")
		handler.DeleteVideo(c)
	})

	video := &domain.Video{ID: "v1", UserID: "user123", StoragePath: "s", ZipPath: StringPtr("z")}
	mockDB.On("GetVideoByID", "v1").Return(video, nil)
	mockMinio.On("DeleteFile", "s").Return(nil)
	mockMinio.On("DeleteFile", "z").Return(nil)
	mockDB.On("DeleteVideo", "v1").Return(nil)
	mockDB.On("CreateAuditLog", mock.Anything).Return(nil)

	req, _ := http.NewRequest("DELETE", "/videos/v1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDownloadZip_Error_NotCompleted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	handler := NewVideoHandler(mockDB, nil, nil)

	r := gin.Default()
	r.GET("/videos/:id/download", handler.DownloadZip)

	video := &domain.Video{ID: "v1", Status: "processing"}
	mockDB.On("GetVideoByID", "v1").Return(video, nil)

	req, _ := http.NewRequest("GET", "/videos/v1/download", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIsValidVideoFile(t *testing.T) {
	assert.True(t, isValidVideoFile("test.mp4"))
	assert.True(t, isValidVideoFile("test.AVI"))
	assert.False(t, isValidVideoFile("test.txt"))
}
