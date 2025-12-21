package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/njt/go365/internal/plugin"
	"github.com/njt/go365/libgo365"
	"github.com/spf13/cobra"
)

var (
	configMgr *libgo365.ConfigManager
	rootCmd   = &cobra.Command{
		Use:   "go365",
		Short: "Microsoft 365 / Microsoft Graph CLI tool",
		Long:  `go365 is a CLI tool for accessing Microsoft 365 and Microsoft Graph functionality.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If no subcommand is provided, show help
			if len(args) == 0 {
				return cmd.Help()
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: false,
	}
)

func init() {
	var err error
	configMgr, err = libgo365.NewConfigManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing config manager: %v\n", err)
		os.Exit(1)
	}

	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(pluginsCmd)
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Microsoft 365",
	Long:  `Authenticate with Microsoft 365 using device code flow`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := configMgr.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if config.ClientID == "" || config.TenantID == "" {
			return fmt.Errorf("client ID and tenant ID must be configured. Use 'go365 config set' to configure")
		}

		authConfig := libgo365.AuthConfig{
			TenantID: config.TenantID,
			ClientID: config.ClientID,
			Scopes:   config.Scopes,
		}

		auth, err := libgo365.NewAuthenticator(authConfig)
		if err != nil {
			return fmt.Errorf("failed to create authenticator: %w", err)
		}

		ctx := context.Background()
		if err := auth.LoginWithDeviceCode(ctx); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		fmt.Println("Successfully authenticated!")
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Sign out from Microsoft 365",
	Long:  `Remove stored authentication tokens`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := configMgr.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		authConfig := libgo365.AuthConfig{
			TenantID: config.TenantID,
			ClientID: config.ClientID,
			Scopes:   config.Scopes,
		}

		auth, err := libgo365.NewAuthenticator(authConfig)
		if err != nil {
			return fmt.Errorf("failed to create authenticator: %w", err)
		}

		ctx := context.Background()
		if err := auth.Logout(ctx); err != nil {
			return fmt.Errorf("logout failed: %w", err)
		}

		fmt.Println("Successfully logged out!")
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Long:  `Display current authentication status and user information`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := configMgr.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		authConfig := libgo365.AuthConfig{
			TenantID: config.TenantID,
			ClientID: config.ClientID,
			Scopes:   config.Scopes,
		}

		auth, err := libgo365.NewAuthenticator(authConfig)
		if err != nil {
			return fmt.Errorf("failed to create authenticator: %w", err)
		}

		ctx := context.Background()
		if !auth.IsAuthenticated(ctx) {
			fmt.Println("Status: Not authenticated")
			return nil
		}

		fmt.Println("Status: Authenticated")

		// Try to get user info from Graph API
		accessToken, err := auth.GetAccessToken(ctx)
		if err != nil {
			return err
		}

		client := libgo365.NewClient(ctx, accessToken)
		userInfo, err := client.GetMe(ctx)
		if err != nil {
			fmt.Printf("Warning: Could not retrieve user info: %v\n", err)
			return nil
		}

		if displayName, ok := userInfo["displayName"].(string); ok {
			fmt.Printf("User: %s\n", displayName)
		}
		if userPrincipalName, ok := userInfo["userPrincipalName"].(string); ok {
			fmt.Printf("Email: %s\n", userPrincipalName)
		}

		return nil
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Manage go365 configuration settings`,
}

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set configuration values",
	Long:  `Set configuration values like tenant ID, client ID, etc.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := configMgr.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		tenantID, _ := cmd.Flags().GetString("tenant-id")
		clientID, _ := cmd.Flags().GetString("client-id")

		if tenantID != "" {
			config.TenantID = tenantID
		}
		if clientID != "" {
			config.ClientID = clientID
		}

		if err := configMgr.Save(config); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Println("Configuration saved successfully!")
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display current configuration settings`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := configMgr.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		fmt.Printf("Tenant ID: %s\n", config.TenantID)
		fmt.Printf("Client ID: %s\n", config.ClientID)
		fmt.Printf("Scopes: %v\n", config.Scopes)

		return nil
	},
}

var pluginsCmd = &cobra.Command{
	Use:   "plugins",
	Short: "List available plugins",
	Long:  `List all available go365-* plugins in PATH`,
	RunE: func(cmd *cobra.Command, args []string) error {
		plugins, err := plugin.ListPlugins()
		if err != nil {
			return fmt.Errorf("failed to list plugins: %w", err)
		}

		if len(plugins) == 0 {
			fmt.Println("No plugins found in PATH")
			return nil
		}

		fmt.Println("Available plugins:")
		for _, p := range plugins {
			fmt.Printf("  - %s\n", p)
		}

		return nil
	},
}

func init() {
	configSetCmd.Flags().String("tenant-id", "", "Azure AD tenant ID")
	configSetCmd.Flags().String("client-id", "", "Azure AD client ID")

	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)
}

func main() {
	// Check if we should try to execute a plugin
	if len(os.Args) > 1 {
		// Check if this is a known command
		cmdName := os.Args[1]
		isKnownCmd := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == cmdName || cmd.HasAlias(cmdName) {
				isKnownCmd = true
				break
			}
		}

		// If not a known command and not a flag, try plugin
		if !isKnownCmd && cmdName != "" && !strings.HasPrefix(cmdName, "-") {
			if err := plugin.ExecutePlugin(cmdName, os.Args[2:]); err == nil {
				return
			}
			// If plugin fails, fall through to normal cobra execution
			// which will show the "unknown command" error
		}
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
