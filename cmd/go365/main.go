package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/njt/go365/internal/dateparse"
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
	rootCmd.AddCommand(calendarCmd)
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

var calendarCmd = &cobra.Command{
	Use:   "calendar",
	Short: "Manage calendar events",
	Long:  `View and manage calendar events for the authenticated user`,
}

var calendarListCmd = &cobra.Command{
	Use:   "list",
	Short: "List calendar events",
	Long:  `List calendar events for a time range. Defaults to today. Accepts natural language dates.`,
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
		startStr, _ := cmd.Flags().GetString("start")
		endStr, _ := cmd.Flags().GetString("end")
		days, _ := cmd.Flags().GetInt("days")
		calendarID, _ := cmd.Flags().GetString("calendar-id")
		allCalendars, _ := cmd.Flags().GetBool("all-calendars")
		top, _ := cmd.Flags().GetInt("top")
		pageToken, _ := cmd.Flags().GetString("page-token")
		jsonOutput, _ := cmd.Flags().GetBool("json")
		userID, _ := cmd.Flags().GetString("user")
		// --markdown is accepted but is a no-op for list (no body content)

		// Parse start date (default: today)
		now := time.Now()
		var startTime time.Time
		if startStr == "" {
			startTime = dateparse.StartOfDay(now)
		} else {
			startTime, err = dateparse.Parse(startStr, now)
			if err != nil {
				return fmt.Errorf("invalid start date: %w", err)
			}
		}

		// Parse end date
		var endTime time.Time
		if days > 0 {
			// --days takes precedence
			endTime = dateparse.AddDays(startTime, days)
		} else if endStr != "" {
			endTime, err = dateparse.Parse(endStr, now)
			if err != nil {
				return fmt.Errorf("invalid end date: %w", err)
			}
		} else {
			// Default: 1 day from start
			endTime = dateparse.AddDays(startTime, 1)
		}

		opts := &libgo365.CalendarViewOptions{
			StartDateTime: dateparse.FormatISO8601(startTime),
			EndDateTime:   dateparse.FormatISO8601(endTime),
			CalendarID:    calendarID,
			AllCalendars:  allCalendars,
			Top:           top,
			PageToken:     pageToken,
			UserID:        userID,
		}

		resp, err := client.CalendarView(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to list events: %w", err)
		}

		if jsonOutput {
			// JSON output matching Graph API structure
			listResp := output.FormatListResponse(resp.Events, resp.Count, resp.NextPageToken)
			return output.WriteJSON(os.Stdout, listResp)
		}

		// Human-readable output
		if len(resp.Events) == 0 {
			fmt.Println("No events found")
			return nil
		}

		for _, event := range resp.Events {
			fmt.Printf("ID: %s\n", event.ID)
			fmt.Printf("Subject: %s\n", event.Subject)
			if event.Start != nil {
				fmt.Printf("Start: %s\n", event.Start.DateTime)
			}
			if event.End != nil {
				fmt.Printf("End: %s\n", event.End.DateTime)
			}
			if event.IsAllDay {
				fmt.Printf("AllDay: true\n")
			}
			if event.Location != nil && event.Location.DisplayName != "" {
				fmt.Printf("Location: %s\n", event.Location.DisplayName)
			}
			if event.Organizer != nil && event.Organizer.EmailAddress != nil {
				fmt.Printf("Organizer: %s <%s>\n", event.Organizer.EmailAddress.Name, event.Organizer.EmailAddress.Address)
			}
			if event.ResponseStatus != nil && event.ResponseStatus.Response != "" {
				fmt.Printf("Response: %s\n", event.ResponseStatus.Response)
			}
			if event.CalendarID != "" {
				fmt.Printf("Calendar: %s\n", event.CalendarID)
			}
			fmt.Println("---")
		}

		// Print pagination hint if there are more results
		output.PrintNextPageHint(os.Stdout, resp.NextPageToken)

		return nil
	},
}

var calendarGetCmd = &cobra.Command{
	Use:   "get <event-id>",
	Short: "Get a specific calendar event",
	Long:  `Retrieve and display a specific calendar event by ID`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eventID := args[0]

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

		calendarID, _ := cmd.Flags().GetString("calendar-id")
		jsonOutput, _ := cmd.Flags().GetBool("json")
		markdownOutput, _ := cmd.Flags().GetBool("markdown")
		userID, _ := cmd.Flags().GetString("user")

		event, err := client.GetEventWithOptions(ctx, &libgo365.GetEventOptions{
			EventID:    eventID,
			CalendarID: calendarID,
			UserID:     userID,
		})
		if err != nil {
			return fmt.Errorf("failed to get event: %w", err)
		}

		// Convert body to markdown if requested and body is HTML
		if markdownOutput && event.Body != nil && strings.EqualFold(event.Body.ContentType, "HTML") {
			event.Body.Content = output.HTMLToMarkdown(event.Body.Content)
			event.Body.ContentType = "Markdown"
		}

		if jsonOutput {
			return output.WriteJSON(os.Stdout, event)
		}

		// Human-readable output
		fmt.Printf("ID: %s\n", event.ID)
		fmt.Printf("Subject: %s\n", event.Subject)
		if event.Start != nil {
			fmt.Printf("Start: %s (%s)\n", event.Start.DateTime, event.Start.TimeZone)
		}
		if event.End != nil {
			fmt.Printf("End: %s (%s)\n", event.End.DateTime, event.End.TimeZone)
		}
		if event.IsAllDay {
			fmt.Printf("AllDay: true\n")
		}
		if event.Location != nil && event.Location.DisplayName != "" {
			fmt.Printf("Location: %s\n", event.Location.DisplayName)
		}
		if event.Organizer != nil && event.Organizer.EmailAddress != nil {
			fmt.Printf("Organizer: %s <%s>\n", event.Organizer.EmailAddress.Name, event.Organizer.EmailAddress.Address)
		}
		if event.ResponseStatus != nil && event.ResponseStatus.Response != "" {
			fmt.Printf("Response: %s\n", event.ResponseStatus.Response)
		}

		// Attendees
		if len(event.Attendees) > 0 {
			fmt.Println("\nAttendees:")
			for _, att := range event.Attendees {
				if att.EmailAddress != nil {
					status := ""
					if att.Status != nil {
						status = att.Status.Response
					}
					fmt.Printf("  - %s <%s> [%s] (%s)\n", att.EmailAddress.Name, att.EmailAddress.Address, att.Type, status)
				}
			}
		}

		// Online meeting
		if event.OnlineMeeting != nil && event.OnlineMeeting.JoinUrl != "" {
			fmt.Printf("\nOnline Meeting: %s\n", event.OnlineMeeting.JoinUrl)
		}

		// Body
		if event.Body != nil && event.Body.Content != "" {
			fmt.Printf("\nBody (%s):\n%s\n", event.Body.ContentType, event.Body.Content)
		}

		return nil
	},
}

var calendarCalendarsCmd = &cobra.Command{
	Use:   "calendars",
	Short: "List available calendars",
	Long:  `List all calendars available to the authenticated user`,
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
		jsonOutput, _ := cmd.Flags().GetBool("json")

		calendars, err := client.ListCalendars(ctx)
		if err != nil {
			return fmt.Errorf("failed to list calendars: %w", err)
		}

		if jsonOutput {
			listResp := output.FormatListResponse(calendars, len(calendars), "")
			return output.WriteJSON(os.Stdout, listResp)
		}

		if len(calendars) == 0 {
			fmt.Println("No calendars found")
			return nil
		}

		fmt.Println("Calendars:")
		for i, cal := range calendars {
			fmt.Printf("%d. %s\n", i+1, cal.Name)
			fmt.Printf("   ID: %s\n", cal.ID)
			if cal.Owner != nil {
				fmt.Printf("   Owner: %s\n", cal.Owner.Address)
			}
			fmt.Println()
		}

		return nil
	},
}

var calendarEventsCmd = &cobra.Command{
	Use:   "events",
	Short: "List raw calendar events",
	Long:  `List raw events including series masters for recurring events. Unlike 'list', this doesn't expand recurring events into occurrences.`,
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

		calendarID, _ := cmd.Flags().GetString("calendar-id")
		top, _ := cmd.Flags().GetInt("top")
		pageToken, _ := cmd.Flags().GetString("page-token")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		opts := &libgo365.ListEventsOptions{
			CalendarID: calendarID,
			Top:        top,
			PageToken:  pageToken,
		}

		resp, err := client.ListEvents(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to list events: %w", err)
		}

		if jsonOutput {
			listResp := output.FormatListResponse(resp.Events, resp.Count, resp.NextPageToken)
			return output.WriteJSON(os.Stdout, listResp)
		}

		if len(resp.Events) == 0 {
			fmt.Println("No events found")
			return nil
		}

		for _, event := range resp.Events {
			fmt.Printf("ID: %s\n", event.ID)
			fmt.Printf("Subject: %s\n", event.Subject)
			if event.Start != nil {
				fmt.Printf("Start: %s\n", event.Start.DateTime)
			}
			if event.End != nil {
				fmt.Printf("End: %s\n", event.End.DateTime)
			}
			fmt.Println("---")
		}

		output.PrintNextPageHint(os.Stdout, resp.NextPageToken)
		return nil
	},
}

var calendarRespondCmd = &cobra.Command{
	Use:   "respond <event-id> <accept|decline|tentative>",
	Short: "Respond to a calendar invitation",
	Long:  `Accept, decline, or tentatively accept a calendar invitation.`,
	Args:  cobra.RangeArgs(1, 2),
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

		respondAll, _ := cmd.Flags().GetBool("all")
		idsStr, _ := cmd.Flags().GetString("ids")
		message, _ := cmd.Flags().GetString("message")

		var eventIDs []string
		var response string

		if respondAll {
			if len(args) < 1 {
				return fmt.Errorf("response type required (accept, decline, or tentative)")
			}
			response = args[0]

			// Get all pending events
			opts := &libgo365.ListEventsOptions{
				Filter: "responseStatus/response eq 'notResponded' or responseStatus/response eq 'none'",
			}
			resp, err := client.ListEvents(ctx, opts)
			if err != nil {
				return fmt.Errorf("failed to list pending events: %w", err)
			}
			for _, e := range resp.Events {
				eventIDs = append(eventIDs, e.ID)
			}
		} else if idsStr != "" {
			if len(args) < 1 {
				return fmt.Errorf("response type required (accept, decline, or tentative)")
			}
			response = args[0]
			parts := strings.Split(idsStr, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					eventIDs = append(eventIDs, p)
				}
			}
		} else {
			if len(args) < 2 {
				return fmt.Errorf("usage: calendar respond <event-id> <accept|decline|tentative>")
			}
			eventIDs = []string{args[0]}
			response = args[1]
		}

		if len(eventIDs) == 0 {
			fmt.Println("No events to respond to")
			return nil
		}

		for _, eventID := range eventIDs {
			err := client.RespondToEvent(ctx, eventID, response, message)
			if err != nil {
				fmt.Printf("Failed to respond to %s: %v\n", eventID, err)
				continue
			}
			fmt.Printf("Responded '%s' to event %s\n", response, eventID)
		}

		return nil
	},
}

var calendarPendingCmd = &cobra.Command{
	Use:   "pending",
	Short: "List pending invitations",
	Long:  `List calendar invitations awaiting your response.`,
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
		jsonOutput, _ := cmd.Flags().GetBool("json")

		// Filter for events where responseStatus is notResponded or none
		opts := &libgo365.ListEventsOptions{
			Filter: "responseStatus/response eq 'notResponded' or responseStatus/response eq 'none'",
		}

		resp, err := client.ListEvents(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to list events: %w", err)
		}

		if jsonOutput {
			listResp := output.FormatListResponse(resp.Events, resp.Count, resp.NextPageToken)
			return output.WriteJSON(os.Stdout, listResp)
		}

		if len(resp.Events) == 0 {
			fmt.Println("No pending invitations")
			return nil
		}

		fmt.Printf("%d pending invitation(s):\n\n", len(resp.Events))

		for i, event := range resp.Events {
			fmt.Printf("%d. %s\n", i+1, event.Subject)
			fmt.Printf("   ID: %s\n", event.ID)
			if event.Start != nil {
				fmt.Printf("   When: %s\n", event.Start.DateTime)
			}
			if event.Organizer != nil && event.Organizer.EmailAddress != nil {
				fmt.Printf("   From: %s\n", event.Organizer.EmailAddress.Address)
			}
			fmt.Println()
		}

		return nil
	},
}

var calendarFreeBusyCmd = &cobra.Command{
	Use:   "free-busy <emails>",
	Short: "Check availability for users",
	Long:  `Check free/busy status for one or more users. Works for anyone in your organization.`,
	Args:  cobra.MinimumNArgs(1),
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

		// Parse emails from args (may be comma-separated or multiple args)
		var emails []string
		for _, arg := range args {
			parts := strings.Split(arg, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					emails = append(emails, p)
				}
			}
		}

		startStr, _ := cmd.Flags().GetString("start")
		endStr, _ := cmd.Flags().GetString("end")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		now := time.Now()
		var startTime, endTime time.Time

		if startStr == "" {
			startTime = now
		} else {
			startTime, err = dateparse.Parse(startStr, now)
			if err != nil {
				return fmt.Errorf("invalid start time: %w", err)
			}
		}

		if endStr == "" {
			endTime = startTime.Add(24 * time.Hour)
		} else {
			endTime, err = dateparse.Parse(endStr, now)
			if err != nil {
				return fmt.Errorf("invalid end time: %w", err)
			}
		}

		resp, err := client.GetSchedule(ctx, emails, dateparse.FormatISO8601(startTime), dateparse.FormatISO8601(endTime))
		if err != nil {
			return fmt.Errorf("failed to get schedule: %w", err)
		}

		if jsonOutput {
			return output.WriteJSON(os.Stdout, resp)
		}

		for _, schedule := range resp.Value {
			fmt.Printf("%s:\n", schedule.ScheduleId)
			if schedule.Error != nil {
				fmt.Printf("  Error: %s\n", schedule.Error.Message)
				continue
			}
			if len(schedule.ScheduleItems) == 0 {
				fmt.Println("  Free")
				continue
			}
			for _, item := range schedule.ScheduleItems {
				startDT := ""
				endDT := ""
				if item.Start != nil {
					startDT = item.Start.DateTime
				}
				if item.End != nil {
					endDT = item.End.DateTime
				}
				fmt.Printf("  %s: %s - %s\n", strings.ToUpper(item.Status[:1])+item.Status[1:], startDT, endDT)
			}
			fmt.Println()
		}

		return nil
	},
}

var calendarFindTimeCmd = &cobra.Command{
	Use:   "find-time",
	Short: "Find available meeting times",
	Long:  `Find available meeting times across attendees' calendars.`,
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

		attendeesStr, _ := cmd.Flags().GetString("attendees")
		durationStr, _ := cmd.Flags().GetString("duration")
		startStr, _ := cmd.Flags().GetString("start")
		endStr, _ := cmd.Flags().GetString("end")
		maxResults, _ := cmd.Flags().GetInt("max-results")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		if attendeesStr == "" {
			return fmt.Errorf("--attendees is required")
		}

		attendees := strings.Split(attendeesStr, ",")
		for i := range attendees {
			attendees[i] = strings.TrimSpace(attendees[i])
		}

		// Parse duration (default 30m)
		duration := 30
		if durationStr != "" {
			d, err := time.ParseDuration(durationStr)
			if err != nil {
				return fmt.Errorf("invalid duration: %w", err)
			}
			duration = int(d.Minutes())
		}

		now := time.Now()
		var startTime, endTime time.Time

		if startStr == "" {
			startTime = now.Add(24 * time.Hour) // tomorrow
		} else {
			startTime, err = dateparse.Parse(startStr, now)
			if err != nil {
				return fmt.Errorf("invalid start time: %w", err)
			}
		}

		if endStr == "" {
			endTime = startTime.Add(7 * 24 * time.Hour) // +7 days
		} else {
			endTime, err = dateparse.Parse(endStr, now)
			if err != nil {
				return fmt.Errorf("invalid end time: %w", err)
			}
		}

		if maxResults == 0 {
			maxResults = 5
		}

		opts := &libgo365.FindTimeOptions{
			Attendees:       attendees,
			DurationMinutes: duration,
			StartDateTime:   dateparse.FormatISO8601(startTime),
			EndDateTime:     dateparse.FormatISO8601(endTime),
			MaxCandidates:   maxResults,
		}

		resp, err := client.FindMeetingTimes(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to find meeting times: %w", err)
		}

		if jsonOutput {
			return output.WriteJSON(os.Stdout, resp)
		}

		if len(resp.Suggestions) == 0 {
			fmt.Println("No available times found")
			if resp.EmptySuggestionsReason != "" {
				fmt.Printf("Reason: %s\n", resp.EmptySuggestionsReason)
			}
			return nil
		}

		fmt.Printf("Found %d available slots for %dm meeting:\n\n", len(resp.Suggestions), duration)

		for i, suggestion := range resp.Suggestions {
			slot := suggestion.MeetingTimeSlot
			if slot == nil || slot.Start == nil {
				continue
			}
			fmt.Printf("%d. %s - %s\n", i+1, slot.Start.DateTime, slot.End.DateTime)
			for _, avail := range suggestion.AttendeeAvailability {
				if avail.Attendee != nil && avail.Attendee.EmailAddress != nil {
					fmt.Printf("   %s: %s\n", avail.Attendee.EmailAddress.Address, avail.Availability)
				}
			}
			fmt.Println()
		}

		return nil
	},
}

var calendarCreateCmd = &cobra.Command{
	Use:   "create <subject>",
	Short: "Create a calendar event",
	Long:  `Create a new calendar event with subject, time, and optional attendees.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		subject := args[0]

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

		// Parse flags
		startStr, _ := cmd.Flags().GetString("start")
		endStr, _ := cmd.Flags().GetString("end")
		durationStr, _ := cmd.Flags().GetString("duration")
		attendeesStr, _ := cmd.Flags().GetString("attendees")
		location, _ := cmd.Flags().GetString("location")
		body, _ := cmd.Flags().GetString("body")
		online, _ := cmd.Flags().GetBool("online")
		allDay, _ := cmd.Flags().GetBool("all-day")
		calendarID, _ := cmd.Flags().GetString("calendar-id")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		if startStr == "" {
			return fmt.Errorf("--start is required")
		}

		if endStr != "" && durationStr != "" {
			return fmt.Errorf("--end and --duration are mutually exclusive")
		}

		now := time.Now()
		startTime, err := dateparse.Parse(startStr, now)
		if err != nil {
			return fmt.Errorf("invalid start time: %w", err)
		}

		var endTime time.Time
		if endStr != "" {
			endTime, err = dateparse.Parse(endStr, now)
			if err != nil {
				return fmt.Errorf("invalid end time: %w", err)
			}
		} else if durationStr != "" {
			duration, err := dateparse.ParseDuration(durationStr)
			if err != nil {
				return fmt.Errorf("invalid duration: %w", err)
			}
			endTime = startTime.Add(duration)
		} else {
			// Default: 30 minutes
			endTime = startTime.Add(30 * time.Minute)
		}

		tz := startTime.Location().String()
		if tz == "Local" {
			tz = "UTC"
		}

		event := &libgo365.Event{
			Subject:         subject,
			IsAllDay:        allDay,
			IsOnlineMeeting: online,
			Start: &libgo365.DateTimeTimeZone{
				DateTime: startTime.Format("2006-01-02T15:04:05"),
				TimeZone: tz,
			},
			End: &libgo365.DateTimeTimeZone{
				DateTime: endTime.Format("2006-01-02T15:04:05"),
				TimeZone: tz,
			},
		}

		if location != "" {
			event.Location = &libgo365.Location{DisplayName: location}
		}

		if body != "" {
			event.Body = &libgo365.ItemBody{
				ContentType: "Text",
				Content:     body,
			}
		}

		if attendeesStr != "" {
			emails := strings.Split(attendeesStr, ",")
			for _, email := range emails {
				email = strings.TrimSpace(email)
				if email != "" {
					event.Attendees = append(event.Attendees, &libgo365.Attendee{
						EmailAddress: &libgo365.EmailAddress{Address: email},
						Type:         "required",
					})
				}
			}
		}

		created, err := client.CreateEvent(ctx, event, calendarID)
		if err != nil {
			return fmt.Errorf("failed to create event: %w", err)
		}

		if jsonOutput {
			return output.WriteJSON(os.Stdout, created)
		}

		fmt.Printf("Created event: %s\n", created.Subject)
		fmt.Printf("ID: %s\n", created.ID)
		if created.Start != nil {
			fmt.Printf("Start: %s\n", created.Start.DateTime)
		}
		if created.End != nil {
			fmt.Printf("End: %s\n", created.End.DateTime)
		}
		if created.OnlineMeeting != nil && created.OnlineMeeting.JoinUrl != "" {
			fmt.Printf("Teams Link: %s\n", created.OnlineMeeting.JoinUrl)
		}

		return nil
	},
}

func init() {
	// calendar list flags
	calendarListCmd.Flags().String("start", "", "Start date/time (default: today, accepts natural language)")
	calendarListCmd.Flags().String("end", "", "End date/time (default: start + 1 day)")
	calendarListCmd.Flags().Int("days", 0, "Number of days from start (overrides --end)")
	calendarListCmd.Flags().String("calendar-id", "", "Query specific calendar (default: primary)")
	calendarListCmd.Flags().Bool("all-calendars", false, "Query all user's calendars")
	calendarListCmd.Flags().Int("top", 0, "Limit number of results")
	calendarListCmd.Flags().String("page-token", "", "Pagination token from previous response")
	calendarListCmd.Flags().Bool("json", false, "Output as JSON")
	calendarListCmd.Flags().Bool("markdown", false, "Convert HTML body to Markdown (no-op for list)")
	calendarListCmd.Flags().String("user", "", "View another user's calendar (email or ID)")

	// calendar get flags
	calendarGetCmd.Flags().String("calendar-id", "", "Calendar containing the event (default: primary)")
	calendarGetCmd.Flags().Bool("json", false, "Output as JSON")
	calendarGetCmd.Flags().Bool("markdown", false, "Convert HTML body to Markdown")
	calendarGetCmd.Flags().String("user", "", "View another user's calendar event (email or ID)")

	calendarCmd.AddCommand(calendarListCmd)
	calendarCmd.AddCommand(calendarGetCmd)

	// calendar calendars flags
	calendarCalendarsCmd.Flags().Bool("json", false, "Output as JSON")
	calendarCalendarsCmd.Flags().Bool("markdown", false, "Convert HTML to Markdown (no-op)")
	calendarCmd.AddCommand(calendarCalendarsCmd)

	// calendar events flags
	calendarEventsCmd.Flags().String("calendar-id", "", "Query specific calendar")
	calendarEventsCmd.Flags().Int("top", 0, "Limit number of results")
	calendarEventsCmd.Flags().String("page-token", "", "Pagination token")
	calendarEventsCmd.Flags().Bool("json", false, "Output as JSON")
	calendarEventsCmd.Flags().Bool("markdown", false, "Convert HTML to Markdown (no-op for list)")
	calendarCmd.AddCommand(calendarEventsCmd)

	// calendar respond flags
	calendarRespondCmd.Flags().String("message", "", "Optional response message")
	calendarRespondCmd.Flags().Bool("all", false, "Respond to all pending invitations")
	calendarRespondCmd.Flags().String("ids", "", "Comma-separated event IDs to respond to")
	calendarCmd.AddCommand(calendarRespondCmd)

	// calendar pending flags
	calendarPendingCmd.Flags().Bool("json", false, "Output as JSON")
	calendarPendingCmd.Flags().Bool("markdown", false, "Convert HTML to Markdown (no-op)")
	calendarCmd.AddCommand(calendarPendingCmd)

	// calendar free-busy flags
	calendarFreeBusyCmd.Flags().String("start", "", "Start date/time (default: now)")
	calendarFreeBusyCmd.Flags().String("end", "", "End date/time (default: start + 1 day)")
	calendarFreeBusyCmd.Flags().Bool("json", false, "Output as JSON")
	calendarFreeBusyCmd.Flags().Bool("markdown", false, "Convert HTML to Markdown (no-op)")
	calendarCmd.AddCommand(calendarFreeBusyCmd)

	// calendar find-time flags
	calendarFindTimeCmd.Flags().String("attendees", "", "Comma-separated email addresses (required)")
	calendarFindTimeCmd.Flags().String("duration", "30m", "Meeting duration (e.g., 30m, 1h)")
	calendarFindTimeCmd.Flags().String("start", "", "Search window start (default: tomorrow)")
	calendarFindTimeCmd.Flags().String("end", "", "Search window end (default: start + 7 days)")
	calendarFindTimeCmd.Flags().Int("max-results", 5, "Maximum suggestions to return")
	calendarFindTimeCmd.Flags().Bool("json", false, "Output as JSON")
	calendarFindTimeCmd.Flags().Bool("markdown", false, "Convert HTML to Markdown (no-op)")
	calendarCmd.AddCommand(calendarFindTimeCmd)

	// calendar create flags
	calendarCreateCmd.Flags().String("start", "", "Start date/time (required, accepts natural language)")
	calendarCreateCmd.Flags().String("end", "", "End date/time")
	calendarCreateCmd.Flags().String("duration", "", "Duration (e.g., 30m, 1h) - alternative to --end")
	calendarCreateCmd.Flags().String("attendees", "", "Comma-separated email addresses")
	calendarCreateCmd.Flags().String("location", "", "Location")
	calendarCreateCmd.Flags().String("body", "", "Description/agenda")
	calendarCreateCmd.Flags().Bool("online", false, "Generate Teams meeting link")
	calendarCreateCmd.Flags().Bool("all-day", false, "All-day event")
	calendarCreateCmd.Flags().String("calendar-id", "", "Target calendar")
	calendarCreateCmd.Flags().Bool("json", false, "Output as JSON")
	calendarCreateCmd.Flags().Bool("markdown", false, "Convert HTML to Markdown (no-op)")
	calendarCmd.AddCommand(calendarCreateCmd)
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
