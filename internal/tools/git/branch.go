package git

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/taaha3244/potus/internal/tools"
)

type BranchTool struct{}

func NewBranchTool() *BranchTool {
	return &BranchTool{}
}

func (t *BranchTool) Name() string {
	return "git_branch"
}

func (t *BranchTool) Description() string {
	return "List, create, or switch branches"
}

func (t *BranchTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"repo_path": map[string]interface{}{
				"type":        "string",
				"description": "Path to git repository (defaults to current directory)",
			},
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"list", "create", "switch"},
				"description": "Action to perform (default: list)",
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Branch name (required for create/switch)",
			},
		},
	}
}

func (t *BranchTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.Result, error) {
	repoPath := "."
	if path, ok := params["repo_path"].(string); ok && path != "" {
		repoPath = path
	}

	action := "list"
	if a, ok := params["action"].(string); ok && a != "" {
		action = a
	}

	name := ""
	if n, ok := params["name"].(string); ok {
		name = n
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	switch action {
	case "list":
		return t.listBranches(repo)
	case "create":
		if name == "" {
			return nil, fmt.Errorf("branch name is required for create action")
		}
		return t.createBranch(repo, name)
	case "switch":
		if name == "" {
			return nil, fmt.Errorf("branch name is required for switch action")
		}
		return t.switchBranch(repo, name)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func (t *BranchTool) listBranches(repo *git.Repository) (*tools.Result, error) {
	branches, err := repo.Branches()
	if err != nil {
		return nil, fmt.Errorf("failed to get branches: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	var output strings.Builder
	currentBranch := head.Name().Short()

	err = branches.ForEach(func(ref *plumbing.Reference) error {
		branchName := ref.Name().Short()
		if branchName == currentBranch {
			output.WriteString(fmt.Sprintf("* %s\n", branchName))
		} else {
			output.WriteString(fmt.Sprintf("  %s\n", branchName))
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to iterate branches: %w", err)
	}

	return &tools.Result{
		Success: true,
		Output:  output.String(),
	}, nil
}

func (t *BranchTool) createBranch(repo *git.Repository, name string) (*tools.Result, error) {
	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	refName := plumbing.NewBranchReferenceName(name)
	ref := plumbing.NewHashReference(refName, head.Hash())

	err = repo.Storer.SetReference(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to create branch: %w", err)
	}

	return &tools.Result{
		Success: true,
		Output:  fmt.Sprintf("Created branch '%s'", name),
	}, nil
}

func (t *BranchTool) switchBranch(repo *git.Repository, name string) (*tools.Result, error) {
	w, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	refName := plumbing.NewBranchReferenceName(name)

	err = w.Checkout(&git.CheckoutOptions{
		Branch: refName,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to switch branch: %w", err)
	}

	return &tools.Result{
		Success: true,
		Output:  fmt.Sprintf("Switched to branch '%s'", name),
	}, nil
}
