package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"auth-service/domain"
	"auth-service/infra/utils"

	_ "github.com/lib/pq"
)

type Database struct {
	db *sql.DB
}

func InitDatabase() *Database {
	host := utils.GetEnv("DB_HOST", "localhost")
	port := utils.GetEnv("DB_PORT", "5432")
	user := utils.GetEnv("DB_USER", "g57")
	password := utils.GetEnv("DB_PASSWORD", "g57123")
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

func (d *Database) CreateUser(user *domain.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, name, role, is_active, email_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := d.db.Exec(query, user.ID, user.Email, user.PasswordHash, user.Name, user.Role,
		user.IsActive, user.EmailVerified, user.CreatedAt, user.UpdatedAt)
	return err
}

func (d *Database) GetUserByEmail(email string) (*domain.User, error) {
	user := &domain.User{}
	query := `SELECT * FROM users WHERE email = $1`
	err := d.db.QueryRow(query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Role,
		&user.IsActive, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (d *Database) GetUserByID(id string) (*domain.User, error) {
	user := &domain.User{}
	query := `SELECT * FROM users WHERE id = $1`
	err := d.db.QueryRow(query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Role,
		&user.IsActive, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (d *Database) UpdateUser(user *domain.User) error {
	query := `
		UPDATE users 
		SET name = $1, role = $2, is_active = $3, email_verified = $4, 
		    updated_at = $5, last_login_at = $6
		WHERE id = $7
	`
	_, err := d.db.Exec(query, user.Name, user.Role, user.IsActive, user.EmailVerified,
		user.UpdatedAt, user.LastLoginAt, user.ID)
	return err
}

func (d *Database) CreateSession(session *domain.Session) error {
	query := `
		INSERT INTO sessions (id, user_id, refresh_token, ip_address, user_agent, expires_at, created_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := d.db.Exec(query, session.ID, session.UserID, session.RefreshToken, session.IPAddress,
		session.UserAgent, session.ExpiresAt, session.CreatedAt, session.LastUsedAt)
	return err
}

func (d *Database) GetSessionByRefreshToken(token string) (*domain.Session, error) {
	session := &domain.Session{}
	query := `SELECT * FROM sessions WHERE refresh_token = $1`
	err := d.db.QueryRow(query, token).Scan(
		&session.ID, &session.UserID, &session.RefreshToken, &session.IPAddress,
		&session.UserAgent, &session.ExpiresAt, &session.CreatedAt, &session.LastUsedAt,
	)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (d *Database) UpdateSession(session *domain.Session) error {
	query := `UPDATE sessions SET last_used_at = $1 WHERE id = $2`
	_, err := d.db.Exec(query, session.LastUsedAt, session.ID)
	return err
}

func (d *Database) DeleteSessionByRefreshToken(token string) error {
	query := `DELETE FROM sessions WHERE refresh_token = $1`
	_, err := d.db.Exec(query, token)
	return err
}

func (d *Database) CreateAuditLog(log *domain.AuditLog) error {
	query := `
		INSERT INTO audit_logs (user_id, action, entity_type, entity_id, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := d.db.Exec(query, log.UserID, log.Action, log.EntityType, log.EntityID, log.IPAddress, log.UserAgent, log.CreatedAt)
	return err
}




