package clients

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetUserByID_Success(t *testing.T) {
	user := &User{ID: "u1", Email: "test@example.com", Name: "Alice"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/internal/users/u1", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(user)
	}))
	defer srv.Close()

	c := NewAuthServiceClient(srv.URL)
	result, err := c.GetUserByID("u1")
	assert.NoError(t, err)
	assert.Equal(t, "u1", result.ID)
	assert.Equal(t, "test@example.com", result.Email)
}

func TestGetUserByID_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))
	defer srv.Close()

	c := NewAuthServiceClient(srv.URL)
	_, err := c.GetUserByID("u1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code 404")
}

func TestGetUserByID_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	c := NewAuthServiceClient(srv.URL)
	_, err := c.GetUserByID("u1")
	assert.Error(t, err)
}

func TestGetUserByID_ConnectionError(t *testing.T) {
	c := NewAuthServiceClient("http://127.0.0.1:0")
	_, err := c.GetUserByID("u1")
	assert.Error(t, err)
}
