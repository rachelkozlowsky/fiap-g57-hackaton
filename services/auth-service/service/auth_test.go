package service

import (
	"errors"
	"testing"
	"time"
	"auth-service/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

// ------------------------------------------------------------------
// Mocks
// ------------------------------------------------------------------

type MockDB struct {
	mock.Mock
}

func (m *MockDB) CreateUser(user *domain.User) error {
	return m.Called(user).Error(0)
}

func (m *MockDB) GetUserByEmail(email string) (*domain.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockDB) GetUserByID(id string) (*domain.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockDB) UpdateUser(user *domain.User) error {
	return m.Called(user).Error(0)
}

func (m *MockDB) CreateSession(session *domain.Session) error {
	return m.Called(session).Error(0)
}

func (m *MockDB) GetSessionByRefreshToken(token string) (*domain.Session, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Session), args.Error(1)
}

func (m *MockDB) UpdateSession(session *domain.Session) error {
	return m.Called(session).Error(0)
}

func (m *MockDB) DeleteSessionByRefreshToken(token string) error {
	return m.Called(token).Error(0)
}

func (m *MockDB) ListUsers() ([]domain.User, error) {
	args := m.Called()
	return args.Get(0).([]domain.User), args.Error(1)
}

func (m *MockDB) DeleteUser(id string) error {
	return m.Called(id).Error(0)
}

func (m *MockDB) CreateAuditLog(log *domain.AuditLog) error {
	return m.Called(log).Error(0)
}

func (m *MockDB) Ping() error {
	return m.Called().Error(0)
}

func (m *MockDB) Close() error {
	return m.Called().Error(0)
}

type MockRedis struct {
	mock.Mock
}

func (m *MockRedis) SetUser(id string, user *domain.User, ttl time.Duration) error {
	return m.Called(id, user, ttl).Error(0)
}

func (m *MockRedis) GetUser(id string) (*domain.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockRedis) DeleteUser(id string) error {
	return m.Called(id).Error(0)
}

func (m *MockRedis) Ping() error {
	return m.Called().Error(0)
}

func (m *MockRedis) Close() error {
	return m.Called().Error(0)
}

// ------------------------------------------------------------------
// Register
// ------------------------------------------------------------------

func TestAuthService_Register_Success(t *testing.T) {
	db := new(MockDB)
	redis := new(MockRedis)
	svc := NewAuthService(db, redis)

	db.On("GetUserByEmail", "new@example.com").Return(nil, errors.New("not found"))
	db.On("CreateUser", mock.AnythingOfType("*domain.User")).Return(nil)
	db.On("CreateSession", mock.AnythingOfType("*domain.Session")).Return(nil)
	db.On("CreateAuditLog", mock.AnythingOfType("*domain.AuditLog")).Return(nil)
	redis.On("SetUser", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	user, accessToken, refreshToken, err := svc.Register("new@example.com", "password123", "New User", "127.0.0.1", "test-agent")

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)
	assert.Equal(t, "new@example.com", user.Email)
	assert.Equal(t, "New User", user.Name)
	assert.Equal(t, "user", user.Role)
	assert.True(t, user.IsActive)

	db.AssertExpectations(t)
	redis.AssertExpectations(t)
}

func TestAuthService_Register_UserAlreadyExists(t *testing.T) {
	db := new(MockDB)
	redis := new(MockRedis)
	svc := NewAuthService(db, redis)

	existing := &domain.User{ID: "123", Email: "existing@example.com"}
	db.On("GetUserByEmail", "existing@example.com").Return(existing, nil)

	user, accessToken, refreshToken, err := svc.Register("existing@example.com", "password123", "Test", "127.0.0.1", "agent")

	assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)
	assert.Nil(t, user)
	assert.Empty(t, accessToken)
	assert.Empty(t, refreshToken)

	db.AssertExpectations(t)
}

func TestAuthService_Register_CreateUserError(t *testing.T) {
	db := new(MockDB)
	redis := new(MockRedis)
	svc := NewAuthService(db, redis)

	db.On("GetUserByEmail", "new@example.com").Return(nil, errors.New("not found"))
	db.On("CreateUser", mock.Anything).Return(errors.New("db write error"))

	user, _, _, err := svc.Register("new@example.com", "password123", "Test", "127.0.0.1", "agent")

	assert.Error(t, err)
	assert.Nil(t, user)

	db.AssertExpectations(t)
}

func TestAuthService_Register_CreateSessionError(t *testing.T) {
	db := new(MockDB)
	redis := new(MockRedis)
	svc := NewAuthService(db, redis)

	db.On("GetUserByEmail", "new@example.com").Return(nil, errors.New("not found"))
	db.On("CreateUser", mock.Anything).Return(nil)
	db.On("CreateSession", mock.Anything).Return(errors.New("session error"))

	user, _, _, err := svc.Register("new@example.com", "password123", "Test", "127.0.0.1", "agent")

	assert.Error(t, err)
	assert.Nil(t, user)

	db.AssertExpectations(t)
}

// ------------------------------------------------------------------
// Login
// ------------------------------------------------------------------

func TestAuthService_Login_Success(t *testing.T) {
	db := new(MockDB)
	redis := new(MockRedis)
	svc := NewAuthService(db, redis)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &domain.User{
		ID:           "user123",
		Email:        "user@example.com",
		PasswordHash: string(hashed),
		IsActive:     true,
		Role:         "user",
	}

	db.On("GetUserByEmail", "user@example.com").Return(user, nil)
	db.On("CreateSession", mock.AnythingOfType("*domain.Session")).Return(nil)
	db.On("UpdateUser", mock.AnythingOfType("*domain.User")).Return(nil)
	db.On("CreateAuditLog", mock.AnythingOfType("*domain.AuditLog")).Return(nil)
	redis.On("SetUser", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	resultUser, accessToken, refreshToken, err := svc.Login("user@example.com", "password123", "127.0.0.1", "agent")

	assert.NoError(t, err)
	assert.NotNil(t, resultUser)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)

	db.AssertExpectations(t)
	redis.AssertExpectations(t)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	db := new(MockDB)
	redis := new(MockRedis)
	svc := NewAuthService(db, redis)

	db.On("GetUserByEmail", "ghost@example.com").Return(nil, errors.New("not found"))

	user, accessToken, refreshToken, err := svc.Login("ghost@example.com", "password123", "127.0.0.1", "agent")

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	assert.Nil(t, user)
	assert.Empty(t, accessToken)
	assert.Empty(t, refreshToken)

	db.AssertExpectations(t)
}

func TestAuthService_Login_AccountDisabled(t *testing.T) {
	db := new(MockDB)
	redis := new(MockRedis)
	svc := NewAuthService(db, redis)

	user := &domain.User{
		ID:       "user123",
		Email:    "user@example.com",
		IsActive: false,
	}
	db.On("GetUserByEmail", "user@example.com").Return(user, nil)

	_, _, _, err := svc.Login("user@example.com", "password123", "127.0.0.1", "agent")

	assert.ErrorIs(t, err, domain.ErrAccountDisabled)

	db.AssertExpectations(t)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	db := new(MockDB)
	redis := new(MockRedis)
	svc := NewAuthService(db, redis)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("correct_password"), bcrypt.DefaultCost)
	user := &domain.User{
		ID:           "user123",
		Email:        "user@example.com",
		PasswordHash: string(hashed),
		IsActive:     true,
	}
	db.On("GetUserByEmail", "user@example.com").Return(user, nil)

	_, _, _, err := svc.Login("user@example.com", "wrong_password", "127.0.0.1", "agent")

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)

	db.AssertExpectations(t)
}

func TestAuthService_Login_CreateSessionError(t *testing.T) {
	db := new(MockDB)
	redis := new(MockRedis)
	svc := NewAuthService(db, redis)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &domain.User{
		ID:           "user123",
		Email:        "user@example.com",
		PasswordHash: string(hashed),
		IsActive:     true,
	}
	db.On("GetUserByEmail", "user@example.com").Return(user, nil)
	db.On("CreateSession", mock.Anything).Return(errors.New("session error"))

	_, _, _, err := svc.Login("user@example.com", "password123", "127.0.0.1", "agent")

	assert.Error(t, err)

	db.AssertExpectations(t)
}

// ------------------------------------------------------------------
// RefreshToken
// ------------------------------------------------------------------

func TestAuthService_RefreshToken_Success(t *testing.T) {
	db := new(MockDB)
	redis := new(MockRedis)
	svc := NewAuthService(db, redis)

	user := &domain.User{ID: "user123", Email: "user@example.com", Role: "user"}

	// Use a second AuthService instance to generate a valid refresh token via Register
	db2 := new(MockDB)
	redis2 := new(MockRedis)
	svc2 := NewAuthService(db2, redis2)

	db2.On("GetUserByEmail", "user@example.com").Return(nil, errors.New("not found"))
	db2.On("CreateUser", mock.Anything).Return(nil)
	var capturedRefreshToken string
	db2.On("CreateSession", mock.MatchedBy(func(s *domain.Session) bool {
		capturedRefreshToken = s.RefreshToken
		return true
	})).Return(nil)
	db2.On("CreateAuditLog", mock.Anything).Return(nil)
	redis2.On("SetUser", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, _, refreshToken, errReg := svc2.Register("user@example.com", "password123", "Test", "127.0.0.1", "agent")
	assert.NoError(t, errReg)
	_ = capturedRefreshToken

	// Now use the real svc
	session := &domain.Session{
		ID:           "sess1",
		UserID:       "user123",
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
	}
	db.On("GetSessionByRefreshToken", refreshToken).Return(session, nil)
	db.On("GetUserByID", mock.Anything).Return(user, nil)
	db.On("UpdateSession", mock.Anything).Return(nil)

	accessToken, err := svc.RefreshToken(refreshToken)

	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken)

	db.AssertExpectations(t)
}

func TestAuthService_RefreshToken_InvalidJWT(t *testing.T) {
	db := new(MockDB)
	redis := new(MockRedis)
	svc := NewAuthService(db, redis)

	_, err := svc.RefreshToken("totally.invalid.token")

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthService_RefreshToken_SessionNotFound(t *testing.T) {
	db := new(MockDB)
	redis := new(MockRedis)
	svc := NewAuthService(db, redis)

	// Generate a valid token first
	db2 := new(MockDB)
	redis2 := new(MockRedis)
	svc2 := NewAuthService(db2, redis2)

	db2.On("GetUserByEmail", mock.Anything).Return(nil, errors.New("not found"))
	db2.On("CreateUser", mock.Anything).Return(nil)
	db2.On("CreateSession", mock.Anything).Return(nil)
	db2.On("CreateAuditLog", mock.Anything).Return(nil)
	redis2.On("SetUser", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, _, refreshToken, _ := svc2.Register("user@example.com", "password123", "T", "ip", "ua")

	db.On("GetSessionByRefreshToken", refreshToken).Return(nil, errors.New("not found"))

	_, err := svc.RefreshToken(refreshToken)

	assert.ErrorIs(t, err, domain.ErrSessionNotFound)
	db.AssertExpectations(t)
}

func TestAuthService_RefreshToken_ExpiredSession(t *testing.T) {
	db := new(MockDB)
	redis := new(MockRedis)
	svc := NewAuthService(db, redis)

	db2 := new(MockDB)
	redis2 := new(MockRedis)
	svc2 := NewAuthService(db2, redis2)

	db2.On("GetUserByEmail", mock.Anything).Return(nil, errors.New("not found"))
	db2.On("CreateUser", mock.Anything).Return(nil)
	db2.On("CreateSession", mock.Anything).Return(nil)
	db2.On("CreateAuditLog", mock.Anything).Return(nil)
	redis2.On("SetUser", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, _, refreshToken, _ := svc2.Register("user@example.com", "password123", "T", "ip", "ua")

	session := &domain.Session{
		ID:           "sess1",
		UserID:       "user123",
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(-1 * time.Hour),
	}
	db.On("GetSessionByRefreshToken", refreshToken).Return(session, nil)

	_, err := svc.RefreshToken(refreshToken)

	assert.ErrorIs(t, err, domain.ErrSessionExpired)
	db.AssertExpectations(t)
}

func TestAuthService_RefreshToken_UserNotFound(t *testing.T) {
	db := new(MockDB)
	redis := new(MockRedis)
	svc := NewAuthService(db, redis)

	db2 := new(MockDB)
	redis2 := new(MockRedis)
	svc2 := NewAuthService(db2, redis2)

	db2.On("GetUserByEmail", mock.Anything).Return(nil, errors.New("not found"))
	db2.On("CreateUser", mock.Anything).Return(nil)
	db2.On("CreateSession", mock.Anything).Return(nil)
	db2.On("CreateAuditLog", mock.Anything).Return(nil)
	redis2.On("SetUser", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, _, refreshToken, _ := svc2.Register("user@example.com", "password123", "T", "ip", "ua")

	session := &domain.Session{
		ID:           "sess1",
		UserID:       "user123",
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
	}
	db.On("GetSessionByRefreshToken", refreshToken).Return(session, nil)
	db.On("GetUserByID", mock.Anything).Return(nil, errors.New("not found"))

	_, err := svc.RefreshToken(refreshToken)

	assert.ErrorIs(t, err, domain.ErrUserNotFound)
	db.AssertExpectations(t)
}

// ------------------------------------------------------------------
// Logout
// ------------------------------------------------------------------

func TestAuthService_Logout_Success(t *testing.T) {
	db := new(MockDB)
	redis := new(MockRedis)
	svc := NewAuthService(db, redis)

	db.On("DeleteSessionByRefreshToken", "token123").Return(nil)
	db.On("CreateAuditLog", mock.AnythingOfType("*domain.AuditLog")).Return(nil)
	redis.On("DeleteUser", "user123").Return(nil)

	err := svc.Logout("token123", "user123", "127.0.0.1", "agent")

	assert.NoError(t, err)

	db.AssertExpectations(t)
	redis.AssertExpectations(t)
}

func TestAuthService_Logout_DeleteSessionError(t *testing.T) {
	db := new(MockDB)
	redis := new(MockRedis)
	svc := NewAuthService(db, redis)

	db.On("DeleteSessionByRefreshToken", "token123").Return(errors.New("db error"))

	err := svc.Logout("token123", "user123", "127.0.0.1", "agent")

	assert.Error(t, err)

	db.AssertExpectations(t)
}

// ------------------------------------------------------------------
// GetCurrentUser
// ------------------------------------------------------------------

func TestAuthService_GetCurrentUser_FromRedis(t *testing.T) {
	db := new(MockDB)
	redis := new(MockRedis)
	svc := NewAuthService(db, redis)

	user := &domain.User{ID: "user123", Email: "user@example.com"}
	redis.On("GetUser", "user123").Return(user, nil)

	result, err := svc.GetCurrentUser("user123")

	assert.NoError(t, err)
	assert.Equal(t, "user@example.com", result.Email)

	redis.AssertExpectations(t)
}

func TestAuthService_GetCurrentUser_FromDB(t *testing.T) {
	db := new(MockDB)
	redis := new(MockRedis)
	svc := NewAuthService(db, redis)

	user := &domain.User{ID: "user123", Email: "user@example.com"}
	redis.On("GetUser", "user123").Return(nil, errors.New("cache miss"))
	db.On("GetUserByID", "user123").Return(user, nil)
	redis.On("SetUser", "user123", user, 24*time.Hour).Return(nil)

	result, err := svc.GetCurrentUser("user123")

	assert.NoError(t, err)
	assert.Equal(t, "user@example.com", result.Email)

	db.AssertExpectations(t)
	redis.AssertExpectations(t)
}

func TestAuthService_GetCurrentUser_NotFound(t *testing.T) {
	db := new(MockDB)
	redis := new(MockRedis)
	svc := NewAuthService(db, redis)

	redis.On("GetUser", "ghost").Return(nil, errors.New("cache miss"))
	db.On("GetUserByID", "ghost").Return(nil, errors.New("not found"))

	result, err := svc.GetCurrentUser("ghost")

	assert.ErrorIs(t, err, domain.ErrUserNotFound)
	assert.Nil(t, result)

	db.AssertExpectations(t)
	redis.AssertExpectations(t)
}
