# Agent-Friendly Output Flags Design

## Summary

Add `--json`, `--markdown`, and pagination flags to make go365 commands useful for LLM agents. These flags reduce token usage and enable programmatic consumption.

## Flags

All list/get commands support these flags:

| Flag | Purpose |
|------|---------|
| `--json` | Output as JSON matching Graph API structure |
| `--markdown` | Convert HTML body content to markdown (no-op if no body) |
| `--skip N` | Skip first N items (offset-based pagination) |
| `--page-token <token>` | Continue from previous response (cursor-based pagination) |

### Flag Behavior

- **Composable**: `--json` and `--markdown` can be combined
- **Silent no-ops**: `--markdown` on commands without body content does nothing (no error)
- **Precedence**: `--page-token` takes precedence over `--skip` if both specified

## Output Formats

### `mail list`

**Human-readable (default):**
```
ID: AAMkAGI2...
Subject: Weekly Report
From: John Smith <john@example.com>
Received: 2025-01-15T10:30:00Z
---
ID: AAMkAGI3...
Subject: Meeting Notes
From: Jane Doe <jane@example.com>
Received: 2025-01-15T09:15:00Z
---

Next page: --page-token eyJza2lwIjo1MH0...
```

The "Next page" line only appears when more results exist.

**JSON (`--json`):**
```json
{
  "value": [
    {
      "id": "AAMkAGI2...",
      "subject": "Weekly Report",
      "from": {
        "emailAddress": {
          "name": "John Smith",
          "address": "john@example.com"
        }
      },
      "receivedDateTime": "2025-01-15T10:30:00Z"
    }
  ],
  "@odata.count": 50,
  "hasMore": true,
  "nextPageToken": "eyJza2lwIjo1MH0..."
}
```

### `mail get`

**Human-readable + `--markdown`:**
```
ID: AAMkAGI2...
Subject: Weekly Report
From: John Smith <john@example.com>
To: team@example.com
Received: 2025-01-15T10:30:00Z

Body (Markdown):
# Weekly Report

Here are the highlights from this week...

- Item one
- Item two
```

**JSON + `--markdown`:**
```json
{
  "id": "AAMkAGI2...",
  "subject": "Weekly Report",
  "from": {
    "emailAddress": {
      "name": "John Smith",
      "address": "john@example.com"
    }
  },
  "body": {
    "contentType": "Markdown",
    "content": "# Weekly Report\n\nHere are the highlights..."
  },
  "receivedDateTime": "2025-01-15T10:30:00Z"
}
```

### `mail send`

**Human-readable (default):**
```
Message sent successfully!
```

**JSON (`--json`):**
```json
{
  "success": true,
  "message": "Message sent successfully"
}
```

`--markdown` is a no-op for send.

## Pagination

### Token Extraction

Graph API returns pagination via `@odata.nextLink`:
```json
{
  "@odata.nextLink": "https://graph.microsoft.com/v1.0/me/messages?$skip=50&$top=50"
}
```

We extract the `$skiptoken` or `$skip` parameter and expose it as `nextPageToken`. On subsequent requests, we append it back to the Graph API query.

### Agent Workflow

```bash
# First page
response=$(go365 mail list --top 50 --json)
token=$(echo "$response" | jq -r '.nextPageToken // empty')

# Continue until no more pages
while [ -n "$token" ]; do
  response=$(go365 mail list --top 50 --json --page-token "$token")
  token=$(echo "$response" | jq -r '.nextPageToken // empty')
done
```

## Implementation

### New Dependency

```
github.com/JohannesKaufmann/html-to-markdown/v2
```

Pure Go, ~25 MB/s throughput, actively maintained.

### Files to Modify

| File | Changes |
|------|---------|
| `cmd/go365/main.go` | Add flags to mail commands, output formatting logic |
| `libgo365/mail.go` | Add `Skip`, `PageToken` to `ListMessagesOptions`; parse `@odata.nextLink` |
| `libgo365/mail_test.go` | Test pagination parsing, token extraction |
| `CLAUDE.md` | Document `--json`/`--markdown` pattern for future commands |

### New Package

```
internal/output/output.go
```

Handles JSON/human formatting and markdown conversion. Keeps formatting logic reusable for future commands (teams, calendar, etc.).

## Testing

### Unit Tests

**`libgo365/mail_test.go`:**
- Parse `@odata.nextLink` → extract token
- Round-trip: token → query param → same results
- Handle missing `@odata.nextLink` (last page)

**`internal/output/output_test.go`:**
- HTML → Markdown conversion (basic tags, tables, links)
- JSON marshaling with Graph API structure
- Edge cases: empty body, Text vs HTML contentType

### Manual Verification

- `mail list --json` returns valid JSON
- `mail list --top 5` then `--page-token` continues correctly
- `mail get <id> --markdown` converts body
- `--json --markdown` composes correctly

## Future Commands

All commands that return structured data (teams, calendar, files, etc.) should implement these same flags for consistency. Document this in CLAUDE.md as a pattern.
