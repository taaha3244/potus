package providers

import (
	"context"
	"testing"
)

type mockProvider struct {
	name string
}

func (m *mockProvider) Chat(ctx context.Context, req *ChatRequest) (<-chan ChatEvent, error) {
	ch := make(chan ChatEvent)
	close(ch)
	return ch, nil
}

func (m *mockProvider) ListModels(ctx context.Context) ([]Model, error) {
	return []Model{}, nil
}

func (m *mockProvider) SupportsTools() bool {
	return true
}

func (m *mockProvider) SupportsVision() bool {
	return false
}

func (m *mockProvider) Name() string {
	return m.name
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	mock := &mockProvider{name: "test"}

	reg.Register("test", mock)

	provider, err := reg.Get("test")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if provider.Name() != "test" {
		t.Errorf("expected provider name 'test', got %s", provider.Name())
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	reg := NewRegistry()

	_, err := reg.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent provider")
	}
}

func TestRegistry_List(t *testing.T) {
	reg := NewRegistry()
	reg.Register("provider1", &mockProvider{name: "provider1"})
	reg.Register("provider2", &mockProvider{name: "provider2"})

	names := reg.List()
	if len(names) != 2 {
		t.Errorf("expected 2 providers, got %d", len(names))
	}
}

func TestParseModelString(t *testing.T) {
	tests := []struct {
		input         string
		wantProvider  string
		wantModel     string
	}{
		{"anthropic/claude-sonnet-4-5", "anthropic", "claude-sonnet-4-5"},
		{"openai/gpt-4o", "openai", "gpt-4o"},
		{"gpt-4o", "", "gpt-4o"},
		{"ollama/qwen2.5-coder:32b", "ollama", "qwen2.5-coder:32b"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			provider, model := ParseModelString(tt.input)
			if provider != tt.wantProvider {
				t.Errorf("provider = %s, want %s", provider, tt.wantProvider)
			}
			if model != tt.wantModel {
				t.Errorf("model = %s, want %s", model, tt.wantModel)
			}
		})
	}
}
