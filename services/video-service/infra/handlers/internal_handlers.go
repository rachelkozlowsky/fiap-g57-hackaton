package handlers

import (
	"net/http"
	"time"
	"video-service/domain"
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

func (h *InternalHandler) GetVideoByID(c *gin.Context) {
	videoID := c.Param("id")

	video, err := h.db.GetVideoByID(videoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Video not found",
		})
		return
	}

	c.JSON(http.StatusOK, video)
}

func (h *InternalHandler) ListUserVideos(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user_id is required",
		})
		return
	}

	videos, err := h.db.GetVideosByUserID(userID, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get videos",
		})
		return
	}

	c.JSON(http.StatusOK, videos)
}

type UpdateStatusRequest struct {
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
}

func (h *InternalHandler) UpdateVideoStatus(c *gin.Context) {
	videoID := c.Param("id")

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	video, err := h.db.GetVideoByID(videoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Video not found",
		})
		return
	}

	video.Status = req.Status
	if req.ErrorMessage != "" {
		video.ErrorMessage = &req.ErrorMessage
	}

	if req.Status == "processing" {
		now := time.Now()
		video.ProcessingStartedAt = &now
	}

	if err := h.db.UpdateVideo(video); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update video",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Video status updated",
	})
}

type CompleteVideoRequest struct {
	ZipPath      string `json:"zip_path"`
	ZipSizeBytes int64  `json:"zip_size_bytes"`
	FrameCount   int    `json:"frame_count"`
}

func (h *InternalHandler) CompleteVideo(c *gin.Context) {
	videoID := c.Param("id")

	var req CompleteVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	video, err := h.db.GetVideoByID(videoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Video not found",
		})
		return
	}

	video.Status = "completed"
	video.ZipPath = &req.ZipPath
	video.ZipSizeBytes = &req.ZipSizeBytes
	video.FrameCount = &req.FrameCount
	now := time.Now()
	video.ProcessingCompletedAt = &now

	if err := h.db.UpdateVideo(video); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update video",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Video marked as completed",
	})
}

type FailVideoRequest struct {
	ErrorMessage string `json:"error_message"`
}

func (h *InternalHandler) FailVideo(c *gin.Context) {
	videoID := c.Param("id")

	var req FailVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	video, err := h.db.GetVideoByID(videoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Video not found",
		})
		return
	}

	video.Status = "failed"
	video.ErrorMessage = &req.ErrorMessage
	now := time.Now()
	video.ProcessingCompletedAt = &now

	if err := h.db.UpdateVideo(video); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update video",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Video marked as failed",
	})
}

func (h *InternalHandler) GetUserStats(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	stats, err := h.db.GetUserStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get user stats",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *InternalHandler) GetSystemStats(c *gin.Context) {
	stats, err := h.db.GetSystemStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get system stats",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}
