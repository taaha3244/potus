package providers

import (
	"fmt"
	"strings"
	"sync"
)

type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

func (r *Registry) Register(name string, provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[strings.ToLower(name)] = provider
}

func (r *Registry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, ok := r.providers[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	return provider, nil
}

func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

func ParseModelString(modelStr string) (provider, model string) {
	parts := strings.SplitN(modelStr, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", modelStr
}
