package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/taaha3244/potus/internal/config"
	"github.com/taaha3244/potus/internal/providers"
	"github.com/taaha3244/potus/internal/tools"
)

// mockProvider implements providers.Provider for testing
type mockProvider struct {
	responses   []mockResponse
	currentResp int
	shouldError bool
	errorMsg    string
}

type mockResponse struct {
	text     string
	toolUses []*providers.ToolUseContent
	usage    *providers.Usage
}

func (m *mockProvider) Chat(ctx context.Context, req *providers.ChatRequest) (<-chan providers.ChatEvent, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMsg)
	}

	events := make(chan providers.ChatEvent, 10)

	go func() {
		defer close(events)

		if m.currentResp >= len(m.responses) {
			// Default response
			events <- providers.ChatEvent{
				Type:    providers.EventTypeTextDelta,
				Content: "Default response",
			}
			events <- providers.ChatEvent{
				Type: providers.EventTypeMessageDone,
			}
			return
		}

		resp := m.responses[m.currentResp]
		m.currentResp++

		// Send text
		if resp.text != "" {
			events <- providers.ChatEvent{
				Type:    providers.EventTypeTextDelta,
				Content: resp.text,
			}
		}

		// Send tool uses
		for _, toolUse := range resp.toolUses {
			events <- providers.ChatEvent{
				Type:    providers.EventTypeToolUse,
				ToolUse: toolUse,
			}
		}

		// Send done
		events <- providers.ChatEvent{
			Type:  providers.EventTypeMessageDone,
			Usage: resp.usage,
		}
	}()

	return events, nil
}

func (m *mockProvider) ListModels(ctx context.Context) ([]providers.Model, error) {
	return []providers.Model{
		{ID: "test-model", Name: "Test Model", Provider: "mock", ContextSize: 100000},
	}, nil
}

func (m *mockProvider) SupportsTools() bool { return true }
func (m *mockProvider) SupportsVision() bool { return true }
func (m *mockProvider) Name() string { return "mock" }

// mockTool implements tools.Tool for testing
type mockTool struct {
	name        string
	shouldError bool
	output      string
}

func (t *mockTool) Name() string        { return t.name }
func (t *mockTool) Description() string { return "Mock tool for testing" }
func (t *mockTool) Schema() map[string]interface{} {
	return map[string]interface{}{"type": "object"}
}
func (t *mockTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.Result, error) {
	if t.shouldError {
		return nil, errors.New("tool execution failed")
	}
	return tools.NewResult(t.output), nil
}

func TestNew(t *testing.T) {
	t.Run("minimal config", func(t *testing.T) {
		provider := &mockProvider{}
		registry := tools.NewRegistry()

		agent := New(&Config{
			Provider:     provider,
			ToolRegistry: registry,
			SystemPrompt: "You are helpful",
			MaxTokens:    1024,
			Model:        "test-model",
		})

		if agent == nil {
			t.Fatal("New returned nil")
		}

		if agent.provider != provider {
			t.Error("provider not set correctly")
		}

		if agent.memory == nil {
			t.Error("memory should not be nil")
		}

		if agent.executor == nil {
			t.Error("executor should not be nil")
		}

		// Without ContextConfig, contextManager should be nil
		if agent.contextManager != nil {
			t.Error("contextManager should be nil without ContextConfig")
		}
	})

	t.Run("with context config", func(t *testing.T) {
		provider := &mockProvider{}
		registry := tools.NewRegistry()

		agent := New(&Config{
			Provider:     provider,
			ToolRegistry: registry,
			SystemPrompt: "You are helpful",
			MaxTokens:    1024,
			Model:        "test-model",
			ContextConfig: &config.ContextConfig{
				MaxTokens:          100000,
				ReserveForResponse: 8192,
				AutoCompact:        true,
				AutoPrune:          true,
				WarnThreshold:      0.80,
				CompactThreshold:   0.90,
			},
		})

		if agent.contextManager == nil {
			t.Error("contextManager should not be nil with ContextConfig")
		}
	})

	t.Run("with model info", func(t *testing.T) {
		provider := &mockProvider{}
		registry := tools.NewRegistry()

		agent := New(&Config{
			Provider:     provider,
			ToolRegistry: registry,
			SystemPrompt: "You are helpful",
			MaxTokens:    1024,
			Model:        "test-model",
			ContextConfig: &config.ContextConfig{
				MaxTokens:          100000,
				ReserveForResponse: 8192,
			},
			ModelInfo: &providers.Model{
				ID:          "test-model",
				ContextSize: 200000,
				Pricing: providers.ModelPricing{
					InputPer1M:  3.00,
					OutputPer1M: 15.00,
				},
			},
		})

		if agent.contextManager == nil {
			t.Error("contextManager should not be nil")
		}
	})

	t.Run("with project context", func(t *testing.T) {
		tmpDir := t.TempDir()
		potusContent := "# Project Context\n\nThis is a test project."
		if err := os.WriteFile(filepath.Join(tmpDir, "POTUS.md"), []byte(potusContent), 0644); err != nil {
			t.Fatal(err)
		}

		provider := &mockProvider{}
		registry := tools.NewRegistry()

		agent := New(&Config{
			Provider:     provider,
			ToolRegistry: registry,
			SystemPrompt: "You are helpful",
			MaxTokens:    1024,
			Model:        "test-model",
			WorkDir:      tmpDir,
			ContextConfig: &config.ContextConfig{
				MaxTokens:          100000,
				ReserveForResponse: 8192,
				LoadProjectContext: true,
			},
		})

		// System prompt should include project context
		if agent.systemPrompt == "You are helpful" {
			t.Error("System prompt should include project context")
		}
	})
}

func TestAgent_ProcessMessage(t *testing.T) {
	t.Run("simple text response", func(t *testing.T) {
		provider := &mockProvider{
			responses: []mockResponse{
				{text: "Hello! How can I help you?"},
			},
		}
		registry := tools.NewRegistry()

		agent := New(&Config{
			Provider:     provider,
			ToolRegistry: registry,
			SystemPrompt: "You are helpful",
			MaxTokens:    1024,
			Model:        "test-model",
		})

		events, err := agent.ProcessMessage(context.Background(), "Hello")
		if err != nil {
			t.Fatalf("ProcessMessage() error = %v", err)
		}

		var gotText string
		var gotDone bool

		for event := range events {
			switch event.Type {
			case EventTypeTextDelta:
				gotText += event.Content
			case EventTypeMessageDone:
				gotDone = true
			case EventTypeError:
				t.Fatalf("Unexpected error: %v", event.Error)
			}
		}

		if gotText != "Hello! How can I help you?" {
			t.Errorf("Text = %s, want 'Hello! How can I help you?'", gotText)
		}

		if !gotDone {
			t.Error("Expected message_done event")
		}
	})

	t.Run("with tool call", func(t *testing.T) {
		provider := &mockProvider{
			responses: []mockResponse{
				{
					text: "Let me read that file.",
					toolUses: []*providers.ToolUseContent{
						{
							ID:    "tool_1",
							Name:  "test_tool",
							Input: map[string]interface{}{"param": "value"},
						},
					},
				},
				{text: "Here's the result."},
			},
		}

		registry := tools.NewRegistry()
		registry.Register(&mockTool{name: "test_tool", output: "Tool output"})

		agent := New(&Config{
			Provider:     provider,
			ToolRegistry: registry,
			SystemPrompt: "You are helpful",
			MaxTokens:    1024,
			Model:        "test-model",
		})

		events, err := agent.ProcessMessage(context.Background(), "Read the file")
		if err != nil {
			t.Fatalf("ProcessMessage() error = %v", err)
		}

		var gotToolCall, gotToolResult bool
		var toolName string
		var toolResultContent string

		for event := range events {
			switch event.Type {
			case EventTypeToolCall:
				gotToolCall = true
				toolName = event.ToolUse.Name
			case EventTypeToolResult:
				gotToolResult = true
				toolResultContent = event.ToolResult.Content
			case EventTypeError:
				t.Fatalf("Unexpected error: %v", event.Error)
			}
		}

		if !gotToolCall {
			t.Error("Expected tool_call event")
		}
		if toolName != "test_tool" {
			t.Errorf("Tool name = %s, want test_tool", toolName)
		}

		if !gotToolResult {
			t.Error("Expected tool_result event")
		}
		if toolResultContent != "Tool output" {
			t.Errorf("Tool result = %s, want 'Tool output'", toolResultContent)
		}
	})

	t.Run("tool execution error", func(t *testing.T) {
		provider := &mockProvider{
			responses: []mockResponse{
				{
					toolUses: []*providers.ToolUseContent{
						{
							ID:    "tool_1",
							Name:  "error_tool",
							Input: map[string]interface{}{},
						},
					},
				},
				{text: "I see there was an error."},
			},
		}

		registry := tools.NewRegistry()
		registry.Register(&mockTool{name: "error_tool", shouldError: true})

		agent := New(&Config{
			Provider:     provider,
			ToolRegistry: registry,
			SystemPrompt: "You are helpful",
			MaxTokens:    1024,
			Model:        "test-model",
		})

		events, err := agent.ProcessMessage(context.Background(), "Do something")
		if err != nil {
			t.Fatalf("ProcessMessage() error = %v", err)
		}

		var toolResult *providers.ToolResultContent
		for event := range events {
			if event.Type == EventTypeToolResult {
				toolResult = event.ToolResult
			}
		}

		if toolResult == nil {
			t.Fatal("Expected tool result")
		}
		if !toolResult.IsError {
			t.Error("Tool result should be an error")
		}
	})

	t.Run("provider error", func(t *testing.T) {
		provider := &mockProvider{
			shouldError: true,
			errorMsg:    "API rate limit exceeded",
		}
		registry := tools.NewRegistry()

		agent := New(&Config{
			Provider:     provider,
			ToolRegistry: registry,
			SystemPrompt: "You are helpful",
			MaxTokens:    1024,
			Model:        "test-model",
		})

		events, err := agent.ProcessMessage(context.Background(), "Hello")
		if err != nil {
			t.Fatalf("ProcessMessage() error = %v", err)
		}

		var gotError bool
		for event := range events {
			if event.Type == EventTypeError {
				gotError = true
			}
		}

		if !gotError {
			t.Error("Expected error event")
		}
	})

	t.Run("with token tracking", func(t *testing.T) {
		provider := &mockProvider{
			responses: []mockResponse{
				{
					text: "Hello!",
					usage: &providers.Usage{
						InputTokens:  100,
						OutputTokens: 50,
					},
				},
			},
		}
		registry := tools.NewRegistry()

		agent := New(&Config{
			Provider:     provider,
			ToolRegistry: registry,
			SystemPrompt: "You are helpful",
			MaxTokens:    1024,
			Model:        "test-model",
			ContextConfig: &config.ContextConfig{
				MaxTokens:          100000,
				ReserveForResponse: 8192,
			},
		})

		events, err := agent.ProcessMessage(context.Background(), "Hello")
		if err != nil {
			t.Fatalf("ProcessMessage() error = %v", err)
		}

		var gotTokenUpdate bool
		for event := range events {
			if event.Type == EventTypeTokenUpdate {
				gotTokenUpdate = true
				if event.TokenInfo == nil {
					t.Error("TokenInfo should not be nil")
				}
			}
		}

		if !gotTokenUpdate {
			t.Error("Expected token_update event")
		}
	})
}

func TestAgent_GetMemory(t *testing.T) {
	provider := &mockProvider{}
	registry := tools.NewRegistry()

	agent := New(&Config{
		Provider:     provider,
		ToolRegistry: registry,
		SystemPrompt: "You are helpful",
		MaxTokens:    1024,
		Model:        "test-model",
	})

	memory := agent.GetMemory()
	if memory == nil {
		t.Error("GetMemory() should not return nil")
	}
}

func TestAgent_GetContextManager(t *testing.T) {
	t.Run("without context config", func(t *testing.T) {
		provider := &mockProvider{}
		registry := tools.NewRegistry()

		agent := New(&Config{
			Provider:     provider,
			ToolRegistry: registry,
			SystemPrompt: "You are helpful",
			MaxTokens:    1024,
			Model:        "test-model",
		})

		if agent.GetContextManager() != nil {
			t.Error("GetContextManager() should return nil without ContextConfig")
		}
	})

	t.Run("with context config", func(t *testing.T) {
		provider := &mockProvider{}
		registry := tools.NewRegistry()

		agent := New(&Config{
			Provider:     provider,
			ToolRegistry: registry,
			SystemPrompt: "You are helpful",
			MaxTokens:    1024,
			Model:        "test-model",
			ContextConfig: &config.ContextConfig{
				MaxTokens:          100000,
				ReserveForResponse: 8192,
			},
		})

		if agent.GetContextManager() == nil {
			t.Error("GetContextManager() should not return nil with ContextConfig")
		}
	})
}

func TestAgent_GetTokenSummary(t *testing.T) {
	provider := &mockProvider{}
	registry := tools.NewRegistry()

	agent := New(&Config{
		Provider:     provider,
		ToolRegistry: registry,
		SystemPrompt: "You are helpful",
		MaxTokens:    1024,
		Model:        "test-model",
	})

	summary := agent.GetTokenSummary()

	// Should start with 0 messages
	if summary.MessageCount != 0 {
		t.Errorf("MessageCount = %d, want 0", summary.MessageCount)
	}
}

func TestTokenUpdateInfo(t *testing.T) {
	info := &TokenUpdateInfo{
		CurrentTokens: 50000,
		MaxTokens:     100000,
		UsagePercent:  50.0,
		SessionTokens: 5000,
		Cost:          0.05,
		AtWarning:     false,
	}

	if info.CurrentTokens != 50000 {
		t.Error("CurrentTokens not set correctly")
	}
	if info.UsagePercent != 50.0 {
		t.Error("UsagePercent not set correctly")
	}
}

func TestEventType(t *testing.T) {
	tests := []struct {
		eventType EventType
		expected  string
	}{
		{EventTypeTextDelta, "text_delta"},
		{EventTypeToolCall, "tool_call"},
		{EventTypeToolResult, "tool_result"},
		{EventTypeMessageDone, "message_done"},
		{EventTypeError, "error"},
		{EventTypeTokenUpdate, "token_update"},
		{EventTypeContextUpdate, "context_update"},
	}

	for _, tt := range tests {
		if string(tt.eventType) != tt.expected {
			t.Errorf("EventType %v = %s, want %s", tt.eventType, string(tt.eventType), tt.expected)
		}
	}
}

func TestMaxToolIterations(t *testing.T) {
	if MaxToolIterations != 10 {
		t.Errorf("MaxToolIterations = %d, want 10", MaxToolIterations)
	}
}
