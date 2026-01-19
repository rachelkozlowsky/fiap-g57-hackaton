package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"status-service/infra/utils"

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

	log.Println("✅ Connected to PostgreSQL database")

	return &Database{db: db}
}

func (d *Database) Ping() error {
	return d.db.Ping()
}

func (d *Database) Close() error {
	return d.db.Close()
}
