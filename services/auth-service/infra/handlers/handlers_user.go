package handlers

import (
	"net/http"
	"time"
	"auth-service/domain"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	db domain.DatabaseInterface
}

func NewUserHandler(db domain.DatabaseInterface) *UserHandler {
	return &UserHandler{
		db: db,
	}
}

func (h *UserHandler) ListUsers(c *gin.Context) {

	users, err := h.db.ListUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	
	dtos := make([]UserDTO, len(users))
	for i, user := range users {
		dtos[i] = UserToDTO(&user)
	}
	
	c.JSON(http.StatusOK, dtos)
}

func (h *UserHandler) GetUser(c *gin.Context) {
	id := c.Param("id")
	user, err := h.db.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, UserToDTO(user))
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	
	currentUserID := c.GetString("user_id")
	role := c.GetString("role")
	
	if currentUserID != id && role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.db.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user.Name = req.Name
	user.UpdatedAt = time.Now()
	
	if err := h.db.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, UserToDTO(user))
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.DeleteUser(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}
