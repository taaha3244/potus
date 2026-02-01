package file

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewDeleteTool(t *testing.T) {
	tool := NewDeleteTool("/work/dir")

	if tool == nil {
		t.Fatal("NewDeleteTool returned nil")
	}

	if tool.workDir != "/work/dir" {
		t.Errorf("workDir = %s, want /work/dir", tool.workDir)
	}
}

func TestDeleteTool_Name(t *testing.T) {
	tool := NewDeleteTool("")
	if tool.Name() != "file_delete" {
		t.Errorf("Name() = %s, want file_delete", tool.Name())
	}
}

func TestDeleteTool_Description(t *testing.T) {
	tool := NewDeleteTool("")
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() should not be empty")
	}
}

func TestDeleteTool_Schema(t *testing.T) {
	tool := NewDeleteTool("")
	schema := tool.Schema()

	if schema["type"] != "object" {
		t.Errorf("schema type = %v, want object", schema["type"])
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties should be a map")
	}

	if _, exists := props["path"]; !exists {
		t.Error("schema should have 'path' property")
	}

	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("required should be a string array")
	}
	if len(required) != 1 || required[0] != "path" {
		t.Error("'path' should be required")
	}
}

func TestDeleteTool_Execute(t *testing.T) {
	t.Run("missing path parameter", func(t *testing.T) {
		tool := NewDeleteTool("")
		result, err := tool.Execute(context.Background(), map[string]interface{}{})

		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if result.Success {
			t.Error("Expected failure for missing path")
		}
	})

	t.Run("invalid path type", func(t *testing.T) {
		tool := NewDeleteTool("")
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"path": 123, // wrong type
		})

		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if result.Success {
			t.Error("Expected failure for invalid path type")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		tool := NewDeleteTool(tmpDir)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"path": "nonexistent.txt",
		})

		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if result.Success {
			t.Error("Expected failure for nonexistent file")
		}
	})

	t.Run("cannot delete directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "subdir")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatal(err)
		}

		tool := NewDeleteTool(tmpDir)
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"path": "subdir",
		})

		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if result.Success {
			t.Error("Expected failure when trying to delete directory")
		}

		if _, statErr := os.Stat(subDir); os.IsNotExist(statErr) {
			t.Error("Directory should not have been deleted")
		}
	})

	t.Run("successful delete", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}

		tool := NewDeleteTool(tmpDir)
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"path": "test.txt",
		})

		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if !result.Success {
			t.Errorf("Expected success, got error: %s", result.Output)
		}

		if _, statErr := os.Stat(testFile); !os.IsNotExist(statErr) {
			t.Error("File should have been deleted")
		}
	})

	t.Run("delete with absolute path", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "absolute.txt")
		if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}

		tool := NewDeleteTool("/different/workdir")
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"path": testFile, // absolute path
		})

		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if !result.Success {
			t.Errorf("Expected success, got error: %s", result.Output)
		}

		if _, statErr := os.Stat(testFile); !os.IsNotExist(statErr) {
			t.Error("File should have been deleted")
		}
	})
}
