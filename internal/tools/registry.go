package tools

import (
	"fmt"
	"sync"

	"github.com/taaha3244/potus/internal/providers"
)

type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
}

func (r *Registry) Get(name string) (Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return tool, nil
}

func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

func (r *Registry) ToProviderTools() []providers.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providerTools := make([]providers.Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		providerTools = append(providerTools, providers.Tool{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: tool.Schema(),
		})
	}
	return providerTools
}
