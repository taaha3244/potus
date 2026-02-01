package cli

import (
	"fmt"

	"github.com/taaha3244/potus/internal/config"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config
)

func Execute(version, commit, date string) error {
	rootCmd := &cobra.Command{
		Use:   "potus",
		Short: "Power Of The Universal Shell - AI Coding Agent",
		Long:  `POTUS is an open-source, provider-agnostic AI coding agent CLI.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
		SilenceUsage: true,
		RunE: runChat,
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: $HOME/.config/potus/config.yaml)")
	rootCmd.PersistentFlags().String("model", "", "model to use (e.g., anthropic/claude-sonnet-4-5)")
	rootCmd.PersistentFlags().String("provider", "", "provider to use (anthropic, openai, ollama)")
	rootCmd.PersistentFlags().String("agent", "default", "agent preset to use")
	rootCmd.PersistentFlags().String("dir", ".", "working directory")

	rootCmd.AddCommand(newAuthCmd())
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newProvidersCmd())
	rootCmd.AddCommand(newToolsCmd())
	rootCmd.AddCommand(newMCPCmd())

	return rootCmd.Execute()
}
