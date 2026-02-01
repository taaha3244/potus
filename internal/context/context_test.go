package context

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/taaha3244/potus/internal/providers"
)

func TestContextAction_String(t *testing.T) {
	tests := []struct {
		action   ContextAction
		expected string
	}{
		{ActionNone, "none"},
		{ActionWarn, "warn"},
		{ActionPrune, "prune"},
		{ActionCompact, "compact"},
		{ContextAction(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.action.String()
			if got != tt.expected {
				t.Errorf("String() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestNewManager(t *testing.T) {
	t.Run("with minimal config", func(t *testing.T) {
		manager := NewManager(ManagerConfig{
			MaxTokens:          100000,
			ReserveForResponse: 8192,
		})

		if manager == nil {
			t.Fatal("NewManager returned nil")
		}

		if manager.estimator == nil {
			t.Error("estimator should not be nil")
		}

		if manager.pruner == nil {
			t.Error("pruner should not be nil")
		}

		if manager.budget == nil {
			t.Error("budget should not be nil")
		}

		if manager.projectFiles == nil {
			t.Error("projectFiles should not be nil")
		}

		// Compactor should be nil without provider
		if manager.compactor != nil {
			t.Error("compactor should be nil without provider")
		}
	})

	t.Run("with provider", func(t *testing.T) {
		provider := &mockProvider{response: "test"}
		manager := NewManager(ManagerConfig{
			MaxTokens:          100000,
			ReserveForResponse: 8192,
			Provider:           provider,
		})

		if manager.compactor == nil {
			t.Error("compactor should not be nil with provider")
		}
	})

	t.Run("with all options", func(t *testing.T) {
		eventChan := make(chan ContextEvent, 10)
		manager := NewManager(ManagerConfig{
			MaxTokens:           100000,
			ReserveForResponse:  8192,
			ModelContextSize:    128000,
			WarnThreshold:       0.75,
			CompactThreshold:    0.85,
			AutoCompact:         true,
			AutoPrune:           true,
			ProtectedTools:      []string{"custom_tool"},
			LoadProjectContext:  true,
			ProjectContextFiles: []string{"CUSTOM.md"},
			MaxProjectTokens:    5000,
			EventChan:           eventChan,
		})

		if !manager.autoCompact {
			t.Error("autoCompact should be true")
		}

		if !manager.autoPrune {
			t.Error("autoPrune should be true")
		}

		if manager.eventChan == nil {
			t.Error("eventChan should be set")
		}
	})
}

func TestManager_LoadProjectContext(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a POTUS.md file
	potusContent := "# Project Context\n\nThis is test project context."
	if err := os.WriteFile(filepath.Join(tmpDir, "POTUS.md"), []byte(potusContent), 0644); err != nil {
		t.Fatal(err)
	}

	manager := NewManager(ManagerConfig{
		MaxTokens:          100000,
		ReserveForResponse: 8192,
	})

	err := manager.LoadProjectContext(tmpDir)
	if err != nil {
		t.Fatalf("LoadProjectContext() error = %v", err)
	}

	if manager.projectContext == nil {
		t.Error("projectContext should be set after loading")
	}

	if len(manager.projectContext.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(manager.projectContext.Files))
	}
}

func TestManager_GetProjectContextForPrompt(t *testing.T) {
	manager := NewManager(ManagerConfig{
		MaxTokens:          100000,
		ReserveForResponse: 8192,
	})

	t.Run("no context loaded", func(t *testing.T) {
		result := manager.GetProjectContextForPrompt()
		if result != "" {
			t.Error("Should return empty string when no context loaded")
		}
	})

	t.Run("with context loaded", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "POTUS.md"), []byte("# Test"), 0644); err != nil {
			t.Fatal(err)
		}

		if err := manager.LoadProjectContext(tmpDir); err != nil {
			t.Fatal(err)
		}

		result := manager.GetProjectContextForPrompt()
		if result == "" {
			t.Error("Should return formatted context")
		}
	})
}

func TestManager_GetProjectContextTokens(t *testing.T) {
	manager := NewManager(ManagerConfig{
		MaxTokens:          100000,
		ReserveForResponse: 8192,
	})

	t.Run("no context loaded", func(t *testing.T) {
		tokens := manager.GetProjectContextTokens()
		if tokens != 0 {
			t.Errorf("Tokens = %d, want 0", tokens)
		}
	})

	t.Run("with context loaded", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "POTUS.md"), []byte("# Test Project"), 0644); err != nil {
			t.Fatal(err)
		}

		if err := manager.LoadProjectContext(tmpDir); err != nil {
			t.Fatal(err)
		}

		tokens := manager.GetProjectContextTokens()
		if tokens <= 0 {
			t.Error("Tokens should be > 0 with context loaded")
		}
	})
}

func TestManager_CheckContext(t *testing.T) {
	manager := NewManager(ManagerConfig{
		MaxTokens:          100000,
		ReserveForResponse: 0, // Simpler math
		WarnThreshold:      0.80,
		CompactThreshold:   0.90,
	})

	tests := []struct {
		name          string
		currentTokens int
		expected      ContextAction
	}{
		{
			name:          "below warning",
			currentTokens: 70000,
			expected:      ActionNone,
		},
		{
			name:          "at warning threshold",
			currentTokens: 80000,
			expected:      ActionWarn,
		},
		{
			name:          "between warning and compact",
			currentTokens: 85000,
			expected:      ActionWarn,
		},
		{
			name:          "at compact threshold",
			currentTokens: 90000,
			expected:      ActionCompact,
		},
		{
			name:          "above compact threshold",
			currentTokens: 95000,
			expected:      ActionCompact,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := manager.CheckContext(tt.currentTokens)
			if action != tt.expected {
				t.Errorf("CheckContext(%d) = %v, want %v", tt.currentTokens, action, tt.expected)
			}
		})
	}
}

func TestManager_PrepareContext_NoAction(t *testing.T) {
	manager := NewManager(ManagerConfig{
		MaxTokens:          100000,
		ReserveForResponse: 0,
		WarnThreshold:      0.80,
		CompactThreshold:   0.90,
	})

	messages := []providers.Message{
		{Role: providers.RoleUser, Content: []providers.ContentBlock{&providers.TextContent{Text: "Hello"}}},
	}

	tokenInfo := []TokenInfo{
		{MessageIndex: 0, Tokens: 10, IsPrunable: false},
	}

	result, err := manager.PrepareContext(context.Background(), messages, tokenInfo)
	if err != nil {
		t.Fatalf("PrepareContext() error = %v", err)
	}

	if len(result) != len(messages) {
		t.Errorf("Expected %d messages, got %d", len(messages), len(result))
	}
}

func TestManager_PrepareContext_Warning(t *testing.T) {
	eventChan := make(chan ContextEvent, 10)
	manager := NewManager(ManagerConfig{
		MaxTokens:          100000,
		ReserveForResponse: 0,
		WarnThreshold:      0.80,
		CompactThreshold:   0.90,
		EventChan:          eventChan,
	})

	messages := []providers.Message{
		{Role: providers.RoleUser, Content: []providers.ContentBlock{&providers.TextContent{Text: "Hello"}}},
	}

	tokenInfo := []TokenInfo{
		{MessageIndex: 0, Tokens: 85000, IsPrunable: false}, // 85% - warning level
	}

	result, err := manager.PrepareContext(context.Background(), messages, tokenInfo)
	if err != nil {
		t.Fatalf("PrepareContext() error = %v", err)
	}

	// Messages should be returned unchanged
	if len(result) != len(messages) {
		t.Errorf("Expected %d messages, got %d", len(messages), len(result))
	}

	// Should have emitted warning event
	select {
	case event := <-eventChan:
		if event.Type != EventTypeWarning {
			t.Errorf("Expected warning event, got %v", event.Type)
		}
	default:
		t.Error("Expected warning event to be emitted")
	}
}

func TestManager_PrepareContext_CompactWithProvider(t *testing.T) {
	provider := &mockProvider{response: "Summary of conversation"}
	eventChan := make(chan ContextEvent, 10)

	manager := NewManager(ManagerConfig{
		MaxTokens:          100000,
		ReserveForResponse: 0,
		WarnThreshold:      0.80,
		CompactThreshold:   0.90,
		Provider:           provider,
		AutoCompact:        true,
		EventChan:          eventChan,
	})

	// Create enough messages to compact
	messages := make([]providers.Message, 10)
	for i := 0; i < 10; i++ {
		role := providers.RoleUser
		if i%2 == 1 {
			role = providers.RoleAssistant
		}
		messages[i] = providers.Message{
			Role: role,
			Content: []providers.ContentBlock{
				&providers.TextContent{Text: "Message content here"},
			},
		}
	}

	tokenInfo := []TokenInfo{
		{MessageIndex: 0, Tokens: 95000, IsPrunable: false}, // 95% - compact level
	}

	result, err := manager.PrepareContext(context.Background(), messages, tokenInfo)
	if err != nil {
		t.Fatalf("PrepareContext() error = %v", err)
	}

	// Should have fewer messages after compaction
	if len(result) >= len(messages) {
		t.Errorf("Expected fewer messages after compaction, got %d (was %d)", len(result), len(messages))
	}

	// Should have emitted compacted event
	select {
	case event := <-eventChan:
		if event.Type != EventTypeCompacted {
			t.Errorf("Expected compacted event, got %v", event.Type)
		}
	default:
		t.Error("Expected compacted event to be emitted")
	}
}

func TestManager_PrepareContext_CompactDisabled(t *testing.T) {
	eventChan := make(chan ContextEvent, 10)

	manager := NewManager(ManagerConfig{
		MaxTokens:          100000,
		ReserveForResponse: 0,
		WarnThreshold:      0.80,
		CompactThreshold:   0.90,
		AutoCompact:        false, // Disabled
		AutoPrune:          false, // Disabled
		EventChan:          eventChan,
	})

	messages := []providers.Message{
		{Role: providers.RoleUser, Content: []providers.ContentBlock{&providers.TextContent{Text: "Hello"}}},
	}

	tokenInfo := []TokenInfo{
		{MessageIndex: 0, Tokens: 95000, IsPrunable: false},
	}

	result, err := manager.PrepareContext(context.Background(), messages, tokenInfo)
	if err != nil {
		t.Fatalf("PrepareContext() error = %v", err)
	}

	// Messages should be unchanged
	if len(result) != len(messages) {
		t.Errorf("Expected %d messages, got %d", len(messages), len(result))
	}

	// Should emit warning about context limit
	select {
	case event := <-eventChan:
		if event.Type != EventTypeWarning {
			t.Errorf("Expected warning event, got %v", event.Type)
		}
	default:
		t.Error("Expected warning event")
	}
}

func TestManager_EstimateTokens(t *testing.T) {
	manager := NewManager(ManagerConfig{
		MaxTokens:          100000,
		ReserveForResponse: 8192,
	})

	messages := []providers.Message{
		{Role: providers.RoleUser, Content: []providers.ContentBlock{&providers.TextContent{Text: "Hello world"}}},
		{Role: providers.RoleAssistant, Content: []providers.ContentBlock{&providers.TextContent{Text: "Hi there!"}}},
	}

	tokens := manager.EstimateTokens(messages)
	if tokens <= 0 {
		t.Error("Tokens should be > 0")
	}
}

func TestManager_EstimateMessage(t *testing.T) {
	manager := NewManager(ManagerConfig{
		MaxTokens:          100000,
		ReserveForResponse: 8192,
	})

	msg := &providers.Message{
		Role:    providers.RoleUser,
		Content: []providers.ContentBlock{&providers.TextContent{Text: "Hello world"}},
	}

	tokens := manager.EstimateMessage(msg)
	if tokens <= 0 {
		t.Error("Tokens should be > 0")
	}
}

func TestManager_RecordUsage(t *testing.T) {
	manager := NewManager(ManagerConfig{
		MaxTokens:          100000,
		ReserveForResponse: 8192,
	})

	manager.SetPricing(3.0, 15.0)
	manager.RecordUsage(1000, 500)

	snapshot := manager.GetBudgetSnapshot(50000)
	if snapshot.SessionInputTokens != 1000 {
		t.Errorf("SessionInputTokens = %d, want 1000", snapshot.SessionInputTokens)
	}
	if snapshot.SessionOutputTokens != 500 {
		t.Errorf("SessionOutputTokens = %d, want 500", snapshot.SessionOutputTokens)
	}
	if snapshot.SessionCost <= 0 {
		t.Error("SessionCost should be > 0")
	}
}

func TestManager_GetEffectiveLimit(t *testing.T) {
	manager := NewManager(ManagerConfig{
		MaxTokens:          100000,
		ReserveForResponse: 8192,
	})

	limit := manager.GetEffectiveLimit()
	expected := 100000 - 8192
	if limit != expected {
		t.Errorf("GetEffectiveLimit() = %d, want %d", limit, expected)
	}
}

func TestManager_UpdateModelContextSize(t *testing.T) {
	manager := NewManager(ManagerConfig{
		MaxTokens:          100000,
		ReserveForResponse: 8192,
		ModelContextSize:   128000,
	})

	initialLimit := manager.GetEffectiveLimit()
	if initialLimit != 100000-8192 {
		t.Errorf("Initial limit = %d, want %d", initialLimit, 100000-8192)
	}

	// Update to smaller context
	manager.UpdateModelContextSize(50000)

	newLimit := manager.GetEffectiveLimit()
	if newLimit != 50000-8192 {
		t.Errorf("New limit = %d, want %d", newLimit, 50000-8192)
	}
}

func TestManager_GetEstimator(t *testing.T) {
	manager := NewManager(ManagerConfig{
		MaxTokens:          100000,
		ReserveForResponse: 8192,
	})

	estimator := manager.GetEstimator()
	if estimator == nil {
		t.Error("GetEstimator() should not return nil")
	}
}

func TestManager_Prune(t *testing.T) {
	manager := NewManager(ManagerConfig{
		MaxTokens:          100000,
		ReserveForResponse: 8192,
		ProtectedTools:     []string{"file_read"},
	})

	messages := []providers.Message{
		{
			Role: providers.RoleTool,
			Content: []providers.ContentBlock{
				&providers.ToolResultContent{
					ToolUseID: "tool_1",
					Content:   "Long tool result content",
				},
			},
		},
		{
			Role: providers.RoleUser,
			Content: []providers.ContentBlock{
				&providers.TextContent{Text: "Recent message"},
			},
		},
	}

	tokenInfo := []TokenInfo{
		{MessageIndex: 0, Tokens: 500, IsPrunable: true, ToolName: "bash"},
		{MessageIndex: 1, Tokens: 10, IsPrunable: false},
	}

	result, pruneResult := manager.Prune(messages, tokenInfo)

	if len(result) != len(messages) {
		t.Errorf("Expected %d messages, got %d", len(messages), len(result))
	}

	if pruneResult.OriginalMessages != 2 {
		t.Errorf("OriginalMessages = %d, want 2", pruneResult.OriginalMessages)
	}
}

func TestManager_Compact(t *testing.T) {
	t.Run("without provider", func(t *testing.T) {
		manager := NewManager(ManagerConfig{
			MaxTokens:          100000,
			ReserveForResponse: 8192,
		})

		messages := []providers.Message{
			{Role: providers.RoleUser, Content: []providers.ContentBlock{&providers.TextContent{Text: "Hello"}}},
		}

		_, _, err := manager.Compact(context.Background(), messages)
		if err == nil {
			t.Error("Expected error without provider")
		}
	})

	t.Run("with provider", func(t *testing.T) {
		provider := &mockProvider{response: "Summary"}
		manager := NewManager(ManagerConfig{
			MaxTokens:          100000,
			ReserveForResponse: 8192,
			Provider:           provider,
		})

		messages := make([]providers.Message, 10)
		for i := 0; i < 10; i++ {
			role := providers.RoleUser
			if i%2 == 1 {
				role = providers.RoleAssistant
			}
			messages[i] = providers.Message{
				Role:    role,
				Content: []providers.ContentBlock{&providers.TextContent{Text: "Message"}},
			}
		}

		result, compactResult, err := manager.Compact(context.Background(), messages)
		if err != nil {
			t.Fatalf("Compact() error = %v", err)
		}

		if len(result) >= len(messages) {
			t.Error("Expected fewer messages after compaction")
		}

		if compactResult.Summary == "" {
			t.Error("Summary should not be empty")
		}
	})
}

func TestManager_GetLoadedProjectFiles(t *testing.T) {
	manager := NewManager(ManagerConfig{
		MaxTokens:          100000,
		ReserveForResponse: 8192,
	})

	t.Run("no context loaded", func(t *testing.T) {
		files := manager.GetLoadedProjectFiles()
		if files != nil {
			t.Error("Should return nil when no context loaded")
		}
	})

	t.Run("with context loaded", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "POTUS.md"), []byte("# Test"), 0644); err != nil {
			t.Fatal(err)
		}

		if err := manager.LoadProjectContext(tmpDir); err != nil {
			t.Fatal(err)
		}

		files := manager.GetLoadedProjectFiles()
		if len(files) != 1 {
			t.Errorf("Expected 1 file, got %d", len(files))
		}
	})
}

func TestManager_EmitEvent(t *testing.T) {
	t.Run("with event channel", func(t *testing.T) {
		eventChan := make(chan ContextEvent, 10)
		manager := NewManager(ManagerConfig{
			MaxTokens: 100000,
			EventChan: eventChan,
		})

		// Trigger an event by checking context at warning level
		manager.PrepareContext(context.Background(), []providers.Message{
			{Role: providers.RoleUser, Content: []providers.ContentBlock{&providers.TextContent{Text: "test"}}},
		}, []TokenInfo{{Tokens: 85000}})

		select {
		case event := <-eventChan:
			if event.Type != EventTypeWarning {
				t.Errorf("Expected warning event, got %v", event.Type)
			}
		default:
			t.Error("Expected event to be emitted")
		}
	})

	t.Run("without event channel", func(t *testing.T) {
		manager := NewManager(ManagerConfig{
			MaxTokens: 100000,
			// No event channel
		})

		// Should not panic
		manager.PrepareContext(context.Background(), []providers.Message{
			{Role: providers.RoleUser, Content: []providers.ContentBlock{&providers.TextContent{Text: "test"}}},
		}, []TokenInfo{{Tokens: 85000}})
	})
}

func TestManager_CalculateTokens(t *testing.T) {
	manager := NewManager(ManagerConfig{
		MaxTokens: 100000,
	})

	tokenInfo := []TokenInfo{
		{Tokens: 100},
		{Tokens: 200},
		{Tokens: 300},
	}

	total := manager.calculateTokens(tokenInfo)
	if total != 600 {
		t.Errorf("calculateTokens() = %d, want 600", total)
	}
}

func TestManager_GetBudgetSnapshot(t *testing.T) {
	manager := NewManager(ManagerConfig{
		MaxTokens:          100000,
		ReserveForResponse: 0,
		WarnThreshold:      0.80,
		CompactThreshold:   0.90,
	})

	manager.SetPricing(3.0, 15.0)
	manager.RecordUsage(1000, 500)

	snapshot := manager.GetBudgetSnapshot(50000)

	if snapshot.CurrentContextTokens != 50000 {
		t.Errorf("CurrentContextTokens = %d, want 50000", snapshot.CurrentContextTokens)
	}

	if snapshot.MaxContextTokens != 100000 {
		t.Errorf("MaxContextTokens = %d, want 100000", snapshot.MaxContextTokens)
	}

	expectedPercent := 50.0
	if snapshot.UsagePercent < expectedPercent-0.1 || snapshot.UsagePercent > expectedPercent+0.1 {
		t.Errorf("UsagePercent = %f, want %f", snapshot.UsagePercent, expectedPercent)
	}

	if snapshot.SessionInputTokens != 1000 {
		t.Errorf("SessionInputTokens = %d, want 1000", snapshot.SessionInputTokens)
	}

	if snapshot.SessionOutputTokens != 500 {
		t.Errorf("SessionOutputTokens = %d, want 500", snapshot.SessionOutputTokens)
	}

	if snapshot.AtWarningLevel {
		t.Error("Should not be at warning level at 50%")
	}

	if snapshot.AtCompactLevel {
		t.Error("Should not be at compact level at 50%")
	}
}
