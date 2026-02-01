package permissions

import (
	"fmt"
	"sync"

	"github.com/taaha3244/potus/internal/config"
)

type Permission string

const (
	PermissionAllow Permission = "allow"
	PermissionAsk   Permission = "ask"
	PermissionDeny  Permission = "deny"
)

type Decision string

const (
	DecisionAllow     Decision = "allow"
	DecisionDeny      Decision = "deny"
	DecisionAllowOnce Decision = "allow_once"
	DecisionDenyOnce  Decision = "deny_once"
)

type Manager struct {
	mu       sync.RWMutex
	config   *config.PermissionConfig
	cache    map[string]Decision
	promptFn PromptFunc
}

type PromptFunc func(tool, action, details string) (Decision, error)

func NewManager(cfg *config.PermissionConfig, promptFn PromptFunc) *Manager {
	return &Manager{
		config:   cfg,
		cache:    make(map[string]Decision),
		promptFn: promptFn,
	}
}

type CheckResult struct {
	Allowed bool
	Reason  string
}

func (m *Manager) Check(toolName string, action string, details string) (*CheckResult, error) {
	permission := m.getToolPermission(toolName)

	switch permission {
	case PermissionAllow:
		return &CheckResult{Allowed: true, Reason: "allowed by configuration"}, nil
	case PermissionDeny:
		return &CheckResult{Allowed: false, Reason: "denied by configuration"}, nil
	case PermissionAsk:
		return m.handleAskPermission(toolName, action, details)
	default:
		return m.handleAskPermission(toolName, action, details)
	}
}

func (m *Manager) handleAskPermission(toolName, action, details string) (*CheckResult, error) {
	cacheKey := fmt.Sprintf("%s:%s", toolName, action)

	m.mu.RLock()
	if decision, ok := m.cache[cacheKey]; ok {
		m.mu.RUnlock()
		switch decision {
		case DecisionAllow:
			return &CheckResult{Allowed: true, Reason: "previously allowed"}, nil
		case DecisionDeny:
			return &CheckResult{Allowed: false, Reason: "previously denied"}, nil
		}
	} else {
		m.mu.RUnlock()
	}

	if m.promptFn == nil {
		return &CheckResult{Allowed: false, Reason: "no prompt handler available"}, nil
	}

	decision, err := m.promptFn(toolName, action, details)
	if err != nil {
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	m.mu.Lock()
	switch decision {
	case DecisionAllow:
		m.cache[cacheKey] = DecisionAllow
	case DecisionDeny:
		m.cache[cacheKey] = DecisionDeny
	}
	m.mu.Unlock()

	switch decision {
	case DecisionAllow, DecisionAllowOnce:
		return &CheckResult{Allowed: true, Reason: "user approved"}, nil
	default:
		return &CheckResult{Allowed: false, Reason: "user denied"}, nil
	}
}

func (m *Manager) getToolPermission(toolName string) Permission {
	if m.config == nil {
		return PermissionAsk
	}

	switch toolName {
	case "file_read":
		return Permission(m.config.FileRead)
	case "file_write":
		return Permission(m.config.FileWrite)
	case "file_edit":
		return Permission(m.config.FileEdit)
	case "file_delete":
		return Permission(m.config.FileDelete)
	case "bash":
		return Permission(m.config.Bash)
	case "git_status", "git_diff", "git_commit", "git_branch", "git_log":
		return Permission(m.config.Git)
	case "web_fetch":
		return Permission(m.config.WebFetch)
	case "web_search":
		return Permission(m.config.WebSearch)
	case "search_files", "search_content":
		return PermissionAllow
	default:
		return PermissionAsk
	}
}

func (m *Manager) ClearCache() {
	m.mu.Lock()
	m.cache = make(map[string]Decision)
	m.mu.Unlock()
}

func (m *Manager) AllowAll() {
	m.mu.Lock()
	m.promptFn = func(tool, action, details string) (Decision, error) {
		return DecisionAllow, nil
	}
	m.mu.Unlock()
}

func (m *Manager) SetPromptFunc(fn PromptFunc) {
	m.mu.Lock()
	m.promptFn = fn
	m.mu.Unlock()
}

func (m *Manager) GetBashAllowlist() []string {
	if m.config == nil {
		return nil
	}
	return m.config.BashAllowlist
}

func (m *Manager) GetBashBlocklist() []string {
	if m.config == nil {
		return nil
	}
	return m.config.BashBlocklist
}
