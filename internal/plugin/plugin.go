package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// FindPlugin looks for a go365-* plugin in the PATH
func FindPlugin(name string) (string, error) {
	pluginName := "go365-" + name
	path, err := exec.LookPath(pluginName)
	if err != nil {
		return "", fmt.Errorf("plugin '%s' not found in PATH", pluginName)
	}
	return path, nil
}

// ExecutePlugin runs a go365-* plugin with the given arguments
func ExecutePlugin(name string, args []string) error {
	pluginPath, err := FindPlugin(name)
	if err != nil {
		return err
	}

	cmd := exec.Command(pluginPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// ListPlugins returns a list of available go365-* plugins in PATH
func ListPlugins() ([]string, error) {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return nil, nil
	}

	paths := strings.Split(pathEnv, string(os.PathListSeparator))
	plugins := make(map[string]bool)

	for _, dir := range paths {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, "go365-") && !entry.IsDir() {
				// Check if executable
				fullPath := dir + string(os.PathSeparator) + name
				info, err := os.Stat(fullPath)
				if err != nil {
					continue
				}

				// Check if file is executable
				if info.Mode()&0111 != 0 {
					pluginName := strings.TrimPrefix(name, "go365-")
					plugins[pluginName] = true
				}
			}
		}
	}

	result := make([]string, 0, len(plugins))
	for plugin := range plugins {
		result = append(result, plugin)
	}

	return result, nil
}
