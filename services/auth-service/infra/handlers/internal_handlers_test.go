package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"auth-service/domain"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupInternalRouter(mockDB *MockDatabase) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	internalHandler := NewInternalHandler(mockDB)

	internal := router.Group("/internal")
	{
		internal.GET("/users/:id", internalHandler.GetUserByID)
		internal.GET("/users/email/:email", internalHandler.GetUserByEmail)
		internal.POST("/audit-logs", internalHandler.CreateAuditLog)
	}
	return router
}

func TestNewInternalHandler(t *testing.T) {
	mockDB := new(MockDatabase)
	handler := NewInternalHandler(mockDB)
	assert.NotNil(t, handler)
}

func TestInternalGetUserByID_Success(t *testing.T) {
	mockDB := new(MockDatabase)
	router := setupInternalRouter(mockDB)

	now := time.Now()
	user := &domain.User{
		ID:            "abc123",
		Email:         "user@example.com",
		Name:          "Test User",
		Role:          "user",
		IsActive:      true,
		EmailVerified: true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	mockDB.On("GetUserByID", "abc123").Return(user, nil)

	req, _ := http.NewRequest("GET", "/internal/users/abc123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result UserResponse
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Equal(t, "abc123", result.ID)
	assert.Equal(t, "user@example.com", result.Email)
	assert.Equal(t, "Test User", result.Name)
	assert.Equal(t, "user", result.Role)
	assert.True(t, result.IsActive)
	assert.True(t, result.EmailVerified)

	mockDB.AssertExpectations(t)
}

func TestInternalGetUserByID_NotFound(t *testing.T) {
	mockDB := new(MockDatabase)
	router := setupInternalRouter(mockDB)

	mockDB.On("GetUserByID", "notexist").Return(nil, errors.New("not found"))

	req, _ := http.NewRequest("GET", "/internal/users/notexist", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, "User not found", result["error"])

	mockDB.AssertExpectations(t)
}

func TestInternalGetUserByEmail_Success(t *testing.T) {
	mockDB := new(MockDatabase)
	router := setupInternalRouter(mockDB)

	now := time.Now()
	user := &domain.User{
		ID:            "abc123",
		Email:         "user@example.com",
		Name:          "Test User",
		Role:          "user",
		IsActive:      true,
		EmailVerified: false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	mockDB.On("GetUserByEmail", "user@example.com").Return(user, nil)

	req, _ := http.NewRequest("GET", "/internal/users/email/user@example.com", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result UserResponse
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Equal(t, "abc123", result.ID)
	assert.Equal(t, "user@example.com", result.Email)

	mockDB.AssertExpectations(t)
}

func TestInternalGetUserByEmail_NotFound(t *testing.T) {
	mockDB := new(MockDatabase)
	router := setupInternalRouter(mockDB)

	mockDB.On("GetUserByEmail", "ghost@example.com").Return(nil, errors.New("not found"))

	req, _ := http.NewRequest("GET", "/internal/users/email/ghost@example.com", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	mockDB.AssertExpectations(t)
}

func TestCreateAuditLog_Success(t *testing.T) {
	mockDB := new(MockDatabase)
	router := setupInternalRouter(mockDB)

	mockDB.On("CreateAuditLog", mock.AnythingOfType("*domain.AuditLog")).Return(nil)

	userID := "user123"
	entityID := "entity456"
	body := CreateAuditLogRequest{
		UserID:     &userID,
		Action:     "user.update",
		EntityType: "user",
		EntityID:   &entityID,
		IPAddress:  "127.0.0.1",
		UserAgent:  "test-agent",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/internal/audit-logs", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, "Audit log created", result["message"])

	mockDB.AssertExpectations(t)
}

func TestCreateAuditLog_InvalidBody(t *testing.T) {
	mockDB := new(MockDatabase)
	router := setupInternalRouter(mockDB)

	req, _ := http.NewRequest("POST", "/internal/audit-logs", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, "Invalid request body", result["error"])
}

func TestCreateAuditLog_DBError(t *testing.T) {
	mockDB := new(MockDatabase)
	router := setupInternalRouter(mockDB)

	mockDB.On("CreateAuditLog", mock.AnythingOfType("*domain.AuditLog")).Return(errors.New("db error"))

	body := CreateAuditLogRequest{
		Action:     "user.update",
		EntityType: "user",
		IPAddress:  "127.0.0.1",
		UserAgent:  "test-agent",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/internal/audit-logs", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, "Failed to create audit log", result["error"])

	mockDB.AssertExpectations(t)
}
