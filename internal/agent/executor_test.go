package agent

import (
	"context"
	"testing"

	"github.com/taaha3244/potus/internal/providers"
	"github.com/taaha3244/potus/internal/tools"
)

type testTool struct {
	name      string
	shouldErr bool
}

func (t *testTool) Name() string {
	return t.name
}

func (t *testTool) Description() string {
	return "test tool"
}

func (t *testTool) Schema() map[string]interface{} {
	return map[string]interface{}{"type": "object"}
}

func (t *testTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.Result, error) {
	if t.shouldErr {
		return tools.NewErrorResult(context.Canceled), nil
	}
	return tools.NewResult("success"), nil
}

func TestExecutor_Execute(t *testing.T) {
	registry := tools.NewRegistry()
	successTool := &testTool{name: "success_tool", shouldErr: false}
	errorTool := &testTool{name: "error_tool", shouldErr: true}

	registry.Register(successTool)
	registry.Register(errorTool)

	executor := NewExecutor(registry)

	tests := []struct {
		name        string
		toolUse     *providers.ToolUseContent
		wantSuccess bool
		wantErr     bool
	}{
		{
			name: "successful execution",
			toolUse: &providers.ToolUseContent{
				ID:    "1",
				Name:  "success_tool",
				Input: map[string]interface{}{},
			},
			wantSuccess: true,
			wantErr:     false,
		},
		{
			name: "tool returns error",
			toolUse: &providers.ToolUseContent{
				ID:    "2",
				Name:  "error_tool",
				Input: map[string]interface{}{},
			},
			wantSuccess: false,
			wantErr:     false,
		},
		{
			name: "tool not found",
			toolUse: &providers.ToolUseContent{
				ID:    "3",
				Name:  "nonexistent_tool",
				Input: map[string]interface{}{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.Execute(context.Background(), tt.toolUse)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			if result.Success != tt.wantSuccess {
				t.Errorf("expected Success = %v, got %v", tt.wantSuccess, result.Success)
			}
		})
	}
}
