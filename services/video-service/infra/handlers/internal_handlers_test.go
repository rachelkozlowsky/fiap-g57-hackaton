package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"video-service/domain"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ---------- GetVideoByID ----------

func TestInternalGetVideoByID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	h := NewInternalHandler(mockDB)

	r := gin.New()
	r.GET("/internal/videos/:id", h.GetVideoByID)

	mockDB.On("GetVideoByID", "v1").Return(&domain.Video{ID: "v1"}, nil)

	req, _ := http.NewRequest("GET", "/internal/videos/v1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInternalGetVideoByID_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	h := NewInternalHandler(mockDB)

	r := gin.New()
	r.GET("/internal/videos/:id", h.GetVideoByID)

	mockDB.On("GetVideoByID", "v99").Return(nil, errors.New("not found"))

	req, _ := http.NewRequest("GET", "/internal/videos/v99", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------- ListUserVideos ----------

func TestInternalListUserVideos_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	h := NewInternalHandler(mockDB)

	r := gin.New()
	r.GET("/internal/videos", h.ListUserVideos)

	mockDB.On("GetVideosByUserID", "u1", "").Return([]*domain.Video{{ID: "v1", UserID: "u1"}}, nil)

	req, _ := http.NewRequest("GET", "/internal/videos?user_id=u1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInternalListUserVideos_MissingUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	h := NewInternalHandler(mockDB)

	r := gin.New()
	r.GET("/internal/videos", h.ListUserVideos)

	req, _ := http.NewRequest("GET", "/internal/videos", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInternalListUserVideos_DBError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	h := NewInternalHandler(mockDB)

	r := gin.New()
	r.GET("/internal/videos", h.ListUserVideos)

	mockDB.On("GetVideosByUserID", "u1", "").Return(nil, errors.New("db error"))

	req, _ := http.NewRequest("GET", "/internal/videos?user_id=u1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------- UpdateVideoStatus ----------

func TestInternalUpdateVideoStatus_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	h := NewInternalHandler(mockDB)

	r := gin.New()
	r.PUT("/internal/videos/:id/status", h.UpdateVideoStatus)

	mockDB.On("GetVideoByID", "v1").Return(&domain.Video{ID: "v1"}, nil)
	mockDB.On("UpdateVideo", mock.Anything).Return(nil)

	body, _ := json.Marshal(UpdateStatusRequest{Status: "processing"})
	req, _ := http.NewRequest("PUT", "/internal/videos/v1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInternalUpdateVideoStatus_WithError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	h := NewInternalHandler(mockDB)

	r := gin.New()
	r.PUT("/internal/videos/:id/status", h.UpdateVideoStatus)

	mockDB.On("GetVideoByID", "v1").Return(&domain.Video{ID: "v1"}, nil)
	mockDB.On("UpdateVideo", mock.Anything).Return(nil)

	body, _ := json.Marshal(UpdateStatusRequest{Status: "failed", ErrorMessage: "processing error"})
	req, _ := http.NewRequest("PUT", "/internal/videos/v1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInternalUpdateVideoStatus_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	h := NewInternalHandler(mockDB)

	r := gin.New()
	r.PUT("/internal/videos/:id/status", h.UpdateVideoStatus)

	req, _ := http.NewRequest("PUT", "/internal/videos/v1/status", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInternalUpdateVideoStatus_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	h := NewInternalHandler(mockDB)

	r := gin.New()
	r.PUT("/internal/videos/:id/status", h.UpdateVideoStatus)

	mockDB.On("GetVideoByID", "v99").Return(nil, errors.New("not found"))

	body, _ := json.Marshal(UpdateStatusRequest{Status: "processing"})
	req, _ := http.NewRequest("PUT", "/internal/videos/v99/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestInternalUpdateVideoStatus_UpdateError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	h := NewInternalHandler(mockDB)

	r := gin.New()
	r.PUT("/internal/videos/:id/status", h.UpdateVideoStatus)

	mockDB.On("GetVideoByID", "v1").Return(&domain.Video{ID: "v1"}, nil)
	mockDB.On("UpdateVideo", mock.Anything).Return(errors.New("db error"))

	body, _ := json.Marshal(UpdateStatusRequest{Status: "processing"})
	req, _ := http.NewRequest("PUT", "/internal/videos/v1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------- CompleteVideo ----------

func TestInternalCompleteVideo_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	h := NewInternalHandler(mockDB)

	r := gin.New()
	r.PUT("/internal/videos/:id/complete", h.CompleteVideo)

	mockDB.On("GetVideoByID", "v1").Return(&domain.Video{ID: "v1"}, nil)
	mockDB.On("UpdateVideo", mock.Anything).Return(nil)

	body, _ := json.Marshal(CompleteVideoRequest{ZipPath: "v1.zip", ZipSizeBytes: 1024, FrameCount: 50})
	req, _ := http.NewRequest("PUT", "/internal/videos/v1/complete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInternalCompleteVideo_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	h := NewInternalHandler(mockDB)

	r := gin.New()
	r.PUT("/internal/videos/:id/complete", h.CompleteVideo)

	req, _ := http.NewRequest("PUT", "/internal/videos/v1/complete", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInternalCompleteVideo_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	h := NewInternalHandler(mockDB)

	r := gin.New()
	r.PUT("/internal/videos/:id/complete", h.CompleteVideo)

	mockDB.On("GetVideoByID", "v99").Return(nil, errors.New("not found"))

	body, _ := json.Marshal(CompleteVideoRequest{ZipPath: "v99.zip"})
	req, _ := http.NewRequest("PUT", "/internal/videos/v99/complete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestInternalCompleteVideo_UpdateError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	h := NewInternalHandler(mockDB)

	r := gin.New()
	r.PUT("/internal/videos/:id/complete", h.CompleteVideo)

	mockDB.On("GetVideoByID", "v1").Return(&domain.Video{ID: "v1"}, nil)
	mockDB.On("UpdateVideo", mock.Anything).Return(errors.New("db error"))

	body, _ := json.Marshal(CompleteVideoRequest{ZipPath: "v1.zip"})
	req, _ := http.NewRequest("PUT", "/internal/videos/v1/complete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------- FailVideo ----------

func TestInternalFailVideo_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	h := NewInternalHandler(mockDB)

	r := gin.New()
	r.PUT("/internal/videos/:id/fail", h.FailVideo)

	mockDB.On("GetVideoByID", "v1").Return(&domain.Video{ID: "v1"}, nil)
	mockDB.On("UpdateVideo", mock.Anything).Return(nil)

	body, _ := json.Marshal(FailVideoRequest{ErrorMessage: "processing failed"})
	req, _ := http.NewRequest("PUT", "/internal/videos/v1/fail", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInternalFailVideo_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	h := NewInternalHandler(mockDB)

	r := gin.New()
	r.PUT("/internal/videos/:id/fail", h.FailVideo)

	req, _ := http.NewRequest("PUT", "/internal/videos/v1/fail", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInternalFailVideo_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	h := NewInternalHandler(mockDB)

	r := gin.New()
	r.PUT("/internal/videos/:id/fail", h.FailVideo)

	mockDB.On("GetVideoByID", "v99").Return(nil, errors.New("not found"))

	body, _ := json.Marshal(FailVideoRequest{ErrorMessage: "err"})
	req, _ := http.NewRequest("PUT", "/internal/videos/v99/fail", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestInternalFailVideo_UpdateError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	h := NewInternalHandler(mockDB)

	r := gin.New()
	r.PUT("/internal/videos/:id/fail", h.FailVideo)

	mockDB.On("GetVideoByID", "v1").Return(&domain.Video{ID: "v1"}, nil)
	mockDB.On("UpdateVideo", mock.Anything).Return(errors.New("db error"))

	body, _ := json.Marshal(FailVideoRequest{ErrorMessage: "err"})
	req, _ := http.NewRequest("PUT", "/internal/videos/v1/fail", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------- GetUserStats ----------

func TestInternalGetUserStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	h := NewInternalHandler(mockDB)

	r := gin.New()
	r.GET("/internal/stats/user/:user_id", h.GetUserStats)

	stats := &domain.UserStats{TotalVideos: 5}
	mockDB.On("GetUserStats", "u1").Return(stats, nil)

	req, _ := http.NewRequest("GET", "/internal/stats/user/u1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInternalGetUserStats_DBError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	h := NewInternalHandler(mockDB)

	r := gin.New()
	r.GET("/internal/stats/user/:user_id", h.GetUserStats)

	mockDB.On("GetUserStats", "u1").Return(nil, errors.New("db error"))

	req, _ := http.NewRequest("GET", "/internal/stats/user/u1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestInternalGetUserStats_EmptyUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewInternalHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/stats", nil)
	// c.Params is empty, so c.Param("user_id") returns ""
	h.GetUserStats(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------- GetSystemStats ----------

func TestInternalGetSystemStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	h := NewInternalHandler(mockDB)

	r := gin.New()
	r.GET("/internal/stats/system", h.GetSystemStats)

	stats := &domain.SystemStats{TotalUsers: 3}
	mockDB.On("GetSystemStats").Return(stats, nil)

	req, _ := http.NewRequest("GET", "/internal/stats/system", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInternalGetSystemStats_DBError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	h := NewInternalHandler(mockDB)

	r := gin.New()
	r.GET("/internal/stats/system", h.GetSystemStats)

	mockDB.On("GetSystemStats").Return(nil, errors.New("db error"))

	req, _ := http.NewRequest("GET", "/internal/stats/system", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
