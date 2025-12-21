package libgo365

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	TenantID string
	ClientID string
	Scopes   []string
}

// TokenCache handles token persistence for MSAL
type TokenCache struct {
	cachePath string
}

// NewTokenCache creates a new token cache
func NewTokenCache() (*TokenCache, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".go365")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return &TokenCache{
		cachePath: filepath.Join(configDir, "msal_cache.bin"),
	}, nil
}

// Replace implements cache.ExportReplace
func (tc *TokenCache) Replace(ctx context.Context, cache cache.Unmarshaler, hints cache.ReplaceHints) error {
	data, err := os.ReadFile(tc.cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No cache to replace with
		}
		return fmt.Errorf("failed to read cache: %w", err)
	}

	if err := cache.Unmarshal(data); err != nil {
		return fmt.Errorf("failed to unmarshal cache: %w", err)
	}

	return nil
}

// Export implements cache.ExportReplace
func (tc *TokenCache) Export(ctx context.Context, cache cache.Marshaler, hints cache.ExportHints) error {
	data, err := cache.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.WriteFile(tc.cachePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write cache: %w", err)
	}

	return nil
}

// DeleteCache removes the stored cache
func (tc *TokenCache) DeleteCache() error {
	if err := os.Remove(tc.cachePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete cache: %w", err)
	}
	return nil
}

// Authenticator handles MSAL authentication
type Authenticator struct {
	app        public.Client
	scopes     []string
	tokenCache *TokenCache
}

// NewAuthenticator creates a new authenticator
func NewAuthenticator(cfg AuthConfig) (*Authenticator, error) {
	tokenCache, err := NewTokenCache()
	if err != nil {
		return nil, err
	}

	// Create MSAL public client
	app, err := public.New(cfg.ClientID,
		public.WithAuthority(fmt.Sprintf("https://login.microsoftonline.com/%s", cfg.TenantID)),
		public.WithCache(tokenCache),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create MSAL client: %w", err)
	}

	return &Authenticator{
		app:        app,
		scopes:     cfg.Scopes,
		tokenCache: tokenCache,
	}, nil
}

// LoginWithDeviceCode performs device code authentication
func (a *Authenticator) LoginWithDeviceCode(ctx context.Context) error {
	// Start device code flow
	deviceCode, err := a.app.AcquireTokenByDeviceCode(ctx, a.scopes)
	if err != nil {
		return fmt.Errorf("failed to initiate device code flow: %w", err)
	}

	// Display device code message with chili pepper emoji (m365 CLI style)
	fmt.Printf("🌶️  %s\n", deviceCode.Result.Message)

	// Wait for user to complete authentication
	_, err = deviceCode.AuthenticationResult(ctx)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	return nil
}

// GetAccessToken retrieves a valid access token, using silent authentication if possible
func (a *Authenticator) GetAccessToken(ctx context.Context) (string, error) {
	// Try to get all cached accounts
	accounts, err := a.app.Accounts(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get accounts: %w", err)
	}

	if len(accounts) == 0 {
		return "", fmt.Errorf("not authenticated: please login first")
	}

	// Use the first account (most recently used)
	account := accounts[0]

	// Try silent authentication first
	result, err := a.app.AcquireTokenSilent(ctx, a.scopes, public.WithSilentAccount(account))
	if err != nil {
		return "", fmt.Errorf("failed to acquire token silently: %w", err)
	}

	return result.AccessToken, nil
}

// Logout removes all cached accounts
func (a *Authenticator) Logout(ctx context.Context) error {
	// Get all accounts
	accounts, err := a.app.Accounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to get accounts: %w", err)
	}

	// Remove each account
	for _, account := range accounts {
		if err := a.app.RemoveAccount(ctx, account); err != nil {
			return fmt.Errorf("failed to remove account: %w", err)
		}
	}

	// Delete the cache file
	return a.tokenCache.DeleteCache()
}

// IsAuthenticated checks if a valid account exists
func (a *Authenticator) IsAuthenticated(ctx context.Context) bool {
	accounts, err := a.app.Accounts(ctx)
	if err != nil {
		return false
	}
	return len(accounts) > 0
}

// GetUserInfo retrieves user information from the token
func (a *Authenticator) GetUserInfo(ctx context.Context) (map[string]interface{}, error) {
	accounts, err := a.app.Accounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts: %w", err)
	}

	if len(accounts) == 0 {
		return nil, fmt.Errorf("not authenticated")
	}

	account := accounts[0]
	return map[string]interface{}{
		"username":       account.PreferredUsername,
		"homeAccountId":  account.HomeAccountID,
		"environment":    account.Environment,
		"localAccountId": account.LocalAccountID,
	}, nil
}
