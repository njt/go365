package libgo365

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

// Message represents an email message from Microsoft Graph
type Message struct {
	ID                   string       `json:"id,omitempty"`
	Subject              string       `json:"subject,omitempty"`
	Body                 *ItemBody    `json:"body,omitempty"`
	BodyPreview          string       `json:"bodyPreview,omitempty"`
	From                 *Recipient   `json:"from,omitempty"`
	ToRecipients         []*Recipient `json:"toRecipients,omitempty"`
	CcRecipients         []*Recipient `json:"ccRecipients,omitempty"`
	BccRecipients        []*Recipient `json:"bccRecipients,omitempty"`
	ReceivedDateTime     *time.Time   `json:"receivedDateTime,omitempty"`
	SentDateTime         *time.Time   `json:"sentDateTime,omitempty"`
	HasAttachments       bool         `json:"hasAttachments,omitempty"`
	Importance           string       `json:"importance,omitempty"`
	IsRead               bool         `json:"isRead,omitempty"`
	IsDraft              bool         `json:"isDraft,omitempty"`
	ConversationID       string       `json:"conversationId,omitempty"`
	InternetMessageID    string       `json:"internetMessageId,omitempty"`
	WebLink              string       `json:"webLink,omitempty"`
}

// ItemBody represents the body of an item
type ItemBody struct {
	ContentType string `json:"contentType,omitempty"`
	Content     string `json:"content,omitempty"`
}

// Recipient represents an email recipient
type Recipient struct {
	EmailAddress *EmailAddress `json:"emailAddress,omitempty"`
}

// EmailAddress represents an email address
type EmailAddress struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"address,omitempty"`
}

// SendMailRequest represents a request to send an email
type SendMailRequest struct {
	Message         *Message `json:"message"`
	SaveToSentItems bool     `json:"saveToSentItems,omitempty"`
}

// MessageList represents a list of messages returned by Graph API
type MessageList struct {
	Value    []*Message `json:"value"`
	NextLink string     `json:"@odata.nextLink,omitempty"`
}

// ListMessagesOptions represents options for listing messages
type ListMessagesOptions struct {
	FolderID  string
	Top       int
	Filter    string
	OrderBy   string
	StartTime *time.Time
	EndTime   *time.Time
}

// ListMessages retrieves messages from the user's mailbox
func (c *Client) ListMessages(ctx context.Context, opts *ListMessagesOptions) ([]*Message, error) {
	path := "/me/messages"
	if opts != nil && opts.FolderID != "" {
		path = fmt.Sprintf("/me/mailFolders/%s/messages", opts.FolderID)
	}

	// Build query parameters
	params := url.Values{}
	params.Set("$top", "100")
	
	if opts != nil {
		if opts.Top > 0 {
			params.Set("$top", fmt.Sprintf("%d", opts.Top))
		}

		filters := []string{}
		if opts.StartTime != nil {
			filters = append(filters, fmt.Sprintf("receivedDateTime ge %s", opts.StartTime.Format(time.RFC3339)))
		}
		if opts.EndTime != nil {
			filters = append(filters, fmt.Sprintf("receivedDateTime lt %s", opts.EndTime.Format(time.RFC3339)))
		}
		if opts.Filter != "" {
			filters = append(filters, opts.Filter)
		}

		if len(filters) > 0 {
			filterStr := filters[0]
			for i := 1; i < len(filters); i++ {
				filterStr += " and " + filters[i]
			}
			params.Set("$filter", filterStr)
		}

		if opts.OrderBy != "" {
			params.Set("$orderby", opts.OrderBy)
		}
	}

	data, err := c.Get(ctx, path+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var messageList MessageList
	if err := json.Unmarshal(data, &messageList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	return messageList.Value, nil
}

// GetMessage retrieves a specific message by ID
func (c *Client) GetMessage(ctx context.Context, messageID string) (*Message, error) {
	if messageID == "" {
		return nil, fmt.Errorf("message ID is required")
	}

	data, err := c.Get(ctx, fmt.Sprintf("/me/messages/%s", messageID))
	if err != nil {
		return nil, err
	}

	var message Message
	if err := json.Unmarshal(data, &message); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return &message, nil
}

// SendMail sends an email message
func (c *Client) SendMail(ctx context.Context, message *Message, saveToSentItems bool) error {
	if message == nil {
		return fmt.Errorf("message is required")
	}

	if message.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if len(message.ToRecipients) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}

	sendRequest := &SendMailRequest{
		Message:         message,
		SaveToSentItems: saveToSentItems,
	}

	_, err := c.Post(ctx, "/me/sendMail", sendRequest)
	return err
}
