package domain

import "time"

type User struct {
	ID              string     `json:"id" db:"id"`
	Email           string     `json:"email" db:"email"`
	PasswordHash    string     `json:"-" db:"password_hash"`
	Name            string     `json:"name" db:"name"`
	Role            string     `json:"role" db:"role"`
	IsActive        bool       `json:"is_active" db:"is_active"`
	EmailVerified   bool       `json:"email_verified" db:"email_verified"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
}

type Video struct {
	ID                    string     `json:"id" db:"id"`
	UserID                string     `json:"user_id" db:"user_id"`
	Filename              string     `json:"filename" db:"filename"`
	OriginalName          string     `json:"original_name" db:"original_name"`
	SizeBytes             int64      `json:"size_bytes" db:"size_bytes"`
	DurationSeconds       *float64   `json:"duration_seconds,omitempty" db:"duration_seconds"`
	Status                string     `json:"status" db:"status"`
	StoragePath           string     `json:"storage_path" db:"storage_path"`
	ZipPath               *string    `json:"zip_path,omitempty" db:"zip_path"`
	ZipSizeBytes          *int64     `json:"zip_size_bytes,omitempty" db:"zip_size_bytes"`
	FrameCount            *int       `json:"frame_count,omitempty" db:"frame_count"`
	ErrorMessage          *string    `json:"error_message,omitempty" db:"error_message"`
	RetryCount            int        `json:"retry_count" db:"retry_count"`
	Priority              int        `json:"priority" db:"priority"`
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at" db:"updated_at"`
	QueuedAt              *time.Time `json:"queued_at,omitempty" db:"queued_at"`
	ProcessingStartedAt   *time.Time `json:"processing_started_at,omitempty" db:"processing_started_at"`
	ProcessingCompletedAt *time.Time `json:"processing_completed_at,omitempty" db:"processing_completed_at"`
}

type Session struct {
	ID           string    `json:"id" db:"id"`
	UserID       string    `json:"user_id" db:"user_id"`
	RefreshToken string    `json:"refresh_token" db:"refresh_token"`
	IPAddress    string    `json:"ip_address" db:"ip_address"`
	UserAgent    string    `json:"user_agent" db:"user_agent"`
	ExpiresAt    time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	LastUsedAt   time.Time `json:"last_used_at" db:"last_used_at"`
}

type ProcessingJob struct {
	ID              string     `json:"id" db:"id"`
	VideoID         string     `json:"video_id" db:"video_id"`
	WorkerID        string     `json:"worker_id" db:"worker_id"`
	Status          string     `json:"status" db:"status"`
	StartedAt       *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	DurationSeconds *int       `json:"duration_seconds,omitempty" db:"duration_seconds"`
	ErrorMessage    *string    `json:"error_message,omitempty" db:"error_message"`
	RetryCount      int        `json:"retry_count" db:"retry_count"`
	Metadata        *string    `json:"metadata,omitempty" db:"metadata"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
}

type Notification struct {
	ID           string     `json:"id" db:"id"`
	UserID       string     `json:"user_id" db:"user_id"`
	VideoID      *string    `json:"video_id,omitempty" db:"video_id"`
	Type         string     `json:"type" db:"type"`
	Status       string     `json:"status" db:"status"`
	Subject      string     `json:"subject" db:"subject"`
	Message      string     `json:"message" db:"message"`
	Recipient    string     `json:"recipient" db:"recipient"`
	SentAt       *time.Time `json:"sent_at,omitempty" db:"sent_at"`
	ErrorMessage *string    `json:"error_message,omitempty" db:"error_message"`
	RetryCount   int        `json:"retry_count" db:"retry_count"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}

type AuditLog struct {
	ID         string    `json:"id" db:"id"`
	UserID     *string   `json:"user_id,omitempty" db:"user_id"`
	Action     string    `json:"action" db:"action"`
	EntityType string    `json:"entity_type" db:"entity_type"`
	EntityID   *string   `json:"entity_id,omitempty" db:"entity_id"`
	IPAddress  string    `json:"ip_address" db:"ip_address"`
	UserAgent  string    `json:"user_agent" db:"user_agent"`
	Metadata   *string   `json:"metadata,omitempty" db:"metadata"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type UserStats struct {
	TotalVideos       int     `json:"total_videos"`
	CompletedVideos   int     `json:"completed_videos"`
	FailedVideos      int     `json:"failed_videos"`
	ProcessingVideos  int     `json:"processing_videos"`
	PendingVideos     int     `json:"pending_videos"`
	TotalStorageMB    float64 `json:"total_storage_mb"`
	AvgProcessingTime float64 `json:"avg_processing_time_seconds"`
}

type SystemStats struct {
	TotalUsers        int     `json:"total_users"`
	TotalVideos       int     `json:"total_videos"`
	PendingVideos     int     `json:"pending_videos"`
	QueuedVideos      int     `json:"queued_videos"`
	ProcessingVideos  int     `json:"processing_videos"`
	CompletedVideos   int     `json:"completed_videos"`
	FailedVideos      int     `json:"failed_videos"`
	AvgProcessingTime float64 `json:"avg_processing_time_seconds"`
}

type VideoProcessingMessage struct {
	VideoID     string `json:"video_id"`
	UserID      string `json:"user_id"`
	Filename    string `json:"filename"`
	StoragePath string `json:"storage_path"`
	Priority    int    `json:"priority"`
}

type NotificationMessage struct {
	UserID  string `json:"user_id"`
	VideoID string `json:"video_id"`
	Type    string `json:"type"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type AuditLogRequest struct {
	UserID     *string `json:"user_id"`
	Action     string  `json:"action"`
	EntityType string  `json:"entity_type"`
	EntityID   *string `json:"entity_id"`
	IPAddress  string  `json:"ip_address"`
	UserAgent  string  `json:"user_agent"`
}

