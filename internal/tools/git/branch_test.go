package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

func TestBranchTool_List(t *testing.T) {
	tmpDir, repo := setupTestRepo(t)
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	commitFile(t, repo, testFile, "content")

	tool := NewBranchTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
		"action":    "list",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}

	if result.Output == "" {
		t.Error("Expected branch list in output")
	}
}

func TestBranchTool_Create(t *testing.T) {
	tmpDir, repo := setupTestRepo(t)
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	commitFile(t, repo, testFile, "content")

	tool := NewBranchTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
		"action":    "create",
		"name":      "feature-branch",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}

	branches, err := repo.Branches()
	if err != nil {
		t.Fatal(err)
	}

	found := false
	err = branches.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().Short() == "feature-branch" {
			found = true
		}
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	if !found {
		t.Error("Branch 'feature-branch' was not created")
	}
}

func TestBranchTool_Switch(t *testing.T) {
	tmpDir, repo := setupTestRepo(t)
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	commitFile(t, repo, testFile, "content")

	w, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("new-branch"),
		Create: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	tool := NewBranchTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
		"action":    "switch",
		"name":      "master",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}

	ref, err := repo.Head()
	if err != nil {
		t.Fatal(err)
	}

	if ref.Name().Short() != "master" {
		t.Errorf("Expected to be on 'master', got %q", ref.Name().Short())
	}
}

func TestBranchTool_CreateMissingName(t *testing.T) {
	tmpDir, repo := setupTestRepo(t)
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	commitFile(t, repo, testFile, "content")

	tool := NewBranchTool()
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
		"action":    "create",
	})

	if err == nil {
		t.Error("Expected error when creating branch without name")
	}
}

func TestBranchTool_SwitchNonexistent(t *testing.T) {
	tmpDir, repo := setupTestRepo(t)
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	commitFile(t, repo, testFile, "content")

	tool := NewBranchTool()
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
		"action":    "switch",
		"name":      "nonexistent",
	})

	if err == nil {
		t.Error("Expected error when switching to nonexistent branch")
	}
}

func TestBranchTool_DefaultAction(t *testing.T) {
	tmpDir, repo := setupTestRepo(t)
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	commitFile(t, repo, testFile, "content")

	tool := NewBranchTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected success for default action (list)")
	}
}

func TestBranchTool_NotARepository(t *testing.T) {
	tmpDir := t.TempDir()

	tool := NewBranchTool()
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]interface{}{
		"repo_path": tmpDir,
		"action":    "list",
	})

	if err == nil {
		t.Error("Expected error for non-repository")
	}
}
