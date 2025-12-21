package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindPlugin(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create a test plugin
	pluginPath := filepath.Join(tmpDir, "go365-testplugin")
	content := []byte("#!/bin/bash\necho test")
	if err := os.WriteFile(pluginPath, content, 0755); err != nil {
		t.Fatalf("Failed to create test plugin: %v", err)
	}

	// Add temp directory to PATH
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+string(os.PathListSeparator)+oldPath)
	defer os.Setenv("PATH", oldPath)

	// Test finding the plugin
	found, err := FindPlugin("testplugin")
	if err != nil {
		t.Errorf("FindPlugin failed: %v", err)
	}

	if found == "" {
		t.Error("Expected to find plugin")
	}

	// Test finding non-existent plugin
	_, err = FindPlugin("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent plugin")
	}
}

func TestListPlugins(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create test plugins
	plugins := []string{"go365-plugin1", "go365-plugin2", "go365-plugin3"}
	for _, name := range plugins {
		pluginPath := filepath.Join(tmpDir, name)
		content := []byte("#!/bin/bash\necho test")
		if err := os.WriteFile(pluginPath, content, 0755); err != nil {
			t.Fatalf("Failed to create test plugin %s: %v", name, err)
		}
	}

	// Create a non-plugin file that shouldn't be listed
	nonPlugin := filepath.Join(tmpDir, "not-a-plugin")
	if err := os.WriteFile(nonPlugin, []byte("test"), 0755); err != nil {
		t.Fatalf("Failed to create non-plugin file: %v", err)
	}

	// Create a non-executable go365- file that shouldn't be listed
	nonExec := filepath.Join(tmpDir, "go365-nonexec")
	if err := os.WriteFile(nonExec, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create non-executable file: %v", err)
	}

	// Add temp directory to PATH
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+string(os.PathListSeparator)+oldPath)
	defer os.Setenv("PATH", oldPath)

	// Test listing plugins
	found, err := ListPlugins()
	if err != nil {
		t.Fatalf("ListPlugins failed: %v", err)
	}

	// Should find 3 plugins
	if len(found) != 3 {
		t.Errorf("Expected to find 3 plugins, got %d", len(found))
	}

	// Check that all plugin names are found
	pluginMap := make(map[string]bool)
	for _, p := range found {
		pluginMap[p] = true
	}

	expectedPlugins := []string{"plugin1", "plugin2", "plugin3"}
	for _, expected := range expectedPlugins {
		if !pluginMap[expected] {
			t.Errorf("Expected to find plugin %s", expected)
		}
	}
}

func TestListPluginsEmptyPath(t *testing.T) {
	// Save and clear PATH
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", oldPath)

	plugins, err := ListPlugins()
	if err != nil {
		t.Fatalf("ListPlugins with empty PATH should not error: %v", err)
	}

	if plugins != nil && len(plugins) != 0 {
		t.Errorf("Expected empty plugin list, got %d plugins", len(plugins))
	}
}
