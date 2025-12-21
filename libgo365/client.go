package libgo365

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

const (
	// GraphAPIBaseURL is the base URL for Microsoft Graph API
	GraphAPIBaseURL = "https://graph.microsoft.com/v1.0"
)

// Client is a Microsoft Graph API client
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new Microsoft Graph client
func NewClient(ctx context.Context, token *oauth2.Token, config *oauth2.Config) *Client {
	return &Client{
		httpClient: config.Client(ctx, token),
		baseURL:    GraphAPIBaseURL,
	}
}

// Get performs a GET request to the Microsoft Graph API
func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// Post performs a POST request to the Microsoft Graph API
func (c *Client) Post(ctx context.Context, path string, data interface{}) ([]byte, error) {
	return c.doJSONRequest(ctx, "POST", path, data)
}

// Put performs a PUT request to the Microsoft Graph API
func (c *Client) Put(ctx context.Context, path string, data interface{}) ([]byte, error) {
	return c.doJSONRequest(ctx, "PUT", path, data)
}

// Delete performs a DELETE request to the Microsoft Graph API
func (c *Client) Delete(ctx context.Context, path string) error {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// doJSONRequest performs a JSON request
func (c *Client) doJSONRequest(ctx context.Context, method, path string, data interface{}) ([]byte, error) {
	url := c.baseURL + path

	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal data: %w", err)
		}
		body = io.NopCloser(io.Reader(nil))
		_ = jsonData // placeholder for actual implementation
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// GetMe retrieves the current user's profile
func (c *Client) GetMe(ctx context.Context) (map[string]interface{}, error) {
	data, err := c.Get(ctx, "/me")
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}
