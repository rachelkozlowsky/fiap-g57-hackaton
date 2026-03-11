package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"	
	"processing-service/domain"
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

type UpdateStatusRequest struct {
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
}

func (c *VideoServiceClient) UpdateVideoStatus(videoID, status string, errorMessage string) error {
	url := fmt.Sprintf("%s/api/internal/videos/%s/status", c.baseURL, videoID)
	
	payload := UpdateStatusRequest{
		Status:       status,
		ErrorMessage: errorMessage,
	}
	
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update video status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

type CompleteVideoRequest struct {
	ZipPath      string `json:"zip_path"`
	ZipSizeBytes int64  `json:"zip_size_bytes"`
	FrameCount   int    `json:"frame_count"`
}

func (c *VideoServiceClient) CompleteVideo(videoID, zipPath string, zipSize int64, frameCount int) error {
	url := fmt.Sprintf("%s/api/internal/videos/%s/complete", c.baseURL, videoID)
	
	payload := CompleteVideoRequest{
		ZipPath:      zipPath,
		ZipSizeBytes: zipSize,
		FrameCount:   frameCount,
	}
	
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	
	resp, err := c.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to complete video: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

type FailVideoRequest struct {
	ErrorMessage string `json:"error_message"`
}

func (c *VideoServiceClient) FailVideo(videoID, errorMessage string) error {
	url := fmt.Sprintf("%s/api/internal/videos/%s/fail", c.baseURL, videoID)
	
	payload := FailVideoRequest{
		ErrorMessage: errorMessage,
	}
	
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	
	resp, err := c.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to fail video: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}
