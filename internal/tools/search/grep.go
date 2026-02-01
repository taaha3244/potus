package search

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/taaha3244/potus/internal/tools"
)

const (
	// DefaultMaxResults limits search results to prevent overwhelming output
	DefaultMaxResults = 100
)

type GrepTool struct{}

func NewGrepTool() *GrepTool {
	return &GrepTool{}
}

func (t *GrepTool) Name() string {
	return "search_content"
}

func (t *GrepTool) Description() string {
	return "Search for text content within files"
}

func (t *GrepTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"root_path": map[string]interface{}{
				"type":        "string",
				"description": "Root directory to search from (defaults to current directory)",
			},
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "Text pattern to search for",
			},
			"file_types": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "File extensions to search (e.g., ['go', 'js'])",
			},
			"case_sensitive": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether search is case-sensitive (default: false)",
			},
			"context_lines": map[string]interface{}{
				"type":        "number",
				"description": "Number of context lines to show (default: 0)",
			},
			"max_results": map[string]interface{}{
				"type":        "number",
				"description": "Maximum number of matches to return (default: 100)",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *GrepTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.Result, error) {
	rootPath := "."
	if path, ok := params["root_path"].(string); ok && path != "" {
		rootPath = path
	}

	pattern, ok := params["pattern"].(string)
	if !ok || pattern == "" {
		return nil, fmt.Errorf("pattern is required")
	}

	caseSensitive := false
	if cs, ok := params["case_sensitive"].(bool); ok {
		caseSensitive = cs
	}

	contextLines := 0
	if cl, ok := params["context_lines"].(float64); ok {
		contextLines = int(cl)
	} else if cl, ok := params["context_lines"].(int); ok {
		contextLines = cl
	}

	maxResults := DefaultMaxResults
	if mr, ok := params["max_results"].(float64); ok {
		maxResults = int(mr)
	} else if mr, ok := params["max_results"].(int); ok {
		maxResults = mr
	}

	var fileTypes []string
	if ft, ok := params["file_types"].([]interface{}); ok {
		for _, item := range ft {
			if s, ok := item.(string); ok {
				fileTypes = append(fileTypes, s)
			}
		}
	}

	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("root path does not exist: %s", rootPath)
	}

	searchPattern := pattern
	if !caseSensitive {
		searchPattern = strings.ToLower(pattern)
	}

	matches := make([]Match, 0)
	totalMatches := 0

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			if strings.HasPrefix(filepath.Base(path), ".") && path != rootPath {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasPrefix(filepath.Base(path), ".") {
			return nil
		}

		if len(fileTypes) > 0 {
			ext := strings.TrimPrefix(filepath.Ext(path), ".")
			found := false
			for _, ft := range fileTypes {
				if ext == ft {
					found = true
					break
				}
			}
			if !found {
				return nil
			}
		}

		fileMatches, err := t.searchFile(path, searchPattern, caseSensitive, contextLines)
		if err != nil {
			return nil
		}

		for _, match := range fileMatches {
			if totalMatches >= maxResults {
				return filepath.SkipAll
			}
			matches = append(matches, match)
			totalMatches++
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	if len(matches) == 0 {
		return &tools.Result{
			Success: true,
			Output:  "No matches found",
		}, nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Found %d match(es):\n\n", len(matches)))

	currentFile := ""
	for _, match := range matches {
		relPath, err := filepath.Rel(rootPath, match.File)
		if err != nil {
			relPath = match.File
		}

		if relPath != currentFile {
			if currentFile != "" {
				output.WriteString("\n")
			}
			output.WriteString(fmt.Sprintf("%s:\n", relPath))
			currentFile = relPath
		}

		output.WriteString(fmt.Sprintf("  %d: %s\n", match.Line, match.Text))

		for _, ctx := range match.Context {
			output.WriteString(fmt.Sprintf("     %s\n", ctx))
		}
	}

	return &tools.Result{
		Success: true,
		Output:  output.String(),
	}, nil
}

type Match struct {
	File    string
	Line    int
	Text    string
	Context []string
}

func (t *GrepTool) searchFile(path, pattern string, caseSensitive bool, contextLines int) ([]Match, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	matches := make([]Match, 0)
	scanner := bufio.NewScanner(file)
	lineNum := 0
	var recentLines []string

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if contextLines > 0 {
			recentLines = append(recentLines, line)
			if len(recentLines) > contextLines*2+1 {
				recentLines = recentLines[1:]
			}
		}

		searchLine := line
		if !caseSensitive {
			searchLine = strings.ToLower(line)
		}

		if strings.Contains(searchLine, pattern) {
			match := Match{
				File: path,
				Line: lineNum,
				Text: line,
			}

			if contextLines > 0 && len(recentLines) > 0 {
				start := 0
				if len(recentLines) > contextLines {
					start = len(recentLines) - contextLines - 1
				}
				for i := start; i < len(recentLines)-1; i++ {
					match.Context = append(match.Context, recentLines[i])
				}
			}

			matches = append(matches, match)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return matches, nil
}
