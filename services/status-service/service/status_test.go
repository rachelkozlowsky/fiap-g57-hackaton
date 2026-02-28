package service

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"status-service/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ---------- mocks ----------

type mockDB struct{ mock.Mock }

func (m *mockDB) Ping() error  { return m.Called().Error(0) }
func (m *mockDB) Close() error { return m.Called().Error(0) }

type mockRedis struct{ mock.Mock }

func (m *mockRedis) Get(key string) (string, error) {
	args := m.Called(key)
	return args.String(0), args.Error(1)
}
func (m *mockRedis) Set(key string, value interface{}, exp time.Duration) error {
	return m.Called(key, value, exp).Error(0)
}
func (m *mockRedis) Ping() error  { return m.Called().Error(0) }
func (m *mockRedis) Close() error { return m.Called().Error(0) }

type mockMinIO struct{ mock.Mock }

func (m *mockMinIO) GetPresignedURL(obj string, exp time.Duration) (string, error) {
	args := m.Called(obj, exp)
	return args.String(0), args.Error(1)
}
func (m *mockMinIO) Ping() error { return m.Called().Error(0) }

type mockVideoClient struct{ mock.Mock }

func (m *mockVideoClient) GetVideosByUserID(userID, status string) ([]*domain.Video, error) {
	args := m.Called(userID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Video), args.Error(1)
}
func (m *mockVideoClient) GetVideoByID(id string) (*domain.Video, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Video), args.Error(1)
}
func (m *mockVideoClient) GetUserStats(userID string) (*domain.UserStats, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserStats), args.Error(1)
}
func (m *mockVideoClient) GetSystemStats() (*domain.SystemStats, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.SystemStats), args.Error(1)
}

func newSvc(r *mockRedis, vc *mockVideoClient, mn *mockMinIO) *StatusService {
	return NewStatusService(nil, r, mn, vc)
}

// ---------- ListVideos ----------

func TestListVideos_CacheHit(t *testing.T) {
	r := new(mockRedis)
	videos := []domain.Video{{ID: "v1"}}
	data, _ := json.Marshal(videos)
	r.On("Get", "videos:user:u1:status:").Return(string(data), nil)

	svc := newSvc(r, nil, nil)
	result, err := svc.ListVideos("u1", "")
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestListVideos_CacheMiss_Success(t *testing.T) {
	r := new(mockRedis)
	vc := new(mockVideoClient)
	r.On("Get", "videos:user:u1:status:completed").Return("", errors.New("miss"))
	vc.On("GetVideosByUserID", "u1", "completed").Return([]*domain.Video{{ID: "v1", Status: "completed"}}, nil)
	r.On("Set", "videos:user:u1:status:completed", mock.Anything, mock.Anything).Return(nil)

	svc := newSvc(r, vc, nil)
	result, err := svc.ListVideos("u1", "completed")
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestListVideos_Error(t *testing.T) {
	r := new(mockRedis)
	vc := new(mockVideoClient)
	r.On("Get", "videos:user:u1:status:").Return("", errors.New("miss"))
	vc.On("GetVideosByUserID", "u1", "").Return(nil, errors.New("service down"))

	svc := newSvc(r, vc, nil)
	_, err := svc.ListVideos("u1", "")
	assert.Error(t, err)
}

// ---------- GetVideo ----------

func TestGetVideo_CacheHit_Owner(t *testing.T) {
	r := new(mockRedis)
	video := &domain.Video{ID: "v1", UserID: "u1"}
	data, _ := json.Marshal(video)
	r.On("Get", "video:v1").Return(string(data), nil)

	svc := newSvc(r, nil, nil)
	result, err := svc.GetVideo("v1", "u1")
	assert.NoError(t, err)
	assert.Equal(t, "v1", result.ID)
}

func TestGetVideo_CacheHit_Forbidden(t *testing.T) {
	r := new(mockRedis)
	video := &domain.Video{ID: "v1", UserID: "other"}
	data, _ := json.Marshal(video)
	r.On("Get", "video:v1").Return(string(data), nil)

	svc := newSvc(r, nil, nil)
	_, err := svc.GetVideo("v1", "u1")
	assert.EqualError(t, err, "access denied")
}

func TestGetVideo_CacheMiss_Success(t *testing.T) {
	r := new(mockRedis)
	vc := new(mockVideoClient)
	video := &domain.Video{ID: "v1", UserID: "u1"}
	r.On("Get", "video:v1").Return("", errors.New("miss"))
	vc.On("GetVideoByID", "v1").Return(video, nil)
	r.On("Set", "video:v1", mock.Anything, mock.Anything).Return(nil)

	svc := newSvc(r, vc, nil)
	result, err := svc.GetVideo("v1", "u1")
	assert.NoError(t, err)
	assert.Equal(t, "v1", result.ID)
}

func TestGetVideo_CacheMiss_Forbidden(t *testing.T) {
	r := new(mockRedis)
	vc := new(mockVideoClient)
	video := &domain.Video{ID: "v1", UserID: "other"}
	r.On("Get", "video:v1").Return("", errors.New("miss"))
	vc.On("GetVideoByID", "v1").Return(video, nil)

	svc := newSvc(r, vc, nil)
	_, err := svc.GetVideo("v1", "u1")
	assert.EqualError(t, err, "access denied")
}

func TestGetVideo_Error(t *testing.T) {
	r := new(mockRedis)
	vc := new(mockVideoClient)
	r.On("Get", "video:v1").Return("", errors.New("miss"))
	vc.On("GetVideoByID", "v1").Return(nil, errors.New("not found"))

	svc := newSvc(r, vc, nil)
	_, err := svc.GetVideo("v1", "u1")
	assert.Error(t, err)
}

// ---------- GetDownloadURL ----------

func zipPathPtr(s string) *string { return &s }

func TestGetDownloadURL_Success(t *testing.T) {
	r := new(mockRedis)
	vc := new(mockVideoClient)
	mn := new(mockMinIO)
	zipPath := "path/v1.zip"
	video := &domain.Video{ID: "v1", UserID: "u1", Status: "completed", ZipPath: &zipPath}
	r.On("Get", "video:v1").Return("", errors.New("miss"))
	vc.On("GetVideoByID", "v1").Return(video, nil)
	r.On("Set", "video:v1", mock.Anything, mock.Anything).Return(nil)
	mn.On("GetPresignedURL", "path/v1.zip", mock.Anything).Return("http://dl", nil)

	svc := NewStatusService(nil, r, mn, vc)
	url, err := svc.GetDownloadURL("v1", "u1")
	assert.NoError(t, err)
	assert.Equal(t, "http://dl", url)
}

func TestGetDownloadURL_NotCompleted(t *testing.T) {
	r := new(mockRedis)
	vc := new(mockVideoClient)
	video := &domain.Video{ID: "v1", UserID: "u1", Status: "processing"}
	r.On("Get", "video:v1").Return("", errors.New("miss"))
	vc.On("GetVideoByID", "v1").Return(video, nil)
	r.On("Set", "video:v1", mock.Anything, mock.Anything).Return(nil)

	svc := NewStatusService(nil, r, nil, vc)
	_, err := svc.GetDownloadURL("v1", "u1")
	assert.EqualError(t, err, "video processing not completed")
}

func TestGetDownloadURL_NoZipPath(t *testing.T) {
	r := new(mockRedis)
	vc := new(mockVideoClient)
	video := &domain.Video{ID: "v1", UserID: "u1", Status: "completed", ZipPath: nil}
	r.On("Get", "video:v1").Return("", errors.New("miss"))
	vc.On("GetVideoByID", "v1").Return(video, nil)
	r.On("Set", "video:v1", mock.Anything, mock.Anything).Return(nil)

	svc := NewStatusService(nil, r, nil, vc)
	_, err := svc.GetDownloadURL("v1", "u1")
	assert.EqualError(t, err, "ZIP file not found")
}

func TestGetDownloadURL_MinIOError(t *testing.T) {
	r := new(mockRedis)
	vc := new(mockVideoClient)
	mn := new(mockMinIO)
	zipPath := "path/v1.zip"
	video := &domain.Video{ID: "v1", UserID: "u1", Status: "completed", ZipPath: &zipPath}
	r.On("Get", "video:v1").Return("", errors.New("miss"))
	vc.On("GetVideoByID", "v1").Return(video, nil)
	r.On("Set", "video:v1", mock.Anything, mock.Anything).Return(nil)
	mn.On("GetPresignedURL", "path/v1.zip", mock.Anything).Return("", errors.New("minio down"))

	svc := NewStatusService(nil, r, mn, vc)
	_, err := svc.GetDownloadURL("v1", "u1")
	assert.Error(t, err)
}

func TestGetDownloadURL_VideoError(t *testing.T) {
	r := new(mockRedis)
	vc := new(mockVideoClient)
	r.On("Get", "video:v1").Return("", errors.New("miss"))
	vc.On("GetVideoByID", "v1").Return(nil, errors.New("not found"))

	svc := NewStatusService(nil, r, nil, vc)
	_, err := svc.GetDownloadURL("v1", "u1")
	assert.Error(t, err)
}

// ---------- GetUserStats ----------

func TestGetUserStats_CacheHit(t *testing.T) {
	r := new(mockRedis)
	stats := &domain.UserStats{TotalVideos: 5}
	data, _ := json.Marshal(stats)
	r.On("Get", "stats:user:u1").Return(string(data), nil)

	svc := newSvc(r, nil, nil)
	result, err := svc.GetUserStats("u1")
	assert.NoError(t, err)
	assert.Equal(t, 5, result.TotalVideos)
}

func TestGetUserStats_CacheMiss_Success(t *testing.T) {
	r := new(mockRedis)
	vc := new(mockVideoClient)
	stats := &domain.UserStats{TotalVideos: 10}
	r.On("Get", "stats:user:u1").Return("", errors.New("miss"))
	vc.On("GetUserStats", "u1").Return(stats, nil)
	r.On("Set", "stats:user:u1", mock.Anything, mock.Anything).Return(nil)

	svc := newSvc(r, vc, nil)
	result, err := svc.GetUserStats("u1")
	assert.NoError(t, err)
	assert.Equal(t, 10, result.TotalVideos)
}

func TestGetUserStats_Error(t *testing.T) {
	r := new(mockRedis)
	vc := new(mockVideoClient)
	r.On("Get", "stats:user:u1").Return("", errors.New("miss"))
	vc.On("GetUserStats", "u1").Return(nil, errors.New("service error"))

	svc := newSvc(r, vc, nil)
	_, err := svc.GetUserStats("u1")
	assert.Error(t, err)
}

// ---------- GetSystemStats ----------

func TestGetSystemStats_CacheHit(t *testing.T) {
	r := new(mockRedis)
	stats := &domain.SystemStats{TotalUsers: 3}
	data, _ := json.Marshal(stats)
	r.On("Get", "stats:system").Return(string(data), nil)

	svc := newSvc(r, nil, nil)
	result, err := svc.GetSystemStats()
	assert.NoError(t, err)
	assert.Equal(t, 3, result.TotalUsers)
}

func TestGetSystemStats_CacheMiss_Success(t *testing.T) {
	r := new(mockRedis)
	vc := new(mockVideoClient)
	stats := &domain.SystemStats{TotalUsers: 7}
	r.On("Get", "stats:system").Return("", errors.New("miss"))
	vc.On("GetSystemStats").Return(stats, nil)
	r.On("Set", "stats:system", mock.Anything, mock.Anything).Return(nil)

	svc := newSvc(r, vc, nil)
	result, err := svc.GetSystemStats()
	assert.NoError(t, err)
	assert.Equal(t, 7, result.TotalUsers)
}

func TestGetSystemStats_Error(t *testing.T) {
	r := new(mockRedis)
	vc := new(mockVideoClient)
	r.On("Get", "stats:system").Return("", errors.New("miss"))
	vc.On("GetSystemStats").Return(nil, errors.New("service error"))

	svc := newSvc(r, vc, nil)
	_, err := svc.GetSystemStats()
	assert.Error(t, err)
}
