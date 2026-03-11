package service

import (
	"encoding/json"
	"fmt"
	"time"
	"status-service/domain"
)

type StatusService struct {
	db          domain.DatabaseInterface
	redis       domain.RedisInterface
	minio       domain.MinIOInterface
	videoClient domain.VideoServiceClient
}

func NewStatusService(db domain.DatabaseInterface, redis domain.RedisInterface, minio domain.MinIOInterface, videoClient domain.VideoServiceClient) *StatusService {
	return &StatusService{
		db:          db,
		redis:       redis,
		minio:       minio,
		videoClient: videoClient,
	}
}

func (s *StatusService) ListVideos(userID, status string) ([]domain.Video, error) {
	cacheKey := fmt.Sprintf("videos:user:%s:status:%s", userID, status)
	cached, err := s.redis.Get(cacheKey)
	if err == nil && cached != "" {
		var videos []domain.Video
		if json.Unmarshal([]byte(cached), &videos) == nil {
			return videos, nil
		}
	}

	videosPtr, err := s.videoClient.GetVideosByUserID(userID, status)
	if err != nil {
		return nil, err
	}

	videos := make([]domain.Video, len(videosPtr))
	for i, v := range videosPtr {
		videos[i] = *v
	}

	jsonData, _ := json.Marshal(videos)
	s.redis.Set(cacheKey, string(jsonData), 5*time.Minute)

	return videos, nil
}

func (s *StatusService) GetVideo(videoID, userID string) (*domain.Video, error) {
	cacheKey := fmt.Sprintf("video:%s", videoID)
	cached, err := s.redis.Get(cacheKey)
	if err == nil && cached != "" {
		var video domain.Video
		if json.Unmarshal([]byte(cached), &video) == nil {
			if video.UserID != userID {
				return nil, fmt.Errorf("access denied")
			}
			return &video, nil
		}
	}

	video, err := s.videoClient.GetVideoByID(videoID)
	if err != nil {
		return nil, err
	}

	if video.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	jsonData, _ := json.Marshal(video)
	s.redis.Set(cacheKey, string(jsonData), 5*time.Minute)

	return video, nil
}

func (s *StatusService) GetDownloadURL(videoID, userID string) (string, error) {
	video, err := s.GetVideo(videoID, userID)
	if err != nil {
		return "", err
	}

	if video.Status != "completed" {
		return "", fmt.Errorf("video processing not completed")
	}

	if video.ZipPath == nil {
		return "", fmt.Errorf("ZIP file not found")
	}

	return s.minio.GetPresignedURL(*video.ZipPath, 1*time.Hour)
}

func (s *StatusService) GetUserStats(userID string) (*domain.UserStats, error) {
	cacheKey := fmt.Sprintf("stats:user:%s", userID)
	cached, err := s.redis.Get(cacheKey)
	if err == nil && cached != "" {
		var stats domain.UserStats
		if json.Unmarshal([]byte(cached), &stats) == nil {
			return &stats, nil
		}
	}

	stats, err := s.videoClient.GetUserStats(userID)
	if err != nil {
		return nil, err
	}

	jsonData, _ := json.Marshal(stats)
	s.redis.Set(cacheKey, string(jsonData), 1*time.Minute)

	return stats, nil
}

func (s *StatusService) GetSystemStats() (*domain.SystemStats, error) {
	cacheKey := "stats:system"
	cached, err := s.redis.Get(cacheKey)
	if err == nil && cached != "" {
		var stats domain.SystemStats
		if json.Unmarshal([]byte(cached), &stats) == nil {
			return &stats, nil
		}
	}

	stats, err := s.videoClient.GetSystemStats()
	if err != nil {
		return nil, err
	}

	jsonData, _ := json.Marshal(stats)
	s.redis.Set(cacheKey, string(jsonData), 30*time.Second)

	return stats, nil
}
