package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"
	"processing-service/domain"
	"processing-service/infra/utils"
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




func (d *Database) CreateProcessingJob(job *domain.ProcessingJob) error {
	query := `
		INSERT INTO processing_jobs (id, video_id, worker_id, status, started_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := d.db.Exec(query, job.ID, job.VideoID, job.WorkerID, job.Status, job.StartedAt, job.CreatedAt)
	return err
}

func (d *Database) UpdateProcessingJob(job *domain.ProcessingJob) error {
	query := `
		UPDATE processing_jobs 
		SET status = $1, completed_at = $2, duration_seconds = $3, error_message = $4, retry_count = $5
		WHERE id = $6
	`
	_, err := d.db.Exec(query, job.Status, job.CompletedAt, job.DurationSeconds, job.ErrorMessage, job.RetryCount, job.ID)
	return err
}




