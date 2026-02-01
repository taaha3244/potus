package search

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/taaha3244/potus/internal/tools"
)

type FileSearchTool struct{}

func NewFileSearchTool() *FileSearchTool {
	return &FileSearchTool{}
}

func (t *FileSearchTool) Name() string {
	return "search_files"
}

func (t *FileSearchTool) Description() string {
	return "Search for files matching a pattern"
}

func (t *FileSearchTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"root_path": map[string]interface{}{
				"type":        "string",
				"description": "Root directory to search from (defaults to current directory)",
			},
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "File pattern to match (e.g., '*.go', '**/*.ts', 'utils/*.js')",
			},
			"include_hidden": map[string]interface{}{
				"type":        "boolean",
				"description": "Include hidden files and directories (default: false)",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *FileSearchTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.Result, error) {
	rootPath := "."
	if path, ok := params["root_path"].(string); ok && path != "" {
		rootPath = path
	}

	pattern, ok := params["pattern"].(string)
	if !ok || pattern == "" {
		return nil, fmt.Errorf("pattern is required")
	}

	includeHidden := false
	if ih, ok := params["include_hidden"].(bool); ok {
		includeHidden = ih
	}

	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("root path does not exist: %s", rootPath)
	}

	matches, err := t.findFiles(rootPath, pattern, includeHidden)
	if err != nil {
		return nil, fmt.Errorf("failed to search files: %w", err)
	}

	if len(matches) == 0 {
		return &tools.Result{
			Success: true,
			Output:  "No files found matching pattern",
		}, nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Found %d file(s):\n\n", len(matches)))
	for _, match := range matches {
		relPath, err := filepath.Rel(rootPath, match)
		if err != nil {
			relPath = match
		}
		output.WriteString(fmt.Sprintf("  %s\n", relPath))
	}

	return &tools.Result{
		Success: true,
		Output:  output.String(),
	}, nil
}

func (t *FileSearchTool) findFiles(rootPath, pattern string, includeHidden bool) ([]string, error) {
	var matches []string

	isRecursive := strings.Contains(pattern, "**")
	if isRecursive {
		pattern = strings.ReplaceAll(pattern, "**/", "")
	}

	hasDir := strings.Contains(pattern, string(filepath.Separator))
	var searchPattern string
	var searchDir string

	if hasDir && !isRecursive {
		searchDir = filepath.Join(rootPath, filepath.Dir(pattern))
		searchPattern = filepath.Base(pattern)
	} else {
		searchDir = rootPath
		searchPattern = pattern
	}

	err := filepath.Walk(searchDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !includeHidden && strings.HasPrefix(filepath.Base(path), ".") && path != searchDir {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			if !isRecursive && path != searchDir {
				return filepath.SkipDir
			}
			return nil
		}

		if matched, _ := t.matchPattern(filepath.Base(path), searchPattern); matched {
			matches = append(matches, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return matches, nil
}

func (t *FileSearchTool) matchPattern(filename, pattern string) (bool, error) {
	if strings.Contains(pattern, "{") && strings.Contains(pattern, "}") {
		start := strings.Index(pattern, "{")
		end := strings.Index(pattern, "}")
		if start != -1 && end != -1 && start < end {
			prefix := pattern[:start]
			suffix := pattern[end+1:]
			extensions := strings.Split(pattern[start+1:end], ",")

			for _, ext := range extensions {
				testPattern := prefix + ext + suffix
				if matched, _ := filepath.Match(testPattern, filename); matched {
					return true, nil
				}
			}
			return false, nil
		}
	}

	return filepath.Match(pattern, filename)
}
