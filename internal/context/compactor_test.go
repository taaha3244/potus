package context

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/taaha3244/potus/internal/providers"
)

// mockProvider implements providers.Provider for testing
type mockProvider struct {
	response    string
	shouldError bool
	errorMsg    string
}

func (m *mockProvider) Chat(ctx context.Context, req *providers.ChatRequest) (<-chan providers.ChatEvent, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMsg)
	}

	events := make(chan providers.ChatEvent, 2)
	go func() {
		defer close(events)
		events <- providers.ChatEvent{
			Type:    providers.EventTypeTextDelta,
			Content: m.response,
		}
		events <- providers.ChatEvent{
			Type: providers.EventTypeMessageDone,
		}
	}()
	return events, nil
}

func (m *mockProvider) Name() string                                            { return "mock" }
func (m *mockProvider) ListModels(ctx context.Context) ([]providers.Model, error) { return nil, nil }
func (m *mockProvider) SupportsTools() bool                                     { return true }
func (m *mockProvider) SupportsVision() bool                                    { return true }

// mockStreamErrorProvider returns an error through the event stream
type mockStreamErrorProvider struct{}

func (m *mockStreamErrorProvider) Chat(ctx context.Context, req *providers.ChatRequest) (<-chan providers.ChatEvent, error) {
	events := make(chan providers.ChatEvent, 1)
	go func() {
		defer close(events)
		events <- providers.ChatEvent{
			Type:  providers.EventTypeError,
			Error: errors.New("stream error"),
		}
	}()
	return events, nil
}

func (m *mockStreamErrorProvider) Name() string                                            { return "mock-error" }
func (m *mockStreamErrorProvider) ListModels(ctx context.Context) ([]providers.Model, error) { return nil, nil }
func (m *mockStreamErrorProvider) SupportsTools() bool                                     { return true }
func (m *mockStreamErrorProvider) SupportsVision() bool                                    { return true }

func TestNewCompactor(t *testing.T) {
	t.Run("with defaults", func(t *testing.T) {
		compactor := NewCompactor(CompactorConfig{})

		if compactor == nil {
			t.Fatal("NewCompactor returned nil")
		}

		if compactor.protectedMessages != DefaultProtectedMessages {
			t.Errorf("protectedMessages = %d, want %d", compactor.protectedMessages, DefaultProtectedMessages)
		}

		if compactor.maxSummaryTokens != DefaultMaxSummaryTokens {
			t.Errorf("maxSummaryTokens = %d, want %d", compactor.maxSummaryTokens, DefaultMaxSummaryTokens)
		}

		if compactor.estimator == nil {
			t.Error("estimator should not be nil")
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		provider := &mockProvider{response: "test"}
		estimator := NewSimpleEstimator()

		compactor := NewCompactor(CompactorConfig{
			Provider:          provider,
			Estimator:         estimator,
			ProtectedMessages: 10,
			MaxSummaryTokens:  500,
		})

		if compactor.protectedMessages != 10 {
			t.Errorf("protectedMessages = %d, want 10", compactor.protectedMessages)
		}

		if compactor.maxSummaryTokens != 500 {
			t.Errorf("maxSummaryTokens = %d, want 500", compactor.maxSummaryTokens)
		}

		if compactor.provider == nil {
			t.Error("provider should not be nil")
		}
	})
}

func TestCompactor_Compact_NotEnoughMessages(t *testing.T) {
	compactor := NewCompactor(CompactorConfig{
		ProtectedMessages: 6,
	})

	// Only 4 messages - less than protected
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: []providers.ContentBlock{&providers.TextContent{Text: "msg1"}}},
		{Role: providers.RoleAssistant, Content: []providers.ContentBlock{&providers.TextContent{Text: "msg2"}}},
		{Role: providers.RoleUser, Content: []providers.ContentBlock{&providers.TextContent{Text: "msg3"}}},
		{Role: providers.RoleAssistant, Content: []providers.ContentBlock{&providers.TextContent{Text: "msg4"}}},
	}

	result, compactResult, err := compactor.Compact(context.Background(), messages)
	if err != nil {
		t.Fatalf("Compact() error = %v", err)
	}

	if len(result) != len(messages) {
		t.Errorf("Expected %d messages, got %d", len(messages), len(result))
	}

	if compactResult.SummarizedMessages != 0 {
		t.Errorf("SummarizedMessages = %d, want 0", compactResult.SummarizedMessages)
	}

	if compactResult.OriginalMessages != 4 {
		t.Errorf("OriginalMessages = %d, want 4", compactResult.OriginalMessages)
	}
}

func TestCompactor_Compact_WithProvider(t *testing.T) {
	provider := &mockProvider{response: "This is a summary of the conversation."}

	compactor := NewCompactor(CompactorConfig{
		Provider:          provider,
		ProtectedMessages: 2,
		MaxSummaryTokens:  1000,
	})

	// 6 messages - 2 will be preserved, 4 will be summarized
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: []providers.ContentBlock{&providers.TextContent{Text: "old msg 1"}}},
		{Role: providers.RoleAssistant, Content: []providers.ContentBlock{&providers.TextContent{Text: "old msg 2"}}},
		{Role: providers.RoleUser, Content: []providers.ContentBlock{&providers.TextContent{Text: "old msg 3"}}},
		{Role: providers.RoleAssistant, Content: []providers.ContentBlock{&providers.TextContent{Text: "old msg 4"}}},
		{Role: providers.RoleUser, Content: []providers.ContentBlock{&providers.TextContent{Text: "recent msg 1"}}},
		{Role: providers.RoleAssistant, Content: []providers.ContentBlock{&providers.TextContent{Text: "recent msg 2"}}},
	}

	result, compactResult, err := compactor.Compact(context.Background(), messages)
	if err != nil {
		t.Fatalf("Compact() error = %v", err)
	}

	// Should have: summary message + acknowledgment + 2 preserved = 4
	if len(result) != 4 {
		t.Errorf("Expected 4 messages, got %d", len(result))
	}

	if compactResult.SummarizedMessages != 4 {
		t.Errorf("SummarizedMessages = %d, want 4", compactResult.SummarizedMessages)
	}

	if compactResult.OriginalMessages != 6 {
		t.Errorf("OriginalMessages = %d, want 6", compactResult.OriginalMessages)
	}

	if compactResult.Summary != "This is a summary of the conversation." {
		t.Errorf("Summary = %q, want %q", compactResult.Summary, "This is a summary of the conversation.")
	}

	// First message should be the summary
	firstContent, ok := result[0].Content[0].(*providers.TextContent)
	if !ok {
		t.Fatal("First message content is not TextContent")
	}
	if result[0].Role != providers.RoleUser {
		t.Error("Summary message should have user role")
	}
	if !contains(firstContent.Text, "[Previous Conversation Summary]") {
		t.Error("Summary should contain summary marker")
	}

	// Second message should be acknowledgment
	if result[1].Role != providers.RoleAssistant {
		t.Error("Acknowledgment should have assistant role")
	}

	// Last two should be preserved messages
	lastContent, _ := result[3].Content[0].(*providers.TextContent)
	if lastContent.Text != "recent msg 2" {
		t.Errorf("Last preserved message = %q, want %q", lastContent.Text, "recent msg 2")
	}
}

func TestCompactor_Compact_ProviderError(t *testing.T) {
	provider := &mockProvider{
		shouldError: true,
		errorMsg:    "provider failed",
	}

	compactor := NewCompactor(CompactorConfig{
		Provider:          provider,
		ProtectedMessages: 2,
	})

	messages := []providers.Message{
		{Role: providers.RoleUser, Content: []providers.ContentBlock{&providers.TextContent{Text: "msg1"}}},
		{Role: providers.RoleAssistant, Content: []providers.ContentBlock{&providers.TextContent{Text: "msg2"}}},
		{Role: providers.RoleUser, Content: []providers.ContentBlock{&providers.TextContent{Text: "msg3"}}},
		{Role: providers.RoleAssistant, Content: []providers.ContentBlock{&providers.TextContent{Text: "msg4"}}},
	}

	_, _, err := compactor.Compact(context.Background(), messages)
	if err == nil {
		t.Error("Expected error from provider")
	}
}

func TestCompactor_Compact_StreamError(t *testing.T) {
	provider := &mockStreamErrorProvider{}

	compactor := NewCompactor(CompactorConfig{
		Provider:          provider,
		ProtectedMessages: 2,
	})

	messages := []providers.Message{
		{Role: providers.RoleUser, Content: []providers.ContentBlock{&providers.TextContent{Text: "msg1"}}},
		{Role: providers.RoleAssistant, Content: []providers.ContentBlock{&providers.TextContent{Text: "msg2"}}},
		{Role: providers.RoleUser, Content: []providers.ContentBlock{&providers.TextContent{Text: "msg3"}}},
		{Role: providers.RoleAssistant, Content: []providers.ContentBlock{&providers.TextContent{Text: "msg4"}}},
	}

	_, _, err := compactor.Compact(context.Background(), messages)
	if err == nil {
		t.Error("Expected error from stream")
	}
}

func TestCompactor_ShouldCompact(t *testing.T) {
	compactor := NewCompactor(CompactorConfig{
		ProtectedMessages: 6,
	})

	tests := []struct {
		name          string
		messageCount  int
		currentTokens int
		maxTokens     int
		expected      bool
	}{
		{
			name:          "not enough messages",
			messageCount:  7,
			currentTokens: 95000,
			maxTokens:     100000,
			expected:      false, // 7 <= 6+2
		},
		{
			name:          "enough messages but below threshold",
			messageCount:  10,
			currentTokens: 80000,
			maxTokens:     100000,
			expected:      false, // 80% < 90%
		},
		{
			name:          "enough messages at threshold",
			messageCount:  10,
			currentTokens: 90000,
			maxTokens:     100000,
			expected:      true, // 90% >= 90%
		},
		{
			name:          "enough messages above threshold",
			messageCount:  10,
			currentTokens: 95000,
			maxTokens:     100000,
			expected:      true, // 95% >= 90%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages := make([]providers.Message, tt.messageCount)
			for i := 0; i < tt.messageCount; i++ {
				messages[i] = providers.Message{
					Role: providers.RoleUser,
					Content: []providers.ContentBlock{
						&providers.TextContent{Text: "test"},
					},
				}
			}

			got := compactor.ShouldCompact(messages, tt.currentTokens, tt.maxTokens)
			if got != tt.expected {
				t.Errorf("ShouldCompact() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCompactor_EstimateSavings(t *testing.T) {
	compactor := NewCompactor(CompactorConfig{
		ProtectedMessages: 2,
	})

	t.Run("not enough messages", func(t *testing.T) {
		messages := []providers.Message{
			{Role: providers.RoleUser, Content: []providers.ContentBlock{&providers.TextContent{Text: "msg1"}}},
		}

		savings := compactor.EstimateSavings(messages)
		if savings != 0 {
			t.Errorf("EstimateSavings = %d, want 0", savings)
		}
	})

	t.Run("with summarizable messages", func(t *testing.T) {
		// Create messages with substantial content (must be > 1000 tokens to overcome 200 token overhead)
		// Each longText is ~500 chars = ~125 tokens. With 4 messages = ~500 tokens to summarize
		// After 80% reduction = ~100 tokens, plus 200 overhead = 300 tokens
		// Need original tokens > 300 for positive savings
		// So we need much longer text
		longText := strings.Repeat("This is a longer message with substantial content that will have many more tokens when we estimate the token count. ", 20)

		messages := []providers.Message{
			{Role: providers.RoleUser, Content: []providers.ContentBlock{&providers.TextContent{Text: longText}}},
			{Role: providers.RoleAssistant, Content: []providers.ContentBlock{&providers.TextContent{Text: longText}}},
			{Role: providers.RoleUser, Content: []providers.ContentBlock{&providers.TextContent{Text: longText}}},
			{Role: providers.RoleAssistant, Content: []providers.ContentBlock{&providers.TextContent{Text: longText}}},
			{Role: providers.RoleUser, Content: []providers.ContentBlock{&providers.TextContent{Text: "recent"}}},
			{Role: providers.RoleAssistant, Content: []providers.ContentBlock{&providers.TextContent{Text: "recent"}}},
		}

		savings := compactor.EstimateSavings(messages)
		// Savings should be positive (original - ~20% - 200 overhead)
		// With ~2000 tokens to summarize: savings = 2000 - (400 + 200) = 1400
		if savings <= 0 {
			t.Errorf("EstimateSavings = %d, want > 0", savings)
		}
	})
}

func TestCompactor_FormatConversation(t *testing.T) {
	compactor := NewCompactor(CompactorConfig{})

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
				&providers.ToolUseContent{
					ID:    "tool_1",
					Name:  "file_read",
					Input: map[string]interface{}{"path": "/test"},
				},
			},
		},
		{
			Role: providers.RoleTool,
			Content: []providers.ContentBlock{
				&providers.ToolResultContent{
					ToolUseID: "tool_1",
					Content:   "File contents",
					IsError:   false,
				},
			},
		},
		{
			Role: providers.RoleTool,
			Content: []providers.ContentBlock{
				&providers.ToolResultContent{
					ToolUseID: "tool_2",
					Content:   "Error occurred",
					IsError:   true,
				},
			},
		},
	}

	result := compactor.formatConversation(messages)

	if !contains(result, "User: Hello") {
		t.Error("Should contain user message")
	}

	if !contains(result, "Assistant: Hi there!") {
		t.Error("Should contain assistant text")
	}

	if !contains(result, "[Called tool: file_read]") {
		t.Error("Should contain tool use")
	}

	if !contains(result, "Tool Result (success): File contents") {
		t.Error("Should contain successful tool result")
	}

	if !contains(result, "Tool Result (error): Error occurred") {
		t.Error("Should contain error tool result")
	}
}

func TestCompactor_FormatConversation_TruncatesLongResults(t *testing.T) {
	compactor := NewCompactor(CompactorConfig{})

	// Create a very long tool result (> 500 chars)
	longContent := make([]byte, 600)
	for i := range longContent {
		longContent[i] = 'x'
	}

	messages := []providers.Message{
		{
			Role: providers.RoleTool,
			Content: []providers.ContentBlock{
				&providers.ToolResultContent{
					ToolUseID: "tool_1",
					Content:   string(longContent),
					IsError:   false,
				},
			},
		},
	}

	result := compactor.formatConversation(messages)

	if !contains(result, "...[truncated]") {
		t.Error("Long content should be truncated")
	}

	// Should not contain the full content
	if contains(result, string(longContent)) {
		t.Error("Full long content should not be present")
	}
}

func TestFormatRole(t *testing.T) {
	tests := []struct {
		role     providers.MessageRole
		expected string
	}{
		{providers.RoleUser, "User"},
		{providers.RoleAssistant, "Assistant"},
		{providers.RoleSystem, "System"},
		{providers.RoleTool, "Tool"},
		{providers.MessageRole("unknown"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			got := formatRole(tt.role)
			if got != tt.expected {
				t.Errorf("formatRole(%s) = %s, want %s", tt.role, got, tt.expected)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
