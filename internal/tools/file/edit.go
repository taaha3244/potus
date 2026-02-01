package file

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/taaha3244/potus/internal/tools"
)

type EditTool struct {
	workDir string
}

func NewEditTool(workDir string) *EditTool {
	return &EditTool{workDir: workDir}
}

func (t *EditTool) Name() string {
	return "file_edit"
}

func (t *EditTool) Description() string {
	return "Edit an existing file by searching for text and replacing it. The search text must match exactly."
}

func (t *EditTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to edit",
			},
			"search": map[string]interface{}{
				"type":        "string",
				"description": "Text to search for (must match exactly)",
			},
			"replace": map[string]interface{}{
				"type":        "string",
				"description": "Text to replace with",
			},
		},
		"required": []string{"path", "search", "replace"},
	}
}

func (t *EditTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.Result, error) {
	path, ok := params["path"].(string)
	if !ok {
		return tools.NewErrorResult(fmt.Errorf("path parameter is required")), nil
	}

	search, ok := params["search"].(string)
	if !ok {
		return tools.NewErrorResult(fmt.Errorf("search parameter is required")), nil
	}

	replace, ok := params["replace"].(string)
	if !ok {
		return tools.NewErrorResult(fmt.Errorf("replace parameter is required")), nil
	}

	fullPath := resolvePath(t.workDir, path)

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return tools.NewErrorResult(fmt.Errorf("failed to read file: %w", err)), nil
	}

	originalContent := string(content)

	if !strings.Contains(originalContent, search) {
		return tools.NewErrorResult(fmt.Errorf("search text not found in file")), nil
	}

	occurrences := strings.Count(originalContent, search)
	if occurrences > 1 {
		return tools.NewErrorResult(fmt.Errorf("search text appears %d times in file; must be unique", occurrences)), nil
	}

	newContent := strings.Replace(originalContent, search, replace, 1)

	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return tools.NewErrorResult(fmt.Errorf("failed to write file: %w", err)), nil
	}

	return tools.NewResult(fmt.Sprintf("File edited: %s (replaced %d characters with %d)", path, len(search), len(replace))), nil
}
