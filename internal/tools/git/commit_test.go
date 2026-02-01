package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCommitTool_Success(t *testing.T) {
	tmpDir, repo := setupTestRepo(t)
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := w.Add("test.txt"); err != nil {
		t.Fatal(err)
	}

	tool := NewCommitTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
		"message":   "Test commit",
		"author":    "Test User",
		"email":     "test@example.com",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}

	if result.Output == "" {
		t.Error("Expected commit hash in output")
	}
}

func TestCommitTool_NothingToCommit(t *testing.T) {
	tmpDir, repo := setupTestRepo(t)
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	commitFile(t, repo, testFile, "content")

	tool := NewCommitTool()
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
		"message":   "Nothing to commit",
		"author":    "Test",
		"email":     "test@example.com",
	})

	if err == nil {
		t.Error("Expected error when nothing to commit")
	}
}

func TestCommitTool_MissingMessage(t *testing.T) {
	tmpDir, repo := setupTestRepo(t)
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "new.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	w.Add("new.txt")

	tool := NewCommitTool()
	ctx := context.Background()

	_, err = tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
	})

	if err == nil {
		t.Error("Expected error for missing commit message")
	}
}

func TestCommitTool_DefaultAuthor(t *testing.T) {
	tmpDir, repo := setupTestRepo(t)
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	w.Add("test.txt")

	tool := NewCommitTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
		"message":   "Test with default author",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success with default author")
	}
}

func TestCommitTool_NotARepository(t *testing.T) {
	tmpDir := t.TempDir()

	tool := NewCommitTool()
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
		"message":   "Test",
	})

	if err == nil {
		t.Error("Expected error for non-repository")
	}
}

func TestCommitTool_VerifyCommit(t *testing.T) {
	tmpDir, repo := setupTestRepo(t)
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	w.Add("test.txt")

	tool := NewCommitTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
		"message":   "Verify commit message",
		"author":    "Tester",
		"email":     "tester@test.com",
	})

	if err != nil {
		t.Fatal(err)
	}

	ref, err := repo.Head()
	if err != nil {
		t.Fatal(err)
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		t.Fatal(err)
	}

	if commit.Message != "Verify commit message" {
		t.Errorf("Expected message 'Verify commit message', got %q", commit.Message)
	}

	if commit.Author.Name != "Tester" {
		t.Errorf("Expected author 'Tester', got %q", commit.Author.Name)
	}

	if !result.Success {
		t.Error("Expected success")
	}
}
