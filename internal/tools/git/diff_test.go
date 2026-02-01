package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDiffTool_Unstaged(t *testing.T) {
	tmpDir, repo := setupTestRepo(t)
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	commitFile(t, repo, testFile, "initial content")

	if err := os.WriteFile(testFile, []byte("modified content"), 0644); err != nil {
		t.Fatal(err)
	}

	tool := NewDiffTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
		"staged":    false,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}

	if result.Output == "" {
		t.Error("Expected diff output for modified file")
	}
}

func TestDiffTool_Staged(t *testing.T) {
	tmpDir, repo := setupTestRepo(t)
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	commitFile(t, repo, testFile, "initial")

	if err := os.WriteFile(testFile, []byte("modified"), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := w.Add("test.txt"); err != nil {
		t.Fatal(err)
	}

	tool := NewDiffTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
		"staged":    true,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}
}

func TestDiffTool_NoDiff(t *testing.T) {
	tmpDir, repo := setupTestRepo(t)
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	commitFile(t, repo, testFile, "content")

	tool := NewDiffTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
		"staged":    false,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}
}

func TestDiffTool_SpecificFile(t *testing.T) {
	tmpDir, repo := setupTestRepo(t)
	defer os.RemoveAll(tmpDir)

	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	commitFile(t, repo, file1, "content1")
	commitFile(t, repo, file2, "content2")

	if err := os.WriteFile(file1, []byte("modified1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("modified2"), 0644); err != nil {
		t.Fatal(err)
	}

	tool := NewDiffTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
		"file_path": "file1.txt",
		"staged":    false,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}
}

func TestDiffTool_NotARepository(t *testing.T) {
	tmpDir := t.TempDir()

	tool := NewDiffTool()
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
	})

	if err == nil {
		t.Error("Expected error for non-repository")
	}
}
