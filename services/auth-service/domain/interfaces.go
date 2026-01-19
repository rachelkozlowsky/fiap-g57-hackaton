package domain

import "time"

type DatabaseInterface interface {
	CreateUser(user *User) error
	GetUserByEmail(email string) (*User, error)
	GetUserByID(id string) (*User, error)
	UpdateUser(user *User) error
	
	CreateSession(session *Session) error
	GetSessionByRefreshToken(token string) (*Session, error)
	UpdateSession(session *Session) error
	DeleteSessionByRefreshToken(token string) error
	
	ListUsers() ([]User, error)
	DeleteUser(id string) error
	
	CreateAuditLog(log *AuditLog) error
	
	Ping() error
	Close() error
}

type RedisClientInterface interface {
	SetUser(id string, user *User, ttl time.Duration) error
	GetUser(id string) (*User, error)
	DeleteUser(id string) error
	Ping() error
	Close() error
}
