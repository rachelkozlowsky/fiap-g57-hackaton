package domain

import (
	"time"
)

type DatabaseInterface interface {
	Ping() error
	Close() error
}

type RedisInterface interface {
	Get(key string) (string, error)
	Set(key string, value interface{}, expiration time.Duration) error
	Ping() error
	Close() error
}

type MinIOInterface interface {
	GetPresignedURL(objectName string, expires time.Duration) (string, error)
	Ping() error
}

type VideoServiceClient interface {
	GetVideosByUserID(userID, status string) ([]*Video, error)
	GetVideoByID(id string) (*Video, error)
	GetUserStats(userID string) (*UserStats, error)
	GetSystemStats() (*SystemStats, error)
}
