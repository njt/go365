package libgo365

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestListMessages(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Authorization header with Bearer token")
		}

		// Check if the path is correct
		expectedPath := "/me/messages"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		// Return mock response
		response := MessageList{
			Value: []*Message{
				{
					ID:      "msg1",
					Subject: "Test Message 1",
					Body: &ItemBody{
						ContentType: "Text",
						Content:     "This is test message 1",
					},
					From: &Recipient{
						EmailAddress: &EmailAddress{
							Name:    "John Doe",
							Address: "john@example.com",
						},
					},
				},
				{
					ID:      "msg2",
					Subject: "Test Message 2",
					Body: &ItemBody{
						ContentType: "HTML",
						Content:     "<p>This is test message 2</p>",
					},
					From: &Recipient{
						EmailAddress: &EmailAddress{
							Name:    "Jane Smith",
							Address: "jane@example.com",
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server
	client := &Client{
		httpClient:  &http.Client{},
		baseURL:     server.URL,
		accessToken: "test-token",
	}

	ctx := context.Background()
	messages, err := client.ListMessages(ctx, nil)

	if err != nil {
		t.Fatalf("ListMessages failed: %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}

	if messages[0].Subject != "Test Message 1" {
		t.Errorf("Expected subject 'Test Message 1', got '%s'", messages[0].Subject)
	}

	if messages[1].Subject != "Test Message 2" {
		t.Errorf("Expected subject 'Test Message 2', got '%s'", messages[1].Subject)
	}
}

func TestListMessagesWithFolder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/me/mailFolders/inbox/messages"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		response := MessageList{
			Value: []*Message{
				{
					ID:      "msg1",
					Subject: "Inbox Message",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient:  &http.Client{},
		baseURL:     server.URL,
		accessToken: "test-token",
	}

	ctx := context.Background()
	opts := &ListMessagesOptions{
		FolderID: "inbox",
	}
	messages, err := client.ListMessages(ctx, opts)

	if err != nil {
		t.Fatalf("ListMessages failed: %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}
}

func TestListMessagesWithTimeFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the filter query parameter is present
		filterParam := r.URL.Query().Get("$filter")
		if filterParam == "" {
			t.Error("Expected filter parameter")
		}

		response := MessageList{
			Value: []*Message{},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient:  &http.Client{},
		baseURL:     server.URL,
		accessToken: "test-token",
	}

	ctx := context.Background()
	startTime := time.Now().Add(-24 * time.Hour)
	opts := &ListMessagesOptions{
		StartTime: &startTime,
	}
	_, err := client.ListMessages(ctx, opts)

	if err != nil {
		t.Fatalf("ListMessages failed: %v", err)
	}
}

func TestGetMessage(t *testing.T) {
	messageID := "test-message-id"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		expectedPath := fmt.Sprintf("/me/messages/%s", messageID)
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		message := Message{
			ID:      messageID,
			Subject: "Test Message",
			Body: &ItemBody{
				ContentType: "Text",
				Content:     "Test content",
			},
			From: &Recipient{
				EmailAddress: &EmailAddress{
					Name:    "Sender Name",
					Address: "sender@example.com",
				},
			},
			ToRecipients: []*Recipient{
				{
					EmailAddress: &EmailAddress{
						Name:    "Recipient Name",
						Address: "recipient@example.com",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(message)
	}))
	defer server.Close()

	client := &Client{
		httpClient:  &http.Client{},
		baseURL:     server.URL,
		accessToken: "test-token",
	}

	ctx := context.Background()
	message, err := client.GetMessage(ctx, messageID)

	if err != nil {
		t.Fatalf("GetMessage failed: %v", err)
	}

	if message.ID != messageID {
		t.Errorf("Expected message ID %s, got %s", messageID, message.ID)
	}

	if message.Subject != "Test Message" {
		t.Errorf("Expected subject 'Test Message', got '%s'", message.Subject)
	}

	if message.From.EmailAddress.Address != "sender@example.com" {
		t.Errorf("Expected sender 'sender@example.com', got '%s'", message.From.EmailAddress.Address)
	}
}

func TestGetMessageEmptyID(t *testing.T) {
	client := &Client{
		httpClient:  &http.Client{},
		baseURL:     "http://localhost",
		accessToken: "test-token",
	}

	ctx := context.Background()
	_, err := client.GetMessage(ctx, "")

	if err == nil {
		t.Error("Expected error for empty message ID")
	}
}

func TestSendMail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		expectedPath := "/me/sendMail"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Expected Content-Type: application/json")
		}

		// Parse the request body
		var sendRequest SendMailRequest
		if err := json.NewDecoder(r.Body).Decode(&sendRequest); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify the message content
		if sendRequest.Message.Subject != "Test Subject" {
			t.Errorf("Expected subject 'Test Subject', got '%s'", sendRequest.Message.Subject)
		}

		if len(sendRequest.Message.ToRecipients) != 1 {
			t.Errorf("Expected 1 recipient, got %d", len(sendRequest.Message.ToRecipients))
		}

		if sendRequest.Message.ToRecipients[0].EmailAddress.Address != "test@example.com" {
			t.Errorf("Expected recipient 'test@example.com', got '%s'",
				sendRequest.Message.ToRecipients[0].EmailAddress.Address)
		}

		if !sendRequest.SaveToSentItems {
			t.Error("Expected SaveToSentItems to be true")
		}

		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := &Client{
		httpClient:  &http.Client{},
		baseURL:     server.URL,
		accessToken: "test-token",
	}

	ctx := context.Background()
	message := &Message{
		Subject: "Test Subject",
		Body: &ItemBody{
			ContentType: "Text",
			Content:     "Test body content",
		},
		ToRecipients: []*Recipient{
			{
				EmailAddress: &EmailAddress{
					Address: "test@example.com",
				},
			},
		},
	}

	err := client.SendMail(ctx, message, true)

	if err != nil {
		t.Fatalf("SendMail failed: %v", err)
	}
}

func TestSendMailNilMessage(t *testing.T) {
	client := &Client{
		httpClient:  &http.Client{},
		baseURL:     "http://localhost",
		accessToken: "test-token",
	}

	ctx := context.Background()
	err := client.SendMail(ctx, nil, false)

	if err == nil {
		t.Error("Expected error for nil message")
	}
}

func TestSendMailEmptySubject(t *testing.T) {
	client := &Client{
		httpClient:  &http.Client{},
		baseURL:     "http://localhost",
		accessToken: "test-token",
	}

	ctx := context.Background()
	message := &Message{
		Subject: "",
		ToRecipients: []*Recipient{
			{EmailAddress: &EmailAddress{Address: "test@example.com"}},
		},
	}

	err := client.SendMail(ctx, message, false)

	if err == nil {
		t.Error("Expected error for empty subject")
	}
}

func TestSendMailNoRecipients(t *testing.T) {
	client := &Client{
		httpClient:  &http.Client{},
		baseURL:     "http://localhost",
		accessToken: "test-token",
	}

	ctx := context.Background()
	message := &Message{
		Subject:      "Test",
		ToRecipients: []*Recipient{},
	}

	err := client.SendMail(ctx, message, false)

	if err == nil {
		t.Error("Expected error for no recipients")
	}
}

func TestSendMailWithCcAndBcc(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sendRequest SendMailRequest
		if err := json.NewDecoder(r.Body).Decode(&sendRequest); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		if len(sendRequest.Message.CcRecipients) != 1 {
			t.Errorf("Expected 1 CC recipient, got %d", len(sendRequest.Message.CcRecipients))
		}

		if len(sendRequest.Message.BccRecipients) != 1 {
			t.Errorf("Expected 1 BCC recipient, got %d", len(sendRequest.Message.BccRecipients))
		}

		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := &Client{
		httpClient:  &http.Client{},
		baseURL:     server.URL,
		accessToken: "test-token",
	}

	ctx := context.Background()
	message := &Message{
		Subject: "Test Subject",
		Body: &ItemBody{
			ContentType: "Text",
			Content:     "Test body",
		},
		ToRecipients: []*Recipient{
			{EmailAddress: &EmailAddress{Address: "to@example.com"}},
		},
		CcRecipients: []*Recipient{
			{EmailAddress: &EmailAddress{Address: "cc@example.com"}},
		},
		BccRecipients: []*Recipient{
			{EmailAddress: &EmailAddress{Address: "bcc@example.com"}},
		},
	}

	err := client.SendMail(ctx, message, false)

	if err != nil {
		t.Fatalf("SendMail failed: %v", err)
	}
}
