package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/taaha3244/potus/internal/providers"
)

func TestNew(t *testing.T) {
	t.Run("with default endpoint", func(t *testing.T) {
		client, err := New("")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		if client.endpoint != "http://localhost:11434" {
			t.Errorf("endpoint = %s, want http://localhost:11434", client.endpoint)
		}
	})

	t.Run("with custom endpoint", func(t *testing.T) {
		client, err := New("http://custom:8080")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		if client.endpoint != "http://custom:8080" {
			t.Errorf("endpoint = %s, want http://custom:8080", client.endpoint)
		}
	})
}

func TestClient_Name(t *testing.T) {
	client := &Client{}
	if client.Name() != "ollama" {
		t.Errorf("Name() = %s, want ollama", client.Name())
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
	t.Run("successful response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/tags" {
				t.Errorf("URL path = %s, want /api/tags", r.URL.Path)
			}

			response := map[string]interface{}{
				"models": []map[string]interface{}{
					{
						"name":        "llama2:latest",
						"modified_at": "2024-01-01T00:00:00Z",
					},
					{
						"name":        "codellama:latest",
						"modified_at": "2024-01-02T00:00:00Z",
					},
				},
			}

			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := &Client{
			endpoint: server.URL,
			client:   &http.Client{},
		}

		models, err := client.ListModels(context.Background())
		if err != nil {
			t.Fatalf("ListModels() error = %v", err)
		}

		if len(models) != 2 {
			t.Errorf("models length = %d, want 2", len(models))
		}

		if models[0].ID != "llama2:latest" {
			t.Errorf("models[0].ID = %s, want llama2:latest", models[0].ID)
		}
		if models[0].Provider != "ollama" {
			t.Errorf("models[0].Provider = %s, want ollama", models[0].Provider)
		}
		if models[0].Pricing.InputPer1M != 0 {
			t.Errorf("models[0].Pricing.InputPer1M = %f, want 0 (ollama is free)", models[0].Pricing.InputPer1M)
		}
	})

	t.Run("error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := &Client{
			endpoint: server.URL,
			client:   &http.Client{},
		}

		_, err := client.ListModels(context.Background())
		if err == nil {
			t.Error("Expected error for server error response")
		}
	})
}

func TestClient_Chat(t *testing.T) {
	t.Run("successful streaming response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/chat" {
				t.Errorf("URL path = %s, want /api/chat", r.URL.Path)
			}

			// Verify request body
			var req map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("Failed to decode request: %v", err)
			}

			if req["model"] != "llama2" {
				t.Errorf("model = %v, want llama2", req["model"])
			}
			if req["stream"] != true {
				t.Error("stream should be true")
			}

			// Send streaming response
			w.Header().Set("Content-Type", "application/x-ndjson")
			w.WriteHeader(http.StatusOK)

			responses := []map[string]interface{}{
				{
					"model": "llama2",
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "Hello",
					},
					"done": false,
				},
				{
					"model": "llama2",
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": " World",
					},
					"done": false,
				},
				{
					"model": "llama2",
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "",
					},
					"done": true,
				},
			}

			for _, resp := range responses {
				data, _ := json.Marshal(resp)
				w.Write(data)
				w.Write([]byte("\n"))
			}
		}))
		defer server.Close()

		client := &Client{
			endpoint: server.URL,
			client:   &http.Client{},
		}

		events, err := client.Chat(context.Background(), &providers.ChatRequest{
			Model: "llama2",
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

	t.Run("with tool calls", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/x-ndjson")
			w.WriteHeader(http.StatusOK)

			responses := []map[string]interface{}{
				{
					"model": "llama2",
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "",
						"tool_calls": []map[string]interface{}{
							{
								"function": map[string]interface{}{
									"name":      "file_read",
									"arguments": map[string]interface{}{"path": "/test.txt"},
								},
							},
						},
					},
					"done": false,
				},
				{
					"model": "llama2",
					"done":  true,
				},
			}

			for _, resp := range responses {
				data, _ := json.Marshal(resp)
				w.Write(data)
				w.Write([]byte("\n"))
			}
		}))
		defer server.Close()

		client := &Client{
			endpoint: server.URL,
			client:   &http.Client{},
		}

		events, err := client.Chat(context.Background(), &providers.ChatRequest{
			Model: "llama2",
			Messages: []providers.Message{
				{
					Role:    providers.RoleUser,
					Content: []providers.ContentBlock{&providers.TextContent{Text: "Read file"}},
				},
			},
			Tools: []providers.Tool{
				{
					Name:        "file_read",
					Description: "Read a file",
					InputSchema: map[string]interface{}{"type": "object"},
				},
			},
		})

		if err != nil {
			t.Fatalf("Chat() error = %v", err)
		}

		var gotToolUse bool
		var toolName string

		for event := range events {
			if event.Type == providers.EventTypeToolUse {
				gotToolUse = true
				toolName = event.ToolUse.Name
			}
		}

		if !gotToolUse {
			t.Error("Expected tool_use event")
		}
		if toolName != "file_read" {
			t.Errorf("toolName = %s, want file_read", toolName)
		}
	})

	t.Run("API error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"model not found"}`))
		}))
		defer server.Close()

		client := &Client{
			endpoint: server.URL,
			client:   &http.Client{},
		}

		_, err := client.Chat(context.Background(), &providers.ChatRequest{
			Model: "nonexistent",
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
			Model: "llama2",
			Messages: []providers.Message{
				{
					Role:    providers.RoleUser,
					Content: []providers.ContentBlock{&providers.TextContent{Text: "Hello"}},
				},
			},
		}

		apiReq := client.buildRequest(req)

		if apiReq["model"] != "llama2" {
			t.Errorf("model = %v, want llama2", apiReq["model"])
		}
		if apiReq["stream"] != true {
			t.Error("stream should be true")
		}
	})

	t.Run("with system prompt", func(t *testing.T) {
		req := &providers.ChatRequest{
			Model:  "llama2",
			System: "You are a helpful assistant",
			Messages: []providers.Message{
				{
					Role:    providers.RoleUser,
					Content: []providers.ContentBlock{&providers.TextContent{Text: "Hello"}},
				},
			},
		}

		apiReq := client.buildRequest(req)

		messages := apiReq["messages"].([]map[string]interface{})
		if messages[0]["role"] != "system" {
			t.Errorf("first message role = %v, want system", messages[0]["role"])
		}
		if messages[0]["content"] != "You are a helpful assistant" {
			t.Errorf("system content = %v", messages[0]["content"])
		}
	})

	t.Run("with temperature", func(t *testing.T) {
		req := &providers.ChatRequest{
			Model:       "llama2",
			Temperature: 0.7,
			Messages: []providers.Message{
				{
					Role:    providers.RoleUser,
					Content: []providers.ContentBlock{&providers.TextContent{Text: "Hello"}},
				},
			},
		}

		apiReq := client.buildRequest(req)

		options, ok := apiReq["options"].(map[string]interface{})
		if !ok {
			t.Fatal("options should be a map")
		}
		if options["temperature"] != 0.7 {
			t.Errorf("temperature = %v, want 0.7", options["temperature"])
		}
	})

	t.Run("with tools", func(t *testing.T) {
		req := &providers.ChatRequest{
			Model: "llama2",
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
	})

	t.Run("with image content", func(t *testing.T) {
		req := &providers.ChatRequest{
			Model: "llava",
			Messages: []providers.Message{
				{
					Role: providers.RoleUser,
					Content: []providers.ContentBlock{
						&providers.TextContent{Text: "What's in this image?"},
						&providers.ImageContent{
							Source: providers.ImageSource{
								Type:      "base64",
								MediaType: "image/png",
								Data:      "base64imagedata",
							},
						},
					},
				},
			},
		}

		apiReq := client.buildRequest(req)

		messages := apiReq["messages"].([]map[string]interface{})
		images := messages[0]["images"].([]string)
		if len(images) != 1 {
			t.Errorf("images length = %d, want 1", len(images))
		}
		if images[0] != "base64imagedata" {
			t.Errorf("images[0] = %s, want base64imagedata", images[0])
		}
	})

	t.Run("with tool use content", func(t *testing.T) {
		req := &providers.ChatRequest{
			Model: "llama2",
			Messages: []providers.Message{
				{
					Role: providers.RoleAssistant,
					Content: []providers.ContentBlock{
						&providers.ToolUseContent{
							ID:    "tool_123",
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
			t.Errorf("tool_calls length = %d, want 1", len(toolCalls))
		}
	})

	t.Run("with tool result content", func(t *testing.T) {
		req := &providers.ChatRequest{
			Model: "llama2",
			Messages: []providers.Message{
				{
					Role: providers.RoleTool,
					Content: []providers.ContentBlock{
						&providers.ToolResultContent{
							ToolUseID: "tool_123",
							Content:   "File contents here",
							IsError:   false,
						},
					},
				},
			},
		}

		apiReq := client.buildRequest(req)

		messages := apiReq["messages"].([]map[string]interface{})
		if messages[0]["content"] != "File contents here" {
			t.Errorf("content = %v, want 'File contents here'", messages[0]["content"])
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
