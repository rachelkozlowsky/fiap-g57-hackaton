package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"auth-service/domain"
	"auth-service/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

// setupAuthRouter creates a router with auth routes (no auth middleware so we can test freely)
func setupAuthRouter() (*gin.Engine, *MockDatabase, *MockRedisClient) {
	gin.SetMode(gin.TestMode)
	mockDB := new(MockDatabase)
	mockRedis := new(MockRedisClient)
	authService := service.NewAuthService(mockDB, mockRedis)
	router := gin.New()
	authHandler := NewAuthHandler(authService)
	api := router.Group("/api/v1/auth")
	api.POST("/register", authHandler.Register)
	api.POST("/login", authHandler.Login)
	api.POST("/refresh", authHandler.RefreshToken)
	api.POST("/logout", authHandler.Logout)
	api.GET("/me", func(c *gin.Context) {
		c.Set("user_id", c.GetHeader("X-User-ID"))
		authHandler.GetCurrentUser(c)
	})
	return router, mockDB, mockRedis
}

// ── Register ────────────────────────────────────────────────────────────────

func TestRegister_InternalServerError(t *testing.T) {
	router, mockDB, mockRedis := setupAuthRouter()

	mockDB.On("GetUserByEmail", "fail@example.com").Return(nil, errors.New("not found"))
	mockDB.On("CreateUser", mock.Anything).Return(nil)
	mockDB.On("CreateSession", mock.Anything).Return(errors.New("session db error"))

	reqBody := RegisterRequest{Email: "fail@example.com", Password: "password123", Name: "Fail User"}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	mockDB.AssertExpectations(t)
	mockRedis.AssertExpectations(t)
}

// ── Login ────────────────────────────────────────────────────────────────────

func TestLogin_InternalServerError(t *testing.T) {
	// Login returns 500 when the service returns an error that is not
	// ErrInvalidCredentials or ErrAccountDisabled (e.g. CreateSession fails).
	router, mockDB, mockRedis := setupAuthRouter()

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &domain.User{
		ID:           "user123",
		Email:        "err@example.com",
		PasswordHash: string(hashedPassword),
		IsActive:     true,
		Role:         "user",
	}

	mockDB.On("GetUserByEmail", "err@example.com").Return(user, nil)
	mockDB.On("CreateSession", mock.Anything).Return(errors.New("unexpected db error"))

	reqBody := LoginRequest{Email: "err@example.com", Password: "password123"}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	mockDB.AssertExpectations(t)
	mockRedis.AssertExpectations(t)
}

func TestLogin_InvalidBody(t *testing.T) {
	router, _, _ := setupAuthRouter()

	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer([]byte("bad json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── RefreshToken ─────────────────────────────────────────────────────────────

func TestRefreshToken_BadRequest(t *testing.T) {
	router, _, _ := setupAuthRouter()

	req, _ := http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRefreshToken_EmptyBody(t *testing.T) {
	router, _, _ := setupAuthRouter()

	req, _ := http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer([]byte("")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── Logout ───────────────────────────────────────────────────────────────────

func TestLogout_BadRequest(t *testing.T) {
	router, _, _ := setupAuthRouter()

	req, _ := http.NewRequest("POST", "/api/v1/auth/logout", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogout_InternalServerError(t *testing.T) {
	router, mockDB, mockRedis := setupAuthRouter()

	mockDB.On("DeleteSessionByRefreshToken", mock.Anything).Return(errors.New("db delete error"))

	reqBody := RefreshTokenRequest{RefreshToken: "some-valid-looking-token"}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/auth/logout", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, "Failed to logout", result["error"])

	mockDB.AssertExpectations(t)
	mockRedis.AssertExpectations(t)
}

// ── GetCurrentUser ────────────────────────────────────────────────────────────

func TestGetCurrentUser_NotFound(t *testing.T) {
	router, mockDB, mockRedis := setupAuthRouter()

	mockRedis.On("GetUser", "missing_user").Return(nil, errors.New("cache miss"))
	mockDB.On("GetUserByID", "missing_user").Return(nil, errors.New("not found"))

	req, _ := http.NewRequest("GET", "/api/v1/auth/me", nil)
	req.Header.Set("X-User-ID", "missing_user")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, domain.ErrUserNotFound.Error(), result["error"])

	mockDB.AssertExpectations(t)
	mockRedis.AssertExpectations(t)
}

// ── UpdateUser (handlers_user.go) ────────────────────────────────────────────

func TestUpdateUser_InvalidBody(t *testing.T) {
	mockDB := new(MockDatabase)
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)

	userHandler := NewUserHandler(mockDB)
	router.PUT("/users/:id", func(ctx *gin.Context) {
		ctx.Set("user_id", "user123")
		ctx.Set("role", "user")
		userHandler.UpdateUser(ctx)
	})

	c.Request, _ = http.NewRequest("PUT", "/users/user123", bytes.NewBuffer([]byte("not json")))
	c.Request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
