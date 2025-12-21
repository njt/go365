# go365

Microsoft 365 / Microsoft Graph CLI tool (m365 ported to Go)

## Overview

`go365` is a command-line interface tool for accessing Microsoft 365 and Microsoft Graph functionality, ported from the Node.js `m365` CLI. It provides a Go-based implementation with:

- **libgo365**: A library package with common OAuth authentication, credentials management, and Microsoft Graph API client functionality
- **go365 CLI**: A command-line tool with the same calling conventions and subcommands as m365
- **Plugin System**: Git-style plugin support - if you run `go365 foo` and `foo` isn't a built-in subcommand, go365 will look for `go365-foo` in your PATH

## Features

- OAuth 2.0 authentication with Microsoft 365
- Token management and automatic refresh
- Configuration management
- Microsoft Graph API client
- Extensible plugin system for custom commands
- Cross-platform support (Linux, macOS, Windows)

## Installation

### Build from source

```bash
git clone https://github.com/njt/go365.git
cd go365
go build -o go365 ./cmd/go365
```

### Install with go install

```bash
go install github.com/njt/go365/cmd/go365@latest
```

## Quick Start

### 1. Configure your Azure AD application

```bash
go365 config set --tenant-id YOUR_TENANT_ID --client-id YOUR_CLIENT_ID
```

### 2. Login

```bash
go365 login
```

This will provide an authentication URL. Visit the URL, authenticate, and paste the authorization code back into the terminal.

### 3. Check status

```bash
go365 status
```

## Commands

### Built-in Commands

- `go365 login` - Authenticate with Microsoft 365
- `go365 logout` - Sign out and remove stored tokens
- `go365 status` - Show authentication status and user information
- `go365 config set` - Set configuration values (tenant-id, client-id, client-secret)
- `go365 config show` - Display current configuration
- `go365 plugins` - List available plugins in PATH

### Mail Commands

- `go365 mail list` - List email messages from your mailbox
  - `--folder-id` - Specify folder (e.g., inbox, sentitems)
  - `--top` - Number of messages to retrieve (default: 10)
- `go365 mail get <message-id>` - Get a specific email message by ID
- `go365 mail send` - Send an email message
  - `--subject` - Email subject (required)
  - `--to` - Recipient email address(es), comma-separated (required)
  - `--body` - Email body content (required)
  - `--body-type` - Body content type: Text or HTML (default: Text)
  - `--cc` - CC recipient email address(es), comma-separated
  - `--bcc` - BCC recipient email address(es), comma-separated
  - `--save-to-sent-items` - Save message to sent items (default: true)

**Example:**

```bash
# List recent emails
go365 mail list --top 20

# Get a specific email
go365 mail get AAMkAGI2THVSAAA=

# Send an email
go365 mail send --subject "Hello" --to "user@example.com" --body "Hello from go365!"

# Send HTML email with CC
go365 mail send \
  --subject "Important Update" \
  --to "user@example.com" \
  --cc "manager@example.com" \
  --body "<h1>Hello</h1><p>This is an HTML email</p>" \
  --body-type HTML
```

### Plugin System

go365 supports a Git-style plugin system. If you run a command that isn't built-in, go365 will look for an executable named `go365-COMMAND` in your PATH.

**Example:**

Create a plugin script `go365-hello`:

```bash
#!/bin/bash
echo "Hello from go365 plugin!"
echo "Arguments: $@"
```

Make it executable and add it to your PATH:

```bash
chmod +x go365-hello
sudo mv go365-hello /usr/local/bin/
```

Run it:

```bash
go365 hello world
# Output: Hello from go365 plugin!
# Output: Arguments: world
```

## Library Usage (libgo365)

You can use `libgo365` as a library in your own Go applications:

```go
package main

import (
    "context"
    "fmt"
    "github.com/njt/go365/libgo365"
)

func main() {
    // Create authenticator
    cfg := libgo365.AuthConfig{
        TenantID:     "your-tenant-id",
        ClientID:     "your-client-id",
        ClientSecret: "your-client-secret",
        RedirectURL:  "http://localhost:8080/callback",
        Scopes:       []string{"https://graph.microsoft.com/.default"},
    }
    
    auth, err := libgo365.NewAuthenticator(cfg)
    if err != nil {
        panic(err)
    }
    
    ctx := context.Background()
    
    // Get token
    token, err := auth.GetToken(ctx)
    if err != nil {
        panic(err)
    }
    
    // Create Graph API client
    client := libgo365.NewClient(ctx, token, auth.GetConfig())
    
    // Get current user
    user, err := client.GetMe(ctx)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("User: %v\n", user)
}
```

## Configuration

Configuration is stored in `~/.go365/config.json` and includes:

- `tenant_id`: Azure AD tenant ID
- `client_id`: Azure AD application client ID
- `client_secret`: Azure AD application client secret (optional)
- `redirect_url`: OAuth redirect URL (default: http://localhost:8080/callback)
- `scopes`: OAuth scopes (default: https://graph.microsoft.com/.default)

Authentication tokens are stored separately in `~/.go365/token.json`.

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o go365 ./cmd/go365
```

## Architecture

- **cmd/go365**: Main CLI application entry point
- **libgo365**: Core library with authentication and API client
  - `auth.go`: OAuth authentication and token management
  - `client.go`: Microsoft Graph API client
  - `config.go`: Configuration management
- **internal/plugin**: Plugin discovery and execution system

## License

See LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

