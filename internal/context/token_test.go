package context

import (
	"testing"

	"github.com/taaha3244/potus/internal/providers"
)

func TestSimpleEstimator_EstimateTokens(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "empty string",
			text:     "",
			expected: 0,
		},
		{
			name:     "short text",
			text:     "hello",
			expected: 1, // 5 chars / 4 = 1.25 -> 1
		},
		{
			name:     "medium text",
			text:     "hello world",
			expected: 2, // 11 chars / 4 = 2.75 -> 2
		},
		{
			name:     "longer text",
			text:     "The quick brown fox jumps over the lazy dog",
			expected: 10, // 43 chars / 4 = 10.75 -> 10
		},
		{
			name:     "code snippet",
			text:     "func main() {\n\tfmt.Println(\"Hello\")\n}",
			expected: 9, // 38 chars / 4 = 9.5 -> 9
		},
	}

	estimator := NewSimpleEstimator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimator.EstimateTokens(tt.text)
			if got != tt.expected {
				t.Errorf("EstimateTokens(%q) = %d, want %d", tt.text, got, tt.expected)
			}
		})
	}
}

func TestSimpleEstimator_EstimateMessage(t *testing.T) {
	estimator := NewSimpleEstimator()

	tests := []struct {
		name    string
		message *providers.Message
		minToks int // minimum expected tokens (due to overhead)
	}{
		{
			name:    "nil message",
			message: nil,
			minToks: 0,
		},
		{
			name: "simple text message",
			message: &providers.Message{
				Role: providers.RoleUser,
				Content: []providers.ContentBlock{
					&providers.TextContent{Text: "Hello, how are you?"},
				},
			},
			minToks: 4, // base overhead + text tokens
		},
		{
			name: "tool use message",
			message: &providers.Message{
				Role: providers.RoleAssistant,
				Content: []providers.ContentBlock{
					&providers.ToolUseContent{
						ID:    "tool_123",
						Name:  "file_read",
						Input: map[string]interface{}{"path": "/tmp/test.go"},
					},
				},
			},
			minToks: 20, // base overhead + tool overhead + content
		},
		{
			name: "tool result message",
			message: &providers.Message{
				Role: providers.RoleTool,
				Content: []providers.ContentBlock{
					&providers.ToolResultContent{
						ToolUseID: "tool_123",
						Content:   "File contents here...",
						IsError:   false,
					},
				},
			},
			minToks: 10, // base overhead + tool result overhead + content
		},
		{
			name: "message with multiple blocks",
			message: &providers.Message{
				Role: providers.RoleAssistant,
				Content: []providers.ContentBlock{
					&providers.TextContent{Text: "Let me read that file."},
					&providers.ToolUseContent{
						ID:    "tool_456",
						Name:  "file_read",
						Input: map[string]interface{}{"path": "/tmp/test.go"},
					},
				},
			},
			minToks: 25, // multiple blocks
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimator.EstimateMessage(tt.message)
			if got < tt.minToks {
				t.Errorf("EstimateMessage() = %d, want at least %d", got, tt.minToks)
			}
		})
	}
}

func TestSimpleEstimator_EstimateMessages(t *testing.T) {
	estimator := NewSimpleEstimator()

	messages := []providers.Message{
		{
			Role: providers.RoleUser,
			Content: []providers.ContentBlock{
				&providers.TextContent{Text: "Hello"},
			},
		},
		{
			Role: providers.RoleAssistant,
			Content: []providers.ContentBlock{
				&providers.TextContent{Text: "Hi there!"},
			},
		},
		{
			Role: providers.RoleUser,
			Content: []providers.ContentBlock{
				&providers.TextContent{Text: "How are you?"},
			},
		},
	}

	total := estimator.EstimateMessages(messages)

	// Should be sum of individual estimates
	expected := 0
	for i := range messages {
		expected += estimator.EstimateMessage(&messages[i])
	}

	if total != expected {
		t.Errorf("EstimateMessages() = %d, want %d", total, expected)
	}
}

func TestSimpleEstimator_CustomCharsPerToken(t *testing.T) {
	// Test with custom chars per token ratio
	estimator := &SimpleEstimator{CharsPerToken: 3.0}

	text := "hello world" // 11 chars
	got := estimator.EstimateTokens(text)
	expected := 3 // 11 / 3 = 3.67 -> 3

	if got != expected {
		t.Errorf("EstimateTokens with custom ratio = %d, want %d", got, expected)
	}
}

func TestSimpleEstimator_ZeroCharsPerToken(t *testing.T) {
	// Should default to 4.0 if CharsPerToken is 0
	estimator := &SimpleEstimator{CharsPerToken: 0}

	text := "hello" // 5 chars
	got := estimator.EstimateTokens(text)
	expected := 1 // 5 / 4 = 1.25 -> 1

	if got != expected {
		t.Errorf("EstimateTokens with zero ratio = %d, want %d", got, expected)
	}
}

func TestNewTokenInfo(t *testing.T) {
	estimator := NewSimpleEstimator()

	t.Run("user message", func(t *testing.T) {
		msg := &providers.Message{
			Role: providers.RoleUser,
			Content: []providers.ContentBlock{
				&providers.TextContent{Text: "Hello"},
			},
		}

		info := NewTokenInfo(0, msg, estimator)

		if info.MessageIndex != 0 {
			t.Errorf("MessageIndex = %d, want 0", info.MessageIndex)
		}
		if info.Role != providers.RoleUser {
			t.Errorf("Role = %s, want user", info.Role)
		}
		if info.IsPrunable {
			t.Error("User message should not be prunable")
		}
		if info.Tokens <= 0 {
			t.Error("Tokens should be > 0")
		}
	})

	t.Run("tool result message", func(t *testing.T) {
		msg := &providers.Message{
			Role: providers.RoleTool,
			Content: []providers.ContentBlock{
				&providers.ToolResultContent{
					ToolUseID: "tool_123",
					Content:   "Result",
				},
			},
		}

		info := NewTokenInfo(5, msg, estimator)

		if info.MessageIndex != 5 {
			t.Errorf("MessageIndex = %d, want 5", info.MessageIndex)
		}
		if !info.IsPrunable {
			t.Error("Tool result should be prunable")
		}
		if info.ToolUseID != "tool_123" {
			t.Errorf("ToolUseID = %s, want tool_123", info.ToolUseID)
		}
	})
}

func TestEstimateSystemPrompt(t *testing.T) {
	prompt := "You are a helpful coding assistant."
	tokens := EstimateSystemPrompt(prompt)

	// Should include some overhead
	estimator := NewSimpleEstimator()
	baseTokens := estimator.EstimateTokens(prompt)

	if tokens <= baseTokens {
		t.Errorf("EstimateSystemPrompt should add overhead, got %d, base %d", tokens, baseTokens)
	}
}

func TestIsPrunableMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  *providers.Message
		expected bool
	}{
		{
			name: "user message",
			message: &providers.Message{
				Role: providers.RoleUser,
				Content: []providers.ContentBlock{
					&providers.TextContent{Text: "Hello"},
				},
			},
			expected: false,
		},
		{
			name: "assistant text message",
			message: &providers.Message{
				Role: providers.RoleAssistant,
				Content: []providers.ContentBlock{
					&providers.TextContent{Text: "Hello"},
				},
			},
			expected: false,
		},
		{
			name: "tool role message",
			message: &providers.Message{
				Role: providers.RoleTool,
				Content: []providers.ContentBlock{
					&providers.ToolResultContent{
						ToolUseID: "123",
						Content:   "result",
					},
				},
			},
			expected: true,
		},
		{
			name: "assistant with tool result",
			message: &providers.Message{
				Role: providers.RoleAssistant,
				Content: []providers.ContentBlock{
					&providers.ToolResultContent{
						ToolUseID: "123",
						Content:   "result",
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPrunableMessage(tt.message)
			if got != tt.expected {
				t.Errorf("isPrunableMessage() = %v, want %v", got, tt.expected)
			}
		})
	}
}
