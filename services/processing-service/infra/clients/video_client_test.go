package clients

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"processing-service/domain"

	"github.com/stretchr/testify/assert"
)

func TestNewVideoServiceClient(t *testing.T) {
	c := NewVideoServiceClient("http://localhost:8080")
	assert.NotNil(t, c)
	assert.Equal(t, "http://localhost:8080", c.baseURL)
}

// ─── GetVideoByID ─────────────────────────────────────────────────────────────

func TestGetVideoByID_Success(t *testing.T) {
	expected := domain.Video{ID: "v1", OriginalName: "test.mp4", Status: "queued"}
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

func TestGetVideoByID_NotFound(t *testing.T) {
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

func TestGetVideoByID_InvalidJSON(t *testing.T) {
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

func TestGetVideoByID_ConnectionError(t *testing.T) {
	c := NewVideoServiceClient("http://127.0.0.1:1")
	video, err := c.GetVideoByID("v1")

	assert.Error(t, err)
	assert.Nil(t, video)
	assert.Contains(t, err.Error(), "failed to get video by ID")
}

// ─── UpdateVideoStatus ────────────────────────────────────────────────────────

func TestUpdateVideoStatus_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/internal/videos/v1/status", r.URL.Path)

		var body UpdateStatusRequest
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "processing", body.Status)

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	err := c.UpdateVideoStatus("v1", "processing", "")

	assert.NoError(t, err)
}

func TestUpdateVideoStatus_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	err := c.UpdateVideoStatus("v1", "processing", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestUpdateVideoStatus_ConnectionError(t *testing.T) {
	c := NewVideoServiceClient("http://127.0.0.1:1")
	err := c.UpdateVideoStatus("v1", "processing", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update video status")
}

// ─── CompleteVideo ────────────────────────────────────────────────────────────

func TestCompleteVideo_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/internal/videos/v1/complete", r.URL.Path)

		var body CompleteVideoRequest
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "processed/frames.zip", body.ZipPath)
		assert.Equal(t, int64(1024), body.ZipSizeBytes)
		assert.Equal(t, 10, body.FrameCount)

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	err := c.CompleteVideo("v1", "processed/frames.zip", 1024, 10)

	assert.NoError(t, err)
}

func TestCompleteVideo_Created(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	err := c.CompleteVideo("v1", "zip/path", 2048, 5)

	assert.NoError(t, err)
}

func TestCompleteVideo_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	err := c.CompleteVideo("v1", "zip/path", 1024, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

func TestCompleteVideo_ConnectionError(t *testing.T) {
	c := NewVideoServiceClient("http://127.0.0.1:1")
	err := c.CompleteVideo("v1", "zip/path", 1024, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to complete video")
}

// ─── FailVideo ────────────────────────────────────────────────────────────────

func TestFailVideo_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/internal/videos/v1/fail", r.URL.Path)

		var body FailVideoRequest
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "ffmpeg crashed", body.ErrorMessage)

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	err := c.FailVideo("v1", "ffmpeg crashed")

	assert.NoError(t, err)
}

func TestFailVideo_Created(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	err := c.FailVideo("v1", "some error")

	assert.NoError(t, err)
}

func TestFailVideo_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewVideoServiceClient(srv.URL)
	err := c.FailVideo("v1", "err")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestFailVideo_ConnectionError(t *testing.T) {
	c := NewVideoServiceClient("http://127.0.0.1:1")
	err := c.FailVideo("v1", "err")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fail video")
}
