package git

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/taaha3244/potus/internal/tools"
)

type DiffTool struct{}

func NewDiffTool() *DiffTool {
	return &DiffTool{}
}

func (t *DiffTool) Name() string {
	return "git_diff"
}

func (t *DiffTool) Description() string {
	return "Show changes between commits, commit and working tree, etc"
}

func (t *DiffTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"repo_path": map[string]interface{}{
				"type":        "string",
				"description": "Path to git repository (defaults to current directory)",
			},
			"staged": map[string]interface{}{
				"type":        "boolean",
				"description": "Show staged changes (default: false shows unstaged)",
			},
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "Specific file to diff (optional)",
			},
		},
	}
}

func (t *DiffTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.Result, error) {
	repoPath := "."
	if path, ok := params["repo_path"].(string); ok && path != "" {
		repoPath = path
	}

	staged := false
	if s, ok := params["staged"].(bool); ok {
		staged = s
	}

	filePath := ""
	if f, ok := params["file_path"].(string); ok {
		filePath = f
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	ref, err := repo.Head()
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return &tools.Result{
				Success: true,
				Output:  "No commits yet",
			}, nil
		}
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	var output strings.Builder

	status, err := w.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	for file, stat := range status {
		if filePath != "" && file != filePath {
			continue
		}

		showFile := false
		if staged && stat.Staging != git.Unmodified {
			showFile = true
		} else if !staged && stat.Worktree != git.Unmodified && stat.Worktree != git.Untracked {
			showFile = true
		}

		if !showFile {
			continue
		}

		output.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", file, file))

		treeEntry, err := tree.File(file)
		var oldContent string
		if err == nil {
			oldContent, _ = treeEntry.Contents()
		}

		newContent := ""
		if stat.Worktree != git.Deleted {
			f, err := w.Filesystem.Open(file)
			if err == nil {
				newBytes, err := io.ReadAll(f)
				f.Close()
				if err == nil {
					newContent = string(newBytes)
				}
			}
		}

		output.WriteString(generateDiff(oldContent, newContent))
	}

	if output.Len() == 0 {
		return &tools.Result{
			Success: true,
			Output:  "No changes",
		}, nil
	}

	return &tools.Result{
		Success: true,
		Output:  output.String(),
	}, nil
}

func generateDiff(old, new string) string {
	var output strings.Builder

	oldLines := strings.Split(old, "\n")
	newLines := strings.Split(new, "\n")

	output.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", 1, len(oldLines), 1, len(newLines)))

	maxLines := len(oldLines)
	if len(newLines) > maxLines {
		maxLines = len(newLines)
	}

	for i := 0; i < maxLines; i++ {
		if i < len(oldLines) && i < len(newLines) {
			if oldLines[i] != newLines[i] {
				output.WriteString(fmt.Sprintf("-%s\n", oldLines[i]))
				output.WriteString(fmt.Sprintf("+%s\n", newLines[i]))
			} else {
				output.WriteString(fmt.Sprintf(" %s\n", oldLines[i]))
			}
		} else if i < len(oldLines) {
			output.WriteString(fmt.Sprintf("-%s\n", oldLines[i]))
		} else if i < len(newLines) {
			output.WriteString(fmt.Sprintf("+%s\n", newLines[i]))
		}
	}

	return output.String()
}
