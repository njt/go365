# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
# Build
go build -o go365 ./cmd/go365

# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific package tests
go test ./libgo365/...
go test ./internal/plugin/...
go test ./internal/output/...
```

## Architecture

**go365** is a Microsoft 365 / Microsoft Graph CLI tool ported from Node.js m365 to Go.

```
cmd/go365/main.go     - CLI entry point, all subcommands (login, logout, status, config, mail, calendar, plugins)
libgo365/             - Reusable library for embedding in other Go projects
  auth.go             - OAuth 2.0 via MSAL, device code flow, token caching (~/.go365/msal_cache.bin)
  client.go           - Graph API HTTP client (GET/POST/PUT/DELETE)
  config.go           - Config management (~/.go365/config.json)
  mail.go             - Email operations (list, get, send) with pagination support
  calendar.go         - Calendar operations (list events, get event) with natural language dates
internal/dateparse/   - Natural language date parsing (uses tj/go-naturaldate)
internal/output/      - Agent-friendly output formatting (JSON, Markdown conversion)
internal/plugin/      - Git-style plugin system: "go365 foo" looks for "go365-foo" in PATH
examples/whoami/      - Example plugin demonstrating libgo365 usage
```

## Key Patterns

**Authentication flow**: ConfigManager.Load() → NewAuthenticator(cfg) → LoginWithDeviceCode() or GetAccessToken() → NewClient(token)

**Graph API calls**: Client wraps HTTP with bearer token. All methods take context.Context for cancellation.

**Error wrapping**: Use `fmt.Errorf("context: %w", err)` pattern throughout.

**File permissions**: Token cache and config use 0600 (user-only).

## Agent-Friendly Output Flags

All list/get commands support these flags for agent consumption:

| Flag | Purpose |
|------|---------|
| `--json` | Output as JSON matching Graph API structure |
| `--markdown` | Convert HTML body content to markdown (reduces tokens) |
| `--skip N` | Skip first N items (offset-based pagination) |
| `--page-token <token>` | Continue from previous response (cursor-based pagination) |

**Design principles:**
- Flags are composable: `--json --markdown` returns JSON with markdown-converted body
- Silent no-ops: `--markdown` on commands without body content does nothing (no error)
- Pagination: `--page-token` takes precedence over `--skip` if both specified
- Consistency: All commands accept these flags even if some are no-ops

**JSON output for lists** includes Graph API structure:
```json
{
  "value": [...],
  "@odata.count": 50,
  "hasMore": true,
  "nextPageToken": "..."
}
```

**New commands** (teams, calendar, files, etc.) should implement the same flags for consistency.

## CLI Structure

Uses spf13/cobra. Each subcommand (login, logout, status, config, mail, calendar, plugins) defined in main.go. Unknown commands trigger plugin lookup.

## Calendar Command

`calendar list` accepts natural language dates via [tj/go-naturaldate](https://github.com/tj/go-naturaldate):
- `today`, `tomorrow`, `yesterday`
- `next week`, `last month`, `next Tuesday`
- `in 3 days`, `5 days ago`
- ISO 8601: `2025-01-15` or `2025-01-15T09:00:00`

```bash
go365 calendar list                              # Today
go365 calendar list --days 7                     # Next 7 days
go365 calendar list --start "next monday" --end "friday"
go365 calendar list --all-calendars --json       # All calendars, JSON output
```

## Dependencies

- `github.com/AzureAD/microsoft-authentication-library-for-go` - MSAL for OAuth
- `github.com/spf13/cobra` - CLI framework
- `github.com/JohannesKaufmann/html-to-markdown/v2` - HTML to Markdown conversion
- `github.com/tj/go-naturaldate` - Natural language date parsing
