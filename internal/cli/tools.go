package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/taaha3244/potus/internal/config"
	"github.com/taaha3244/potus/internal/tools"
	"github.com/taaha3244/potus/internal/tools/bash"
	"github.com/taaha3244/potus/internal/tools/file"
	"github.com/taaha3244/potus/internal/tools/git"
	"github.com/taaha3244/potus/internal/tools/search"
	"github.com/taaha3244/potus/internal/tools/web"
)

func newToolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tools",
		Short: "Manage available tools",
		Long:  "List and inspect available tools that the AI agent can use.",
	}

	cmd.AddCommand(newToolsListCmd())
	cmd.AddCommand(newToolsInfoCmd())

	return cmd
}

func newToolsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available tools",
		Long:  "Show all tools available to the AI agent along with their permission status.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			registry := buildToolRegistry()

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "TOOL\tDESCRIPTION\tPERMISSION")
			fmt.Fprintln(w, "----\t-----------\t----------")

			for _, tool := range registry.List() {
				permission := getToolPermission(tool.Name(), cfg)
				fmt.Fprintf(w, "%s\t%s\t%s\n",
					tool.Name(),
					truncate(tool.Description(), 50),
					permission)
			}

			w.Flush()
			return nil
		},
	}
}

func newToolsInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info [tool-name]",
		Short: "Show tool details",
		Long: `Show detailed information about a specific tool including its schema.

Examples:
  potus tools info file_read
  potus tools info bash
  potus tools info git_commit`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			toolName := args[0]

			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			registry := buildToolRegistry()
			tool, err := registry.Get(toolName)
			if err != nil {
				return fmt.Errorf("tool not found: %s", toolName)
			}

			fmt.Printf("Name: %s\n", tool.Name())
			fmt.Printf("Description: %s\n", tool.Description())
			fmt.Printf("Permission: %s\n\n", getToolPermission(toolName, cfg))

			fmt.Println("Schema:")
			schema := tool.Schema()
			schemaJSON, err := json.MarshalIndent(schema, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal schema: %w", err)
			}
			fmt.Println(string(schemaJSON))

			return nil
		},
	}
}

func buildToolRegistry() *tools.Registry {
	workDir, _ := os.Getwd()
	registry := tools.NewRegistry()

	// File tools
	registry.Register(file.NewReadTool(workDir))
	registry.Register(file.NewWriteTool(workDir))
	registry.Register(file.NewEditTool(workDir))
	registry.Register(file.NewDeleteTool(workDir))

	// Bash tool
	registry.Register(bash.NewExecutorTool(workDir, 0, nil, bash.DefaultBlocklist))

	// Git tools
	registry.Register(git.NewStatusTool())
	registry.Register(git.NewDiffTool())
	registry.Register(git.NewCommitTool())
	registry.Register(git.NewBranchTool())
	registry.Register(git.NewLogTool())

	// Search tools
	registry.Register(search.NewFileSearchTool())
	registry.Register(search.NewGrepTool())

	// Web tools
	registry.Register(web.NewFetchTool())
	registry.Register(web.NewSearchTool())

	return registry
}

func getToolPermission(toolName string, cfg *config.Config) string {
	switch toolName {
	case "file_read":
		return string(cfg.Permissions.FileRead)
	case "file_write":
		return string(cfg.Permissions.FileWrite)
	case "file_edit":
		return string(cfg.Permissions.FileEdit)
	case "file_delete":
		return string(cfg.Permissions.FileDelete)
	case "bash":
		return string(cfg.Permissions.Bash)
	case "git_status", "git_diff", "git_commit", "git_branch", "git_log":
		return string(cfg.Permissions.Git)
	case "web_fetch":
		return string(cfg.Permissions.WebFetch)
	case "web_search":
		return string(cfg.Permissions.WebSearch)
	default:
		return "ask"
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
