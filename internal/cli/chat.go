package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/taaha3244/potus/internal/agent"
	"github.com/taaha3244/potus/internal/auth"
	"github.com/taaha3244/potus/internal/config"
	"github.com/taaha3244/potus/internal/permissions"
	"github.com/taaha3244/potus/internal/providers"
	"github.com/taaha3244/potus/internal/providers/anthropic"
	"github.com/taaha3244/potus/internal/providers/ollama"
	"github.com/taaha3244/potus/internal/providers/openai"
	"github.com/taaha3244/potus/internal/tools"
	"github.com/taaha3244/potus/internal/tools/bash"
	"github.com/taaha3244/potus/internal/tools/file"
	"github.com/taaha3244/potus/internal/tools/git"
	"github.com/taaha3244/potus/internal/tools/search"
	"github.com/taaha3244/potus/internal/tools/web"
	"github.com/taaha3244/potus/internal/tui"
)

func runChat(cmd *cobra.Command, args []string) error {
	modelFlag, _ := cmd.Flags().GetString("model")
	dirFlag, _ := cmd.Flags().GetString("dir")

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	workDir := dirFlag
	if workDir == "" {
		workDir, _ = os.Getwd()
	} else {
		if err := os.Chdir(dirFlag); err != nil {
			return fmt.Errorf("failed to change directory: %w", err)
		}
		workDir, _ = os.Getwd()
	}

	providerRegistry := providers.NewRegistry()
	authStore := auth.NewStore()

	if apiKey := auth.ResolveAPIKey(authStore, "anthropic", cfg.Providers["anthropic"].APIKeyEnv); apiKey != "" {
		anthropicClient, err := anthropic.New(apiKey)
		if err == nil {
			providerRegistry.Register("anthropic", anthropicClient)
		}
	}

	if apiKey := auth.ResolveAPIKey(authStore, "openai", cfg.Providers["openai"].APIKeyEnv); apiKey != "" {
		openaiClient, err := openai.New(apiKey, cfg.Providers["openai"].Organization)
		if err == nil {
			providerRegistry.Register("openai", openaiClient)
		}
	}

	ollamaClient, err := ollama.New(cfg.Providers["ollama"].Endpoint)
	if err == nil {
		providerRegistry.Register("ollama", ollamaClient)
	}

	modelStr := modelFlag
	if modelStr == "" {
		modelStr = cfg.Agents["default"].Model
	}

	providerName, modelName := providers.ParseModelString(modelStr)
	if providerName == "" {
		providerName = "anthropic"
		modelName = modelStr
	}

	provider, err := providerRegistry.Get(providerName)
	if err != nil {
		return fmt.Errorf("provider not available: %w", err)
	}

	// Get model info for context size and pricing
	var modelInfo *providers.Model
	if models, err := provider.ListModels(cmd.Context()); err == nil {
		for _, m := range models {
			if m.ID == modelName || m.Name == modelName {
				modelInfo = &m
				break
			}
		}
	}

	// Load permission settings
	permSettings := permissions.LoadSettings(workDir)

	// Create confirmation channel for tool approval
	confirmChan := make(chan agent.Decision, 1)

	// Register tools
	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(file.NewReadTool(workDir))
	toolRegistry.Register(file.NewWriteTool(workDir))
	toolRegistry.Register(file.NewEditTool(workDir))
	toolRegistry.Register(file.NewDeleteTool(workDir))
	toolRegistry.Register(bash.NewExecutorTool(workDir, 0, nil, bash.DefaultBlocklist))
	toolRegistry.Register(git.NewStatusTool())
	toolRegistry.Register(git.NewDiffTool())
	toolRegistry.Register(git.NewCommitTool())
	toolRegistry.Register(git.NewBranchTool())
	toolRegistry.Register(git.NewLogTool())
	toolRegistry.Register(search.NewFileSearchTool())
	toolRegistry.Register(search.NewGrepTool())
	toolRegistry.Register(web.NewFetchTool())
	toolRegistry.Register(web.NewSearchTool())

	systemPrompt := `You are POTUS (Power Of The Universal Shell), an AI coding assistant.

You have access to tools to read, write, and edit files, execute bash commands, work with git repositories, search code, and fetch web content.
You can help with coding tasks, debugging, refactoring, and more.

When editing files:
- Always read the file first before editing
- Use file_edit for precise changes (search and replace must match exactly)
- Use file_write only for new files

When executing bash commands:
- Be cautious with destructive operations
- Provide clear explanations of what you're doing

When working with git:
- Use git_status to check repository state
- Use git_diff to see changes
- Use git_commit to create commits (stage files with bash first)
- Use git_branch to manage branches
- Use git_log to view commit history

When searching code:
- Use search_files to find files by pattern (*.go, **/*.ts, etc.)
- Use search_content to search for text within files
- Combine searches with file type filters for precision

When fetching web content:
- Use web_fetch to retrieve documentation or web pages
- Use web_search to search for information online

Be helpful, accurate, and concise in your responses.`

	ag := agent.New(&agent.Config{
		Provider:      provider,
		ToolRegistry:  toolRegistry,
		SystemPrompt:  systemPrompt,
		MaxTokens:     cfg.Agents["default"].MaxTokens,
		Temperature:   cfg.Agents["default"].Temperature,
		Model:         modelName,
		ContextConfig: &cfg.Context,
		ModelInfo:     modelInfo,
		WorkDir:       workDir,
		ConfirmChan:   confirmChan,
		Settings:      permSettings,
	})

	return tui.Run(ag, modelStr, confirmChan)
}
