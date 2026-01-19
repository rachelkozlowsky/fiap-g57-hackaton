package handlers

import (
	"fmt"
	"net/http"
	"time"
	"status-service/domain"
	"status-service/service"
	"github.com/gin-gonic/gin"
)

type StatusHandler struct {
	statusService *service.StatusService
}

func NewStatusHandler(statusService *service.StatusService) *StatusHandler {
	return &StatusHandler{
		statusService: statusService,
	}
}

type VideoStatusResponse struct {
	ID                    string     `json:"id"`
	Filename              string     `json:"filename"`
	OriginalName          string     `json:"original_name"`
	SizeBytes             int64      `json:"size_bytes"`
	Status                string     `json:"status"`
	FrameCount            *int       `json:"frame_count,omitempty"`
	ZipSizeBytes          *int64     `json:"zip_size_bytes,omitempty"`
	DownloadURL           *string    `json:"download_url,omitempty"`
	ErrorMessage          *string    `json:"error_message,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	ProcessingStartedAt   *time.Time `json:"processing_started_at,omitempty"`
	ProcessingCompletedAt *time.Time `json:"processing_completed_at,omitempty"`
	ProcessingDuration    *int       `json:"processing_duration_seconds,omitempty"`
}

type ListVideosResponse struct {
	Videos []VideoStatusResponse `json:"videos"`
	Total  int                   `json:"total"`
}

type UserStatsResponse struct {
	TotalVideos       int     `json:"total_videos"`
	CompletedVideos   int     `json:"completed_videos"`
	FailedVideos      int     `json:"failed_videos"`
	ProcessingVideos  int     `json:"processing_videos"`
	PendingVideos     int     `json:"pending_videos"`
	TotalStorageMB    float64 `json:"total_storage_mb"`
	AvgProcessingTime float64 `json:"avg_processing_time_seconds"`
}

type SystemStatsResponse struct {
	TotalUsers        int     `json:"total_users"`
	TotalVideos       int     `json:"total_videos"`
	PendingVideos     int     `json:"pending_videos"`
	QueuedVideos      int     `json:"queued_videos"`
	ProcessingVideos  int     `json:"processing_videos"`
	CompletedVideos   int     `json:"completed_videos"`
	FailedVideos      int     `json:"failed_videos"`
	AvgProcessingTime float64 `json:"avg_processing_time_seconds"`
}

func (h *StatusHandler) ListVideos(c *gin.Context) {
	userID := c.GetString("user_id")
	status := c.Query("status")

	videos, err := h.statusService.ListVideos(userID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch videos"})
		return
	}

	response := ListVideosResponse{
		Videos: make([]VideoStatusResponse, 0, len(videos)),
		Total:  len(videos),
	}

	for _, video := range videos {
		response.Videos = append(response.Videos, h.videoToResponse(&video))
	}

	c.JSON(http.StatusOK, response)
}

func (h *StatusHandler) GetVideo(c *gin.Context) {
	videoID := c.Param("id")
	userID := c.GetString("user_id")

	video, err := h.statusService.GetVideo(videoID, userID)
	if err != nil {
		if err.Error() == "access denied" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
		}
		return
	}

	c.JSON(http.StatusOK, h.videoToResponse(video))
}

func (h *StatusHandler) DownloadZip(c *gin.Context) {
	videoID := c.Param("id")
	userID := c.GetString("user_id")

	downloadURL, err := h.statusService.GetDownloadURL(videoID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"download_url": downloadURL,
		"expires_in":   3600,
	})
}

func (h *StatusHandler) GetUserStats(c *gin.Context) {
	userID := c.GetString("user_id")

	stats, err := h.statusService.GetUserStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"})
		return
	}

	c.JSON(http.StatusOK, UserStatsResponse{
		TotalVideos:       stats.TotalVideos,
		CompletedVideos:   stats.CompletedVideos,
		FailedVideos:      stats.FailedVideos,
		ProcessingVideos:  stats.ProcessingVideos,
		PendingVideos:     stats.PendingVideos,
		TotalStorageMB:    stats.TotalStorageMB,
		AvgProcessingTime: stats.AvgProcessingTime,
	})
}

func (h *StatusHandler) GetSystemStats(c *gin.Context) {
	stats, err := h.statusService.GetSystemStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"})
		return
	}

	c.JSON(http.StatusOK, SystemStatsResponse{
		TotalUsers:        stats.TotalUsers,
		TotalVideos:       stats.TotalVideos,
		PendingVideos:     stats.PendingVideos,
		QueuedVideos:      stats.QueuedVideos,
		ProcessingVideos:  stats.ProcessingVideos,
		CompletedVideos:   stats.CompletedVideos,
		FailedVideos:      stats.FailedVideos,
		AvgProcessingTime: stats.AvgProcessingTime,
	})
}

func (h *StatusHandler) videoToResponse(video *domain.Video) VideoStatusResponse {
	response := VideoStatusResponse{
		ID:           video.ID,
		Filename:     video.Filename,
		OriginalName: video.OriginalName,
		SizeBytes:    video.SizeBytes,
		Status:       video.Status,
		CreatedAt:    video.CreatedAt,
	}

	if video.FrameCount != nil {
		response.FrameCount = video.FrameCount
	}

	if video.ZipSizeBytes != nil {
		response.ZipSizeBytes = video.ZipSizeBytes
	}

	if video.Status == "completed" && video.ZipPath != nil {
		downloadURL := fmt.Sprintf("/api/v1/videos/%s/download", video.ID)
		response.DownloadURL = &downloadURL
	}

	if video.ErrorMessage != nil {
		response.ErrorMessage = video.ErrorMessage
	}

	if video.ProcessingStartedAt != nil {
		response.ProcessingStartedAt = video.ProcessingStartedAt
	}

	if video.ProcessingCompletedAt != nil {
		response.ProcessingCompletedAt = video.ProcessingCompletedAt

		if video.ProcessingStartedAt != nil {
			duration := int(video.ProcessingCompletedAt.Sub(*video.ProcessingStartedAt).Seconds())
			response.ProcessingDuration = &duration
		}
	}

	return response
}
