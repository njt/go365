package libgo365

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the application configuration
type Config struct {
	TenantID string   `json:"tenant_id,omitempty"`
	ClientID string   `json:"client_id,omitempty"`
	Scopes   []string `json:"scopes,omitempty"`
}

// ConfigManager handles configuration persistence
type ConfigManager struct {
	configPath string
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() (*ConfigManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".go365")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return &ConfigManager{
		configPath: filepath.Join(configDir, "config.json"),
	}, nil
}

// Save saves the configuration to disk
func (cm *ConfigManager) Save(config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(cm.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Load loads the configuration from disk
func (cm *ConfigManager) Load() (*Config, error) {
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default configuration
			return &Config{
				Scopes: []string{
					"https://graph.microsoft.com/.default",
				},
			}, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set defaults if not specified
	if len(config.Scopes) == 0 {
		config.Scopes = []string{"https://graph.microsoft.com/.default"}
	}

	return &config, nil
}
