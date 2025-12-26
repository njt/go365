package libgo365

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	// GraphAPIBaseURL is the base URL for Microsoft Graph API
	GraphAPIBaseURL = "https://graph.microsoft.com/v1.0"
)

// Client is a Microsoft Graph API client
type Client struct {
	httpClient  *http.Client
	baseURL     string
	accessToken string
}

// NewClient creates a new Microsoft Graph client
func NewClient(ctx context.Context, accessToken string) *Client {
	return &Client{
		httpClient:  &http.Client{},
		baseURL:     GraphAPIBaseURL,
		accessToken: accessToken,
	}
}

// addAuthHeader adds the authorization header to a request
func (c *Client) addAuthHeader(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
}

// Get performs a GET request to the Microsoft Graph API
func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeader(req)

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

	c.addAuthHeader(req)

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
		body = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeader(req)

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

// MailboxSettings represents user mailbox settings from Graph API
type MailboxSettings struct {
	TimeZone   string `json:"timeZone"`
	DateFormat string `json:"dateFormat"`
	TimeFormat string `json:"timeFormat"`
}

// GetMailboxSettings retrieves the current user's mailbox settings (including timezone)
func (c *Client) GetMailboxSettings(ctx context.Context) (*MailboxSettings, error) {
	data, err := c.Get(ctx, "/me/mailboxSettings")
	if err != nil {
		return nil, err
	}

	var settings MailboxSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mailbox settings: %w", err)
	}

	return &settings, nil
}
