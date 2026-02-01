package file

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/taaha3244/potus/internal/tools"
)

type ReadTool struct {
	workDir string
}

func NewReadTool(workDir string) *ReadTool {
	return &ReadTool{workDir: workDir}
}

func (t *ReadTool) Name() string {
	return "file_read"
}

func (t *ReadTool) Description() string {
	return "Read the contents of a file. Optionally specify line range."
}

func (t *ReadTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to read (relative to working directory)",
			},
			"start_line": map[string]interface{}{
				"type":        "integer",
				"description": "Optional: start line number (1-indexed)",
			},
			"end_line": map[string]interface{}{
				"type":        "integer",
				"description": "Optional: end line number (inclusive)",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ReadTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.Result, error) {
	path, ok := params["path"].(string)
	if !ok {
		return tools.NewErrorResult(fmt.Errorf("path parameter is required")), nil
	}

	fullPath := resolvePath(t.workDir, path)

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return tools.NewErrorResult(fmt.Errorf("failed to read file: %w", err)), nil
	}

	lines := strings.Split(string(content), "\n")

	startLine := 1
	endLine := len(lines)

	if start, ok := params["start_line"].(float64); ok {
		startLine = int(start)
	}
	if end, ok := params["end_line"].(float64); ok {
		endLine = int(end)
	}

	if startLine < 1 {
		startLine = 1
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	if startLine > endLine {
		return tools.NewErrorResult(fmt.Errorf("start_line must be <= end_line")), nil
	}

	selectedLines := lines[startLine-1 : endLine]
	var numbered strings.Builder
	for i, line := range selectedLines {
		numbered.WriteString(fmt.Sprintf("%4d  %s\n", startLine+i, line))
	}

	return tools.NewResult(numbered.String()), nil
}
