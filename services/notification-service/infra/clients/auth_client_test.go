package clients

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"notification-service/domain"

	"github.com/stretchr/testify/assert"
)

func TestAuthServiceClient_GetUserByID_Success(t *testing.T) {
	expected := domain.User{ID: "u1", Email: "user@example.com", Name: "Test User"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/internal/users/u1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	c := NewAuthServiceClient(srv.URL)
	user, err := c.GetUserByID("u1")

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, expected.ID, user.ID)
	assert.Equal(t, expected.Email, user.Email)
	assert.Equal(t, expected.Name, user.Name)
}

func TestAuthServiceClient_GetUserByID_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewAuthServiceClient(srv.URL)
	user, err := c.GetUserByID("missing")

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "404")
}

func TestAuthServiceClient_GetUserByID_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewAuthServiceClient(srv.URL)
	user, err := c.GetUserByID("u1")

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "500")
}

func TestAuthServiceClient_GetUserByID_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	c := NewAuthServiceClient(srv.URL)
	user, err := c.GetUserByID("u1")

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestAuthServiceClient_GetUserByID_ConnectionError(t *testing.T) {
	// Point to a server that is not listening
	c := NewAuthServiceClient("http://127.0.0.1:1")
	user, err := c.GetUserByID("u1")

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "failed to get user by ID")
}

func TestNewAuthServiceClient(t *testing.T) {
	c := NewAuthServiceClient("http://localhost:8080")
	assert.NotNil(t, c)
	assert.Equal(t, "http://localhost:8080", c.baseURL)
}
