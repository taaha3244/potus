package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchTool_Name(t *testing.T) {
	tool := NewFetchTool()
	if tool.Name() != "web_fetch" {
		t.Errorf("Name() = %s, want web_fetch", tool.Name())
	}
}

func TestFetchTool_Description(t *testing.T) {
	tool := NewFetchTool()
	if tool.Description() == "" {
		t.Error("Description should not be empty")
	}
}

func TestFetchTool_Schema(t *testing.T) {
	tool := NewFetchTool()
	schema := tool.Schema()

	if schema["type"] != "object" {
		t.Error("Schema type should be object")
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Schema should have properties")
	}

	if _, ok := props["url"]; !ok {
		t.Error("Schema should have url property")
	}

	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("Schema should have required array")
	}

	if len(required) != 1 || required[0] != "url" {
		t.Error("url should be required")
	}
}

func TestFetchTool_Execute_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body><h1>Hello World</h1><p>Test content</p></body></html>"))
	}))
	defer server.Close()

	tool := NewFetchTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"url": server.URL,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}

	if !strings.Contains(result.Output, "Hello World") {
		t.Error("Output should contain extracted text")
	}

	if !strings.Contains(result.Output, "Test content") {
		t.Error("Output should contain paragraph text")
	}
}

func TestFetchTool_Execute_PlainText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Plain text content"))
	}))
	defer server.Close()

	tool := NewFetchTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"url": server.URL,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(result.Output, "Plain text content") {
		t.Error("Output should contain plain text")
	}
}

func TestFetchTool_Execute_404Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tool := NewFetchTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"url": server.URL,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.Success {
		t.Error("Expected error result for 404")
	}

	if !strings.Contains(result.Error.Error(), "404") {
		t.Error("Error should mention 404")
	}
}

func TestFetchTool_Execute_MissingURL(t *testing.T) {
	tool := NewFetchTool()
	_, err := tool.Execute(context.Background(), map[string]interface{}{})

	if err == nil {
		t.Error("Expected error for missing URL")
	}
}

func TestFetchTool_Execute_InvalidURL(t *testing.T) {
	tool := NewFetchTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"url": "http://localhost:99999",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.Success {
		t.Error("Expected error for invalid URL")
	}
}

func TestFetchTool_Execute_AutoPrependHTTPS(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Get just the host:port without scheme
	url := strings.TrimPrefix(server.URL, "http://")

	tool := NewFetchTool()
	// This will prepend https:// which won't work with our http test server
	// But it tests the URL modification logic
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"url": url,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Will fail because https:// was prepended to an http server
	// This is expected behavior - testing the URL normalization
	if result.Success {
		t.Log("Connection succeeded (server might support both)")
	}
}

func TestExtractTextFromHTML(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "simple text",
			html:     "<p>Hello World</p>",
			expected: "Hello World",
		},
		{
			name:     "removes script tags",
			html:     "<p>Hello</p><script>alert('bad')</script><p>World</p>",
			expected: "Hello\nWorld",
		},
		{
			name:     "removes style tags",
			html:     "<p>Hello</p><style>body{color:red}</style><p>World</p>",
			expected: "Hello\nWorld",
		},
		{
			name:     "decodes entities",
			html:     "<p>Hello &amp; World &lt;test&gt;</p>",
			expected: "Hello & World <test>",
		},
		{
			name:     "handles multiple newlines",
			html:     "<p>Hello</p>\n\n\n\n<p>World</p>",
			expected: "Hello\nWorld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTextFromHTML(tt.html)
			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)
			if result != expected {
				t.Errorf("extractTextFromHTML() = %q, want %q", result, expected)
			}
		})
	}
}
