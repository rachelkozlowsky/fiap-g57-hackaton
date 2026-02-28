package main

import (
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

func init() {
	gin.SetMode(gin.TestMode)
}

// ─── Mocks ────────────────────────────────────────────────────────────────────

type mockDB struct{ mock.Mock }

func (m *mockDB) CreateUser(u *domain.User) error                         { return m.Called(u).Error(0) }
func (m *mockDB) GetUserByEmail(e string) (*domain.User, error)           { return nil, errors.New("not found") }
func (m *mockDB) GetUserByID(id string) (*domain.User, error)             { return nil, errors.New("not found") }
func (m *mockDB) UpdateUser(u *domain.User) error                         { return nil }
func (m *mockDB) CreateSession(s *domain.Session) error                   { return nil }
func (m *mockDB) GetSessionByRefreshToken(t string) (*domain.Session, error) { return nil, nil }
func (m *mockDB) UpdateSession(s *domain.Session) error                   { return nil }
func (m *mockDB) DeleteSessionByRefreshToken(t string) error              { return nil }
func (m *mockDB) ListUsers() ([]domain.User, error)                       { return nil, nil }
func (m *mockDB) DeleteUser(id string) error                              { return nil }
func (m *mockDB) CreateAuditLog(l *domain.AuditLog) error                 { return nil }
func (m *mockDB) Close() error                                            { return nil }
func (m *mockDB) Ping() error {
	return m.Called().Error(0)
}

type mockRedis struct{ mock.Mock }

func (m *mockRedis) SetUser(id string, u *domain.User, ttl time.Duration) error { return nil }
func (m *mockRedis) GetUser(id string) (*domain.User, error)                    { return nil, nil }
func (m *mockRedis) DeleteUser(id string) error                                 { return nil }
func (m *mockRedis) Close() error                                               { return nil }
func (m *mockRedis) Ping() error {
	return m.Called().Error(0)
}

// ─── healthCheck ──────────────────────────────────────────────────────────────

func TestHealthCheck_StatusOK(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/health", nil)

	healthCheck(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHealthCheck_Body(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/health", nil)

	healthCheck(c)

	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "ok", body["status"])
	assert.Equal(t, "auth-service", body["service"])
	assert.Equal(t, "1.0.0", body["version"])
	assert.NotNil(t, body["time"])
}

// ─── livenessProbe ────────────────────────────────────────────────────────────

func TestLivenessProbe_StatusOK(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/health/live", nil)

	livenessProbe(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLivenessProbe_Body(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/health/live", nil)

	livenessProbe(c)

	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "alive", body["status"])
}

// ─── readinessProbe ───────────────────────────────────────────────────────────

func TestReadinessProbe_Ready(t *testing.T) {
	db := new(mockDB)
	redis := new(mockRedis)
	db.On("Ping").Return(nil)
	redis.On("Ping").Return(nil)

	handler := readinessProbe(db, redis)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/health/ready", nil)
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "ready", body["status"])

	db.AssertExpectations(t)
	redis.AssertExpectations(t)
}

func TestReadinessProbe_DBDown(t *testing.T) {
	db := new(mockDB)
	redis := new(mockRedis)
	db.On("Ping").Return(errors.New("connection refused"))

	handler := readinessProbe(db, redis)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/health/ready", nil)
	handler(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "not ready", body["status"])
	assert.Equal(t, "database connection failed", body["error"])

	db.AssertExpectations(t)
}

func TestReadinessProbe_RedisDown(t *testing.T) {
	db := new(mockDB)
	redis := new(mockRedis)
	db.On("Ping").Return(nil)
	redis.On("Ping").Return(errors.New("redis unreachable"))

	handler := readinessProbe(db, redis)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/health/ready", nil)
	handler(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "not ready", body["status"])
	assert.Equal(t, "redis connection failed", body["error"])

	db.AssertExpectations(t)
	redis.AssertExpectations(t)
}

// ─── setupRouter ──────────────────────────────────────────────────────────────

func newRouter() *gin.Engine {
	db := new(mockDB)
	redis := new(mockRedis)
	return setupRouter(db, redis)
}

func TestSetupRouter_HealthRoute(t *testing.T) {
	router := newRouter()
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSetupRouter_LivenessRoute(t *testing.T) {
	router := newRouter()
	req, _ := http.NewRequest("GET", "/health/live", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSetupRouter_ReadinessRoute_DBDown(t *testing.T) {
	db := new(mockDB)
	redis := new(mockRedis)
	db.On("Ping").Return(errors.New("down"))
	router := setupRouter(db, redis)

	req, _ := http.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestSetupRouter_ReadinessRoute_Ready(t *testing.T) {
	db := new(mockDB)
	redis := new(mockRedis)
	db.On("Ping").Return(nil)
	redis.On("Ping").Return(nil)
	router := setupRouter(db, redis)

	req, _ := http.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSetupRouter_MetricsRoute(t *testing.T) {
	router := newRouter()
	req, _ := http.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")
}

func TestSetupRouter_UnknownRoute_NotFound(t *testing.T) {
	router := newRouter()
	req, _ := http.NewRequest("GET", "/does-not-exist", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSetupRouter_CORSHeaders(t *testing.T) {
	router := newRouter()
	req, _ := http.NewRequest("OPTIONS", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

// ─── rotas protegidas retornam 401 sem token ──────────────────────────────────

func TestSetupRouter_LogoutRequiresAuth(t *testing.T) {
	router := newRouter()
	req, _ := http.NewRequest("POST", "/api/v1/auth/logout", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSetupRouter_MeRequiresAuth(t *testing.T) {
	router := newRouter()
	req, _ := http.NewRequest("GET", "/api/v1/auth/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSetupRouter_UsersRequiresAuth(t *testing.T) {
	router := newRouter()
	req, _ := http.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSetupRouter_UserByIDRequiresAuth(t *testing.T) {
	router := newRouter()
	req, _ := http.NewRequest("GET", "/api/v1/users/abc", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSetupRouter_DeleteUserRequiresAuth(t *testing.T) {
	router := newRouter()
	req, _ := http.NewRequest("DELETE", "/api/v1/users/abc", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ─── rotas públicas retornam 400 com body inválido (validação antes do DB) ────

func TestSetupRouter_RegisterBadBody(t *testing.T) {
	router := newRouter()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSetupRouter_LoginBadBody(t *testing.T) {
	router := newRouter()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSetupRouter_RefreshBadBody(t *testing.T) {
	router := newRouter()
	req, _ := http.NewRequest("POST", "/api/v1/auth/refresh", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── rotas internas ───────────────────────────────────────────────────────────

func TestSetupRouter_InternalGetUserByID(t *testing.T) {
	router := newRouter()
	// handler vai chamar db.GetUserByID — mas o mockDB retorna nil,nil (not found path sim)
	// o InternalHandler responde 404 para user não encontrado
	req, _ := http.NewRequest("GET", "/api/internal/users/someID", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// 404 é a resposta esperada quando GetUserByID retorna erro
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSetupRouter_InternalGetUserByEmail(t *testing.T) {
	router := newRouter()
	req, _ := http.NewRequest("GET", "/api/internal/users/email/test@example.com", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// mockDB.GetUserByEmail retorna nil,nil → handler retorna 404 "User not found"
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSetupRouter_ReleaseModeEnv(t *testing.T) {
	t.Setenv("GIN_MODE", "release")
	router := newRouter()
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	// restaura para teste
	gin.SetMode(gin.TestMode)
}
