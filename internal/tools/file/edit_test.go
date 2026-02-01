package file

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditTool_Execute(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewEditTool(tmpDir)

	tests := []struct {
		name         string
		fileContent  string
		params       map[string]interface{}
		wantErr      bool
		wantContains string
	}{
		{
			name:        "successful edit",
			fileContent: "Hello world\nGoodbye world",
			params: map[string]interface{}{
				"path":    "test.txt",
				"search":  "Hello",
				"replace": "Hi",
			},
			wantErr:      false,
			wantContains: "Hi world",
		},
		{
			name:        "search text not found",
			fileContent: "Hello world",
			params: map[string]interface{}{
				"path":    "test.txt",
				"search":  "Nonexistent",
				"replace": "Replacement",
			},
			wantErr: true,
		},
		{
			name:        "multiple occurrences",
			fileContent: "test test test",
			params: map[string]interface{}{
				"path":    "test.txt",
				"search":  "test",
				"replace": "new",
			},
			wantErr: true,
		},
		{
			name:        "file not found",
			fileContent: "",
			params: map[string]interface{}{
				"path":    "nonexistent.txt",
				"search":  "test",
				"replace": "new",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, tt.params["path"].(string))

			if tt.fileContent != "" {
				if err := os.WriteFile(testFile, []byte(tt.fileContent), 0644); err != nil {
					t.Fatal(err)
				}
			}

			result, err := tool.Execute(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			if tt.wantErr && result.Success {
				t.Error("expected error result")
			}

			if !tt.wantErr {
				if !result.Success {
					t.Errorf("unexpected error: %v", result.Error)
				}

				content, err := os.ReadFile(testFile)
				if err != nil {
					t.Fatal(err)
				}

				if !strings.Contains(string(content), tt.wantContains) {
					t.Errorf("expected content to contain %q, got %q", tt.wantContains, string(content))
				}
			}

			os.Remove(testFile)
		})
	}
}
