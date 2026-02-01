package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/taaha3244/potus/internal/providers"
)

func TestNew(t *testing.T) {
	t.Run("missing API key", func(t *testing.T) {
		_, err := New("", "")
		if err == nil {
			t.Error("Expected error for missing API key")
		}
	})

	t.Run("with API key", func(t *testing.T) {
		client, err := New("test-key-123", "")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		if client == nil {
			t.Fatal("Expected non-nil client")
		}

		if client.apiKey != "test-key-123" {
			t.Errorf("apiKey = %s, want test-key-123", client.apiKey)
		}
	})

	t.Run("with organization", func(t *testing.T) {
		client, err := New("test-key-123", "org-123")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		if client.organization != "org-123" {
			t.Errorf("organization = %s, want org-123", client.organization)
		}
	})
}

func TestClient_Name(t *testing.T) {
	client := &Client{}
	if client.Name() != "openai" {
		t.Errorf("Name() = %s, want openai", client.Name())
	}
}

func TestClient_SupportsTools(t *testing.T) {
	client := &Client{}
	if !client.SupportsTools() {
		t.Error("SupportsTools() should return true")
	}
}

func TestClient_SupportsVision(t *testing.T) {
	client := &Client{}
	if !client.SupportsVision() {
		t.Error("SupportsVision() should return true")
	}
}

func TestClient_ListModels(t *testing.T) {
	client := &Client{}
	models, err := client.ListModels(context.Background())

	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}

	if len(models) == 0 {
		t.Error("Expected at least one model")
	}

	for _, model := range models {
		if model.ID == "" {
			t.Error("Model ID should not be empty")
		}
		if model.Name == "" {
			t.Error("Model Name should not be empty")
		}
		if model.Provider != "openai" {
			t.Errorf("Model Provider = %s, want openai", model.Provider)
		}
		if model.ContextSize <= 0 {
			t.Error("Model ContextSize should be > 0")
		}
	}
}

func TestClient_Chat(t *testing.T) {
	t.Run("successful streaming response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request headers
			if r.Header.Get("Content-Type") != "application/json" {
				t.Error("Missing Content-Type header")
			}
			if r.Header.Get("Authorization") != "Bearer test-key" {
				t.Error("Missing or incorrect Authorization header")
			}

			// Verify request body
			var req map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("Failed to decode request: %v", err)
			}

			if req["model"] != "gpt-4" {
				t.Errorf("model = %v, want gpt-4", req["model"])
			}
			if req["stream"] != true {
				t.Error("stream should be true")
			}

			// Send streaming response
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			responses := []string{
				`data: {"id":"chatcmpl-123","choices":[{"index":0,"delta":{"role":"assistant"}}]}`,
				`data: {"id":"chatcmpl-123","choices":[{"index":0,"delta":{"content":"Hello"}}]}`,
				`data: {"id":"chatcmpl-123","choices":[{"index":0,"delta":{"content":" World"}}]}`,
				`data: {"id":"chatcmpl-123","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
				`data: [DONE]`,
			}

			for _, resp := range responses {
				w.Write([]byte(resp + "\n"))
			}
		}))
		defer server.Close()

		client := &Client{
			apiKey:   "test-key",
			endpoint: server.URL,
			client:   &http.Client{},
		}

		events, err := client.Chat(context.Background(), &providers.ChatRequest{
			Model:     "gpt-4",
			MaxTokens: 1024,
			Messages: []providers.Message{
				{
					Role:    providers.RoleUser,
					Content: []providers.ContentBlock{&providers.TextContent{Text: "Hello"}},
				},
			},
		})

		if err != nil {
			t.Fatalf("Chat() error = %v", err)
		}

		var textContent string
		var gotMessageStart, gotMessageDone bool

		for event := range events {
			switch event.Type {
			case providers.EventTypeMessageStart:
				gotMessageStart = true
			case providers.EventTypeTextDelta:
				textContent += event.Content
			case providers.EventTypeMessageDone:
				gotMessageDone = true
			case providers.EventTypeError:
				t.Errorf("Unexpected error: %v", event.Error)
			}
		}

		if !gotMessageStart {
			t.Error("Expected message_start event")
		}
		if !gotMessageDone {
			t.Error("Expected message_done event")
		}
		if textContent != "Hello World" {
			t.Errorf("textContent = %s, want 'Hello World'", textContent)
		}
	})

	t.Run("with organization header", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("OpenAI-Organization") != "org-123" {
				t.Error("Missing or incorrect OpenAI-Organization header")
			}

			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("data: [DONE]\n"))
		}))
		defer server.Close()

		client := &Client{
			apiKey:       "test-key",
			organization: "org-123",
			endpoint:     server.URL,
			client:       &http.Client{},
		}

		events, err := client.Chat(context.Background(), &providers.ChatRequest{
			Model: "gpt-4",
			Messages: []providers.Message{
				{
					Role:    providers.RoleUser,
					Content: []providers.ContentBlock{&providers.TextContent{Text: "Hello"}},
				},
			},
		})

		if err != nil {
			t.Fatalf("Chat() error = %v", err)
		}

		// Drain the channel
		for range events {
		}
	})

	t.Run("API error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":{"message":"Incorrect API key"}}`))
		}))
		defer server.Close()

		client := &Client{
			apiKey:   "invalid-key",
			endpoint: server.URL,
			client:   &http.Client{},
		}

		_, err := client.Chat(context.Background(), &providers.ChatRequest{
			Model: "gpt-4",
			Messages: []providers.Message{
				{
					Role:    providers.RoleUser,
					Content: []providers.ContentBlock{&providers.TextContent{Text: "Hello"}},
				},
			},
		})

		if err == nil {
			t.Error("Expected error for API error response")
		}
	})
}

func TestClient_BuildRequest(t *testing.T) {
	client := &Client{}

	t.Run("basic request", func(t *testing.T) {
		req := &providers.ChatRequest{
			Model:     "gpt-4",
			MaxTokens: 1024,
			Messages: []providers.Message{
				{
					Role:    providers.RoleUser,
					Content: []providers.ContentBlock{&providers.TextContent{Text: "Hello"}},
				},
			},
		}

		apiReq := client.buildRequest(req)

		if apiReq["model"] != "gpt-4" {
			t.Errorf("model = %v, want gpt-4", apiReq["model"])
		}
		if apiReq["max_tokens"] != 1024 {
			t.Errorf("max_tokens = %v, want 1024", apiReq["max_tokens"])
		}
		if apiReq["stream"] != true {
			t.Error("stream should be true")
		}
	})

	t.Run("with system prompt", func(t *testing.T) {
		req := &providers.ChatRequest{
			Model:     "gpt-4",
			MaxTokens: 1024,
			System:    "You are a helpful assistant",
			Messages: []providers.Message{
				{
					Role:    providers.RoleUser,
					Content: []providers.ContentBlock{&providers.TextContent{Text: "Hello"}},
				},
			},
		}

		apiReq := client.buildRequest(req)

		messages := apiReq["messages"].([]map[string]interface{})
		// First message should be system
		if messages[0]["role"] != "system" {
			t.Errorf("first message role = %v, want system", messages[0]["role"])
		}
		if messages[0]["content"] != "You are a helpful assistant" {
			t.Errorf("system content = %v, want 'You are a helpful assistant'", messages[0]["content"])
		}
	})

	t.Run("with tools", func(t *testing.T) {
		req := &providers.ChatRequest{
			Model:     "gpt-4",
			MaxTokens: 1024,
			Messages: []providers.Message{
				{
					Role:    providers.RoleUser,
					Content: []providers.ContentBlock{&providers.TextContent{Text: "Hello"}},
				},
			},
			Tools: []providers.Tool{
				{
					Name:        "file_read",
					Description: "Read a file",
					InputSchema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"path": map[string]interface{}{"type": "string"},
						},
					},
				},
			},
		}

		apiReq := client.buildRequest(req)

		tools, ok := apiReq["tools"].([]map[string]interface{})
		if !ok {
			t.Fatal("tools should be array of maps")
		}
		if len(tools) != 1 {
			t.Errorf("tools length = %d, want 1", len(tools))
		}
		if tools[0]["type"] != "function" {
			t.Errorf("tool type = %v, want function", tools[0]["type"])
		}

		function := tools[0]["function"].(map[string]interface{})
		if function["name"] != "file_read" {
			t.Errorf("function name = %v, want file_read", function["name"])
		}
	})

	t.Run("with tool result message", func(t *testing.T) {
		req := &providers.ChatRequest{
			Model:     "gpt-4",
			MaxTokens: 1024,
			Messages: []providers.Message{
				{
					Role: providers.RoleTool,
					Content: []providers.ContentBlock{
						&providers.ToolResultContent{
							ToolUseID: "call_123",
							Content:   "File contents here",
							IsError:   false,
						},
					},
				},
			},
		}

		apiReq := client.buildRequest(req)

		messages := apiReq["messages"].([]map[string]interface{})
		if messages[0]["tool_call_id"] != "call_123" {
			t.Errorf("tool_call_id = %v, want call_123", messages[0]["tool_call_id"])
		}
		if messages[0]["content"] != "File contents here" {
			t.Errorf("content = %v, want 'File contents here'", messages[0]["content"])
		}
	})

	t.Run("with assistant tool use", func(t *testing.T) {
		req := &providers.ChatRequest{
			Model:     "gpt-4",
			MaxTokens: 1024,
			Messages: []providers.Message{
				{
					Role: providers.RoleAssistant,
					Content: []providers.ContentBlock{
						&providers.ToolUseContent{
							ID:    "call_123",
							Name:  "file_read",
							Input: map[string]interface{}{"path": "/test.txt"},
						},
					},
				},
			},
		}

		apiReq := client.buildRequest(req)

		messages := apiReq["messages"].([]map[string]interface{})
		toolCalls := messages[0]["tool_calls"].([]map[string]interface{})

		if len(toolCalls) != 1 {
			t.Fatalf("tool_calls length = %d, want 1", len(toolCalls))
		}
		if toolCalls[0]["id"] != "call_123" {
			t.Errorf("tool_call id = %v, want call_123", toolCalls[0]["id"])
		}
	})
}

func TestClient_ConvertRole(t *testing.T) {
	client := &Client{}

	tests := []struct {
		input    providers.MessageRole
		expected string
	}{
		{providers.RoleUser, "user"},
		{providers.RoleAssistant, "assistant"},
		{providers.RoleSystem, "system"},
		{providers.RoleTool, "tool"},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			got := client.convertRole(tt.input)
			if got != tt.expected {
				t.Errorf("convertRole(%s) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}

func TestClient_ConvertContent(t *testing.T) {
	client := &Client{}

	t.Run("single text content", func(t *testing.T) {
		blocks := []providers.ContentBlock{
			&providers.TextContent{Text: "Hello"},
		}

		result := client.convertContent(blocks)

		if str, ok := result.(string); !ok || str != "Hello" {
			t.Errorf("result = %v, want 'Hello'", result)
		}
	})

	t.Run("empty blocks", func(t *testing.T) {
		blocks := []providers.ContentBlock{}

		result := client.convertContent(blocks)

		if str, ok := result.(string); !ok || str != "" {
			t.Errorf("result = %v, want empty string", result)
		}
	})

	t.Run("multiple text blocks", func(t *testing.T) {
		blocks := []providers.ContentBlock{
			&providers.TextContent{Text: "Hello"},
			&providers.TextContent{Text: "World"},
		}

		result := client.convertContent(blocks)

		arr, ok := result.([]map[string]interface{})
		if !ok {
			t.Fatal("result should be array of maps")
		}
		if len(arr) != 2 {
			t.Errorf("result length = %d, want 2", len(arr))
		}
	})

	t.Run("image content", func(t *testing.T) {
		blocks := []providers.ContentBlock{
			&providers.ImageContent{
				Source: providers.ImageSource{
					Type:      "base64",
					MediaType: "image/png",
					Data:      "base64data",
				},
			},
		}

		result := client.convertContent(blocks)

		arr := result.([]map[string]interface{})
		if arr[0]["type"] != "image_url" {
			t.Errorf("type = %v, want image_url", arr[0]["type"])
		}
	})
}

func TestClient_ExtractToolCalls(t *testing.T) {
	client := &Client{}

	t.Run("no tool uses", func(t *testing.T) {
		blocks := []providers.ContentBlock{
			&providers.TextContent{Text: "Hello"},
		}

		result := client.extractToolCalls(blocks)

		if len(result) != 0 {
			t.Errorf("result length = %d, want 0", len(result))
		}
	})

	t.Run("with tool use", func(t *testing.T) {
		blocks := []providers.ContentBlock{
			&providers.ToolUseContent{
				ID:    "call_123",
				Name:  "file_read",
				Input: map[string]interface{}{"path": "/test.txt"},
			},
		}

		result := client.extractToolCalls(blocks)

		if len(result) != 1 {
			t.Fatalf("result length = %d, want 1", len(result))
		}
		if result[0]["id"] != "call_123" {
			t.Errorf("id = %v, want call_123", result[0]["id"])
		}
		if result[0]["type"] != "function" {
			t.Errorf("type = %v, want function", result[0]["type"])
		}

		function := result[0]["function"].(map[string]interface{})
		if function["name"] != "file_read" {
			t.Errorf("name = %v, want file_read", function["name"])
		}
	})
}
