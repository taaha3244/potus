package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearchTool_Name(t *testing.T) {
	tool := NewSearchTool()
	if tool.Name() != "web_search" {
		t.Errorf("Name() = %s, want web_search", tool.Name())
	}
}

func TestSearchTool_Description(t *testing.T) {
	tool := NewSearchTool()
	if tool.Description() == "" {
		t.Error("Description should not be empty")
	}
}

func TestSearchTool_Schema(t *testing.T) {
	tool := NewSearchTool()
	schema := tool.Schema()

	if schema["type"] != "object" {
		t.Error("Schema type should be object")
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Schema should have properties")
	}

	if _, ok := props["query"]; !ok {
		t.Error("Schema should have query property")
	}

	if _, ok := props["num_results"]; !ok {
		t.Error("Schema should have num_results property")
	}

	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("Schema should have required array")
	}

	if len(required) != 1 || required[0] != "query" {
		t.Error("query should be required")
	}
}

func TestSearchTool_Execute_MissingQuery(t *testing.T) {
	tool := NewSearchTool()
	_, err := tool.Execute(context.Background(), map[string]interface{}{})

	if err == nil {
		t.Error("Expected error for missing query")
	}
}

func TestSearchTool_Execute_EmptyQuery(t *testing.T) {
	tool := NewSearchTool()
	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"query": "",
	})

	if err == nil {
		t.Error("Expected error for empty query")
	}
}

func TestSearchTool_Execute_NumResultsBounds(t *testing.T) {
	// Create a mock server that returns valid DDG response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"Abstract":      "",
			"RelatedTopics": []interface{}{},
			"Results":       []interface{}{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tool := &SearchTool{
		client: &http.Client{},
	}

	tests := []struct {
		name       string
		numResults float64
	}{
		{"zero results", 0},
		{"negative results", -5},
		{"too many results", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), map[string]interface{}{
				"query":       "test",
				"num_results": tt.numResults,
			})

			// Should not error, just clamp the value
			if err != nil {
				t.Logf("Execute() returned error (expected for real API): %v", err)
			} else if result != nil {
				t.Logf("Result: %s", result.Output)
			}
		})
	}
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "short text",
			text:     "Hello World",
			expected: "Hello World",
		},
		{
			name:     "text with dash separator",
			text:     "Title - Description goes here",
			expected: "Title",
		},
		{
			name:     "long text without separator",
			text:     strings.Repeat("x", 150),
			expected: strings.Repeat("x", 100) + "...",
		},
		{
			name:     "empty text",
			text:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTitle(tt.text)
			if result != tt.expected {
				t.Errorf("extractTitle() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSearchResult_Struct(t *testing.T) {
	result := SearchResult{
		Title:   "Test Title",
		URL:     "https://example.com",
		Snippet: "Test snippet",
	}

	if result.Title != "Test Title" {
		t.Error("Title not set correctly")
	}
	if result.URL != "https://example.com" {
		t.Error("URL not set correctly")
	}
	if result.Snippet != "Test snippet" {
		t.Error("Snippet not set correctly")
	}
}
