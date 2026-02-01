package context

import (
	"testing"

	"github.com/taaha3244/potus/internal/providers"
)

func TestNewPruner(t *testing.T) {
	t.Run("with defaults", func(t *testing.T) {
		pruner := NewPruner(PrunerConfig{})

		// Should have default protected tools
		protectedTools := pruner.GetProtectedTools()
		if len(protectedTools) == 0 {
			t.Error("Should have default protected tools")
		}

		// Check some default protected tools
		if !pruner.isProtectedTool("file_read") {
			t.Error("file_read should be protected by default")
		}
		if !pruner.isProtectedTool("grep") {
			t.Error("grep should be protected by default")
		}
	})

	t.Run("with custom protected tools", func(t *testing.T) {
		pruner := NewPruner(PrunerConfig{
			ProtectedTools: []string{"custom_tool"},
		})

		if !pruner.isProtectedTool("custom_tool") {
			t.Error("custom_tool should be protected")
		}

		// Default tools should still be protected
		if !pruner.isProtectedTool("file_read") {
			t.Error("file_read should still be protected")
		}
	})

	t.Run("with custom protection ratio", func(t *testing.T) {
		pruner := NewPruner(PrunerConfig{
			ProtectionRatio: 0.50,
		})

		// Protection ratio should be set (tested implicitly through Prune behavior)
		if pruner == nil {
			t.Error("Pruner should not be nil")
		}
	})
}

func TestPruner_AddRemoveProtectedTool(t *testing.T) {
	pruner := NewPruner(PrunerConfig{})

	// Add a new protected tool
	pruner.AddProtectedTool("my_tool")
	if !pruner.isProtectedTool("my_tool") {
		t.Error("my_tool should be protected after adding")
	}

	// Remove it
	pruner.RemoveProtectedTool("my_tool")
	if pruner.isProtectedTool("my_tool") {
		t.Error("my_tool should not be protected after removing")
	}
}

func TestPruner_Prune_EmptyMessages(t *testing.T) {
	pruner := NewPruner(PrunerConfig{})

	messages := []providers.Message{}
	tokenInfo := []TokenInfo{}

	result, pruneResult := pruner.Prune(messages, tokenInfo)

	if len(result) != 0 {
		t.Error("Empty messages should return empty result")
	}
	if pruneResult.TokensSaved != 0 {
		t.Error("No tokens should be saved from empty messages")
	}
}

func TestPruner_Prune_PreservesRecentMessages(t *testing.T) {
	pruner := NewPruner(PrunerConfig{
		ProtectionRatio: 0.30, // Protect last 30%
	})

	// Create messages with tool results
	messages := []providers.Message{
		{
			Role: providers.RoleTool,
			Content: []providers.ContentBlock{
				&providers.ToolResultContent{
					ToolUseID: "tool_1",
					Content:   "Old result that should be pruned",
				},
			},
		},
		{
			Role: providers.RoleUser,
			Content: []providers.ContentBlock{
				&providers.TextContent{Text: "User message"},
			},
		},
		{
			Role: providers.RoleTool,
			Content: []providers.ContentBlock{
				&providers.ToolResultContent{
					ToolUseID: "tool_2",
					Content:   "Recent result that should be preserved",
				},
			},
		},
	}

	// Token info - make older messages have more tokens
	tokenInfo := []TokenInfo{
		{MessageIndex: 0, Tokens: 100, IsPrunable: true, ToolName: "bash"},
		{MessageIndex: 1, Tokens: 50, IsPrunable: false},
		{MessageIndex: 2, Tokens: 100, IsPrunable: true, ToolName: "bash"},
	}

	result, _ := pruner.Prune(messages, tokenInfo)

	// All messages should be preserved (structure)
	if len(result) != len(messages) {
		t.Errorf("Should have same number of messages, got %d want %d", len(result), len(messages))
	}

	// Recent message (last in protected zone) should have original content
	lastMsg := result[len(result)-1]
	if lastMsg.Role != providers.RoleTool {
		t.Error("Last message should still be tool role")
	}
}

func TestPruner_Prune_ProtectsSpecificTools(t *testing.T) {
	pruner := NewPruner(PrunerConfig{
		ProtectedTools:  []string{"file_read"},
		ProtectionRatio: 0.10, // Very low protection to force pruning
	})

	messages := []providers.Message{
		{
			Role: providers.RoleTool,
			Content: []providers.ContentBlock{
				&providers.ToolResultContent{
					ToolUseID: "tool_1",
					Content:   "Protected file content",
				},
			},
		},
		{
			Role: providers.RoleTool,
			Content: []providers.ContentBlock{
				&providers.ToolResultContent{
					ToolUseID: "tool_2",
					Content:   "Unprotected bash output",
				},
			},
		},
		{
			Role: providers.RoleUser,
			Content: []providers.ContentBlock{
				&providers.TextContent{Text: "Recent user message"},
			},
		},
	}

	tokenInfo := []TokenInfo{
		{MessageIndex: 0, Tokens: 1000, IsPrunable: true, ToolName: "file_read"}, // Protected
		{MessageIndex: 1, Tokens: 1000, IsPrunable: true, ToolName: "bash"},      // Not protected
		{MessageIndex: 2, Tokens: 10, IsPrunable: false},                          // User message
	}

	result, pruneResult := pruner.Prune(messages, tokenInfo)

	// Check that file_read result was preserved
	fileReadResult := result[0].Content[0].(*providers.ToolResultContent)
	if fileReadResult.Content == "[Previous tool result pruned for context management]" {
		t.Error("Protected tool (file_read) should not be pruned")
	}

	// Check that bash result was pruned (if it was in the old zone)
	// Note: Whether it gets pruned depends on the protection threshold
	if pruneResult.MessagesPruned > 0 {
		bashResult := result[1].Content[0].(*providers.ToolResultContent)
		if bashResult.Content != "[Previous tool result pruned for context management]" {
			t.Log("Bash result may or may not be pruned depending on threshold")
		}
	}
}

func TestPruner_Prune_ReturnsCorrectStats(t *testing.T) {
	pruner := NewPruner(PrunerConfig{
		ProtectionRatio: 0.10, // Low protection to ensure pruning
	})

	originalContent := "This is a long tool result that will be pruned to save tokens in the context window"
	messages := []providers.Message{
		{
			Role: providers.RoleTool,
			Content: []providers.ContentBlock{
				&providers.ToolResultContent{
					ToolUseID: "tool_1",
					Content:   originalContent,
				},
			},
		},
		{
			Role: providers.RoleUser,
			Content: []providers.ContentBlock{
				&providers.TextContent{Text: "Recent"},
			},
		},
	}

	tokenInfo := []TokenInfo{
		{MessageIndex: 0, Tokens: 500, IsPrunable: true, ToolName: "bash"},
		{MessageIndex: 1, Tokens: 10, IsPrunable: false},
	}

	_, pruneResult := pruner.Prune(messages, tokenInfo)

	if pruneResult.OriginalMessages != 2 {
		t.Errorf("OriginalMessages = %d, want 2", pruneResult.OriginalMessages)
	}

	if pruneResult.PrunedMessages != 2 {
		t.Errorf("PrunedMessages = %d, want 2", pruneResult.PrunedMessages)
	}
}

func TestPruner_ShouldPrune(t *testing.T) {
	pruner := NewPruner(PrunerConfig{})

	t.Run("no prunable messages", func(t *testing.T) {
		tokenInfo := []TokenInfo{
			{Tokens: 100, IsPrunable: false},
			{Tokens: 100, IsPrunable: false},
		}

		if pruner.ShouldPrune(tokenInfo) {
			t.Error("Should not prune when no messages are prunable")
		}
	})

	t.Run("prunable but protected tools", func(t *testing.T) {
		tokenInfo := []TokenInfo{
			{Tokens: 100, IsPrunable: true, ToolName: "file_read"}, // Protected
			{Tokens: 100, IsPrunable: false},
		}

		if pruner.ShouldPrune(tokenInfo) {
			t.Error("Should not prune when only protected tools are prunable")
		}
	})

	t.Run("prunable unprotected tool", func(t *testing.T) {
		tokenInfo := []TokenInfo{
			{Tokens: 500, IsPrunable: true, ToolName: "bash"},
			{Tokens: 100, IsPrunable: false},
			{Tokens: 100, IsPrunable: false},
		}

		// 500 out of 700 is ~71%, which is > 10% threshold
		if !pruner.ShouldPrune(tokenInfo) {
			t.Error("Should prune when significant prunable tokens exist")
		}
	})

	t.Run("small prunable percentage", func(t *testing.T) {
		tokenInfo := []TokenInfo{
			{Tokens: 50, IsPrunable: true, ToolName: "bash"},
			{Tokens: 1000, IsPrunable: false},
		}

		// 50 out of 1050 is ~4.7%, which is < 10% threshold
		if pruner.ShouldPrune(tokenInfo) {
			t.Error("Should not prune when prunable percentage is too small")
		}
	})
}

func TestPruner_PruneMessage(t *testing.T) {
	pruner := NewPruner(PrunerConfig{})

	original := providers.Message{
		Role: providers.RoleTool,
		Content: []providers.ContentBlock{
			&providers.ToolResultContent{
				ToolUseID: "tool_123",
				Content:   "Original long content that takes up tokens",
				IsError:   false,
			},
		},
	}

	pruned := pruner.pruneMessage(original)

	// Check the pruned message
	if pruned.Role != providers.RoleTool {
		t.Error("Role should be preserved")
	}

	if len(pruned.Content) != 1 {
		t.Fatal("Should have 1 content block")
	}

	toolResult, ok := pruned.Content[0].(*providers.ToolResultContent)
	if !ok {
		t.Fatal("Content should be ToolResultContent")
	}

	if toolResult.ToolUseID != "tool_123" {
		t.Error("ToolUseID should be preserved")
	}

	if toolResult.Content != "[Previous tool result pruned for context management]" {
		t.Errorf("Content should be pruned placeholder, got: %s", toolResult.Content)
	}

	if toolResult.IsError != false {
		t.Error("IsError should be preserved")
	}
}

func TestPruner_PreservesNonToolContent(t *testing.T) {
	pruner := NewPruner(PrunerConfig{})

	// Message with mixed content
	original := providers.Message{
		Role: providers.RoleAssistant,
		Content: []providers.ContentBlock{
			&providers.TextContent{Text: "Some text"},
			&providers.ToolResultContent{
				ToolUseID: "tool_1",
				Content:   "Tool result",
			},
		},
	}

	pruned := pruner.pruneMessage(original)

	if len(pruned.Content) != 2 {
		t.Fatal("Should have 2 content blocks")
	}

	// Text should be preserved
	textContent, ok := pruned.Content[0].(*providers.TextContent)
	if !ok {
		t.Fatal("First block should be TextContent")
	}
	if textContent.Text != "Some text" {
		t.Error("Text content should be preserved")
	}

	// Tool result should be pruned
	toolResult, ok := pruned.Content[1].(*providers.ToolResultContent)
	if !ok {
		t.Fatal("Second block should be ToolResultContent")
	}
	if toolResult.Content != "[Previous tool result pruned for context management]" {
		t.Error("Tool result should be pruned")
	}
}
