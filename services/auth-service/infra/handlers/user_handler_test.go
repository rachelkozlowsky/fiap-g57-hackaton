package handlers

import (
	"auth-service/domain"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupUserHandlerRouter(mockDB *MockDatabase) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	userHandler := NewUserHandler(mockDB)

	api := router.Group("/api/v1")
	{
		users := api.Group("/users")
		{
			users.GET("", userHandler.ListUsers)
			users.GET("/:id", userHandler.GetUser)
			users.PUT("/:id", userHandler.UpdateUser)
			users.DELETE("/:id", userHandler.DeleteUser)
		}
	}
	return router
}

func TestNewUserHandler(t *testing.T) {
	mockDB := new(MockDatabase)
	handler := NewUserHandler(mockDB)
	assert.NotNil(t, handler)
}

func TestListUsers_Success(t *testing.T) {
	mockDB := new(MockDatabase)
	router := setupUserHandlerRouter(mockDB)

	users := []domain.User{
		{ID: "1", Email: "a@example.com", Name: "Alice", Role: "user", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "2", Email: "b@example.com", Name: "Bob", Role: "admin", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	mockDB.On("ListUsers").Return(users, nil)

	req, _ := http.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []UserDTO
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "a@example.com", result[0].Email)
	assert.Equal(t, "b@example.com", result[1].Email)

	mockDB.AssertExpectations(t)
}

func TestListUsers_Error(t *testing.T) {
	mockDB := new(MockDatabase)
	router := setupUserHandlerRouter(mockDB)

	mockDB.On("ListUsers").Return([]domain.User{}, errors.New("db error"))

	req, _ := http.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, "Failed to fetch users", result["error"])

	mockDB.AssertExpectations(t)
}

func TestGetUser_Success(t *testing.T) {
	mockDB := new(MockDatabase)
	router := setupUserHandlerRouter(mockDB)

	user := &domain.User{
		ID:        "abc123",
		Email:     "user@example.com",
		Name:      "Test User",
		Role:      "user",
		CreatedAt: time.Now(),
	}
	mockDB.On("GetUserByID", "abc123").Return(user, nil)

	req, _ := http.NewRequest("GET", "/api/v1/users/abc123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result UserDTO
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Equal(t, "abc123", result.ID)
	assert.Equal(t, "user@example.com", result.Email)

	mockDB.AssertExpectations(t)
}

func TestGetUser_NotFound(t *testing.T) {
	mockDB := new(MockDatabase)
	router := setupUserHandlerRouter(mockDB)

	mockDB.On("GetUserByID", "notexist").Return(nil, errors.New("not found"))

	req, _ := http.NewRequest("GET", "/api/v1/users/notexist", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, "User not found", result["error"])

	mockDB.AssertExpectations(t)
}

func TestUpdateUser_Success_OwnProfile(t *testing.T) {
	mockDB := new(MockDatabase)
	gin.SetMode(gin.TestMode)

	user := &domain.User{
		ID:        "user123",
		Email:     "user@example.com",
		Name:      "Old Name",
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mockDB.On("GetUserByID", "user123").Return(user, nil)
	mockDB.On("UpdateUser", mock.AnythingOfType("*domain.User")).Return(nil)

	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)

	userHandler := NewUserHandler(mockDB)
	router.PUT("/users/:id", func(ctx *gin.Context) {
		ctx.Set("user_id", "user123")
		ctx.Set("role", "user")
		userHandler.UpdateUser(ctx)
	})

	body, _ := json.Marshal(map[string]string{"name": "New Name"})
	c.Request, _ = http.NewRequest("PUT", "/users/user123", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusOK, w.Code)

	var result UserDTO
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, "New Name", result.Name)

	mockDB.AssertExpectations(t)
}

func TestUpdateUser_Success_AdminRole(t *testing.T) {
	mockDB := new(MockDatabase)
	gin.SetMode(gin.TestMode)

	user := &domain.User{
		ID:        "user123",
		Email:     "user@example.com",
		Name:      "Old Name",
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mockDB.On("GetUserByID", "user123").Return(user, nil)
	mockDB.On("UpdateUser", mock.AnythingOfType("*domain.User")).Return(nil)

	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)

	userHandler := NewUserHandler(mockDB)
	router.PUT("/users/:id", func(ctx *gin.Context) {
		ctx.Set("user_id", "admin999")
		ctx.Set("role", "admin")
		userHandler.UpdateUser(ctx)
	})

	body, _ := json.Marshal(map[string]string{"name": "Admin Updated"})
	c.Request, _ = http.NewRequest("PUT", "/users/user123", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusOK, w.Code)

	mockDB.AssertExpectations(t)
}

func TestUpdateUser_Forbidden(t *testing.T) {
	mockDB := new(MockDatabase)
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)

	userHandler := NewUserHandler(mockDB)
	router.PUT("/users/:id", func(ctx *gin.Context) {
		ctx.Set("user_id", "other_user")
		ctx.Set("role", "user")
		userHandler.UpdateUser(ctx)
	})

	body, _ := json.Marshal(map[string]string{"name": "Hacked"})
	c.Request, _ = http.NewRequest("PUT", "/users/user123", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, "Access denied", result["error"])
}

func TestUpdateUser_UserNotFound(t *testing.T) {
	mockDB := new(MockDatabase)
	gin.SetMode(gin.TestMode)

	mockDB.On("GetUserByID", "user123").Return(nil, errors.New("not found"))

	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)

	userHandler := NewUserHandler(mockDB)
	router.PUT("/users/:id", func(ctx *gin.Context) {
		ctx.Set("user_id", "user123")
		ctx.Set("role", "user")
		userHandler.UpdateUser(ctx)
	})

	body, _ := json.Marshal(map[string]string{"name": "New Name"})
	c.Request, _ = http.NewRequest("PUT", "/users/user123", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusNotFound, w.Code)

	mockDB.AssertExpectations(t)
}

func TestUpdateUser_UpdateError(t *testing.T) {
	mockDB := new(MockDatabase)
	gin.SetMode(gin.TestMode)

	user := &domain.User{
		ID:        "user123",
		Email:     "user@example.com",
		Name:      "Old Name",
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mockDB.On("GetUserByID", "user123").Return(user, nil)
	mockDB.On("UpdateUser", mock.AnythingOfType("*domain.User")).Return(errors.New("db error"))

	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)

	userHandler := NewUserHandler(mockDB)
	router.PUT("/users/:id", func(ctx *gin.Context) {
		ctx.Set("user_id", "user123")
		ctx.Set("role", "user")
		userHandler.UpdateUser(ctx)
	})

	body, _ := json.Marshal(map[string]string{"name": "New Name"})
	c.Request, _ = http.NewRequest("PUT", "/users/user123", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	mockDB.AssertExpectations(t)
}

func TestDeleteUser_Success(t *testing.T) {
	mockDB := new(MockDatabase)
	router := setupUserHandlerRouter(mockDB)

	mockDB.On("DeleteUser", "user123").Return(nil)

	req, _ := http.NewRequest("DELETE", "/api/v1/users/user123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, "User deleted", result["message"])

	mockDB.AssertExpectations(t)
}

func TestDeleteUser_Error(t *testing.T) {
	mockDB := new(MockDatabase)
	router := setupUserHandlerRouter(mockDB)

	mockDB.On("DeleteUser", "user123").Return(errors.New("db error"))

	req, _ := http.NewRequest("DELETE", "/api/v1/users/user123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, "Failed to delete user", result["error"])

	mockDB.AssertExpectations(t)
}
