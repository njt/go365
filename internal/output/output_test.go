package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestHTMLToMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		contains []string // Substrings that should appear in output
	}{
		{
			name:     "empty string",
			html:     "",
			contains: nil,
		},
		{
			name:     "plain text",
			html:     "Hello world",
			contains: []string{"Hello world"},
		},
		{
			name:     "heading",
			html:     "<h1>Title</h1>",
			contains: []string{"# Title"},
		},
		{
			name:     "paragraph",
			html:     "<p>This is a paragraph.</p>",
			contains: []string{"This is a paragraph."},
		},
		{
			name:     "link",
			html:     `<a href="https://example.com">Example</a>`,
			contains: []string{"[Example]", "(https://example.com)"},
		},
		{
			name:     "bold",
			html:     "<strong>bold text</strong>",
			contains: []string{"**bold text**"},
		},
		{
			name:     "list",
			html:     "<ul><li>Item 1</li><li>Item 2</li></ul>",
			contains: []string{"- Item 1", "- Item 2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HTMLToMarkdown(tt.html)
			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("HTMLToMarkdown(%q) = %q, expected to contain %q", tt.html, result, substr)
				}
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer

	data := map[string]string{"key": "value"}
	err := WriteJSON(&buf, data)
	if err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if result["key"] != "value" {
		t.Errorf("Expected key=value, got key=%s", result["key"])
	}
}

func TestFormatListResponse(t *testing.T) {
	items := []string{"a", "b", "c"}

	t.Run("with next page token", func(t *testing.T) {
		resp := FormatListResponse(items, 3, "token123")
		if !resp.HasMore {
			t.Error("Expected HasMore=true")
		}
		if resp.NextPageToken == nil || *resp.NextPageToken != "token123" {
			t.Errorf("Expected NextPageToken=token123, got %v", resp.NextPageToken)
		}
		if resp.Count != 3 {
			t.Errorf("Expected Count=3, got %d", resp.Count)
		}
	})

	t.Run("without next page token", func(t *testing.T) {
		resp := FormatListResponse(items, 3, "")
		if resp.HasMore {
			t.Error("Expected HasMore=false")
		}
		if resp.NextPageToken != nil {
			t.Errorf("Expected NextPageToken=nil, got %v", resp.NextPageToken)
		}
	})
}

func TestFormatActionResponse(t *testing.T) {
	resp := FormatActionResponse(true, "Success!")
	if !resp.Success {
		t.Error("Expected Success=true")
	}
	if resp.Message != "Success!" {
		t.Errorf("Expected Message=Success!, got %s", resp.Message)
	}
}

func TestPrintNextPageHint(t *testing.T) {
	t.Run("with token", func(t *testing.T) {
		var buf bytes.Buffer
		PrintNextPageHint(&buf, "abc123")
		output := buf.String()
		if !strings.Contains(output, "--page-token abc123") {
			t.Errorf("Expected output to contain --page-token abc123, got %s", output)
		}
	})

	t.Run("empty token", func(t *testing.T) {
		var buf bytes.Buffer
		PrintNextPageHint(&buf, "")
		if buf.Len() > 0 {
			t.Errorf("Expected no output for empty token, got %s", buf.String())
		}
	})
}

func TestConvertBodyToMarkdown(t *testing.T) {
	t.Run("nil body", func(t *testing.T) {
		result := ConvertBodyToMarkdown(nil)
		if result != nil {
			t.Error("Expected nil result for nil input")
		}
	})

	t.Run("HTML body", func(t *testing.T) {
		body := &BodyContent{
			ContentType: "HTML",
			Content:     "<h1>Hello</h1><p>World</p>",
		}
		result := ConvertBodyToMarkdown(body)
		if result.ContentType != "Markdown" {
			t.Errorf("Expected ContentType=Markdown, got %s", result.ContentType)
		}
		if !strings.Contains(result.Content, "# Hello") {
			t.Errorf("Expected markdown heading, got %s", result.Content)
		}
	})

	t.Run("Text body unchanged", func(t *testing.T) {
		body := &BodyContent{
			ContentType: "Text",
			Content:     "Plain text content",
		}
		result := ConvertBodyToMarkdown(body)
		if result.ContentType != "Text" {
			t.Errorf("Expected ContentType=Text, got %s", result.ContentType)
		}
		if result.Content != "Plain text content" {
			t.Errorf("Expected unchanged content, got %s", result.Content)
		}
	})

	t.Run("case insensitive HTML check", func(t *testing.T) {
		body := &BodyContent{
			ContentType: "html",
			Content:     "<b>Bold</b>",
		}
		result := ConvertBodyToMarkdown(body)
		if result.ContentType != "Markdown" {
			t.Errorf("Expected ContentType=Markdown, got %s", result.ContentType)
		}
	})
}
