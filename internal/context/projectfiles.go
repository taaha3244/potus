package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var DefaultContextFileNames = []string{
	"POTUS.md",
	"CLAUDE.md",
	"AGENTS.md",
	"CONTEXT.md",
	".potus/context.md",
	".claude/context.md",
}

type ProjectFiles struct {
	contextFileNames []string
	maxTokens        int
}

type ProjectContext struct {
	Files       []ContextFile
	TotalTokens int
}

type ContextFile struct {
	Path    string
	Name    string
	Content string
	Tokens  int
}

type ProjectFilesConfig struct {
	ContextFileNames []string
	MaxTokens        int
}

func NewProjectFiles(cfg ProjectFilesConfig) *ProjectFiles {
	fileNames := cfg.ContextFileNames
	if len(fileNames) == 0 {
		fileNames = DefaultContextFileNames
	}

	return &ProjectFiles{
		contextFileNames: fileNames,
		maxTokens:        cfg.MaxTokens,
	}
}

func (pf *ProjectFiles) Load(workDir string, estimator TokenEstimator) (*ProjectContext, error) {
	ctx := &ProjectContext{
		Files: make([]ContextFile, 0),
	}

	searchPaths := pf.buildSearchPaths(workDir)
	loaded := make(map[string]bool)

	for _, dir := range searchPaths {
		for _, name := range pf.contextFileNames {
			baseName := filepath.Base(name)
			if loaded[baseName] {
				continue
			}

			path := filepath.Join(dir, name)
			content, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			tokens := estimator.EstimateTokens(string(content))

			if pf.maxTokens > 0 && ctx.TotalTokens+tokens > pf.maxTokens {
				continue
			}

			ctx.Files = append(ctx.Files, ContextFile{
				Path:    path,
				Name:    baseName,
				Content: string(content),
				Tokens:  tokens,
			})
			ctx.TotalTokens += tokens
			loaded[baseName] = true
		}
	}

	return ctx, nil
}

func (pf *ProjectFiles) buildSearchPaths(workDir string) []string {
	paths := make([]string, 0)

	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		absWorkDir = workDir
	}

	current := absWorkDir
	for {
		paths = append(paths, current)

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "potus"))
		paths = append(paths, filepath.Join(home, ".potus"))
	}

	return paths
}

func (pf *ProjectFiles) FormatForSystemPrompt(ctx *ProjectContext) string {
	if ctx == nil || len(ctx.Files) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("\n\n## Project Context\n\n")
	builder.WriteString("The following project-specific context has been loaded:\n\n")

	for _, file := range ctx.Files {
		builder.WriteString(fmt.Sprintf("### From %s\n", file.Name))
		builder.WriteString(fmt.Sprintf("(Source: %s)\n\n", file.Path))
		builder.WriteString(file.Content)
		builder.WriteString("\n\n")
	}

	return builder.String()
}

func (pf *ProjectFiles) GetLoadedFiles(ctx *ProjectContext) []string {
	if ctx == nil {
		return nil
	}

	files := make([]string, len(ctx.Files))
	for i, f := range ctx.Files {
		files[i] = f.Path
	}
	return files
}

func (pf *ProjectFiles) FindContextFile(workDir, fileName string) (string, error) {
	searchPaths := pf.buildSearchPaths(workDir)

	for _, dir := range searchPaths {
		path := filepath.Join(dir, fileName)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("context file %s not found", fileName)
}
