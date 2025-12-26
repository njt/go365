# OneDrive Command Design

## Overview

Full OneDrive file management suite for go365, following Unix conventions for file operations.

## Command Structure

```
go365 drive                     # Show drive info (quota, owner)
go365 drive ls [path]           # List folder contents (default: root)
go365 drive info <path-or-id>   # Get item metadata
go365 drive cat <path-or-id>    # Output file to stdout
go365 drive get <path-or-id>    # Download to --output file
go365 drive put <local> [remote]# Upload file
go365 drive mkdir <path>        # Create folder
go365 drive rm <path-or-id>     # Delete item
go365 drive mv <src> <dest>     # Move or rename
go365 drive cp <src> <dest>     # Copy item
go365 drive find <query>        # Search across drive
go365 drive share <path-or-id>  # Create/manage share links
```

### Path Conventions

- `/` = root
- `/Documents/file.txt` = path from root
- `ABC123` (no slash) = item ID directly
- Paths are case-insensitive (OneDrive behavior)

### Multi-Drive Access (Flags)

```
go365 drive ls                          # Your OneDrive (default)
go365 drive ls --user pete              # Pete's OneDrive (admin)
go365 drive ls --site "Team Site"       # SharePoint site
go365 drive ls --shared                 # Shared with me
```

### Consistent Flags

- `--json` - structured output
- `--user <email>` - access another user's drive (admin)
- `--site <name>` - access SharePoint document library
- `--shared` - access items shared with you

## Output Format

### Human-readable (default)

Detailed format with IDs visible for scripting:

```
drwx  Documents/              2024-12-20         -  ABC123
-rw-  Budget.xlsx             2024-12-15    245 KB  DEF456
-rw-  Report.pdf              2024-12-26    1.2 MB  GHI789
```

### JSON (`--json`)

Full metadata including Graph API fields.

## Library Structure (libgo365/drive.go)

### Types

```go
type DriveItem struct {
    ID           string
    Name         string
    Size         int64
    IsFolder     bool
    MimeType     string
    CreatedAt    time.Time
    ModifiedAt   time.Time
    WebURL       string
    DownloadURL  string  // For files, direct download link
    ParentPath   string  // e.g., "/Documents/Projects"
}

type Drive struct {
    ID        string
    Name      string
    DriveType string  // "personal", "business", "sharepoint"
    Owner     string
    Quota     *DriveQuota
}

type DriveQuota struct {
    Total     int64
    Used      int64
    Remaining int64
}
```

### Methods

```go
GetDrive(ctx) (*Drive, error)
ListItems(ctx, path string, opts *ListItemsOptions) (*ListItemsResponse, error)
GetItem(ctx, pathOrID string) (*DriveItem, error)
DownloadItem(ctx, pathOrID string, w io.Writer) error
UploadItem(ctx, parentPath, filename string, r io.Reader, size int64) (*DriveItem, error)
CreateFolder(ctx, parentPath, name string) (*DriveItem, error)
DeleteItem(ctx, pathOrID string) error
MoveItem(ctx, pathOrID, destPath string) (*DriveItem, error)
CopyItem(ctx, pathOrID, destPath string) (*DriveItem, error)
SearchItems(ctx, query string) ([]*DriveItem, error)
CreateShareLink(ctx, pathOrID string, opts *ShareOptions) (*ShareLink, error)
```

## Path Resolution

### Path to API Endpoint Mapping

| Input | API Endpoint |
|-------|--------------|
| `/` | `/me/drive/root/children` |
| `/Documents` | `/me/drive/root:/Documents:/children` |
| `/Documents/file.txt` | `/me/drive/root:/Documents/file.txt` |
| `ABC123` (no `/`) | `/me/drive/items/ABC123` |

### Flag Prefixes

| Flag | API Prefix |
|------|------------|
| (none) | `/me/drive` |
| `--user pete` | `/users/pete@domain/drive` |
| `--site "Team"` | `/sites/{site-id}/drive` (requires lookup) |
| `--shared` | `/me/drive/sharedWithMe` |

### Large File Uploads

- Graph API requires upload sessions for files over 4MB
- `UploadItem` auto-detects size and uses resumable upload when needed
- Progress callback for CLI progress bar

## Implementation Phases

### Phase 1 - Read Operations (MVP)

- `drive` - show drive info/quota
- `drive ls [path]` - list folder
- `drive info <path>` - item metadata
- `drive cat <path>` - stream to stdout
- `drive get <path>` - download to file
- `drive find <query>` - search

### Phase 2 - Write Operations

- `drive put <local> [remote]` - upload (with large file support)
- `drive mkdir <path>` - create folder
- `drive rm <path>` - delete
- `drive mv <src> <dest>` - move/rename
- `drive cp <src> <dest>` - copy

### Phase 3 - Sharing

- `drive share <path>` - create anonymous view link
- `drive share <path> --edit` - create edit link
- `drive share <path> --to pete` - share with specific user
- `drive share <path> --list` - show existing permissions
- `drive share <path> --revoke <id>` - remove permission

### All Phases Include

- `--json` output
- `--user`, `--site`, `--shared` flags
- Consistent error messages
- Pagination support where applicable
