package file

import (
	"context"
	"fmt"
	"os"

	"github.com/taaha3244/potus/internal/tools"
)

type DeleteTool struct {
	workDir string
}

func NewDeleteTool(workDir string) *DeleteTool {
	return &DeleteTool{workDir: workDir}
}

func (t *DeleteTool) Name() string {
	return "file_delete"
}

func (t *DeleteTool) Description() string {
	return "Delete a file. Use with caution as this cannot be undone."
}

func (t *DeleteTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to delete",
			},
		},
		"required": []string{"path"},
	}
}

func (t *DeleteTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.Result, error) {
	path, ok := params["path"].(string)
	if !ok {
		return tools.NewErrorResult(fmt.Errorf("path parameter is required")), nil
	}

	fullPath := resolvePath(t.workDir, path)

	info, err := os.Stat(fullPath)
	if err != nil {
		return tools.NewErrorResult(fmt.Errorf("file not found: %w", err)), nil
	}

	if info.IsDir() {
		return tools.NewErrorResult(fmt.Errorf("cannot delete directory, file expected")), nil
	}

	if err := os.Remove(fullPath); err != nil {
		return tools.NewErrorResult(fmt.Errorf("failed to delete file: %w", err)), nil
	}

	return tools.NewResult(fmt.Sprintf("File deleted: %s", path)), nil
}
