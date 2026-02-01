package git

import (
	"context"
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/taaha3244/potus/internal/tools"
)

type CommitTool struct{}

func NewCommitTool() *CommitTool {
	return &CommitTool{}
}

func (t *CommitTool) Name() string {
	return "git_commit"
}

func (t *CommitTool) Description() string {
	return "Create a new commit with staged changes"
}

func (t *CommitTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"repo_path": map[string]interface{}{
				"type":        "string",
				"description": "Path to git repository (defaults to current directory)",
			},
			"message": map[string]interface{}{
				"type":        "string",
				"description": "Commit message (required)",
			},
			"author": map[string]interface{}{
				"type":        "string",
				"description": "Author name (optional, defaults to git config)",
			},
			"email": map[string]interface{}{
				"type":        "string",
				"description": "Author email (optional, defaults to git config)",
			},
		},
		"required": []string{"message"},
	}
}

func (t *CommitTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.Result, error) {
	repoPath := "."
	if path, ok := params["repo_path"].(string); ok && path != "" {
		repoPath = path
	}

	message, ok := params["message"].(string)
	if !ok || message == "" {
		return nil, fmt.Errorf("commit message is required")
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	author := ""
	if a, ok := params["author"].(string); ok && a != "" {
		author = a
	}

	email := ""
	if e, ok := params["email"].(string); ok && e != "" {
		email = e
	}

	if author == "" || email == "" {
		cfg, err := repo.Config()
		if err == nil && cfg.User.Name != "" && cfg.User.Email != "" {
			if author == "" {
				author = cfg.User.Name
			}
			if email == "" {
				email = cfg.User.Email
			}
		} else {
			if author == "" {
				author = "Unknown"
			}
			if email == "" {
				email = "unknown@localhost"
			}
		}
	}

	w, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := w.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	hasStaged := false
	for _, stat := range status {
		if stat.Staging != git.Unmodified {
			hasStaged = true
			break
		}
	}

	if !hasStaged {
		return nil, fmt.Errorf("nothing to commit (no staged changes)")
	}

	hash, err := w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  author,
			Email: email,
			When:  time.Now(),
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	return &tools.Result{
		Success: true,
		Output:  fmt.Sprintf("Created commit %s\n%s", hash.String()[:7], message),
	}, nil
}
