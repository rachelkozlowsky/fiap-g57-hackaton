package handlers

import (
	"net/http"
	"time"
	"auth-service/domain"
	"github.com/gin-gonic/gin"
)

type InternalHandler struct {
	db domain.DatabaseInterface
}

func NewInternalHandler(db domain.DatabaseInterface) *InternalHandler {
	return &InternalHandler{
		db: db,
	}
}

type UserResponse struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Role          string `json:"role"`
	IsActive      bool   `json:"is_active"`
	EmailVerified bool   `json:"email_verified"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

func (h *InternalHandler) GetUserByID(c *gin.Context) {
	userID := c.Param("id")

	user, err := h.db.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	response := UserResponse{
		ID:            user.ID,
		Email:         user.Email,
		Name:          user.Name,
		Role:          user.Role,
		IsActive:      user.IsActive,
		EmailVerified: user.EmailVerified,
		CreatedAt:     user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	c.JSON(http.StatusOK, response)
}

func (h *InternalHandler) GetUserByEmail(c *gin.Context) {
	email := c.Param("email")

	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Email is required",
		})
		return
	}

	user, err := h.db.GetUserByEmail(email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	response := UserResponse{
		ID:            user.ID,
		Email:         user.Email,
		Name:          user.Name,
		Role:          user.Role,
		IsActive:      user.IsActive,
		EmailVerified: user.EmailVerified,
		CreatedAt:     user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	c.JSON(http.StatusOK, response)
}

type CreateAuditLogRequest struct {
	UserID     *string `json:"user_id"`
	Action     string  `json:"action"`
	EntityType string  `json:"entity_type"`
	EntityID   *string `json:"entity_id"`
	IPAddress  string  `json:"ip_address"`
	UserAgent  string  `json:"user_agent"`
}

func (h *InternalHandler) CreateAuditLog(c *gin.Context) {
	var req CreateAuditLogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	log := &domain.AuditLog{
		UserID:     req.UserID,
		Action:     req.Action,
		EntityType: req.EntityType,
		EntityID:   req.EntityID,
		IPAddress:  req.IPAddress,
		UserAgent:  req.UserAgent,
		CreatedAt:  time.Now(),
	}

	if err := h.db.CreateAuditLog(log); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create audit log"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Audit log created"})
}