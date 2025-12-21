package libgo365

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func TestTokenStore(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	ts := &TokenStore{
		tokenPath: filepath.Join(tmpDir, "token.json"),
	}

	// Test saving and loading a token
	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		TokenType:    "Bearer",
		RefreshToken: "test-refresh-token",
	}

	if err := ts.SaveToken(token); err != nil {
		t.Fatalf("SaveToken failed: %v", err)
	}

	loadedToken, err := ts.LoadToken()
	if err != nil {
		t.Fatalf("LoadToken failed: %v", err)
	}

	if loadedToken.AccessToken != token.AccessToken {
		t.Errorf("Expected access token %s, got %s", token.AccessToken, loadedToken.AccessToken)
	}

	if loadedToken.RefreshToken != token.RefreshToken {
		t.Errorf("Expected refresh token %s, got %s", token.RefreshToken, loadedToken.RefreshToken)
	}

	// Test deleting token
	if err := ts.DeleteToken(); err != nil {
		t.Fatalf("DeleteToken failed: %v", err)
	}

	loadedToken, err = ts.LoadToken()
	if err != nil {
		t.Fatalf("LoadToken after delete failed: %v", err)
	}

	if loadedToken != nil {
		t.Errorf("Expected nil token after delete, got %v", loadedToken)
	}
}

func TestConfigManager(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	cm := &ConfigManager{
		configPath: filepath.Join(tmpDir, "config.json"),
	}

	// Test saving and loading config
	config := &Config{
		TenantID:     "test-tenant",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:8080",
		Scopes:       []string{"scope1", "scope2"},
	}

	if err := cm.Save(config); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loadedConfig, err := cm.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loadedConfig.TenantID != config.TenantID {
		t.Errorf("Expected tenant ID %s, got %s", config.TenantID, loadedConfig.TenantID)
	}

	if loadedConfig.ClientID != config.ClientID {
		t.Errorf("Expected client ID %s, got %s", config.ClientID, loadedConfig.ClientID)
	}

	if loadedConfig.ClientSecret != config.ClientSecret {
		t.Errorf("Expected client secret %s, got %s", config.ClientSecret, loadedConfig.ClientSecret)
	}

	if loadedConfig.RedirectURL != config.RedirectURL {
		t.Errorf("Expected redirect URL %s, got %s", config.RedirectURL, loadedConfig.RedirectURL)
	}

	if len(loadedConfig.Scopes) != len(config.Scopes) {
		t.Errorf("Expected %d scopes, got %d", len(config.Scopes), len(loadedConfig.Scopes))
	}
}

func TestConfigManagerDefaults(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	cm := &ConfigManager{
		configPath: filepath.Join(tmpDir, "config.json"),
	}

	// Load non-existent config to test defaults
	config, err := cm.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if config.RedirectURL == "" {
		t.Error("Expected default redirect URL to be set")
	}

	if len(config.Scopes) == 0 {
		t.Error("Expected default scopes to be set")
	}
}

func TestNewAuthenticator(t *testing.T) {
	// This test just ensures we can create an authenticator
	// We can't test full OAuth flow without a real server
	cfg := AuthConfig{
		TenantID:     "test-tenant",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:8080",
		Scopes:       []string{"https://graph.microsoft.com/.default"},
	}

	// Set up a temporary directory for the token store
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	auth, err := NewAuthenticator(cfg)
	if err != nil {
		t.Fatalf("NewAuthenticator failed: %v", err)
	}

	if auth == nil {
		t.Error("Expected authenticator to be created")
	}

	authURL := auth.GetAuthURL("test-state")
	if authURL == "" {
		t.Error("Expected auth URL to be generated")
	}

	// Verify the URL contains expected parameters
	if !strings.Contains(authURL, "client_id=test-client") {
		t.Error("Auth URL should contain client_id")
	}
}
