package tools

import (
	"context"
	"testing"
)

type mockTool struct {
	name string
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Description() string {
	return "mock tool"
}

func (m *mockTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
	}
}

func (m *mockTool) Execute(ctx context.Context, params map[string]interface{}) (*Result, error) {
	return NewResult("success"), nil
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	tool := &mockTool{name: "test_tool"}

	reg.Register(tool)

	retrieved, err := reg.Get("test_tool")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.Name() != "test_tool" {
		t.Errorf("expected tool name 'test_tool', got %s", retrieved.Name())
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	reg := NewRegistry()

	_, err := reg.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent tool")
	}
}

func TestRegistry_List(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockTool{name: "tool1"})
	reg.Register(&mockTool{name: "tool2"})
	reg.Register(&mockTool{name: "tool3"})

	tools := reg.List()
	if len(tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(tools))
	}
}

func TestRegistry_ToProviderTools(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockTool{name: "tool1"})
	reg.Register(&mockTool{name: "tool2"})

	providerTools := reg.ToProviderTools()
	if len(providerTools) != 2 {
		t.Errorf("expected 2 provider tools, got %d", len(providerTools))
	}

	for _, pt := range providerTools {
		if pt.Name == "" {
			t.Error("provider tool name should not be empty")
		}
		if pt.Description == "" {
			t.Error("provider tool description should not be empty")
		}
	}
}

func TestNewResult(t *testing.T) {
	result := NewResult("test output")

	if !result.Success {
		t.Error("expected Success = true")
	}

	if result.Output != "test output" {
		t.Errorf("expected output 'test output', got %s", result.Output)
	}

	if result.Error != nil {
		t.Error("expected no error")
	}
}

func TestNewErrorResult(t *testing.T) {
	testErr := context.Canceled
	result := NewErrorResult(testErr)

	if result.Success {
		t.Error("expected Success = false")
	}

	if result.Error != testErr {
		t.Errorf("expected error %v, got %v", testErr, result.Error)
	}

	if result.Output == "" {
		t.Error("expected non-empty output")
	}
}
