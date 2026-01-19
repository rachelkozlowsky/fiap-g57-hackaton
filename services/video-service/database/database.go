package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"
	"video-service/domain"
	"video-service/infra/utils"
	_ "github.com/lib/pq"
)

type Database struct {
	db *sql.DB
}

func InitDatabase() *Database {
	host := utils.GetEnv("DB_HOST", "localhost")
	port := utils.GetEnv("DB_PORT", "5432")
	user := utils.GetEnv("DB_USER", "g57")
	password := utils.GetEnv("DB_PASSWORD", "g57123456")
	dbname := utils.GetEnv("DB_NAME", "g57")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to PostgreSQL database")

	return &Database{db: db}
}

func (d *Database) Ping() error {
	return d.db.Ping()
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) CreateVideo(video *domain.Video) error {
	query := `
		INSERT INTO videos (id, user_id, filename, original_name, size_bytes, status, 
		                    storage_path, priority, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := d.db.Exec(query, video.ID, video.UserID, video.Filename, video.OriginalName,
		video.SizeBytes, video.Status, video.StoragePath, video.Priority, video.CreatedAt, video.UpdatedAt)
	return err
}

func (d *Database) GetVideoByID(id string) (*domain.Video, error) {
	video := &domain.Video{}
	query := `SELECT * FROM videos WHERE id = $1`
	err := d.db.QueryRow(query, id).Scan(
		&video.ID, &video.UserID, &video.Filename, &video.OriginalName, &video.SizeBytes,
		&video.DurationSeconds, &video.Status, &video.StoragePath, &video.ZipPath, &video.ZipSizeBytes,
		&video.FrameCount, &video.ErrorMessage, &video.RetryCount, &video.Priority,
		&video.CreatedAt, &video.UpdatedAt, &video.QueuedAt, &video.ProcessingStartedAt, &video.ProcessingCompletedAt,
	)
	if err != nil {
		return nil, err
	}
	return video, nil
}

func (d *Database) GetVideosByUserID(userID, status string) ([]*domain.Video, error) {
	var query string
	var rows *sql.Rows
	var err error

	if status != "" {
		query = `SELECT * FROM videos WHERE user_id = $1 AND status = $2 ORDER BY created_at DESC`
		rows, err = d.db.Query(query, userID, status)
	} else {
		query = `SELECT * FROM videos WHERE user_id = $1 ORDER BY created_at DESC`
		rows, err = d.db.Query(query, userID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	videos := []*domain.Video{}
	for rows.Next() {
		video := &domain.Video{}
		err := rows.Scan(
			&video.ID, &video.UserID, &video.Filename, &video.OriginalName, &video.SizeBytes,
			&video.DurationSeconds, &video.Status, &video.StoragePath, &video.ZipPath, &video.ZipSizeBytes,
			&video.FrameCount, &video.ErrorMessage, &video.RetryCount, &video.Priority,
			&video.CreatedAt, &video.UpdatedAt, &video.QueuedAt, &video.ProcessingStartedAt, &video.ProcessingCompletedAt,
		)
		if err != nil {
			return nil, err
		}
		videos = append(videos, video)
	}

	return videos, nil
}

func (d *Database) UpdateVideo(video *domain.Video) error {
	query := `
		UPDATE videos 
		SET status = $1, zip_path = $2, zip_size_bytes = $3, frame_count = $4, 
		    error_message = $5, retry_count = $6, updated_at = $7, queued_at = $8,
		    processing_started_at = $9, processing_completed_at = $10
		WHERE id = $11
	`
	_, err := d.db.Exec(query, video.Status, video.ZipPath, video.ZipSizeBytes, video.FrameCount,
		video.ErrorMessage, video.RetryCount, video.UpdatedAt, video.QueuedAt,
		video.ProcessingStartedAt, video.ProcessingCompletedAt, video.ID)
	return err
}

func (d *Database) DeleteVideo(id string) error {
	query := `DELETE FROM videos WHERE id = $1`
	_, err := d.db.Exec(query, id)
	return err
}

func (d *Database) GetUserStats(userID string) (*domain.UserStats, error) {
	stats := &domain.UserStats{}
	query := `
		SELECT 
			COUNT(*) as total_videos,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_videos,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_videos,
			COUNT(CASE WHEN status = 'processing' THEN 1 END) as processing_videos,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_videos,
			COALESCE(SUM(size_bytes) / 1024.0 / 1024.0, 0) as total_storage_mb,
			COALESCE(AVG(EXTRACT(EPOCH FROM (processing_completed_at - processing_started_at))), 0) as avg_processing_time
		FROM videos
		WHERE user_id = $1
	`
	err := d.db.QueryRow(query, userID).Scan(
		&stats.TotalVideos, &stats.CompletedVideos, &stats.FailedVideos,
		&stats.ProcessingVideos, &stats.PendingVideos, &stats.TotalStorageMB, &stats.AvgProcessingTime,
	)
	return stats, err
}

func (d *Database) GetSystemStats() (*domain.SystemStats, error) {
	stats := &domain.SystemStats{}
	
	stats.TotalUsers = 0 
	
	query := `
		SELECT 
			COUNT(*) as total_videos,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_videos,
			COUNT(CASE WHEN status = 'queued' THEN 1 END) as queued_videos,
			COUNT(CASE WHEN status = 'processing' THEN 1 END) as processing_videos,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_videos,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_videos,
			COALESCE(AVG(EXTRACT(EPOCH FROM (processing_completed_at - processing_started_at))), 0) as avg_processing_time
		FROM videos
		WHERE created_at > NOW() - INTERVAL '24 hours'
	`
	err := d.db.QueryRow(query).Scan(
		&stats.TotalVideos, &stats.PendingVideos, &stats.QueuedVideos,
		&stats.ProcessingVideos, &stats.CompletedVideos, &stats.FailedVideos, &stats.AvgProcessingTime,
	)
	
	return stats, err
}
