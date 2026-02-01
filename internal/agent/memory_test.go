package agent

import (
	"testing"

	"github.com/taaha3244/potus/internal/context"
	"github.com/taaha3244/potus/internal/providers"
)

func TestNewMemory(t *testing.T) {
	t.Run("with nil estimator uses default", func(t *testing.T) {
		mem := NewMemory(nil)
		if mem == nil {
			t.Fatal("NewMemory returned nil")
		}
		if mem.estimator == nil {
			t.Error("estimator should not be nil")
		}
	})

	t.Run("with custom estimator", func(t *testing.T) {
		estimator := context.NewSimpleEstimator()
		mem := NewMemory(estimator)
		if mem.estimator != estimator {
			t.Error("estimator not set correctly")
		}
	})
}

func TestMemory_AddUserMessage(t *testing.T) {
	mem := NewMemory(nil)

	tokens := mem.AddUserMessage("Hello")

	if mem.Count() != 1 {
		t.Errorf("expected 1 message, got %d", mem.Count())
	}

	if tokens <= 0 {
		t.Error("AddUserMessage should return positive token count")
	}

	msgs := mem.GetMessages()
	if msgs[0].Role != providers.RoleUser {
		t.Errorf("expected user role, got %s", msgs[0].Role)
	}

	if len(msgs[0].Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(msgs[0].Content))
	}

	textContent, ok := msgs[0].Content[0].(*providers.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}

	if textContent.Text != "Hello" {
		t.Errorf("expected 'Hello', got %s", textContent.Text)
	}
}

func TestMemory_AddMessage(t *testing.T) {
	mem := NewMemory(nil)

	msg := &providers.Message{
		Role: providers.RoleAssistant,
		Content: []providers.ContentBlock{
			&providers.TextContent{Text: "Response"},
		},
	}

	tokens := mem.AddMessage(msg)

	if mem.Count() != 1 {
		t.Errorf("expected 1 message, got %d", mem.Count())
	}

	if tokens <= 0 {
		t.Error("AddMessage should return positive token count")
	}

	msgs := mem.GetMessages()
	if msgs[0].Role != providers.RoleAssistant {
		t.Errorf("expected assistant role, got %s", msgs[0].Role)
	}
}

func TestMemory_AddMessage_ToolResult(t *testing.T) {
	mem := NewMemory(nil)

	msg := &providers.Message{
		Role: providers.RoleTool,
		Content: []providers.ContentBlock{
			&providers.ToolResultContent{
				ToolUseID: "tool_123",
				Content:   "Tool result content",
				IsError:   false,
			},
		},
	}

	mem.AddMessage(msg)

	tokenInfo := mem.GetTokenInfo()
	if len(tokenInfo) != 1 {
		t.Fatalf("expected 1 token info, got %d", len(tokenInfo))
	}

	if !tokenInfo[0].IsPrunable {
		t.Error("Tool result should be prunable")
	}

	if tokenInfo[0].ToolUseID != "tool_123" {
		t.Errorf("ToolUseID = %s, want tool_123", tokenInfo[0].ToolUseID)
	}
}

func TestMemory_AddMessage_ToolUse(t *testing.T) {
	mem := NewMemory(nil)

	msg := &providers.Message{
		Role: providers.RoleAssistant,
		Content: []providers.ContentBlock{
			&providers.ToolUseContent{
				ID:    "tool_123",
				Name:  "file_read",
				Input: map[string]interface{}{"path": "/test.txt"},
			},
		},
	}

	mem.AddMessage(msg)

	tokenInfo := mem.GetTokenInfo()
	if tokenInfo[0].ToolName != "file_read" {
		t.Errorf("ToolName = %s, want file_read", tokenInfo[0].ToolName)
	}
}

func TestMemory_GetMessages(t *testing.T) {
	mem := NewMemory(nil)

	mem.AddUserMessage("Message 1")
	mem.AddUserMessage("Message 2")
	mem.AddUserMessage("Message 3")

	msgs := mem.GetMessages()
	if len(msgs) != 3 {
		t.Errorf("expected 3 messages, got %d", len(msgs))
	}

	// Verify copy is returned
	msgs[0].Content = []providers.ContentBlock{}
	if len(mem.GetMessages()[0].Content) == 0 {
		t.Error("GetMessages() should return a copy, not the original slice")
	}
}

func TestMemory_GetTokenInfo(t *testing.T) {
	mem := NewMemory(nil)

	mem.AddUserMessage("Message 1")
	mem.AddUserMessage("Message 2")

	info := mem.GetTokenInfo()
	if len(info) != 2 {
		t.Errorf("expected 2 token info entries, got %d", len(info))
	}

	// Verify copy is returned
	originalInfo := mem.GetTokenInfo()
	info[0].Tokens = 9999
	if originalInfo[0].Tokens == 9999 {
		t.Error("GetTokenInfo() should return a copy")
	}
}

func TestMemory_GetTotalTokens(t *testing.T) {
	mem := NewMemory(nil)

	// Set system tokens
	mem.SetSystemTokens(100)

	// Add message
	mem.AddUserMessage("Hello world")

	total := mem.GetTotalTokens()
	messageTokens := mem.GetMessageTokens()

	// Total should be system + message tokens
	if total != 100+messageTokens {
		t.Errorf("GetTotalTokens() = %d, want %d", total, 100+messageTokens)
	}
}

func TestMemory_SystemTokens(t *testing.T) {
	mem := NewMemory(nil)

	mem.SetSystemTokens(500)

	if mem.GetSystemTokens() != 500 {
		t.Errorf("GetSystemTokens() = %d, want 500", mem.GetSystemTokens())
	}
}

func TestMemory_ReplaceMessages(t *testing.T) {
	mem := NewMemory(nil)

	// Add initial messages
	mem.AddUserMessage("Message 1")
	mem.AddUserMessage("Message 2")
	mem.AddUserMessage("Message 3")

	if mem.Count() != 3 {
		t.Fatalf("expected 3 messages before replace, got %d", mem.Count())
	}

	originalTokens := mem.GetMessageTokens()

	// Replace with fewer messages
	newMsgs := []providers.Message{
		{
			Role:    providers.RoleUser,
			Content: []providers.ContentBlock{&providers.TextContent{Text: "Summary"}},
		},
	}

	mem.ReplaceMessages(newMsgs)

	if mem.Count() != 1 {
		t.Errorf("expected 1 message after replace, got %d", mem.Count())
	}

	// Token count should be recalculated
	if mem.GetMessageTokens() >= originalTokens {
		t.Error("Token count should decrease after replacing with fewer messages")
	}

	// Token info should be rebuilt
	info := mem.GetTokenInfo()
	if len(info) != 1 {
		t.Errorf("expected 1 token info entry, got %d", len(info))
	}
	if info[0].MessageIndex != 0 {
		t.Errorf("MessageIndex = %d, want 0", info[0].MessageIndex)
	}
}

func TestMemory_Clear(t *testing.T) {
	mem := NewMemory(nil)

	mem.AddUserMessage("Message 1")
	mem.AddUserMessage("Message 2")
	mem.SetSystemTokens(100)

	if mem.Count() != 2 {
		t.Errorf("expected 2 messages before clear, got %d", mem.Count())
	}

	mem.Clear()

	if mem.Count() != 0 {
		t.Errorf("expected 0 messages after clear, got %d", mem.Count())
	}

	if mem.GetMessageTokens() != 0 {
		t.Error("Message tokens should be 0 after clear")
	}

	// System tokens should be preserved
	if mem.GetSystemTokens() != 100 {
		t.Error("System tokens should be preserved after clear")
	}
}

func TestMemory_LastMessage(t *testing.T) {
	t.Run("empty memory", func(t *testing.T) {
		mem := NewMemory(nil)
		last := mem.LastMessage()
		if last != nil {
			t.Error("LastMessage() should return nil for empty memory")
		}
	})

	t.Run("with messages", func(t *testing.T) {
		mem := NewMemory(nil)
		mem.AddUserMessage("First")
		mem.AddUserMessage("Last")

		last := mem.LastMessage()
		if last == nil {
			t.Fatal("LastMessage() should not return nil")
		}

		textContent := last.Content[0].(*providers.TextContent)
		if textContent.Text != "Last" {
			t.Errorf("LastMessage text = %s, want 'Last'", textContent.Text)
		}
	})
}

func TestMemory_GetTokenSummary(t *testing.T) {
	mem := NewMemory(nil)

	mem.SetSystemTokens(100)
	mem.AddUserMessage("User message")

	// Add a prunable message (tool result)
	toolMsg := &providers.Message{
		Role: providers.RoleTool,
		Content: []providers.ContentBlock{
			&providers.ToolResultContent{
				ToolUseID: "tool_1",
				Content:   "Tool result content",
			},
		},
	}
	mem.AddMessage(toolMsg)

	summary := mem.GetTokenSummary()

	if summary.SystemTokens != 100 {
		t.Errorf("SystemTokens = %d, want 100", summary.SystemTokens)
	}

	if summary.MessageCount != 2 {
		t.Errorf("MessageCount = %d, want 2", summary.MessageCount)
	}

	if summary.TotalTokens != summary.SystemTokens+summary.MessageTokens {
		t.Error("TotalTokens should equal SystemTokens + MessageTokens")
	}

	if summary.PrunableTokens <= 0 {
		t.Error("PrunableTokens should be > 0 (tool result is prunable)")
	}
}

func TestMemory_IsPrunable(t *testing.T) {
	mem := NewMemory(nil)

	tests := []struct {
		name     string
		msg      *providers.Message
		expected bool
	}{
		{
			name: "user message",
			msg: &providers.Message{
				Role:    providers.RoleUser,
				Content: []providers.ContentBlock{&providers.TextContent{Text: "Hello"}},
			},
			expected: false,
		},
		{
			name: "assistant text message",
			msg: &providers.Message{
				Role:    providers.RoleAssistant,
				Content: []providers.ContentBlock{&providers.TextContent{Text: "Response"}},
			},
			expected: false,
		},
		{
			name: "tool role message",
			msg: &providers.Message{
				Role: providers.RoleTool,
				Content: []providers.ContentBlock{
					&providers.ToolResultContent{ToolUseID: "1", Content: "Result"},
				},
			},
			expected: true,
		},
		{
			name: "assistant with tool result",
			msg: &providers.Message{
				Role: providers.RoleAssistant,
				Content: []providers.ContentBlock{
					&providers.ToolResultContent{ToolUseID: "1", Content: "Result"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mem.isPrunable(tt.msg)
			if got != tt.expected {
				t.Errorf("isPrunable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMemory_ExtractToolName(t *testing.T) {
	mem := NewMemory(nil)

	t.Run("no tool use", func(t *testing.T) {
		msg := &providers.Message{
			Role:    providers.RoleUser,
			Content: []providers.ContentBlock{&providers.TextContent{Text: "Hello"}},
		}

		name := mem.extractToolName(msg)
		if name != "" {
			t.Errorf("extractToolName() = %s, want empty string", name)
		}
	})

	t.Run("with tool use", func(t *testing.T) {
		msg := &providers.Message{
			Role: providers.RoleAssistant,
			Content: []providers.ContentBlock{
				&providers.ToolUseContent{
					ID:   "tool_1",
					Name: "file_read",
				},
			},
		}

		name := mem.extractToolName(msg)
		if name != "file_read" {
			t.Errorf("extractToolName() = %s, want file_read", name)
		}
	})
}

func TestMemory_ExtractToolUseID(t *testing.T) {
	mem := NewMemory(nil)

	t.Run("no tool result", func(t *testing.T) {
		msg := &providers.Message{
			Role:    providers.RoleUser,
			Content: []providers.ContentBlock{&providers.TextContent{Text: "Hello"}},
		}

		id := mem.extractToolUseID(msg)
		if id != "" {
			t.Errorf("extractToolUseID() = %s, want empty string", id)
		}
	})

	t.Run("with tool result", func(t *testing.T) {
		msg := &providers.Message{
			Role: providers.RoleTool,
			Content: []providers.ContentBlock{
				&providers.ToolResultContent{
					ToolUseID: "tool_123",
					Content:   "Result",
				},
			},
		}

		id := mem.extractToolUseID(msg)
		if id != "tool_123" {
			t.Errorf("extractToolUseID() = %s, want tool_123", id)
		}
	})
}

func TestMemory_GetEstimator(t *testing.T) {
	estimator := context.NewSimpleEstimator()
	mem := NewMemory(estimator)

	if mem.GetEstimator() != estimator {
		t.Error("GetEstimator() should return the configured estimator")
	}
}

func TestMemory_Concurrency(t *testing.T) {
	mem := NewMemory(nil)

	done := make(chan bool, 20)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(n int) {
			mem.AddUserMessage("concurrent message")
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			_ = mem.GetMessages()
			_ = mem.Count()
			_ = mem.GetTokenInfo()
			_ = mem.GetTotalTokens()
			_ = mem.GetTokenSummary()
			done <- true
		}()
	}

	for i := 0; i < 20; i++ {
		<-done
	}

	if mem.Count() != 10 {
		t.Errorf("expected 10 messages after concurrent writes, got %d", mem.Count())
	}
}
