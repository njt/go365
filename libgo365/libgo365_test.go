package libgo365

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTokenCache(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	tc := &TokenCache{
		cachePath: filepath.Join(tmpDir, "msal_cache.bin"),
	}

	// Test deleting non-existent cache
	if err := tc.DeleteCache(); err != nil {
		t.Fatalf("DeleteCache failed: %v", err)
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
		TenantID: "test-tenant",
		ClientID: "test-client",
		Scopes:   []string{"scope1", "scope2"},
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

	if len(config.Scopes) == 0 {
		t.Error("Expected default scopes to be set")
	}
}

func TestNewAuthenticator(t *testing.T) {
	// This test just ensures we can create an authenticator
	// We can't test full device code flow without a real server
	cfg := AuthConfig{
		TenantID: "test-tenant",
		ClientID: "test-client",
		Scopes:   []string{"https://graph.microsoft.com/.default"},
	}

	// Set up a temporary directory for the token cache
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

	// Verify authenticator has the right configuration
	if auth.scopes == nil || len(auth.scopes) == 0 {
		t.Error("Expected scopes to be set")
	}
}
