package libgo365

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetDrive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/me/drive" {
			t.Errorf("Expected path /me/drive, got %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		drive := Drive{
			ID:        "drive123",
			Name:      "OneDrive",
			DriveType: "personal",
			Quota: &DriveQuota{
				Total:     1099511627776, // 1 TB
				Used:      536870912000,  // 500 GB
				Remaining: 562640715776,
				State:     "normal",
			},
		}
		json.NewEncoder(w).Encode(drive)
	}))
	defer server.Close()

	client := &Client{
		httpClient:  server.Client(),
		baseURL:     server.URL,
		accessToken: "test-token",
	}

	ctx := context.Background()
	drive, err := client.GetDrive(ctx, nil)
	if err != nil {
		t.Fatalf("GetDrive failed: %v", err)
	}

	if drive.ID != "drive123" {
		t.Errorf("Expected ID 'drive123', got '%s'", drive.ID)
	}
	if drive.Quota.Total != 1099511627776 {
		t.Errorf("Expected quota total 1099511627776, got %d", drive.Quota.Total)
	}
}

func TestListItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/me/drive/root/children"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		items := DriveItemList{
			Value: []*DriveItem{
				{ID: "folder1", Name: "Documents", Folder: &FolderFacet{ChildCount: 5}},
				{ID: "file1", Name: "report.pdf", Size: 1024, File: &FileFacet{MimeType: "application/pdf"}},
			},
		}
		json.NewEncoder(w).Encode(items)
	}))
	defer server.Close()

	client := &Client{
		httpClient:  server.Client(),
		baseURL:     server.URL,
		accessToken: "test-token",
	}

	ctx := context.Background()
	resp, err := client.ListItems(ctx, "/", nil)
	if err != nil {
		t.Fatalf("ListItems failed: %v", err)
	}

	if len(resp.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(resp.Items))
	}
	if resp.Items[0].Name != "Documents" {
		t.Errorf("Expected first item 'Documents', got '%s'", resp.Items[0].Name)
	}
	if !resp.Items[0].IsFolder() {
		t.Error("Expected first item to be a folder")
	}
}

func TestListItemsWithPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/me/drive/root:/Documents:/children"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		items := DriveItemList{Value: []*DriveItem{}}
		json.NewEncoder(w).Encode(items)
	}))
	defer server.Close()

	client := &Client{
		httpClient:  server.Client(),
		baseURL:     server.URL,
		accessToken: "test-token",
	}

	ctx := context.Background()
	_, err := client.ListItems(ctx, "/Documents", nil)
	if err != nil {
		t.Fatalf("ListItems failed: %v", err)
	}
}

func TestGetItem(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/me/drive/root:/Documents/report.pdf:"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		item := DriveItem{
			ID:   "file123",
			Name: "report.pdf",
			Size: 1024,
			File: &FileFacet{MimeType: "application/pdf"},
		}
		json.NewEncoder(w).Encode(item)
	}))
	defer server.Close()

	client := &Client{
		httpClient:  server.Client(),
		baseURL:     server.URL,
		accessToken: "test-token",
	}

	ctx := context.Background()
	item, err := client.GetItem(ctx, "/Documents/report.pdf", nil)
	if err != nil {
		t.Fatalf("GetItem failed: %v", err)
	}

	if item.Name != "report.pdf" {
		t.Errorf("Expected name 'report.pdf', got '%s'", item.Name)
	}
}

func TestGetItemByID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/me/drive/items/ABC123"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		item := DriveItem{ID: "ABC123", Name: "file.txt"}
		json.NewEncoder(w).Encode(item)
	}))
	defer server.Close()

	client := &Client{
		httpClient:  server.Client(),
		baseURL:     server.URL,
		accessToken: "test-token",
	}

	ctx := context.Background()
	item, err := client.GetItem(ctx, "ABC123", nil)
	if err != nil {
		t.Fatalf("GetItem failed: %v", err)
	}

	if item.ID != "ABC123" {
		t.Errorf("Expected ID 'ABC123', got '%s'", item.ID)
	}
}
