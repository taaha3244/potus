package bash

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/taaha3244/potus/internal/tools"
)

const (
	// DefaultTimeout is the default command execution timeout
	DefaultTimeout = 30 * time.Second
)

type ExecutorTool struct {
	workDir   string
	timeout   time.Duration
	allowlist []string
	blocklist []string
}

func NewExecutorTool(workDir string, timeout time.Duration, allowlist, blocklist []string) *ExecutorTool {
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	return &ExecutorTool{
		workDir:   workDir,
		timeout:   timeout,
		allowlist: allowlist,
		blocklist: blocklist,
	}
}

func (t *ExecutorTool) Name() string {
	return "bash"
}

func (t *ExecutorTool) Description() string {
	return "Execute a bash command in the working directory. Returns stdout and stderr."
}

func (t *ExecutorTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The bash command to execute",
			},
		},
		"required": []string{"command"},
	}
}

func (t *ExecutorTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.Result, error) {
	command, ok := params["command"].(string)
	if !ok {
		return tools.NewErrorResult(fmt.Errorf("command parameter is required")), nil
	}

	if err := t.validateCommand(command); err != nil {
		return tools.NewErrorResult(err), nil
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, "bash", "-c", command)
	cmd.Dir = t.workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\nSTDERR:\n" + stderr.String()
	}

	if err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return tools.NewErrorResult(fmt.Errorf("command timed out after %v", t.timeout)), nil
		}
		return tools.NewErrorResult(fmt.Errorf("command failed: %w\n%s", err, output)), nil
	}

	return tools.NewResult(output), nil
}

func (t *ExecutorTool) validateCommand(command string) error {
	command = strings.TrimSpace(command)

	for _, blocked := range t.blocklist {
		if strings.Contains(command, blocked) {
			return fmt.Errorf("command contains blocked pattern: %s", blocked)
		}
	}

	if len(t.allowlist) == 0 {
		return nil
	}

	for _, allowed := range t.allowlist {
		if strings.HasPrefix(command, allowed) {
			return nil
		}
	}

	return fmt.Errorf("command not in allowlist")
}
