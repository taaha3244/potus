package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/taaha3244/potus/internal/permissions"
	"github.com/taaha3244/potus/internal/providers"
	"github.com/taaha3244/potus/internal/tools"
)

type Decision string

const (
	DecisionApprove     Decision = "approve"
	DecisionDeny        Decision = "deny"
	DecisionAlwaysAllow Decision = "always_allow"
)

type ConfirmFunc func(toolName, action, preview string) (Decision, error)

type Executor struct {
	registry  *tools.Registry
	confirmFn ConfirmFunc
	settings  *permissions.Settings
	workDir   string
}

type ExecutorConfig struct {
	Registry  *tools.Registry
	ConfirmFn ConfirmFunc
	Settings  *permissions.Settings
	WorkDir   string
}

func NewExecutor(registry *tools.Registry) *Executor {
	return &Executor{
		registry: registry,
	}
}

func NewExecutorWithConfig(cfg *ExecutorConfig) *Executor {
	return &Executor{
		registry:  cfg.Registry,
		confirmFn: cfg.ConfirmFn,
		settings:  cfg.Settings,
		workDir:   cfg.WorkDir,
	}
}

func (e *Executor) Execute(ctx context.Context, toolUse *providers.ToolUseContent) (*tools.Result, error) {
	tool, err := e.registry.Get(toolUse.Name)
	if err != nil {
		return nil, fmt.Errorf("tool not found: %s", toolUse.Name)
	}

	if e.needsConfirmation(toolUse.Name) && e.confirmFn != nil {
		preview := e.generatePreview(toolUse)
		action := e.describeAction(toolUse)

		decision, err := e.confirmFn(toolUse.Name, action, preview)
		if err != nil {
			return tools.NewErrorResult(fmt.Errorf("confirmation failed: %w", err)), nil
		}

		switch decision {
		case DecisionDeny:
			return tools.NewErrorResult(fmt.Errorf("operation denied by user")), nil
		case DecisionAlwaysAllow:
			if e.settings != nil {
				e.settings.SetAllow(toolUse.Name)
				e.settings.Save()
			}
		}
	}

	result, err := tool.Execute(ctx, toolUse.Input)
	if err != nil {
		return nil, fmt.Errorf("tool execution failed: %w", err)
	}

	return result, nil
}

func (e *Executor) needsConfirmation(name string) bool {
	if e.settings != nil && e.settings.IsAllowed(name) {
		return false
	}

	switch name {
	case "file_write", "file_edit", "file_delete", "bash":
		return true
	default:
		return false
	}
}

func (e *Executor) generatePreview(toolUse *providers.ToolUseContent) string {
	switch toolUse.Name {
	case "file_write":
		return e.previewFileWrite(toolUse.Input)
	case "file_edit":
		return e.previewFileEdit(toolUse.Input)
	case "file_delete":
		return e.previewFileDelete(toolUse.Input)
	case "bash":
		return e.previewBash(toolUse.Input)
	default:
		return fmt.Sprintf("Tool: %s\nParams: %v", toolUse.Name, toolUse.Input)
	}
}

func (e *Executor) describeAction(toolUse *providers.ToolUseContent) string {
	switch toolUse.Name {
	case "file_write":
		if path, ok := toolUse.Input["path"].(string); ok {
			return fmt.Sprintf("Create file: %s", path)
		}
		return "Create file"
	case "file_edit":
		if path, ok := toolUse.Input["path"].(string); ok {
			return fmt.Sprintf("Edit file: %s", path)
		}
		return "Edit file"
	case "file_delete":
		if path, ok := toolUse.Input["path"].(string); ok {
			return fmt.Sprintf("Delete file: %s", path)
		}
		return "Delete file"
	case "bash":
		if cmd, ok := toolUse.Input["command"].(string); ok {
			if len(cmd) > 50 {
				return fmt.Sprintf("Run: %s...", cmd[:50])
			}
			return fmt.Sprintf("Run: %s", cmd)
		}
		return "Execute bash command"
	default:
		return toolUse.Name
	}
}

func (e *Executor) previewFileWrite(params map[string]interface{}) string {
	path, _ := params["path"].(string)
	content, _ := params["content"].(string)

	return tools.FormatNewFile(path, content)
}

func (e *Executor) previewFileEdit(params map[string]interface{}) string {
	path, _ := params["path"].(string)
	search, _ := params["search"].(string)
	replace, _ := params["replace"].(string)

	fullPath := e.resolvePath(path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Sprintf("Cannot preview: %v", err)
	}

	oldContent := string(content)
	if !strings.Contains(oldContent, search) {
		return fmt.Sprintf("Search text not found in %s", path)
	}

	newContent := strings.Replace(oldContent, search, replace, 1)
	return tools.GenerateUnifiedDiff(oldContent, newContent, path)
}

func (e *Executor) previewFileDelete(params map[string]interface{}) string {
	path, _ := params["path"].(string)

	fullPath := e.resolvePath(path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Sprintf("--- %s (to be deleted)", path)
	}

	return tools.FormatDeleteFile(path, string(content))
}

func (e *Executor) previewBash(params map[string]interface{}) string {
	command, _ := params["command"].(string)
	return tools.FormatBashCommand(command)
}

func (e *Executor) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if e.workDir != "" {
		return filepath.Join(e.workDir, path)
	}
	return path
}
