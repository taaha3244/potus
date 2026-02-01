package git

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/taaha3244/potus/internal/tools"
)

type LogTool struct{}

func NewLogTool() *LogTool {
	return &LogTool{}
}

func (t *LogTool) Name() string {
	return "git_log"
}

func (t *LogTool) Description() string {
	return "Show commit logs"
}

func (t *LogTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"repo_path": map[string]interface{}{
				"type":        "string",
				"description": "Path to git repository (defaults to current directory)",
			},
			"limit": map[string]interface{}{
				"type":        "number",
				"description": "Maximum number of commits to show (default: 10)",
			},
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "Filter commits by file path (optional)",
			},
		},
	}
}

func (t *LogTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.Result, error) {
	repoPath := "."
	if path, ok := params["repo_path"].(string); ok && path != "" {
		repoPath = path
	}

	limit := 10
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
	} else if l, ok := params["limit"].(int); ok {
		limit = l
	}

	filePath := ""
	if f, ok := params["file_path"].(string); ok {
		filePath = f
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
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

	commitIter, err := repo.Log(&git.LogOptions{
		From: ref.Hash(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get log: %w", err)
	}
	defer commitIter.Close()

	var output strings.Builder
	count := 0

	err = commitIter.ForEach(func(commit *object.Commit) error {
		if count >= limit {
			return io.EOF
		}

		if filePath != "" {
			if !commitContainsFile(commit, filePath) {
				return nil
			}
		}

		output.WriteString(fmt.Sprintf("commit %s\n", commit.Hash.String()))
		output.WriteString(fmt.Sprintf("Author: %s <%s>\n", commit.Author.Name, commit.Author.Email))
		output.WriteString(fmt.Sprintf("Date:   %s\n", commit.Author.When.Format("Mon Jan 2 15:04:05 2006 -0700")))
		output.WriteString(fmt.Sprintf("\n    %s\n\n", strings.ReplaceAll(commit.Message, "\n", "\n    ")))

		count++
		return nil
	})

	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}

	if output.Len() == 0 {
		return &tools.Result{
			Success: true,
			Output:  "No commits found",
		}, nil
	}

	return &tools.Result{
		Success: true,
		Output:  output.String(),
	}, nil
}

func commitContainsFile(commit *object.Commit, filePath string) bool {
	tree, err := commit.Tree()
	if err != nil {
		return false
	}

	_, err = tree.File(filePath)
	return err == nil
}
