package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/njt/go365/internal/output"
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
	rootCmd.AddCommand(mailCmd)
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

var mailCmd = &cobra.Command{
	Use:   "mail",
	Short: "Manage email messages",
	Long:  `Read and send email messages as the authenticated user`,
}

var mailListCmd = &cobra.Command{
	Use:   "list",
	Short: "List email messages",
	Long:  `List email messages from the authenticated user's mailbox`,
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
			return fmt.Errorf("not authenticated. Please run 'go365 login' first")
		}

		accessToken, err := auth.GetAccessToken(ctx)
		if err != nil {
			return fmt.Errorf("failed to get access token: %w", err)
		}

		client := libgo365.NewClient(ctx, accessToken)

		// Get options from flags
		folderID, _ := cmd.Flags().GetString("folder-id")
		top, _ := cmd.Flags().GetInt("top")
		skip, _ := cmd.Flags().GetInt("skip")
		pageToken, _ := cmd.Flags().GetString("page-token")
		jsonOutput, _ := cmd.Flags().GetBool("json")
		// --markdown is accepted but is a no-op for list (no body content)

		opts := &libgo365.ListMessagesOptions{
			FolderID:  folderID,
			Top:       top,
			Skip:      skip,
			PageToken: pageToken,
		}

		resp, err := client.ListMessagesWithPagination(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to list messages: %w", err)
		}

		if jsonOutput {
			// JSON output matching Graph API structure
			listResp := output.FormatListResponse(resp.Messages, resp.Count, resp.NextPageToken)
			return output.WriteJSON(os.Stdout, listResp)
		}

		// Human-readable output
		if len(resp.Messages) == 0 {
			fmt.Println("No messages found")
			return nil
		}

		for _, msg := range resp.Messages {
			fmt.Printf("ID: %s\n", msg.ID)
			fmt.Printf("Subject: %s\n", msg.Subject)
			if msg.From != nil && msg.From.EmailAddress != nil {
				fmt.Printf("From: %s <%s>\n", msg.From.EmailAddress.Name, msg.From.EmailAddress.Address)
			}
			if msg.ReceivedDateTime != nil {
				fmt.Printf("Received: %s\n", msg.ReceivedDateTime.Format(time.RFC3339))
			}
			fmt.Println("---")
		}

		// Print pagination hint if there are more results
		output.PrintNextPageHint(os.Stdout, resp.NextPageToken)

		return nil
	},
}

var mailGetCmd = &cobra.Command{
	Use:   "get <message-id>",
	Short: "Get a specific email message",
	Long:  `Retrieve and display a specific email message by ID`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		messageID := args[0]

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
			return fmt.Errorf("not authenticated. Please run 'go365 login' first")
		}

		accessToken, err := auth.GetAccessToken(ctx)
		if err != nil {
			return fmt.Errorf("failed to get access token: %w", err)
		}

		client := libgo365.NewClient(ctx, accessToken)

		message, err := client.GetMessage(ctx, messageID)
		if err != nil {
			return fmt.Errorf("failed to get message: %w", err)
		}

		// Get output format flags
		jsonOutput, _ := cmd.Flags().GetBool("json")
		markdownOutput, _ := cmd.Flags().GetBool("markdown")

		// Convert body to markdown if requested and body is HTML
		if markdownOutput && message.Body != nil && strings.EqualFold(message.Body.ContentType, "HTML") {
			message.Body.Content = output.HTMLToMarkdown(message.Body.Content)
			message.Body.ContentType = "Markdown"
		}

		if jsonOutput {
			return output.WriteJSON(os.Stdout, message)
		}

		// Human-readable output
		fmt.Printf("ID: %s\n", message.ID)
		fmt.Printf("Subject: %s\n", message.Subject)
		if message.From != nil && message.From.EmailAddress != nil {
			fmt.Printf("From: %s <%s>\n", message.From.EmailAddress.Name, message.From.EmailAddress.Address)
		}
		if len(message.ToRecipients) > 0 {
			fmt.Printf("To: ")
			for i, recipient := range message.ToRecipients {
				if i > 0 {
					fmt.Printf(", ")
				}
				if recipient.EmailAddress != nil {
					fmt.Printf("%s <%s>", recipient.EmailAddress.Name, recipient.EmailAddress.Address)
				}
			}
			fmt.Println()
		}
		if message.ReceivedDateTime != nil {
			fmt.Printf("Received: %s\n", message.ReceivedDateTime.Format(time.RFC3339))
		}
		if message.Body != nil {
			fmt.Printf("\nBody (%s):\n", message.Body.ContentType)
			fmt.Println(message.Body.Content)
		}

		return nil
	},
}

var mailSendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send an email message",
	Long:  `Send an email message as the authenticated user`,
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
			return fmt.Errorf("not authenticated. Please run 'go365 login' first")
		}

		accessToken, err := auth.GetAccessToken(ctx)
		if err != nil {
			return fmt.Errorf("failed to get access token: %w", err)
		}

		client := libgo365.NewClient(ctx, accessToken)

		// Get required flags
		subject, _ := cmd.Flags().GetString("subject")
		to, _ := cmd.Flags().GetString("to")
		body, _ := cmd.Flags().GetString("body")
		bodyType, _ := cmd.Flags().GetString("body-type")
		cc, _ := cmd.Flags().GetString("cc")
		bcc, _ := cmd.Flags().GetString("bcc")
		saveToSentItems, _ := cmd.Flags().GetBool("save-to-sent-items")

		if subject == "" {
			return fmt.Errorf("subject is required")
		}
		if to == "" {
			return fmt.Errorf("to is required")
		}
		if body == "" {
			return fmt.Errorf("body is required")
		}

		// Parse recipients
		parseRecipients := func(addresses string) []*libgo365.Recipient {
			if addresses == "" {
				return nil
			}
			addrs := strings.Split(addresses, ",")
			recipients := make([]*libgo365.Recipient, 0, len(addrs))
			for _, addr := range addrs {
				addr = strings.TrimSpace(addr)
				if addr != "" {
					recipients = append(recipients, &libgo365.Recipient{
						EmailAddress: &libgo365.EmailAddress{
							Address: addr,
						},
					})
				}
			}
			return recipients
		}

		message := &libgo365.Message{
			Subject: subject,
			Body: &libgo365.ItemBody{
				ContentType: bodyType,
				Content:     body,
			},
			ToRecipients:  parseRecipients(to),
			CcRecipients:  parseRecipients(cc),
			BccRecipients: parseRecipients(bcc),
		}

		err = client.SendMail(ctx, message, saveToSentItems)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		// --markdown is accepted but is a no-op for send

		if jsonOutput {
			return output.WriteJSON(os.Stdout, output.FormatActionResponse(true, "Message sent successfully"))
		}

		fmt.Println("Message sent successfully!")
		return nil
	},
}

func init() {
	// mail list flags
	mailListCmd.Flags().String("folder-id", "", "Folder ID (e.g., inbox, sentitems)")
	mailListCmd.Flags().Int("top", 0, "Number of messages to retrieve (default: 100)")
	mailListCmd.Flags().Int("skip", 0, "Skip first N messages (offset-based pagination)")
	mailListCmd.Flags().String("page-token", "", "Continue from previous response (cursor-based pagination)")
	mailListCmd.Flags().Bool("json", false, "Output as JSON")
	mailListCmd.Flags().Bool("markdown", false, "Convert HTML body to Markdown (no-op for list)")

	// mail get flags
	mailGetCmd.Flags().Bool("json", false, "Output as JSON")
	mailGetCmd.Flags().Bool("markdown", false, "Convert HTML body to Markdown")

	// mail send flags
	mailSendCmd.Flags().String("subject", "", "Email subject (required)")
	mailSendCmd.Flags().String("to", "", "Recipient email address(es), comma-separated (required)")
	mailSendCmd.Flags().String("body", "", "Email body content (required)")
	mailSendCmd.Flags().String("body-type", "Text", "Body content type (Text or HTML)")
	mailSendCmd.Flags().String("cc", "", "CC recipient email address(es), comma-separated")
	mailSendCmd.Flags().String("bcc", "", "BCC recipient email address(es), comma-separated")
	mailSendCmd.Flags().Bool("save-to-sent-items", true, "Save message to sent items")
	mailSendCmd.Flags().Bool("json", false, "Output as JSON")
	mailSendCmd.Flags().Bool("markdown", false, "No-op for send command (accepted for consistency)")

	mailCmd.AddCommand(mailListCmd)
	mailCmd.AddCommand(mailGetCmd)
	mailCmd.AddCommand(mailSendCmd)
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
