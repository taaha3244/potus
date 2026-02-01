package search

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestDir(t *testing.T) string {
	tmpDir := t.TempDir()

	files := map[string]string{
		"main.go":           "package main",
		"utils/helper.go":   "package utils",
		"utils/parser.go":   "package utils",
		"cmd/app/main.go":   "package main",
		"README.md":         "# Project",
		"internal/core.go":  "package internal",
		".hidden/file.go":   "package hidden",
		"test/data.txt":     "test data",
		"config.yaml":       "config: true",
		"Makefile":          "build:",
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

func TestFileSearchTool_FindGoFiles(t *testing.T) {
	tmpDir := setupTestDir(t)

	tool := NewFileSearchTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"root_path": tmpDir,
		"pattern":   "*.go",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}

	// Pattern *.go is non-recursive, only matches root directory
	if !strings.Contains(result.Output, "main.go") {
		t.Error("Expected to find main.go")
	}

	// helper.go is in utils/ subdirectory, should NOT be found with non-recursive pattern
	if strings.Contains(result.Output, "helper.go") {
		t.Error("Non-recursive pattern should not find files in subdirectories")
	}
}

func TestFileSearchTool_RecursiveSearch(t *testing.T) {
	tmpDir := setupTestDir(t)

	tool := NewFileSearchTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"root_path": tmpDir,
		"pattern":   "**/*.go",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}

	goFileCount := strings.Count(result.Output, ".go")
	if goFileCount < 5 {
		t.Errorf("Expected to find at least 5 .go files, found %d", goFileCount)
	}
}

func TestFileSearchTool_SpecificDirectory(t *testing.T) {
	tmpDir := setupTestDir(t)

	tool := NewFileSearchTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"root_path": tmpDir,
		"pattern":   "utils/*.go",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}

	if !strings.Contains(result.Output, "helper.go") {
		t.Error("Expected to find utils/helper.go")
	}

	if strings.Contains(result.Output, "cmd/app/main.go") {
		t.Error("Should not find files outside utils directory")
	}
}

func TestFileSearchTool_NoMatches(t *testing.T) {
	tmpDir := setupTestDir(t)

	tool := NewFileSearchTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"root_path": tmpDir,
		"pattern":   "*.rs",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success even with no matches")
	}

	if !strings.Contains(result.Output, "No files found") {
		t.Error("Expected 'No files found' message")
	}
}

func TestFileSearchTool_MultipleExtensions(t *testing.T) {
	tmpDir := setupTestDir(t)

	tool := NewFileSearchTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"root_path": tmpDir,
		"pattern":   "*.{go,md}",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}

	if !strings.Contains(result.Output, ".go") {
		t.Error("Expected to find .go files")
	}

	if !strings.Contains(result.Output, "README.md") {
		t.Error("Expected to find .md files")
	}
}

func TestFileSearchTool_IgnoreHidden(t *testing.T) {
	tmpDir := setupTestDir(t)

	tool := NewFileSearchTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"root_path":     tmpDir,
		"pattern":       "**/*.go",
		"include_hidden": false,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if strings.Contains(result.Output, ".hidden") {
		t.Error("Should not include hidden directories")
	}
}

func TestFileSearchTool_IncludeHidden(t *testing.T) {
	tmpDir := setupTestDir(t)

	tool := NewFileSearchTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"root_path":     tmpDir,
		"pattern":       "**/*.go",
		"include_hidden": true,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(result.Output, ".hidden") {
		t.Error("Expected to include hidden directories when requested")
	}
}

func TestFileSearchTool_InvalidPath(t *testing.T) {
	tool := NewFileSearchTool()
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]interface{}{
		"root_path": "/nonexistent/path",
		"pattern":   "*.go",
	})

	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

func TestFileSearchTool_DefaultCurrentDir(t *testing.T) {
	tool := NewFileSearchTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"pattern": "*.go",
	})

	if err != nil {
		t.Logf("Expected potential error when current dir has no matches: %v", err)
	} else if result != nil {
		t.Logf("Found files in current directory: %s", result.Output)
	}
}
