package permissions

import (
	"errors"
	"testing"

	"github.com/taaha3244/potus/internal/config"
)

func TestNewManager(t *testing.T) {
	cfg := &config.PermissionConfig{
		FileRead:  config.PermissionAllow,
		FileWrite: config.PermissionAsk,
		Bash:      config.PermissionDeny,
	}

	mgr := NewManager(cfg, nil)
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	if mgr.config != cfg {
		t.Error("Config not set correctly")
	}

	if mgr.cache == nil {
		t.Error("Cache not initialized")
	}
}

func TestManager_Check_Allow(t *testing.T) {
	cfg := &config.PermissionConfig{
		FileRead: config.PermissionAllow,
	}

	mgr := NewManager(cfg, nil)
	result, err := mgr.Check("file_read", "read", "test.txt")

	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if !result.Allowed {
		t.Error("Expected to be allowed")
	}

	if result.Reason != "allowed by configuration" {
		t.Errorf("Unexpected reason: %s", result.Reason)
	}
}

func TestManager_Check_Deny(t *testing.T) {
	cfg := &config.PermissionConfig{
		Bash: config.PermissionDeny,
	}

	mgr := NewManager(cfg, nil)
	result, err := mgr.Check("bash", "execute", "rm -rf /")

	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if result.Allowed {
		t.Error("Expected to be denied")
	}

	if result.Reason != "denied by configuration" {
		t.Errorf("Unexpected reason: %s", result.Reason)
	}
}

func TestManager_Check_Ask_WithPrompt(t *testing.T) {
	cfg := &config.PermissionConfig{
		FileWrite: config.PermissionAsk,
	}

	promptCalled := false
	promptFn := func(tool, action, details string) (Decision, error) {
		promptCalled = true
		if tool != "file_write" {
			t.Errorf("Wrong tool: %s", tool)
		}
		return DecisionAllow, nil
	}

	mgr := NewManager(cfg, promptFn)
	result, err := mgr.Check("file_write", "write", "test.txt")

	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if !promptCalled {
		t.Error("Prompt function should have been called")
	}

	if !result.Allowed {
		t.Error("Expected to be allowed")
	}
}

func TestManager_Check_Ask_Denied(t *testing.T) {
	cfg := &config.PermissionConfig{
		FileDelete: config.PermissionAsk,
	}

	promptFn := func(tool, action, details string) (Decision, error) {
		return DecisionDeny, nil
	}

	mgr := NewManager(cfg, promptFn)
	result, err := mgr.Check("file_delete", "delete", "important.txt")

	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if result.Allowed {
		t.Error("Expected to be denied")
	}
}

func TestManager_Check_Ask_NoPromptFunction(t *testing.T) {
	cfg := &config.PermissionConfig{
		FileWrite: config.PermissionAsk,
	}

	mgr := NewManager(cfg, nil)
	result, err := mgr.Check("file_write", "write", "test.txt")

	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if result.Allowed {
		t.Error("Expected to be denied when no prompt function")
	}

	if result.Reason != "no prompt handler available" {
		t.Errorf("Unexpected reason: %s", result.Reason)
	}
}

func TestManager_Check_Ask_PromptError(t *testing.T) {
	cfg := &config.PermissionConfig{
		FileWrite: config.PermissionAsk,
	}

	promptFn := func(tool, action, details string) (Decision, error) {
		return "", errors.New("prompt failed")
	}

	mgr := NewManager(cfg, promptFn)
	_, err := mgr.Check("file_write", "write", "test.txt")

	if err == nil {
		t.Error("Expected error when prompt fails")
	}
}

func TestManager_Check_Caching(t *testing.T) {
	cfg := &config.PermissionConfig{
		FileWrite: config.PermissionAsk,
	}

	promptCount := 0
	promptFn := func(tool, action, details string) (Decision, error) {
		promptCount++
		return DecisionAllow, nil
	}

	mgr := NewManager(cfg, promptFn)

	// First check - should prompt
	_, err := mgr.Check("file_write", "write", "test.txt")
	if err != nil {
		t.Fatalf("First check error = %v", err)
	}

	if promptCount != 1 {
		t.Errorf("Expected 1 prompt, got %d", promptCount)
	}

	// Second check - should use cache
	result, err := mgr.Check("file_write", "write", "test.txt")
	if err != nil {
		t.Fatalf("Second check error = %v", err)
	}

	if promptCount != 1 {
		t.Error("Second check should not prompt again")
	}

	if result.Reason != "previously allowed" {
		t.Errorf("Expected cached reason, got: %s", result.Reason)
	}
}

func TestManager_Check_AllowOnce_NotCached(t *testing.T) {
	cfg := &config.PermissionConfig{
		FileWrite: config.PermissionAsk,
	}

	promptCount := 0
	promptFn := func(tool, action, details string) (Decision, error) {
		promptCount++
		return DecisionAllowOnce, nil
	}

	mgr := NewManager(cfg, promptFn)

	// First check
	result, _ := mgr.Check("file_write", "write", "test.txt")
	if !result.Allowed {
		t.Error("First check should be allowed")
	}

	// Second check - should prompt again because AllowOnce is not cached
	_, _ = mgr.Check("file_write", "write", "test.txt")

	if promptCount != 2 {
		t.Errorf("Expected 2 prompts for AllowOnce, got %d", promptCount)
	}
}

func TestManager_ClearCache(t *testing.T) {
	cfg := &config.PermissionConfig{
		FileWrite: config.PermissionAsk,
	}

	promptCount := 0
	promptFn := func(tool, action, details string) (Decision, error) {
		promptCount++
		return DecisionAllow, nil
	}

	mgr := NewManager(cfg, promptFn)

	// First check - populates cache
	mgr.Check("file_write", "write", "test.txt")

	// Clear cache
	mgr.ClearCache()

	// Check again - should prompt again
	mgr.Check("file_write", "write", "test.txt")

	if promptCount != 2 {
		t.Errorf("Expected 2 prompts after cache clear, got %d", promptCount)
	}
}

func TestManager_AllowAll(t *testing.T) {
	cfg := &config.PermissionConfig{
		FileWrite:  config.PermissionAsk,
		FileDelete: config.PermissionAsk,
		Bash:       config.PermissionAsk,
	}

	mgr := NewManager(cfg, nil)
	mgr.AllowAll()

	tests := []string{"file_write", "file_delete", "bash"}
	for _, tool := range tests {
		result, err := mgr.Check(tool, "test", "")
		if err != nil {
			t.Errorf("Check(%s) error = %v", tool, err)
		}
		if !result.Allowed {
			t.Errorf("Check(%s) should be allowed in AllowAll mode", tool)
		}
	}
}

func TestManager_GetBashLists(t *testing.T) {
	cfg := &config.PermissionConfig{
		BashAllowlist: []string{"ls", "cat"},
		BashBlocklist: []string{"rm -rf /"},
	}

	mgr := NewManager(cfg, nil)

	allowlist := mgr.GetBashAllowlist()
	if len(allowlist) != 2 {
		t.Errorf("Expected 2 allowlist items, got %d", len(allowlist))
	}

	blocklist := mgr.GetBashBlocklist()
	if len(blocklist) != 1 {
		t.Errorf("Expected 1 blocklist item, got %d", len(blocklist))
	}
}

func TestManager_GetBashLists_NilConfig(t *testing.T) {
	mgr := NewManager(nil, nil)

	if mgr.GetBashAllowlist() != nil {
		t.Error("Expected nil allowlist with nil config")
	}

	if mgr.GetBashBlocklist() != nil {
		t.Error("Expected nil blocklist with nil config")
	}
}

func TestManager_Check_SearchToolsAlwaysAllowed(t *testing.T) {
	cfg := &config.PermissionConfig{} // All defaults

	mgr := NewManager(cfg, nil)

	tools := []string{"search_files", "search_content"}
	for _, tool := range tools {
		result, err := mgr.Check(tool, "search", "*.go")
		if err != nil {
			t.Errorf("Check(%s) error = %v", tool, err)
		}
		if !result.Allowed {
			t.Errorf("Check(%s) should always be allowed", tool)
		}
	}
}

func TestManager_Check_GitTools(t *testing.T) {
	cfg := &config.PermissionConfig{
		Git: config.PermissionAllow,
	}

	mgr := NewManager(cfg, nil)

	tools := []string{"git_status", "git_diff", "git_commit", "git_branch", "git_log"}
	for _, tool := range tools {
		result, err := mgr.Check(tool, "git", "")
		if err != nil {
			t.Errorf("Check(%s) error = %v", tool, err)
		}
		if !result.Allowed {
			t.Errorf("Check(%s) should be allowed when git=allow", tool)
		}
	}
}

func TestManager_Check_WebTools(t *testing.T) {
	cfg := &config.PermissionConfig{
		WebFetch:  config.PermissionAllow,
		WebSearch: config.PermissionDeny,
	}

	mgr := NewManager(cfg, nil)

	result, _ := mgr.Check("web_fetch", "fetch", "https://example.com")
	if !result.Allowed {
		t.Error("web_fetch should be allowed")
	}

	result, _ = mgr.Check("web_search", "search", "golang tutorial")
	if result.Allowed {
		t.Error("web_search should be denied")
	}
}

func TestManager_SetPromptFunc(t *testing.T) {
	cfg := &config.PermissionConfig{
		FileWrite: config.PermissionAsk,
	}

	mgr := NewManager(cfg, nil)

	// Initially should deny (no prompt function)
	result, _ := mgr.Check("file_write", "write", "test.txt")
	if result.Allowed {
		t.Error("Should be denied without prompt function")
	}

	// Set prompt function
	mgr.SetPromptFunc(func(tool, action, details string) (Decision, error) {
		return DecisionAllow, nil
	})

	// Clear cache and check again
	mgr.ClearCache()
	result, _ = mgr.Check("file_write", "write", "test.txt")
	if !result.Allowed {
		t.Error("Should be allowed after setting prompt function")
	}
}
