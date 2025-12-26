package libgo365

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
)

// Drive represents a OneDrive drive
type Drive struct {
	ID        string      `json:"id,omitempty"`
	Name      string      `json:"name,omitempty"`
	DriveType string      `json:"driveType,omitempty"` // personal, business, documentLibrary
	Owner     *Identity   `json:"owner,omitempty"`
	Quota     *DriveQuota `json:"quota,omitempty"`
	WebURL    string      `json:"webUrl,omitempty"`
}

// DriveQuota represents storage quota information
type DriveQuota struct {
	Total     int64  `json:"total,omitempty"`
	Used      int64  `json:"used,omitempty"`
	Remaining int64  `json:"remaining,omitempty"`
	State     string `json:"state,omitempty"` // normal, nearing, critical, exceeded
}

// Identity represents an identity (user, application, etc.)
type Identity struct {
	User *IdentityUser `json:"user,omitempty"`
}

// IdentityUser represents a user identity
type IdentityUser struct {
	ID          string `json:"id,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Email       string `json:"email,omitempty"`
}

// DriveItem represents a file or folder in OneDrive
type DriveItem struct {
	ID                   string         `json:"id,omitempty"`
	Name                 string         `json:"name,omitempty"`
	Size                 int64          `json:"size,omitempty"`
	CreatedDateTime      *time.Time     `json:"createdDateTime,omitempty"`
	LastModifiedDateTime *time.Time     `json:"lastModifiedDateTime,omitempty"`
	WebURL               string         `json:"webUrl,omitempty"`
	Folder               *FolderFacet   `json:"folder,omitempty"`
	File                 *FileFacet     `json:"file,omitempty"`
	ParentReference      *ItemReference `json:"parentReference,omitempty"`
	DownloadURL          string         `json:"@microsoft.graph.downloadUrl,omitempty"`
}

// IsFolder returns true if the item is a folder
func (d *DriveItem) IsFolder() bool {
	return d.Folder != nil
}

// FolderFacet indicates an item is a folder
type FolderFacet struct {
	ChildCount int32 `json:"childCount,omitempty"`
}

// FileFacet indicates an item is a file
type FileFacet struct {
	MimeType string  `json:"mimeType,omitempty"`
	Hashes   *Hashes `json:"hashes,omitempty"`
}

// Hashes contains hash values for a file
type Hashes struct {
	SHA1Hash     string `json:"sha1Hash,omitempty"`
	QuickXorHash string `json:"quickXorHash,omitempty"`
}

// ItemReference contains information about a parent item
type ItemReference struct {
	DriveID   string `json:"driveId,omitempty"`
	DriveType string `json:"driveType,omitempty"`
	ID        string `json:"id,omitempty"`
	Path      string `json:"path,omitempty"`
}

// DriveItemList represents a list of drive items from Graph API
type DriveItemList struct {
	Value    []*DriveItem `json:"value"`
	NextLink string       `json:"@odata.nextLink,omitempty"`
}

// ListItemsOptions represents options for listing drive items
type ListItemsOptions struct {
	UserID    string // Access another user's drive
	SiteID    string // Access SharePoint site drive
	DriveID   string // Access specific drive by ID
	Shared    bool   // Access shared items
	Top       int
	PageToken string
	OrderBy   string
}

// ListItemsResponse represents the response from ListItems
type ListItemsResponse struct {
	Items         []*DriveItem
	Count         int
	HasMore       bool
	NextPageToken string
}

// GetDriveOptions represents options for getting a drive
type GetDriveOptions struct {
	UserID  string // Access another user's drive
	SiteID  string // Access SharePoint site drive
	DriveID string // Access specific drive by ID
}

// GetDrive retrieves drive information
func (c *Client) GetDrive(ctx context.Context, opts *GetDriveOptions) (*Drive, error) {
	path := "/me/drive"
	if opts != nil {
		if opts.DriveID != "" {
			path = fmt.Sprintf("/drives/%s", opts.DriveID)
		} else if opts.UserID != "" {
			path = fmt.Sprintf("/users/%s/drive", opts.UserID)
		} else if opts.SiteID != "" {
			path = fmt.Sprintf("/sites/%s/drive", opts.SiteID)
		}
	}

	data, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var drive Drive
	if err := json.Unmarshal(data, &drive); err != nil {
		return nil, fmt.Errorf("failed to unmarshal drive: %w", err)
	}

	return &drive, nil
}

// Silence unused import warnings - will be used in later tasks
var (
	_ = io.EOF
	_ = url.Values{}
	_ = strings.TrimSpace
	_ = time.Now
)
