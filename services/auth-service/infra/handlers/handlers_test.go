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
	"auth-service/security"
	"auth-service/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
    "golang.org/x/crypto/bcrypt"
)

type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) CreateUser(user *domain.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockDatabase) GetUserByEmail(email string) (*domain.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockDatabase) GetUserByID(id string) (*domain.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockDatabase) UpdateUser(user *domain.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockDatabase) CreateSession(session *domain.Session) error {
	args := m.Called(session)
	return args.Error(0)
}

func (m *MockDatabase) GetSessionByRefreshToken(token string) (*domain.Session, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Session), args.Error(1)
}

func (m *MockDatabase) CreateAuditLog(log *domain.AuditLog) error {
	args := m.Called(log)
	return args.Error(0)
}

func (m *MockDatabase) UpdateSession(session *domain.Session) error {
	args := m.Called(session)
	return args.Error(0)
}

func (m *MockDatabase) DeleteSessionByRefreshToken(token string) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *MockDatabase) ListUsers() ([]domain.User, error) {
	args := m.Called()
	return args.Get(0).([]domain.User), args.Error(1)
}

func (m *MockDatabase) DeleteUser(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockDatabase) Ping() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDatabase) Close() error {
	args := m.Called()
	return args.Error(0)
}

type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) SetUser(id string, user *domain.User, ttl time.Duration) error {
	args := m.Called(id, user, ttl)
	return args.Error(0)
}

func (m *MockRedisClient) GetUser(id string) (*domain.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockRedisClient) DeleteUser(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockRedisClient) Ping() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockRedisClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func setupTestRouter() (*gin.Engine, *MockDatabase, *MockRedisClient) {
	gin.SetMode(gin.TestMode)
	
	mockDB := new(MockDatabase)
	mockRedis := new(MockRedisClient)
	
	authService := service.NewAuthService(mockDB, mockRedis)
	
	router := gin.New()
	authHandler := NewAuthHandler(authService)
	
	api := router.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/logout", authHandler.Logout)
			auth.GET("/me", authHandler.GetCurrentUser)
		}
	}
	
	return router, mockDB, mockRedis
}

func TestRegister_Success(t *testing.T) {
	router, mockDB, mockRedis := setupTestRouter()
	
	mockDB.On("GetUserByEmail", "test@example.com").Return(nil, errors.New("not found"))
	mockDB.On("CreateUser", mock.AnythingOfType("*domain.User")).Return(nil)
	mockDB.On("CreateSession", mock.AnythingOfType("*domain.Session")).Return(nil)
	mockDB.On("CreateAuditLog", mock.AnythingOfType("*domain.AuditLog")).Return(nil)
	mockRedis.On("SetUser", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	
	reqBody := RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}
	jsonBody, _ := json.Marshal(reqBody)
	
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response AuthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.AccessToken)
	assert.NotEmpty(t, response.RefreshToken)
	assert.Equal(t, "test@example.com", response.User.Email)
	assert.Equal(t, "Test User", response.User.Name)
	
	mockDB.AssertExpectations(t)
	mockRedis.AssertExpectations(t)
}

func TestRegister_UserAlreadyExists(t *testing.T) {
	router, mockDB, _ := setupTestRouter()
	
	existingUser := &domain.User{
		ID:    "123",
		Email: "test@example.com",
		Name:  "Existing User",
	}
	
	mockDB.On("GetUserByEmail", "test@example.com").Return(existingUser, nil)
	
	reqBody := RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}
	jsonBody, _ := json.Marshal(reqBody)
	
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusConflict, w.Code)
	
	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "user already exists", response["error"])
	
	mockDB.AssertExpectations(t)
}

func TestRegister_InvalidEmail(t *testing.T) {
	router, _, _ := setupTestRouter()
	
	reqBody := RegisterRequest{
		Email:    "invalid-email",
		Password: "password123",
		Name:     "Test User",
	}
	jsonBody, _ := json.Marshal(reqBody)
	
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegister_ShortPassword(t *testing.T) {
	router, _, _ := setupTestRouter()
	
	reqBody := RegisterRequest{
		Email:    "test@example.com",
		Password: "short",
		Name:     "Test User",
	}
	jsonBody, _ := json.Marshal(reqBody)
	
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin_Success(t *testing.T) {
	router, mockDB, mockRedis := setupTestRouter()
	
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &domain.User{
		ID:           "123",
		Email:        "test@example.com",
		PasswordHash: string(hashedPassword),
		Name:         "Test User",
		Role:         "user",
		IsActive:     true,
	}
	
	mockDB.On("GetUserByEmail", "test@example.com").Return(user, nil)
	mockDB.On("CreateSession", mock.AnythingOfType("*domain.Session")).Return(nil)
	mockDB.On("UpdateUser", mock.AnythingOfType("*domain.User")).Return(nil)
	mockDB.On("CreateAuditLog", mock.AnythingOfType("*domain.AuditLog")).Return(nil)
	mockRedis.On("SetUser", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	
	reqBody := LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	jsonBody, _ := json.Marshal(reqBody)
	
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response AuthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.AccessToken)
	assert.NotEmpty(t, response.RefreshToken)
	
	mockDB.AssertExpectations(t)
	mockRedis.AssertExpectations(t)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	router, mockDB, _ := setupTestRouter()
	
	mockDB.On("GetUserByEmail", "test@example.com").Return(nil, errors.New("not found"))
	
	reqBody := LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}
	jsonBody, _ := json.Marshal(reqBody)
	
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	
	mockDB.AssertExpectations(t)
}

func TestLogin_InactiveUser(t *testing.T) {
	router, mockDB, _ := setupTestRouter()
	
	user := &domain.User{
		ID:       "123",
		Email:    "test@example.com",
		Name:     "Test User",
		IsActive: false,
	}
	
	mockDB.On("GetUserByEmail", "test@example.com").Return(user, nil)
	
	reqBody := LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	jsonBody, _ := json.Marshal(reqBody)
	
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusForbidden, w.Code)
	
	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "account is disabled", response["error"])
	
	mockDB.AssertExpectations(t)
}

func TestRefreshToken_Success(t *testing.T) {
	router, mockDB, _ := setupTestRouter()
	
	user := &domain.User{
		ID:    "123",
		Email: "test@example.com",
		Name:  "Test User",
		Role:  "user",
	}
	
	refreshToken, _ := security.GenerateRefreshToken(user)
	
	session := &domain.Session{
		ID:           "session123",
		UserID:       "123",
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
	}
	
	mockDB.On("GetSessionByRefreshToken", refreshToken).Return(session, nil)
	mockDB.On("GetUserByID", "123").Return(user, nil)
	mockDB.On("UpdateSession", mock.AnythingOfType("*domain.Session")).Return(nil)
	
	reqBody := RefreshTokenRequest{
		RefreshToken: refreshToken,
	}
	jsonBody, _ := json.Marshal(reqBody)
	
	req, _ := http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.NotEmpty(t, response["access_token"])
	
	mockDB.AssertExpectations(t)
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	router, _, _ := setupTestRouter()
	
	reqBody := RefreshTokenRequest{
		RefreshToken: "invalid-token",
	}
	jsonBody, _ := json.Marshal(reqBody)
	
	req, _ := http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRefreshToken_ExpiredSession(t *testing.T) {
	router, mockDB, _ := setupTestRouter()
	
	user := &domain.User{
		ID:    "123",
		Email: "test@example.com",
	}
	
	refreshToken, _ := security.GenerateRefreshToken(user)
	
	session := &domain.Session{
		ID:           "session123",
		UserID:       "123",
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(-1 * time.Hour), // Expired
	}
	
	mockDB.On("GetSessionByRefreshToken", refreshToken).Return(session, nil)
	
	reqBody := RefreshTokenRequest{
		RefreshToken: refreshToken,
	}
	jsonBody, _ := json.Marshal(reqBody)
	
	req, _ := http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	
	mockDB.AssertExpectations(t)
}

func TestLogout_Success(t *testing.T) {
	router, mockDB, mockRedis := setupTestRouter()
	
	user := &domain.User{
		ID:    "123",
		Email: "test@example.com",
	}
	
	refreshToken, _ := security.GenerateRefreshToken(user)
	
	mockDB.On("DeleteSessionByRefreshToken", refreshToken).Return(nil)
	mockDB.On("CreateAuditLog", mock.AnythingOfType("*domain.AuditLog")).Return(nil)
	mockRedis.On("DeleteUser", mock.Anything).Return(nil)
	
	reqBody := RefreshTokenRequest{
		RefreshToken: refreshToken,
	}
	jsonBody, _ := json.Marshal(reqBody)
	
	req, _ := http.NewRequest("POST", "/api/v1/auth/logout", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "Logged out successfully", response["message"])
	
	mockDB.AssertExpectations(t)
	mockRedis.AssertExpectations(t)
}

func TestGetCurrentUser_Success(t *testing.T) {
	router, mockDB, mockRedis := setupTestRouter()
	
	user := &domain.User{
		ID:    "123",
		Email: "test@example.com",
		Name:  "Test User",
		Role:  "user",
	}
	
	mockRedis.On("GetUser", "123").Return(nil, errors.New("not found"))
	mockDB.On("GetUserByID", "123").Return(user, nil)
	mockRedis.On("SetUser", "123", user, 24*time.Hour).Return(nil)
	
	req, _ := http.NewRequest("GET", "/api/v1/auth/me", nil)
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("user_id", "123")
	
	authService := service.NewAuthService(mockDB, mockRedis)
	authHandler := NewAuthHandler(authService)
	authHandler.GetCurrentUser(c)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response UserDTO
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "test@example.com", response.Email)
	
	mockDB.AssertExpectations(t)
	mockRedis.AssertExpectations(t)
}

func TestGetCurrentUser_FromCache(t *testing.T) {
	router, _, mockRedis := setupTestRouter()
	
	user := &domain.User{
		ID:    "123",
		Email: "test@example.com",
		Name:  "Test User",
		Role:  "user",
	}
	
	mockRedis.On("GetUser", "123").Return(user, nil)
	
	req, _ := http.NewRequest("GET", "/api/v1/auth/me", nil)
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("user_id", "123")
	
	authService := service.NewAuthService(nil, mockRedis)
	authHandler := NewAuthHandler(authService)
	authHandler.GetCurrentUser(c)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	mockRedis.AssertExpectations(t)
}

func TestGenerateAndValidateAccessToken(t *testing.T) {
	user := &domain.User{
		ID:    "123",
		Email: "test@example.com",
		Role:  "user",
	}
	
	token, err := security.GenerateAccessToken(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	
	claims, err := security.ValidateAccessToken(token)
	assert.NoError(t, err)
	assert.Equal(t, "123", claims.UserID)
	assert.Equal(t, "test@example.com", claims.Email)
}

func TestGenerateAndValidateRefreshToken(t *testing.T) {
	user := &domain.User{
		ID:    "123",
		Email: "test@example.com",
	}
	
	token, err := security.GenerateRefreshToken(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	
	claims, err := security.ValidateRefreshToken(token)
	assert.NoError(t, err)
	assert.Equal(t, "123", claims.UserID)
}

func TestValidateAccessToken_Invalid(t *testing.T) {
	_, err := security.ValidateAccessToken("invalid-token")
	assert.Error(t, err)
}

func TestValidateRefreshToken_Invalid(t *testing.T) {
	_, err := security.ValidateRefreshToken("invalid-token")
	assert.Error(t, err)
}

func TestUserToDTO(t *testing.T) {
	user := &domain.User{
		ID:        "123",
		Email:     "test@example.com",
		Name:      "Test User",
		Role:      "user",
		CreatedAt: time.Now(),
	}
	
	dto := UserToDTO(user)
	assert.Equal(t, user.ID, dto.ID)
	assert.Equal(t, user.Email, dto.Email)
	assert.Equal(t, user.Name, dto.Name)
	assert.Equal(t, user.Role, dto.Role)
}

func TestTimePtr(t *testing.T) {
	now := time.Now()
	ptr := TimePtr(now)
	assert.NotNil(t, ptr)
	assert.Equal(t, now, *ptr)
}

func BenchmarkRegister(b *testing.B) {
	router, mockDB, mockRedis := setupTestRouter()
	
	mockDB.On("GetUserByEmail", mock.Anything).Return(nil, errors.New("not found"))
	mockDB.On("CreateUser", mock.Anything).Return(nil)
	mockDB.On("CreateSession", mock.Anything).Return(nil)
	mockDB.On("CreateAuditLog", mock.Anything).Return(nil)
	mockRedis.On("SetUser", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	
	reqBody := RegisterRequest{
		Email:    "bench@example.com",
		Password: "password123",
		Name:     "Bench User",
	}
	jsonBody, _ := json.Marshal(reqBody)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkLogin(b *testing.B) {
	router, mockDB, mockRedis := setupTestRouter()
	
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &domain.User{
		ID:           "123",
		Email:        "test@example.com",
		PasswordHash: string(hashedPassword),
		IsActive:     true,
	}
	
	mockDB.On("GetUserByEmail", mock.Anything).Return(user, nil)
	mockDB.On("CreateSession", mock.Anything).Return(nil)
	mockDB.On("UpdateUser", mock.Anything).Return(nil)
	mockDB.On("CreateAuditLog", mock.Anything).Return(nil)
	mockRedis.On("SetUser", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	
	reqBody := LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	jsonBody, _ := json.Marshal(reqBody)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
