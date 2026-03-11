package handlers

import (
"encoding/json"
"errors"
"net/http"
"net/http/httptest"
"testing"
"time"

"status-service/domain"
"status-service/infra/utils"
"status-service/service"

"github.com/gin-gonic/gin"
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/mock"
)

// MockDatabase implements domain.DatabaseInterface (only Ping/Close).
type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) Ping() error {
	return m.Called().Error(0)
}

func (m *MockDatabase) Close() error {
	return m.Called().Error(0)
}

// MockVideoClient implements domain.VideoServiceClient.
type MockVideoClient struct {
	mock.Mock
}

func (m *MockVideoClient) GetVideosByUserID(userID, status string) ([]*domain.Video, error) {
	args := m.Called(userID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Video), args.Error(1)
}

func (m *MockVideoClient) GetVideoByID(id string) (*domain.Video, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Video), args.Error(1)
}

func (m *MockVideoClient) GetUserStats(userID string) (*domain.UserStats, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserStats), args.Error(1)
}

func (m *MockVideoClient) GetSystemStats() (*domain.SystemStats, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.SystemStats), args.Error(1)
}

// MockRedis implements domain.RedisInterface.
type MockRedis struct {
	mock.Mock
}

func (m *MockRedis) Get(key string) (string, error) {
	args := m.Called(key)
	return args.String(0), args.Error(1)
}

func (m *MockRedis) Set(key string, value interface{}, expiration time.Duration) error {
	args := m.Called(key, value, expiration)
	return args.Error(0)
}

func (m *MockRedis) Ping() error {
	return m.Called().Error(0)
}

func (m *MockRedis) Close() error {
	return m.Called().Error(0)
}

// MockMinIO implements domain.MinIOInterface.
type MockMinIO struct {
	mock.Mock
}

func (m *MockMinIO) GetPresignedURL(objectName string, expires time.Duration) (string, error) {
	args := m.Called(objectName, expires)
	return args.String(0), args.Error(1)
}

func (m *MockMinIO) Ping() error {
	return m.Called().Error(0)
}

func setupTestRouter(db domain.DatabaseInterface, redis domain.RedisInterface, minio domain.MinIOInterface, vc domain.VideoServiceClient) (*gin.Engine, *StatusHandler) {
	statusService := service.NewStatusService(db, redis, minio, vc)
	handler := NewStatusHandler(statusService)

	r := gin.New()
	r.GET("/videos", func(context *gin.Context) {
context.Set("user_id", "user123")
handler.ListVideos(context)
})
	r.GET("/videos/:id", func(context *gin.Context) {
context.Set("user_id", "user123")
handler.GetVideo(context)
})
	r.GET("/videos/:id/download", func(context *gin.Context) {
context.Set("user_id", "user123")
handler.DownloadZip(context)
})
	r.GET("/stats", func(context *gin.Context) {
context.Set("user_id", "user123")
handler.GetUserStats(context)
})
	r.GET("/system/stats", handler.GetSystemStats)

	return r, handler
}

func TestListVideos_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockVC := new(MockVideoClient)
	mockRedis := new(MockRedis)
	r, _ := setupTestRouter(nil, mockRedis, nil, mockVC)

	videos := []*domain.Video{
		{
			ID: "v1", Filename: "f1.mp4", Status: "completed", ZipPath: utils.StringPtr("z1.zip"),
			FrameCount: utils.IntPtr(100), ZipSizeBytes: utils.Int64Ptr(1024), ErrorMessage: utils.StringPtr(""),
			ProcessingStartedAt: utils.TimePtr(time.Now().Add(-1 * time.Minute)), ProcessingCompletedAt: utils.TimePtr(time.Now()),
		},
	}

	mockRedis.On("Get", "videos:user:user123:status:").Return("", errors.New("cache miss"))
	mockVC.On("GetVideosByUserID", "user123", "").Return(videos, nil)
	mockRedis.On("Set", "videos:user:user123:status:", mock.Anything, mock.Anything).Return(nil)

	req, _ := http.NewRequest("GET", "/videos", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListVideos_CacheHit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockRedis := new(MockRedis)
	r, _ := setupTestRouter(nil, mockRedis, nil, nil)

	videos := []domain.Video{{ID: "v-cached"}}
	cachedData, _ := json.Marshal(videos)

	mockRedis.On("Get", "videos:user:user123:status:").Return(string(cachedData), nil)

	req, _ := http.NewRequest("GET", "/videos", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListVideos_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockVC := new(MockVideoClient)
	mockRedis := new(MockRedis)
	r, _ := setupTestRouter(nil, mockRedis, nil, mockVC)

	mockRedis.On("Get", "videos:user:user123:status:").Return("", errors.New("cache miss"))
	mockVC.On("GetVideosByUserID", "user123", "").Return(nil, errors.New("service error"))

	req, _ := http.NewRequest("GET", "/videos", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetVideo_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockVC := new(MockVideoClient)
	mockRedis := new(MockRedis)
	r, _ := setupTestRouter(nil, mockRedis, nil, mockVC)

	video := &domain.Video{
		ID: "v1", UserID: "user123", Status: "completed", StoragePath: "path/v1.mp4",
		ProcessingStartedAt:   utils.TimePtr(time.Now().Add(-1 * time.Minute)),
		ProcessingCompletedAt: utils.TimePtr(time.Now()),
	}

	mockRedis.On("Get", "video:v1").Return("", errors.New("miss"))
	mockVC.On("GetVideoByID", "v1").Return(video, nil)
	mockRedis.On("Set", "video:v1", mock.Anything, mock.Anything).Return(nil)

	req, _ := http.NewRequest("GET", "/videos/v1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetVideo_CacheHit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockRedis := new(MockRedis)
	r, _ := setupTestRouter(nil, mockRedis, nil, nil)

	video := &domain.Video{ID: "v1", UserID: "user123"}
	cachedData, _ := json.Marshal(video)

	mockRedis.On("Get", "video:v1").Return(string(cachedData), nil)

	req, _ := http.NewRequest("GET", "/videos/v1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetVideo_AccessDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockVC := new(MockVideoClient)
	mockRedis := new(MockRedis)
	r, _ := setupTestRouter(nil, mockRedis, nil, mockVC)

	video := &domain.Video{ID: "v1", UserID: "other_user"}

	mockRedis.On("Get", "video:v1").Return("", errors.New("miss"))
	mockVC.On("GetVideoByID", "v1").Return(video, nil)

	req, _ := http.NewRequest("GET", "/videos/v1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetVideo_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockVC := new(MockVideoClient)
	mockRedis := new(MockRedis)
	r, _ := setupTestRouter(nil, mockRedis, nil, mockVC)

	mockRedis.On("Get", "video:v1").Return("", errors.New("miss"))
	mockVC.On("GetVideoByID", "v1").Return(nil, errors.New("video not found"))

	req, _ := http.NewRequest("GET", "/videos/v1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDownloadZip_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockVC := new(MockVideoClient)
	mockMinio := new(MockMinIO)
	mockRedis := new(MockRedis)
	r, _ := setupTestRouter(nil, mockRedis, mockMinio, mockVC)

	video := &domain.Video{
		ID: "v1", UserID: "user123", Status: "completed", ZipPath: utils.StringPtr("path/v1.zip"),
	}

	mockRedis.On("Get", "video:v1").Return("", errors.New("miss"))
	mockVC.On("GetVideoByID", "v1").Return(video, nil)
	mockRedis.On("Set", "video:v1", mock.Anything, mock.Anything).Return(nil)
	mockMinio.On("GetPresignedURL", "path/v1.zip", mock.Anything).Return("http://download", nil)

	req, _ := http.NewRequest("GET", "/videos/v1/download", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDownloadZip_NotCompleted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockVC := new(MockVideoClient)
	mockRedis := new(MockRedis)
	r, _ := setupTestRouter(nil, mockRedis, nil, mockVC)

	video := &domain.Video{ID: "v1", UserID: "user123", Status: "processing"}

	mockRedis.On("Get", "video:v1").Return("", errors.New("miss"))
	mockVC.On("GetVideoByID", "v1").Return(video, nil)
	mockRedis.On("Set", "video:v1", mock.Anything, mock.Anything).Return(nil)

	req, _ := http.NewRequest("GET", "/videos/v1/download", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDownloadZip_MinIOError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockVC := new(MockVideoClient)
	mockMinio := new(MockMinIO)
	mockRedis := new(MockRedis)
	r, _ := setupTestRouter(nil, mockRedis, mockMinio, mockVC)

	video := &domain.Video{
		ID: "v1", UserID: "user123", Status: "completed", ZipPath: utils.StringPtr("path/v1.zip"),
	}

	mockRedis.On("Get", "video:v1").Return("", errors.New("miss"))
	mockVC.On("GetVideoByID", "v1").Return(video, nil)
	mockRedis.On("Set", "video:v1", mock.Anything, mock.Anything).Return(nil)
	mockMinio.On("GetPresignedURL", "path/v1.zip", mock.Anything).Return("", errors.New("minio error"))

	req, _ := http.NewRequest("GET", "/videos/v1/download", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetUserStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockVC := new(MockVideoClient)
	mockRedis := new(MockRedis)
	r, _ := setupTestRouter(nil, mockRedis, nil, mockVC)

	stats := &domain.UserStats{TotalVideos: 10, CompletedVideos: 8}

	mockRedis.On("Get", "stats:user:user123").Return("", errors.New("miss"))
	mockVC.On("GetUserStats", "user123").Return(stats, nil)
	mockRedis.On("Set", "stats:user:user123", mock.Anything, mock.Anything).Return(nil)

	req, _ := http.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetUserStats_CacheHit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockRedis := new(MockRedis)
	r, _ := setupTestRouter(nil, mockRedis, nil, nil)

	stats := domain.UserStats{TotalVideos: 10}
	cachedData, _ := json.Marshal(stats)

	mockRedis.On("Get", "stats:user:user123").Return(string(cachedData), nil)

	req, _ := http.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetUserStats_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockVC := new(MockVideoClient)
	mockRedis := new(MockRedis)
	r, _ := setupTestRouter(nil, mockRedis, nil, mockVC)

	mockRedis.On("Get", "stats:user:user123").Return("", errors.New("miss"))
	mockVC.On("GetUserStats", "user123").Return(nil, errors.New("stats error"))

	req, _ := http.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetSystemStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockVC := new(MockVideoClient)
	mockRedis := new(MockRedis)
	r, _ := setupTestRouter(nil, mockRedis, nil, mockVC)

	stats := &domain.SystemStats{TotalUsers: 5, TotalVideos: 50}

	mockRedis.On("Get", "stats:system").Return("", errors.New("miss"))
	mockVC.On("GetSystemStats").Return(stats, nil)
	mockRedis.On("Set", "stats:system", mock.Anything, mock.Anything).Return(nil)

	req, _ := http.NewRequest("GET", "/system/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetSystemStats_CacheHit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockRedis := new(MockRedis)
	r, _ := setupTestRouter(nil, mockRedis, nil, nil)

	stats := domain.SystemStats{TotalUsers: 5}
	cachedData, _ := json.Marshal(stats)

	mockRedis.On("Get", "stats:system").Return(string(cachedData), nil)

	req, _ := http.NewRequest("GET", "/system/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetSystemStats_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockVC := new(MockVideoClient)
	mockRedis := new(MockRedis)
	r, _ := setupTestRouter(nil, mockRedis, nil, mockVC)

	mockRedis.On("Get", "stats:system").Return("", errors.New("miss"))
	mockVC.On("GetSystemStats").Return(nil, errors.New("system stats error"))

	req, _ := http.NewRequest("GET", "/system/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
