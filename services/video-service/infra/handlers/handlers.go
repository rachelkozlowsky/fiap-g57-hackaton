package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"video-service/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type VideoHandler struct {
	db         domain.DatabaseInterface
	minio      domain.MinIOInterface
	rabbitmq   domain.RabbitMQInterface
	authClient domain.AuthServiceClient
}

type UploadResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	VideoID string `json:"video_id,omitempty"`
	Status  string `json:"status,omitempty"`
}

type VideoResponse struct {
	ID                  string     `json:"id"`
	UserID              string     `json:"user_id"`
	Filename            string     `json:"filename"`
	OriginalName        string     `json:"original_name"`
	SizeBytes           int64      `json:"size_bytes"`
	Status              string     `json:"status"`
	FrameCount          *int       `json:"frame_count,omitempty"`
	ZipPath             *string    `json:"zip_path,omitempty"`
	DownloadURL         *string    `json:"download_url,omitempty"`
	ErrorMessage        *string    `json:"error_message,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	ProcessingStarted   *time.Time `json:"processing_started_at,omitempty"`
	ProcessingCompleted *time.Time `json:"processing_completed_at,omitempty"`
}

func NewVideoHandler(db domain.DatabaseInterface, minio domain.MinIOInterface, rabbitmq domain.RabbitMQInterface, authClient domain.AuthServiceClient) *VideoHandler {
	return &VideoHandler{
		db:         db,
		minio:      minio,
		rabbitmq:   rabbitmq,
		authClient: authClient,
	}
}

func (h *VideoHandler) Upload(c *gin.Context) {
	userID := c.GetString("user_id")

	file, header, err := c.Request.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, UploadResponse{
			Success: false,
			Message: "Failed to get video file: " + err.Error(),
		})
		return
	}
	defer file.Close()

	if !isValidVideoFile(header.Filename) {
		c.JSON(http.StatusBadRequest, UploadResponse{
			Success: false,
			Message: "Invalid video format. Supported: mp4, avi, mov, mkv, wmv, flv, webm",
		})
		return
	}

	maxSize := int64(500 * 1024 * 1024)
	if header.Size > maxSize {
		c.JSON(http.StatusBadRequest, UploadResponse{
			Success: false,
			Message: fmt.Sprintf("File too large. Max size: 500MB, got: %.2fMB", float64(header.Size)/1024/1024),
		})
		return
	}

	videoID := uuid.New().String()
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%s_%s%s", time.Now().Format("20060102_150405"), videoID, ext)

	storagePath, err := h.minio.UploadFile(file, filename, header.Size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, UploadResponse{
			Success: false,
			Message: "Failed to upload file: " + err.Error(),
		})
		return
	}

	video := &domain.Video{
		ID:           videoID,
		UserID:       userID,
		Filename:     filename,
		OriginalName: header.Filename,
		SizeBytes:    header.Size,
		Status:       "pending",
		StoragePath:  storagePath,
		Priority:     1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := h.db.CreateVideo(video); err != nil {
		h.minio.DeleteFile(storagePath)
		c.JSON(http.StatusInternalServerError, UploadResponse{
			Success: false,
			Message: "Failed to create video record: " + err.Error(),
		})
		return
	}

	message := domain.VideoProcessingMessage{
		VideoID:     videoID,
		UserID:      userID,
		StoragePath: storagePath,
		Filename:    filename,
		Priority:    5,
	}

	if err := h.rabbitmq.PublishVideoUpload(message); err != nil {
		video.Status = "failed"
		video.ErrorMessage = StringPtr("Failed to queue for processing")
		h.db.UpdateVideo(video)

		c.JSON(http.StatusInternalServerError, UploadResponse{
			Success: false,
			Message: "Failed to queue video for processing: " + err.Error(),
		})
		return
	}

	video.Status = "queued"
	video.QueuedAt = TimePtr(time.Now())
	h.db.UpdateVideo(video)

	auditReq := domain.AuditLogRequest{
		UserID:     &userID,
		Action:     "video.upload",
		EntityType: "video",
		EntityID:   &videoID,
		IPAddress:  c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
	}
	go func() {
		if err := h.authClient.CreateAuditLog(auditReq); err != nil {
			fmt.Printf("Failed to create audit log: %v\n", err)
		}
	}()

	c.JSON(http.StatusOK, UploadResponse{
		Success: true,
		Message: "Video uploaded successfully and queued for processing",
		VideoID: videoID,
		Status:  "queued",
	})
}

func (h *VideoHandler) GetVideo(c *gin.Context) {
	videoID := c.Param("id")
	userID := c.GetString("user_id")

	video, err := h.db.GetVideoByID(videoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
		return
	}

	if video.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	response := VideoResponse{
		ID:           video.ID,
		UserID:       video.UserID,
		Filename:     video.Filename,
		OriginalName: video.OriginalName,
		SizeBytes:    video.SizeBytes,
		Status:       video.Status,
		CreatedAt:    video.CreatedAt,
	}

	if video.FrameCount != nil {
		response.FrameCount = video.FrameCount
	}

	if video.ZipPath != nil && video.Status == "completed" {
		downloadURL := fmt.Sprintf("/api/v1/videos/%s/download", videoID)
		response.DownloadURL = &downloadURL
		response.ZipPath = video.ZipPath
	}

	if video.ErrorMessage != nil {
		response.ErrorMessage = video.ErrorMessage
	}

	if video.ProcessingStartedAt != nil {
		response.ProcessingStarted = video.ProcessingStartedAt
	}

	if video.ProcessingCompletedAt != nil {
		response.ProcessingCompleted = video.ProcessingCompletedAt
	}

	c.JSON(http.StatusOK, response)
}

func (h *VideoHandler) List(c *gin.Context) {
	userID := c.GetString("user_id")
	status := c.Query("status")

	videos, err := h.db.GetVideosByUserID(userID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch videos"})
		return
	}

	responses := make([]VideoResponse, 0)
	for _, v := range videos {
		resp := VideoResponse{
			ID:           v.ID,
			UserID:       v.UserID,
			Filename:     v.Filename,
			OriginalName: v.OriginalName,
			SizeBytes:    v.SizeBytes,
			Status:       v.Status,
			CreatedAt:    v.CreatedAt,
		}

		if v.FrameCount != nil {
			resp.FrameCount = v.FrameCount
		}

		if v.ZipPath != nil && v.Status == "completed" {
			downloadURL := fmt.Sprintf("/api/v1/videos/%s/download", v.ID)
			resp.DownloadURL = &downloadURL
			resp.ZipPath = v.ZipPath
		}

		if v.ErrorMessage != nil {
			resp.ErrorMessage = v.ErrorMessage
		}

		if v.ProcessingStartedAt != nil {
			resp.ProcessingStarted = v.ProcessingStartedAt
		}

		if v.ProcessingCompletedAt != nil {
			resp.ProcessingCompleted = v.ProcessingCompletedAt
		}

		responses = append(responses, resp)
	}

	c.JSON(http.StatusOK, responses)
}

func (h *VideoHandler) DeleteVideo(c *gin.Context) {
	videoID := c.Param("id")
	userID := c.GetString("user_id")

	video, err := h.db.GetVideoByID(videoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
		return
	}

	if video.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if video.StoragePath != "" {
		h.minio.DeleteFile(video.StoragePath)
	}
	if video.ZipPath != nil && *video.ZipPath != "" {
		h.minio.DeleteFile(*video.ZipPath)
	}

	if err := h.db.DeleteVideo(videoID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete video"})
		return
	}

	auditReq := domain.AuditLogRequest{
		UserID:     &userID,
		Action:     "video.delete",
		EntityType: "video",
		EntityID:   &videoID,
		IPAddress:  c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
	}
	go func() {
		if err := h.authClient.CreateAuditLog(auditReq); err != nil {
			fmt.Printf("Failed to create audit log: %v\n", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"message": "Video deleted successfully"})
}

func (h *VideoHandler) DownloadZip(c *gin.Context) {
	videoID := c.Param("id")

	video, err := h.db.GetVideoByID(videoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
		return
	}

	if video.Status != "completed" || video.ZipPath == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Video processing not completed"})
		return
	}

	object, err := h.minio.GetFileStream(*video.ZipPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get file stream: %v", err)})
		return
	}
	defer object.Close()

	info, err := object.Stat()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get file info"})
		return
	}

	extraHeaders := map[string]string{
		"Content-Disposition": fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(*video.ZipPath)),
	}

	c.DataFromReader(http.StatusOK, info.Size, "application/zip", object, extraHeaders)
}

func isValidVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExts := []string{".mp4", ".avi", ".mov", ".mkv", ".wmv", ".flv", ".webm"}
	for _, validExt := range validExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

func StringPtr(s string) *string {
	return &s
}

func TimePtr(t time.Time) *time.Time {
	return &t
}
