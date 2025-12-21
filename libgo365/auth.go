package libgo365

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	TenantID     string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// TokenStore handles token persistence
type TokenStore struct {
	tokenPath string
}

// NewTokenStore creates a new token store
func NewTokenStore() (*TokenStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".go365")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return &TokenStore{
		tokenPath: filepath.Join(configDir, "token.json"),
	}, nil
}

// SaveToken saves a token to disk
func (ts *TokenStore) SaveToken(token *oauth2.Token) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	if err := os.WriteFile(ts.tokenPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token: %w", err)
	}

	return nil
}

// LoadToken loads a token from disk
func (ts *TokenStore) LoadToken() (*oauth2.Token, error) {
	data, err := os.ReadFile(ts.tokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read token: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}

// DeleteToken removes the stored token
func (ts *TokenStore) DeleteToken() error {
	if err := os.Remove(ts.tokenPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete token: %w", err)
	}
	return nil
}

// Authenticator handles OAuth authentication
type Authenticator struct {
	config     *oauth2.Config
	tokenStore *TokenStore
}

// NewAuthenticator creates a new authenticator
func NewAuthenticator(cfg AuthConfig) (*Authenticator, error) {
	tokenStore, err := NewTokenStore()
	if err != nil {
		return nil, err
	}

	oauth2Config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       cfg.Scopes,
		Endpoint:     microsoft.AzureADEndpoint(cfg.TenantID),
	}

	return &Authenticator{
		config:     oauth2Config,
		tokenStore: tokenStore,
	}, nil
}

// GetAuthURL returns the URL for user authentication
func (a *Authenticator) GetAuthURL(state string) string {
	return a.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// ExchangeCode exchanges an authorization code for a token
func (a *Authenticator) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := a.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	if err := a.tokenStore.SaveToken(token); err != nil {
		return nil, err
	}

	return token, nil
}

// GetToken retrieves the current token, refreshing if needed
func (a *Authenticator) GetToken(ctx context.Context) (*oauth2.Token, error) {
	token, err := a.tokenStore.LoadToken()
	if err != nil {
		return nil, err
	}

	if token == nil {
		return nil, fmt.Errorf("not authenticated: please login first")
	}

	// Refresh token if needed
	tokenSource := a.config.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	// Save if token was refreshed
	if newToken.AccessToken != token.AccessToken {
		if err := a.tokenStore.SaveToken(newToken); err != nil {
			return nil, err
		}
	}

	return newToken, nil
}

// Logout removes the stored token
func (a *Authenticator) Logout() error {
	return a.tokenStore.DeleteToken()
}

// IsAuthenticated checks if a valid token exists
func (a *Authenticator) IsAuthenticated(ctx context.Context) bool {
	token, err := a.GetToken(ctx)
	if err != nil {
		return false
	}
	return token.Valid()
}

// GetConfig returns the OAuth2 config
func (a *Authenticator) GetConfig() *oauth2.Config {
	return a.config
}
