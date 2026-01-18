package clients

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	
	"notification-service/domain"
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

func (c *VideoServiceClient) GetVideoByID(videoID string) (*domain.Video, error) {
	url := fmt.Sprintf("%s/api/internal/videos/%s", c.baseURL, videoID)
	
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get video by ID: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}
	
	var video domain.Video
	if err := json.NewDecoder(resp.Body).Decode(&video); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &video, nil
}
