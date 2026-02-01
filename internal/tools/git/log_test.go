package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestLogTool_Basic(t *testing.T) {
	tmpDir, repo := setupTestRepo(t)
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	commitFile(t, repo, testFile, "content")

	tool := NewLogTool()
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

	if !strings.Contains(result.Output, "test commit") {
		t.Error("Expected commit message in log output")
	}
}

func TestLogTool_WithLimit(t *testing.T) {
	tmpDir, repo := setupTestRepo(t)
	defer os.RemoveAll(tmpDir)

	for i := 0; i < 5; i++ {
		testFile := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
		commitFile(t, repo, testFile, fmt.Sprintf("commit %d", i))
		time.Sleep(time.Millisecond)
	}

	tool := NewLogTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
		"limit":     2,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}

	lines := strings.Split(strings.TrimSpace(result.Output), "\n")
	commitCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "commit ") {
			commitCount++
		}
	}

	if commitCount > 2 {
		t.Errorf("Expected at most 2 commits in output, got %d", commitCount)
	}
}

func TestLogTool_SpecificFile(t *testing.T) {
	tmpDir, repo := setupTestRepo(t)
	defer os.RemoveAll(tmpDir)

	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	commitFile(t, repo, file1, "file1 content")
	commitFile(t, repo, file2, "file2 content")

	tool := NewLogTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
		"file_path": "file1.txt",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}
}

func TestLogTool_NoCommits(t *testing.T) {
	tmpDir, _ := setupTestRepo(t)
	defer os.RemoveAll(tmpDir)

	tool := NewLogTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success even with no commits")
	}

	if !strings.Contains(result.Output, "No commits") {
		t.Error("Expected 'No commits' message")
	}
}

func TestLogTool_NotARepository(t *testing.T) {
	tmpDir := t.TempDir()

	tool := NewLogTool()
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
	})

	if err == nil {
		t.Error("Expected error for non-repository")
	}
}

func TestLogTool_MultipleCommits(t *testing.T) {
	tmpDir, repo := setupTestRepo(t)
	defer os.RemoveAll(tmpDir)

	w, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		testFile := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
		if err := os.WriteFile(testFile, []byte(fmt.Sprintf("content%d", i)), 0644); err != nil {
			t.Fatal(err)
		}

		if _, err := w.Add(filepath.Base(testFile)); err != nil {
			t.Fatal(err)
		}

		_, err = w.Commit(fmt.Sprintf("Commit %d", i), &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Test",
				Email: "test@example.com",
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Millisecond)
	}

	tool := NewLogTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for i := 0; i < 3; i++ {
		expectedMsg := fmt.Sprintf("Commit %d", i)
		if !strings.Contains(result.Output, expectedMsg) {
			t.Errorf("Expected to find '%s' in log", expectedMsg)
		}
	}
}
