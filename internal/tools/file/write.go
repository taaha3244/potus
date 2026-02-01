package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/taaha3244/potus/internal/tools"
)

type WriteTool struct {
	workDir string
}

func NewWriteTool(workDir string) *WriteTool {
	return &WriteTool{workDir: workDir}
}

func (t *WriteTool) Name() string {
	return "file_write"
}

func (t *WriteTool) Description() string {
	return "Create a new file with the specified content. Fails if file already exists."
}

func (t *WriteTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to create",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Content to write to the file",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (t *WriteTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.Result, error) {
	path, ok := params["path"].(string)
	if !ok {
		return tools.NewErrorResult(fmt.Errorf("path parameter is required")), nil
	}

	content, ok := params["content"].(string)
	if !ok {
		return tools.NewErrorResult(fmt.Errorf("content parameter is required")), nil
	}

	fullPath := resolvePath(t.workDir, path)

	if _, err := os.Stat(fullPath); err == nil {
		return tools.NewErrorResult(fmt.Errorf("file already exists: %s", path)), nil
	}

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return tools.NewErrorResult(fmt.Errorf("failed to create directory: %w", err)), nil
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return tools.NewErrorResult(fmt.Errorf("failed to write file: %w", err)), nil
	}

	return tools.NewResult(fmt.Sprintf("File created: %s (%d bytes)", path, len(content))), nil
}
