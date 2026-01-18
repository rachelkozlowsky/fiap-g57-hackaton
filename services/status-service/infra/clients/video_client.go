package clients

import (
	"encoding/json"
	"fmt"

	"net/http"
	"time"

	"status-service/domain"
)

type VideoServiceClient struct {
	baseURL string
	client  *http.Client
}

func NewVideoServiceClient(baseURL string) *VideoServiceClient {
	return &VideoServiceClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *VideoServiceClient) GetVideosByUserID(userID, status string) ([]*domain.Video, error) {
	url := fmt.Sprintf("%s/api/internal/videos?user_id=%s", c.baseURL, userID)
	
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get videos: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var videos []*domain.Video
	if err := json.NewDecoder(resp.Body).Decode(&videos); err != nil {
		return nil, fmt.Errorf("failed to decode videos: %w", err)
	}
	
	if status != "" {
		var filtered []*domain.Video
		for _, v := range videos {
			if v.Status == status {
				filtered = append(filtered, v)
			}
		}
		return filtered, nil
	}

	return videos, nil
}

func (c *VideoServiceClient) GetVideoByID(id string) (*domain.Video, error) {
	url := fmt.Sprintf("%s/api/internal/videos/%s", c.baseURL, id)
	
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get video: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("video not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var video domain.Video
	if err := json.NewDecoder(resp.Body).Decode(&video); err != nil {
		return nil, fmt.Errorf("failed to decode video: %w", err)
	}
	return &video, nil
}

func (c *VideoServiceClient) GetUserStats(userID string) (*domain.UserStats, error) {
	url := fmt.Sprintf("%s/api/internal/stats/user/%s", c.baseURL, userID)
	
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var stats domain.UserStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode user stats: %w", err)
	}

	return &stats, nil
}

func (c *VideoServiceClient) GetSystemStats() (*domain.SystemStats, error) {
	url := fmt.Sprintf("%s/api/internal/stats/system", c.baseURL)
	
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get system stats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var stats domain.SystemStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode system stats: %w", err)
	}

	return &stats, nil
}
