package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"notification-service/domain"
	"notification-service/infra/utils"

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



func (d *Database) CreateNotification(notification *domain.Notification) error {
	query := `
		INSERT INTO notifications (id, user_id, video_id, type, status, subject, message, recipient, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := d.db.Exec(query, notification.ID, notification.UserID, notification.VideoID, notification.Type,
		notification.Status, notification.Subject, notification.Message, notification.Recipient, notification.CreatedAt)
	return err
}

func (d *Database) UpdateNotification(notification *domain.Notification) error {
	query := `
		UPDATE notifications 
		SET status = $1, sent_at = $2, error_message = $3, retry_count = $4
		WHERE id = $5
	`
	_, err := d.db.Exec(query, notification.Status, notification.SentAt, notification.ErrorMessage, notification.RetryCount, notification.ID)
	return err
}


