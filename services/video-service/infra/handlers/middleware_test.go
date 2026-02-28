package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCorsMiddleware_GET(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CorsMiddleware())
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCorsMiddleware_OPTIONS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.OPTIONS("/test", CorsMiddleware())

	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestAuthMiddleware_XUserIDHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthMiddleware())
	r.GET("/test", func(c *gin.Context) {
		userID := c.GetString("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-User-Id", "u-from-gw")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "u-from-gw")
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthMiddleware())
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": c.GetString("user_id")})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer sometoken")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestValidateToken_Empty(t *testing.T) {
	_, err := validateToken("")
	assert.Error(t, err)
}

func TestValidateToken_NonEmpty(t *testing.T) {
	userID, err := validateToken("sometoken")
	assert.NoError(t, err)
	assert.NotEmpty(t, userID)
}
