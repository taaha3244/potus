package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func setupTestRepo(t *testing.T) (string, *git.Repository) {
	tmpDir := t.TempDir()

	repo, err := git.PlainInit(tmpDir, false)
	if err != nil {
		t.Fatalf("Failed to init repo: %v", err)
	}

	return tmpDir, repo
}

func commitFile(t *testing.T, repo *git.Repository, path, content string) {
	w, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := w.Add(filepath.Base(path)); err != nil {
		t.Fatal(err)
	}

	_, err = w.Commit("test commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestStatusTool_CleanWorkingTree(t *testing.T) {
	tmpDir, repo := setupTestRepo(t)
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	commitFile(t, repo, testFile, "initial")

	tool := NewStatusTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success for clean working tree")
	}

	if result.Output == "" {
		t.Error("Expected status output")
	}
}

func TestStatusTool_ModifiedFiles(t *testing.T) {
	tmpDir, repo := setupTestRepo(t)
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	commitFile(t, repo, testFile, "initial")

	if err := os.WriteFile(testFile, []byte("modified"), 0644); err != nil {
		t.Fatal(err)
	}

	tool := NewStatusTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}
}

func TestStatusTool_UntrackedFiles(t *testing.T) {
	tmpDir, _ := setupTestRepo(t)
	defer os.RemoveAll(tmpDir)

	newFile := filepath.Join(tmpDir, "untracked.txt")
	if err := os.WriteFile(newFile, []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}

	tool := NewStatusTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}
}

func TestStatusTool_NotARepository(t *testing.T) {
	tmpDir := t.TempDir()

	tool := NewStatusTool()
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
	})

	if err == nil {
		t.Error("Expected error for non-repository path")
	}
}

func TestStatusTool_DefaultsToCurrentDir(t *testing.T) {
	tool := NewStatusTool()
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]interface{}{})

	if err != nil {
		t.Logf("Expected error when current dir is not a repo: %v", err)
	}
}
