package clients

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	
	"notification-service/domain"
)

type AuthServiceClient struct {
	baseURL string
	client  *http.Client
}

func NewAuthServiceClient(baseURL string) *AuthServiceClient {
	return &AuthServiceClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *AuthServiceClient) GetUserByID(userID string) (*domain.User, error) {
	url := fmt.Sprintf("%s/api/internal/users/%s", c.baseURL, userID)
	
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}
	
	var user domain.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &user, nil
}
