package search

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupGrepTestDir(t *testing.T) string {
	tmpDir := t.TempDir()

	files := map[string]string{
		"main.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}`,
		"utils/helper.go": `package utils

func Helper() string {
	return "helper function"
}`,
		"README.md": `# Project

This is a test project.
It has multiple lines.
Some lines contain the word test.`,
		"config.yaml": `database:
  host: localhost
  port: 5432`,
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	return tmpDir
}

func TestGrepTool_BasicSearch(t *testing.T) {
	tmpDir := setupGrepTestDir(t)

	tool := NewGrepTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"root_path": tmpDir,
		"pattern":   "main",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}

	if !strings.Contains(result.Output, "main.go") {
		t.Error("Expected to find matches in main.go")
	}
}

func TestGrepTool_CaseSensitive(t *testing.T) {
	tmpDir := setupGrepTestDir(t)

	tool := NewGrepTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"root_path":      tmpDir,
		"pattern":        "MAIN",
		"case_sensitive": true,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if strings.Contains(result.Output, "main.go") {
		t.Error("Case-sensitive search should not match 'main' with pattern 'MAIN'")
	}
}

func TestGrepTool_CaseInsensitive(t *testing.T) {
	tmpDir := setupGrepTestDir(t)

	tool := NewGrepTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"root_path":      tmpDir,
		"pattern":        "MAIN",
		"case_sensitive": false,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(result.Output, "main.go") {
		t.Error("Case-insensitive search should match 'main' with pattern 'MAIN'")
	}
}

func TestGrepTool_FileTypeFilter(t *testing.T) {
	tmpDir := setupGrepTestDir(t)

	tool := NewGrepTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"root_path":  tmpDir,
		"pattern":    "test",
		"file_types": []interface{}{"md"},
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}

	if !strings.Contains(result.Output, "README.md") {
		t.Error("Expected to find matches in README.md")
	}

	if strings.Contains(result.Output, "main.go") {
		t.Error("Should not search in .go files when filtering for .md")
	}
}

func TestGrepTool_WithContext(t *testing.T) {
	tmpDir := setupGrepTestDir(t)

	tool := NewGrepTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"root_path":     tmpDir,
		"pattern":       "Println",
		"context_lines": 2,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}

	if !strings.Contains(result.Output, "Println") {
		t.Error("Expected to find 'Println'")
	}
}

func TestGrepTool_NoMatches(t *testing.T) {
	tmpDir := setupGrepTestDir(t)

	tool := NewGrepTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"root_path": tmpDir,
		"pattern":   "nonexistent_pattern_xyz",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success even with no matches")
	}

	if !strings.Contains(result.Output, "No matches found") {
		t.Error("Expected 'No matches found' message")
	}
}

func TestGrepTool_MissingPattern(t *testing.T) {
	tmpDir := setupGrepTestDir(t)

	tool := NewGrepTool()
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]interface{}{
		"root_path": tmpDir,
	})

	if err == nil {
		t.Error("Expected error when pattern is missing")
	}
}

func TestGrepTool_InvalidPath(t *testing.T) {
	tool := NewGrepTool()
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]interface{}{
		"root_path": "/nonexistent/path",
		"pattern":   "test",
	})

	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

func TestGrepTool_MultipleMatches(t *testing.T) {
	tmpDir := setupGrepTestDir(t)

	tool := NewGrepTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"root_path": tmpDir,
		"pattern":   "package",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	matchCount := strings.Count(result.Output, "package")
	if matchCount < 2 {
		t.Errorf("Expected at least 2 matches for 'package', got %d", matchCount)
	}
}

func TestGrepTool_MaxResults(t *testing.T) {
	tmpDir := setupGrepTestDir(t)

	tool := NewGrepTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"root_path":   tmpDir,
		"pattern":     "test",
		"max_results": 1,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}
}
