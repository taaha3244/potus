package anthropic

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
		_, err := New("")
		if err == nil {
			t.Error("Expected error for missing API key")
		}
	})

	t.Run("with API key", func(t *testing.T) {
		client, err := New("test-key-123")
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
}

func TestClient_Name(t *testing.T) {
	client := &Client{}
	if client.Name() != "anthropic" {
		t.Errorf("Name() = %s, want anthropic", client.Name())
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

	// Check that all models have required fields
	for _, model := range models {
		if model.ID == "" {
			t.Error("Model ID should not be empty")
		}
		if model.Name == "" {
			t.Error("Model Name should not be empty")
		}
		if model.Provider != "anthropic" {
			t.Errorf("Model Provider = %s, want anthropic", model.Provider)
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
			if r.Header.Get("X-API-Key") != "test-key" {
				t.Error("Missing or incorrect X-API-Key header")
			}
			if r.Header.Get("Anthropic-Version") != apiVersion {
				t.Error("Missing or incorrect Anthropic-Version header")
			}

			// Verify request body
			var req map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("Failed to decode request: %v", err)
			}

			if req["model"] != "claude-3-sonnet" {
				t.Errorf("model = %v, want claude-3-sonnet", req["model"])
			}
			if req["stream"] != true {
				t.Error("stream should be true")
			}

			// Send streaming response
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			responses := []string{
				`data: {"type":"message_start","message":{"id":"msg_123"}}`,
				`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
				`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" World"}}`,
				`data: {"type":"message_stop"}`,
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
			Model:     "claude-3-sonnet",
			MaxTokens: 1024,
			Messages: []providers.Message{
				{
					Role: providers.RoleUser,
					Content: []providers.ContentBlock{
						&providers.TextContent{Text: "Hello"},
					},
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

	t.Run("API error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":{"message":"Invalid API key"}}`))
		}))
		defer server.Close()

		client := &Client{
			apiKey:   "invalid-key",
			endpoint: server.URL,
			client:   &http.Client{},
		}

		_, err := client.Chat(context.Background(), &providers.ChatRequest{
			Model:     "claude-3-sonnet",
			MaxTokens: 1024,
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
			Model:     "claude-3-sonnet",
			MaxTokens: 1024,
			Messages: []providers.Message{
				{
					Role:    providers.RoleUser,
					Content: []providers.ContentBlock{&providers.TextContent{Text: "Hello"}},
				},
			},
		}

		apiReq := client.buildRequest(req)

		if apiReq["model"] != "claude-3-sonnet" {
			t.Errorf("model = %v, want claude-3-sonnet", apiReq["model"])
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
			Model:     "claude-3-sonnet",
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

		if apiReq["system"] != "You are a helpful assistant" {
			t.Errorf("system = %v, want 'You are a helpful assistant'", apiReq["system"])
		}
	})

	t.Run("with temperature", func(t *testing.T) {
		req := &providers.ChatRequest{
			Model:       "claude-3-sonnet",
			MaxTokens:   1024,
			Temperature: 0.7,
			Messages: []providers.Message{
				{
					Role:    providers.RoleUser,
					Content: []providers.ContentBlock{&providers.TextContent{Text: "Hello"}},
				},
			},
		}

		apiReq := client.buildRequest(req)

		if apiReq["temperature"] != 0.7 {
			t.Errorf("temperature = %v, want 0.7", apiReq["temperature"])
		}
	})

	t.Run("with tools", func(t *testing.T) {
		req := &providers.ChatRequest{
			Model:     "claude-3-sonnet",
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
		if tools[0]["name"] != "file_read" {
			t.Errorf("tool name = %v, want file_read", tools[0]["name"])
		}
	})

	t.Run("skips system role messages", func(t *testing.T) {
		req := &providers.ChatRequest{
			Model:     "claude-3-sonnet",
			MaxTokens: 1024,
			Messages: []providers.Message{
				{
					Role:    providers.RoleSystem,
					Content: []providers.ContentBlock{&providers.TextContent{Text: "System message"}},
				},
				{
					Role:    providers.RoleUser,
					Content: []providers.ContentBlock{&providers.TextContent{Text: "Hello"}},
				},
			},
		}

		apiReq := client.buildRequest(req)

		messages := apiReq["messages"].([]map[string]interface{})
		// Should only have user message, system role should be skipped
		if len(messages) != 1 {
			t.Errorf("messages length = %d, want 1 (system should be skipped)", len(messages))
		}
	})
}

func TestClient_ConvertContent(t *testing.T) {
	client := &Client{}

	t.Run("text content", func(t *testing.T) {
		blocks := []providers.ContentBlock{
			&providers.TextContent{Text: "Hello"},
		}

		result := client.convertContent(blocks)

		if len(result) != 1 {
			t.Fatalf("result length = %d, want 1", len(result))
		}
		if result[0]["type"] != "text" {
			t.Errorf("type = %v, want text", result[0]["type"])
		}
		if result[0]["text"] != "Hello" {
			t.Errorf("text = %v, want Hello", result[0]["text"])
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

		if len(result) != 1 {
			t.Fatalf("result length = %d, want 1", len(result))
		}
		if result[0]["type"] != "image" {
			t.Errorf("type = %v, want image", result[0]["type"])
		}

		source := result[0]["source"].(map[string]interface{})
		if source["type"] != "base64" {
			t.Errorf("source.type = %v, want base64", source["type"])
		}
		if source["media_type"] != "image/png" {
			t.Errorf("source.media_type = %v, want image/png", source["media_type"])
		}
	})

	t.Run("tool result content", func(t *testing.T) {
		blocks := []providers.ContentBlock{
			&providers.ToolResultContent{
				ToolUseID: "tool_123",
				Content:   "File contents here",
				IsError:   false,
			},
		}

		result := client.convertContent(blocks)

		if len(result) != 1 {
			t.Fatalf("result length = %d, want 1", len(result))
		}
		if result[0]["type"] != "tool_result" {
			t.Errorf("type = %v, want tool_result", result[0]["type"])
		}
		if result[0]["tool_use_id"] != "tool_123" {
			t.Errorf("tool_use_id = %v, want tool_123", result[0]["tool_use_id"])
		}
	})
}

func TestClient_HandleEvent(t *testing.T) {
	client := &Client{}

	t.Run("message_start event", func(t *testing.T) {
		eventChan := make(chan providers.ChatEvent, 10)

		event := map[string]interface{}{
			"type": "message_start",
		}

		client.handleEvent(event, eventChan)
		close(eventChan)

		received := <-eventChan
		if received.Type != providers.EventTypeMessageStart {
			t.Errorf("Type = %v, want message_start", received.Type)
		}
	})

	t.Run("text_delta event", func(t *testing.T) {
		eventChan := make(chan providers.ChatEvent, 10)

		event := map[string]interface{}{
			"type": "content_block_delta",
			"delta": map[string]interface{}{
				"type": "text_delta",
				"text": "Hello",
			},
		}

		client.handleEvent(event, eventChan)
		close(eventChan)

		received := <-eventChan
		if received.Type != providers.EventTypeTextDelta {
			t.Errorf("Type = %v, want text_delta", received.Type)
		}
		if received.Content != "Hello" {
			t.Errorf("Content = %v, want Hello", received.Content)
		}
	})

	t.Run("message_stop event", func(t *testing.T) {
		eventChan := make(chan providers.ChatEvent, 10)

		event := map[string]interface{}{
			"type": "message_stop",
		}

		client.handleEvent(event, eventChan)
		close(eventChan)

		received := <-eventChan
		if received.Type != providers.EventTypeMessageDone {
			t.Errorf("Type = %v, want message_done", received.Type)
		}
	})
}
