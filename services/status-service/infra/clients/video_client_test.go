package clients

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"status-service/domain"

	"github.com/stretchr/testify/assert"
)

func TestGetVideosByUserID_Success(t *testing.T) {
	videos := []*domain.Video{
		{ID: "v1", UserID: "u1", Status: "completed"},
		{ID: "v2", UserID: "u1", Status: "processing"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/internal/videos", r.URL.Path)
		assert.Equal(t, "u1", r.URL.Query().Get("user_id"))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(videos)
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	result, err := c.GetVideosByUserID("u1", "")
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestGetVideosByUserID_FilterByStatus(t *testing.T) {
	videos := []*domain.Video{
		{ID: "v1", UserID: "u1", Status: "completed"},
		{ID: "v2", UserID: "u1", Status: "processing"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(videos)
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	result, err := c.GetVideosByUserID("u1", "completed")
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "completed", result[0].Status)
}

func TestGetVideosByUserID_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	_, err := c.GetVideosByUserID("u1", "")
	assert.Error(t, err)
}

func TestGetVideosByUserID_ConnectionError(t *testing.T) {
	c := NewVideoServiceClient("http://127.0.0.1:0")
	_, err := c.GetVideosByUserID("u1", "")
	assert.Error(t, err)
}

func TestGetVideosByUserID_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	_, err := c.GetVideosByUserID("u1", "")
	assert.Error(t, err)
}

func TestGetVideoByID_Success(t *testing.T) {
	video := &domain.Video{ID: "v1", UserID: "u1"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/internal/videos/v1", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(video)
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	result, err := c.GetVideoByID("v1")
	assert.NoError(t, err)
	assert.Equal(t, "v1", result.ID)
}

func TestGetVideoByID_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	_, err := c.GetVideoByID("v1")
	assert.EqualError(t, err, "video not found")
}

func TestGetVideoByID_OtherError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	_, err := c.GetVideoByID("v1")
	assert.Error(t, err)
}

func TestGetVideoByID_ConnectionError(t *testing.T) {
	c := NewVideoServiceClient("http://127.0.0.1:0")
	_, err := c.GetVideoByID("v1")
	assert.Error(t, err)
}

func TestGetVideoByID_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	_, err := c.GetVideoByID("v1")
	assert.Error(t, err)
}

func TestGetUserStats_Success(t *testing.T) {
	stats := &domain.UserStats{TotalVideos: 5}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/internal/stats/user/u1", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(stats)
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	result, err := c.GetUserStats("u1")
	assert.NoError(t, err)
	assert.Equal(t, 5, result.TotalVideos)
}

func TestGetUserStats_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	_, err := c.GetUserStats("u1")
	assert.Error(t, err)
}

func TestGetUserStats_ConnectionError(t *testing.T) {
	c := NewVideoServiceClient("http://127.0.0.1:0")
	_, err := c.GetUserStats("u1")
	assert.Error(t, err)
}

func TestGetUserStats_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	_, err := c.GetUserStats("u1")
	assert.Error(t, err)
}

func TestGetSystemStats_Success(t *testing.T) {
	stats := &domain.SystemStats{TotalUsers: 3, TotalVideos: 30}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/internal/stats/system", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(stats)
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	result, err := c.GetSystemStats()
	assert.NoError(t, err)
	assert.Equal(t, 3, result.TotalUsers)
}

func TestGetSystemStats_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	_, err := c.GetSystemStats()
	assert.Error(t, err)
}

func TestGetSystemStats_ConnectionError(t *testing.T) {
	c := NewVideoServiceClient("http://127.0.0.1:0")
	_, err := c.GetSystemStats()
	assert.Error(t, err)
}

func TestGetSystemStats_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	_, err := c.GetSystemStats()
	assert.Error(t, err)
}
