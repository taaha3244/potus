package git

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/taaha3244/potus/internal/tools"
)

type StatusTool struct{}

func NewStatusTool() *StatusTool {
	return &StatusTool{}
}

func (t *StatusTool) Name() string {
	return "git_status"
}

func (t *StatusTool) Description() string {
	return "Show the working tree status"
}

func (t *StatusTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"repo_path": map[string]interface{}{
				"type":        "string",
				"description": "Path to git repository (defaults to current directory)",
			},
		},
	}
}

func (t *StatusTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.Result, error) {
	repoPath := "."
	if path, ok := params["repo_path"].(string); ok && path != "" {
		repoPath = path
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := w.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	var output strings.Builder

	if status.IsClean() {
		output.WriteString("Working tree clean\n")
	} else {
		staged := []string{}
		modified := []string{}
		untracked := []string{}

		for file, stat := range status {
			if stat.Staging != git.Unmodified {
				staged = append(staged, fmt.Sprintf("  %s: %s", file, statusCode(stat.Staging)))
			}
			if stat.Worktree != git.Unmodified && stat.Worktree != git.Untracked {
				modified = append(modified, fmt.Sprintf("  %s: %s", file, statusCode(stat.Worktree)))
			}
			if stat.Worktree == git.Untracked {
				untracked = append(untracked, fmt.Sprintf("  %s", file))
			}
		}

		if len(staged) > 0 {
			output.WriteString("Changes to be committed:\n")
			output.WriteString(strings.Join(staged, "\n"))
			output.WriteString("\n\n")
		}

		if len(modified) > 0 {
			output.WriteString("Changes not staged for commit:\n")
			output.WriteString(strings.Join(modified, "\n"))
			output.WriteString("\n\n")
		}

		if len(untracked) > 0 {
			output.WriteString("Untracked files:\n")
			output.WriteString(strings.Join(untracked, "\n"))
			output.WriteString("\n")
		}
	}

	return &tools.Result{
		Success: true,
		Output:  output.String(),
	}, nil
}

func statusCode(code git.StatusCode) string {
	switch code {
	case git.Unmodified:
		return "unmodified"
	case git.Untracked:
		return "untracked"
	case git.Modified:
		return "modified"
	case git.Added:
		return "added"
	case git.Deleted:
		return "deleted"
	case git.Renamed:
		return "renamed"
	case git.Copied:
		return "copied"
	case git.UpdatedButUnmerged:
		return "updated but unmerged"
	default:
		return "unknown"
	}
}
