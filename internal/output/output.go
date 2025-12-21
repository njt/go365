// Package output provides formatting utilities for agent-friendly CLI output.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
)

// Options controls output formatting.
type Options struct {
	JSON     bool // Output as JSON
	Markdown bool // Convert HTML body content to markdown
}

// HTMLToMarkdown converts HTML content to Markdown.
// Returns the original content if conversion fails or content is empty.
func HTMLToMarkdown(html string) string {
	if html == "" {
		return ""
	}

	md, err := htmltomarkdown.ConvertString(html)
	if err != nil {
		// Fall back to original content on error
		return html
	}

	return strings.TrimSpace(md)
}

// ListResponse represents a paginated list response matching Graph API structure.
type ListResponse struct {
	Value         any     `json:"value"`
	Count         int     `json:"@odata.count,omitempty"`
	HasMore       bool    `json:"hasMore"`
	NextPageToken *string `json:"nextPageToken,omitempty"`
}

// ActionResponse represents the response from an action command (e.g., send).
type ActionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// WriteJSON writes a value as JSON to the writer.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// WriteJSONString returns a value as a JSON string.
func WriteJSONString(v any) (string, error) {
	var sb strings.Builder
	if err := WriteJSON(&sb, v); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// FormatListResponse creates a ListResponse with the given values.
func FormatListResponse(value any, count int, nextPageToken string) *ListResponse {
	resp := &ListResponse{
		Value:   value,
		Count:   count,
		HasMore: nextPageToken != "",
	}
	if nextPageToken != "" {
		resp.NextPageToken = &nextPageToken
	}
	return resp
}

// FormatActionResponse creates an ActionResponse.
func FormatActionResponse(success bool, message string) *ActionResponse {
	return &ActionResponse{
		Success: success,
		Message: message,
	}
}

// PrintNextPageHint prints the pagination hint for human-readable output.
func PrintNextPageHint(w io.Writer, token string) {
	if token != "" {
		fmt.Fprintf(w, "\nNext page: --page-token %s\n", token)
	}
}

// BodyContent represents message body content with optional markdown conversion.
type BodyContent struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

// ConvertBodyToMarkdown converts HTML body content to Markdown.
// Returns a new BodyContent with ContentType set to "Markdown".
// If the original content type is not HTML, returns unchanged.
func ConvertBodyToMarkdown(body *BodyContent) *BodyContent {
	if body == nil {
		return nil
	}

	// Only convert HTML content
	if !strings.EqualFold(body.ContentType, "HTML") {
		return body
	}

	return &BodyContent{
		ContentType: "Markdown",
		Content:     HTMLToMarkdown(body.Content),
	}
}
