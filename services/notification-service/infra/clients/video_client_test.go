package clients

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"notification-service/domain"

	"github.com/stretchr/testify/assert"
)

func TestVideoServiceClient_GetVideoByID_Success(t *testing.T) {
	expected := domain.Video{ID: "v1", OriginalName: "test.mp4", Status: "completed"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/internal/videos/v1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	video, err := c.GetVideoByID("v1")

	assert.NoError(t, err)
	assert.NotNil(t, video)
	assert.Equal(t, expected.ID, video.ID)
	assert.Equal(t, expected.OriginalName, video.OriginalName)
}

func TestVideoServiceClient_GetVideoByID_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	video, err := c.GetVideoByID("missing")

	assert.Error(t, err)
	assert.Nil(t, video)
	assert.Contains(t, err.Error(), "404")
}

func TestVideoServiceClient_GetVideoByID_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	video, err := c.GetVideoByID("v1")

	assert.Error(t, err)
	assert.Nil(t, video)
	assert.Contains(t, err.Error(), "500")
}

func TestVideoServiceClient_GetVideoByID_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	video, err := c.GetVideoByID("v1")

	assert.Error(t, err)
	assert.Nil(t, video)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestVideoServiceClient_GetVideoByID_ConnectionError(t *testing.T) {
	c := NewVideoServiceClient("http://127.0.0.1:1")
	video, err := c.GetVideoByID("v1")

	assert.Error(t, err)
	assert.Nil(t, video)
	assert.Contains(t, err.Error(), "failed to get video by ID")
}

func TestNewVideoServiceClient(t *testing.T) {
	c := NewVideoServiceClient("http://localhost:8081")
	assert.NotNil(t, c)
	assert.Equal(t, "http://localhost:8081", c.baseURL)
}
