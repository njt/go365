package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/njt/go365/libgo365"
)

func main() {
	ctx := context.Background()

	// Load configuration
	configMgr, err := libgo365.NewConfigManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	config, err := configMgr.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create authenticator
	authConfig := libgo365.AuthConfig{
		TenantID: config.TenantID,
		ClientID: config.ClientID,
		Scopes:   config.Scopes,
	}

	auth, err := libgo365.NewAuthenticator(authConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating authenticator: %v\n", err)
		os.Exit(1)
	}

	// Get access token
	accessToken, err := auth.GetAccessToken(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Not authenticated. Please run 'go365 login' first.\n")
		os.Exit(1)
	}

	// Create Microsoft Graph client
	client := libgo365.NewClient(ctx, accessToken)

	// Get current user information
	userInfo, err := client.GetMe(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting user info: %v\n", err)
		os.Exit(1)
	}

	// Pretty print user information
	prettyJSON, err := json.MarshalIndent(userInfo, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Current User Information:")
	fmt.Println(string(prettyJSON))
}
