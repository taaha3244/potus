package file

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteTool_Execute(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewWriteTool(tmpDir)

	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
		setup   func()
	}{
		{
			name: "create new file",
			params: map[string]interface{}{
				"path":    "new.txt",
				"content": "hello world",
			},
			wantErr: false,
		},
		{
			name: "file already exists",
			params: map[string]interface{}{
				"path":    "existing.txt",
				"content": "content",
			},
			wantErr: true,
			setup: func() {
				os.WriteFile(filepath.Join(tmpDir, "existing.txt"), []byte("old"), 0644)
			},
		},
		{
			name: "create with nested directory",
			params: map[string]interface{}{
				"path":    "nested/dir/file.txt",
				"content": "nested content",
			},
			wantErr: false,
		},
		{
			name:    "missing path",
			params:  map[string]interface{}{"content": "test"},
			wantErr: true,
		},
		{
			name:    "missing content",
			params:  map[string]interface{}{"path": "test.txt"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
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

				path := tt.params["path"].(string)
				fullPath := filepath.Join(tmpDir, path)
				if _, err := os.Stat(fullPath); os.IsNotExist(err) {
					t.Errorf("file was not created: %s", fullPath)
				}
			}
		})
	}
}
