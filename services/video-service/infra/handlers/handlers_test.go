package handlers

import (
"bytes"
"errors"
"io"
"mime/multipart"
"net/http"
"net/http/httptest"
"testing"
"time"

"video-service/domain"

"github.com/gin-gonic/gin"
"github.com/minio/minio-go/v7"
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/mock"
)

// MockDatabase implements domain.DatabaseInterface.
type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) CreateVideo(video *domain.Video) error {
	return m.Called(video).Error(0)
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
	return m.Called(video).Error(0)
}

func (m *MockDatabase) DeleteVideo(id string) error {
	return m.Called(id).Error(0)
}

func (m *MockDatabase) GetUserStats(userID string) (*domain.UserStats, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserStats), args.Error(1)
}

func (m *MockDatabase) GetSystemStats() (*domain.SystemStats, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.SystemStats), args.Error(1)
}

func (m *MockDatabase) Ping() error {
	return m.Called().Error(0)
}

func (m *MockDatabase) Close() error {
	return m.Called().Error(0)
}

// MockMinIO implements domain.MinIOInterface.
type MockMinIO struct {
	mock.Mock
}

func (m *MockMinIO) UploadFile(reader io.Reader, filename string, size int64) (string, error) {
	args := m.Called(reader, filename, size)
	return args.String(0), args.Error(1)
}

func (m *MockMinIO) DeleteFile(objectName string) error {
	return m.Called(objectName).Error(0)
}

func (m *MockMinIO) GetFileStream(objectName string) (*minio.Object, error) {
	args := m.Called(objectName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*minio.Object), args.Error(1)
}

// MockRabbitMQ implements domain.RabbitMQInterface.
type MockRabbitMQ struct {
	mock.Mock
}

func (m *MockRabbitMQ) PublishVideoUpload(message domain.VideoProcessingMessage) error {
	return m.Called(message).Error(0)
}

func (m *MockRabbitMQ) Ping() error {
	return m.Called().Error(0)
}

func (m *MockRabbitMQ) Close() error {
	return m.Called().Error(0)
}

// MockAuthClient implements domain.AuthServiceClient.
type MockAuthClient struct {
	mock.Mock
}

func (m *MockAuthClient) CreateAuditLog(req domain.AuditLogRequest) error {
	return m.Called(req).Error(0)
}

// ---------- Upload ----------

func TestUpload_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	mockMinio := new(MockMinIO)
	mockRabbit := new(MockRabbitMQ)
	mockAuth := new(MockAuthClient)
	handler := NewVideoHandler(mockDB, mockMinio, mockRabbit, mockAuth)

	r := gin.New()
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
	mockAuth.On("CreateAuditLog", mock.Anything).Return(nil)

	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	time.Sleep(20 * time.Millisecond) // let audit goroutine finish
}

func TestUpload_InvalidFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewVideoHandler(nil, nil, nil, nil)

	r := gin.New()
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

func TestUpload_NoFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewVideoHandler(nil, nil, nil, nil)

	r := gin.New()
	r.POST("/upload", func(c *gin.Context) {
c.Set("user_id", "user123")
handler.Upload(c)
})

	req, _ := http.NewRequest("POST", "/upload", nil)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=xxx")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpload_MinIOError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockMinio := new(MockMinIO)
	handler := NewVideoHandler(nil, mockMinio, nil, nil)

	r := gin.New()
	r.POST("/upload", func(c *gin.Context) {
c.Set("user_id", "user123")
handler.Upload(c)
})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("video", "test.mp4")
	part.Write([]byte("fake video content"))
	writer.Close()

	mockMinio.On("UploadFile", mock.Anything, mock.Anything, mock.Anything).Return("", errors.New("upload failed"))

	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpload_DBCreateError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	mockMinio := new(MockMinIO)
	handler := NewVideoHandler(mockDB, mockMinio, nil, nil)

	r := gin.New()
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
	mockDB.On("CreateVideo", mock.Anything).Return(errors.New("db error"))
	mockMinio.On("DeleteFile", "path/test.mp4").Return(nil)

	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpload_RabbitMQError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	mockMinio := new(MockMinIO)
	mockRabbit := new(MockRabbitMQ)
	handler := NewVideoHandler(mockDB, mockMinio, mockRabbit, nil)

	r := gin.New()
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
	mockRabbit.On("PublishVideoUpload", mock.Anything).Return(errors.New("rabbitmq down"))
	mockDB.On("UpdateVideo", mock.Anything).Return(nil)

	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------- GetVideo ----------

func TestGetVideo_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	handler := NewVideoHandler(mockDB, nil, nil, nil)

	r := gin.New()
	r.GET("/videos/:id", func(c *gin.Context) {
c.Set("user_id", "user123")
handler.GetVideo(c)
})

	fc := 10
	zipPath := "z.zip"
	video := &domain.Video{ID: "v1", UserID: "user123", Status: "completed", ZipPath: &zipPath, FrameCount: &fc}
	mockDB.On("GetVideoByID", "v1").Return(video, nil)

	req, _ := http.NewRequest("GET", "/videos/v1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetVideo_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	handler := NewVideoHandler(mockDB, nil, nil, nil)

	r := gin.New()
	r.GET("/videos/:id", func(c *gin.Context) {
c.Set("user_id", "user123")
handler.GetVideo(c)
})

	mockDB.On("GetVideoByID", "v99").Return(nil, errors.New("not found"))

	req, _ := http.NewRequest("GET", "/videos/v99", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetVideo_AccessDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	handler := NewVideoHandler(mockDB, nil, nil, nil)

	r := gin.New()
	r.GET("/videos/:id", func(c *gin.Context) {
c.Set("user_id", "user123")
handler.GetVideo(c)
})

	video := &domain.Video{ID: "v1", UserID: "other_user"}
	mockDB.On("GetVideoByID", "v1").Return(video, nil)

	req, _ := http.NewRequest("GET", "/videos/v1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetVideo_WithTimestamps(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	handler := NewVideoHandler(mockDB, nil, nil, nil)

	r := gin.New()
	r.GET("/videos/:id", func(c *gin.Context) {
c.Set("user_id", "user123")
handler.GetVideo(c)
})

	now := time.Now()
	errMsg := "some error"
	video := &domain.Video{
		ID: "v1", UserID: "user123", Status: "failed",
		ErrorMessage:          &errMsg,
		ProcessingStartedAt:   &now,
		ProcessingCompletedAt: &now,
	}
	mockDB.On("GetVideoByID", "v1").Return(video, nil)

	req, _ := http.NewRequest("GET", "/videos/v1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------- List ----------

func TestList_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	handler := NewVideoHandler(mockDB, nil, nil, nil)

	r := gin.New()
	r.GET("/videos", func(c *gin.Context) {
c.Set("user_id", "user123")
handler.List(c)
})

	zipPath := "z.zip"
	fc := 5
	now := time.Now()
	errMsg := "err"
	videos := []*domain.Video{
		{
			ID: "v1", UserID: "user123", Status: "completed",
			ZipPath: &zipPath, FrameCount: &fc,
			ErrorMessage:          &errMsg,
			ProcessingStartedAt:   &now,
			ProcessingCompletedAt: &now,
		},
	}
	mockDB.On("GetVideosByUserID", "user123", "").Return(videos, nil)

	req, _ := http.NewRequest("GET", "/videos", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestList_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	handler := NewVideoHandler(mockDB, nil, nil, nil)

	r := gin.New()
	r.GET("/videos", func(c *gin.Context) {
c.Set("user_id", "user123")
handler.List(c)
})

	mockDB.On("GetVideosByUserID", "user123", "").Return(nil, errors.New("db error"))

	req, _ := http.NewRequest("GET", "/videos", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------- DeleteVideo ----------

func TestDeleteVideo_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	mockMinio := new(MockMinIO)
	mockAuth := new(MockAuthClient)
	handler := NewVideoHandler(mockDB, mockMinio, nil, mockAuth)

	r := gin.New()
	r.DELETE("/videos/:id", func(c *gin.Context) {
c.Set("user_id", "user123")
handler.DeleteVideo(c)
})

	zipPath := "z.zip"
	video := &domain.Video{ID: "v1", UserID: "user123", StoragePath: "s", ZipPath: &zipPath}
	mockDB.On("GetVideoByID", "v1").Return(video, nil)
	mockMinio.On("DeleteFile", "s").Return(nil)
	mockMinio.On("DeleteFile", "z.zip").Return(nil)
	mockDB.On("DeleteVideo", "v1").Return(nil)
	mockAuth.On("CreateAuditLog", mock.Anything).Return(nil)

	req, _ := http.NewRequest("DELETE", "/videos/v1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	time.Sleep(20 * time.Millisecond)
}

func TestDeleteVideo_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	handler := NewVideoHandler(mockDB, nil, nil, nil)

	r := gin.New()
	r.DELETE("/videos/:id", func(c *gin.Context) {
c.Set("user_id", "user123")
handler.DeleteVideo(c)
})

	mockDB.On("GetVideoByID", "v99").Return(nil, errors.New("not found"))

	req, _ := http.NewRequest("DELETE", "/videos/v99", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteVideo_AccessDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	handler := NewVideoHandler(mockDB, nil, nil, nil)

	r := gin.New()
	r.DELETE("/videos/:id", func(c *gin.Context) {
c.Set("user_id", "user123")
handler.DeleteVideo(c)
})

	video := &domain.Video{ID: "v1", UserID: "other_user"}
	mockDB.On("GetVideoByID", "v1").Return(video, nil)

	req, _ := http.NewRequest("DELETE", "/videos/v1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestDeleteVideo_DBError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	mockMinio := new(MockMinIO)
	handler := NewVideoHandler(mockDB, mockMinio, nil, nil)

	r := gin.New()
	r.DELETE("/videos/:id", func(c *gin.Context) {
c.Set("user_id", "user123")
handler.DeleteVideo(c)
})

	video := &domain.Video{ID: "v1", UserID: "user123", StoragePath: ""}
	mockDB.On("GetVideoByID", "v1").Return(video, nil)
	mockDB.On("DeleteVideo", "v1").Return(errors.New("db error"))

	req, _ := http.NewRequest("DELETE", "/videos/v1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDeleteVideo_EmptyZipPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	mockMinio := new(MockMinIO)
	mockAuth := new(MockAuthClient)
	handler := NewVideoHandler(mockDB, mockMinio, nil, mockAuth)

	r := gin.New()
	r.DELETE("/videos/:id", func(c *gin.Context) {
		c.Set("user_id", "user123")
		handler.DeleteVideo(c)
	})

	emptyZip := ""
	video := &domain.Video{ID: "v1", UserID: "user123", StoragePath: "s", ZipPath: &emptyZip}
	mockDB.On("GetVideoByID", "v1").Return(video, nil)
	mockMinio.On("DeleteFile", "s").Return(nil)
	mockDB.On("DeleteVideo", "v1").Return(nil)
	mockAuth.On("CreateAuditLog", mock.Anything).Return(nil)

	req, _ := http.NewRequest("DELETE", "/videos/v1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	time.Sleep(20 * time.Millisecond)
}

func TestDownloadZip_Error_NotCompleted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	handler := NewVideoHandler(mockDB, nil, nil, nil)

	r := gin.New()
	r.GET("/videos/:id/download", handler.DownloadZip)

	video := &domain.Video{ID: "v1", Status: "processing"}
	mockDB.On("GetVideoByID", "v1").Return(video, nil)

	req, _ := http.NewRequest("GET", "/videos/v1/download", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDownloadZip_Error_GetStreamFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	mockMinio := new(MockMinIO)
	handler := NewVideoHandler(mockDB, mockMinio, nil, nil)

	r := gin.New()
	r.GET("/videos/:id/download", handler.DownloadZip)

	zipPath := "z.zip"
	video := &domain.Video{ID: "v1", Status: "completed", ZipPath: &zipPath}
	mockDB.On("GetVideoByID", "v1").Return(video, nil)
	mockMinio.On("GetFileStream", "z.zip").Return(nil, errors.New("stream error"))

	req, _ := http.NewRequest("GET", "/videos/v1/download", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------- helpers ----------

func TestIsValidVideoFile(t *testing.T) {
	assert.True(t, isValidVideoFile("test.mp4"))
	assert.True(t, isValidVideoFile("test.AVI"))
	assert.True(t, isValidVideoFile("test.MOV"))
	assert.True(t, isValidVideoFile("test.mkv"))
	assert.True(t, isValidVideoFile("test.wmv"))
	assert.True(t, isValidVideoFile("test.flv"))
	assert.True(t, isValidVideoFile("test.webm"))
	assert.False(t, isValidVideoFile("test.txt"))
	assert.False(t, isValidVideoFile("test.pdf"))
}

func TestStringPtr(t *testing.T) {
	p := StringPtr("hello")
	assert.NotNil(t, p)
	assert.Equal(t, "hello", *p)
}

func TestTimePtr(t *testing.T) {
	now := time.Now()
	p := TimePtr(now)
	assert.NotNil(t, p)
	assert.Equal(t, now, *p)
}
