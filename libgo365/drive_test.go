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
