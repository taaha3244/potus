package file

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadTool_Execute(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	content := "line 1\nline 2\nline 3\nline 4\nline 5"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	tool := NewReadTool(tmpDir)

	tests := []struct {
		name      string
		params    map[string]interface{}
		wantErr   bool
		wantLines int
	}{
		{
			name:      "read entire file",
			params:    map[string]interface{}{"path": "test.txt"},
			wantErr:   false,
			wantLines: 5,
		},
		{
			name: "read with line range",
			params: map[string]interface{}{
				"path":       "test.txt",
				"start_line": float64(2),
				"end_line":   float64(4),
			},
			wantErr:   false,
			wantLines: 3,
		},
		{
			name:    "file not found",
			params:  map[string]interface{}{"path": "nonexistent.txt"},
			wantErr: true,
		},
		{
			name:    "missing path parameter",
			params:  map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			if tt.wantErr && result.Success {
				t.Error("expected error result")
			}

			if !tt.wantErr && !result.Success {
				t.Errorf("unexpected error: %v", result.Error)
			}

			if !tt.wantErr && tt.wantLines > 0 {
				lines := strings.Count(result.Output, "\n")
				if lines != tt.wantLines {
					t.Errorf("expected %d lines, got %d", tt.wantLines, lines)
				}
			}
		})
	}
}

func TestReadTool_Schema(t *testing.T) {
	tool := NewReadTool(".")
	schema := tool.Schema()

	if schema["type"] != "object" {
		t.Error("expected object type")
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected properties map")
	}

	if _, ok := props["path"]; !ok {
		t.Error("expected path property")
	}
}
