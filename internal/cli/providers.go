package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/taaha3244/potus/internal/auth"
	"github.com/taaha3244/potus/internal/config"
	"github.com/taaha3244/potus/internal/providers"
	"github.com/taaha3244/potus/internal/providers/anthropic"
	"github.com/taaha3244/potus/internal/providers/ollama"
	"github.com/taaha3244/potus/internal/providers/openai"
)

func newProvidersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "Manage LLM providers",
		Long:  "List, test, and inspect available LLM providers and their models.",
	}

	cmd.AddCommand(newProvidersListCmd())
	cmd.AddCommand(newProvidersTestCmd())
	cmd.AddCommand(newProvidersModelsCmd())

	return cmd
}

func newProvidersListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured providers",
		Long:  "Show all configured providers and their connection status.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			authStore := auth.NewStore()

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "PROVIDER\tSTATUS\tDEFAULT MODEL\tENDPOINT")
			fmt.Fprintln(w, "--------\t------\t-------------\t--------")

			// Check Anthropic
			anthropicStatus := "not configured"
			if apiKey := auth.ResolveAPIKey(authStore, "anthropic", cfg.Providers["anthropic"].APIKeyEnv); apiKey != "" {
				anthropicStatus = "ready"
			}
			fmt.Fprintf(w, "anthropic\t%s\t%s\t%s\n",
				anthropicStatus,
				cfg.Providers["anthropic"].DefaultModel,
				"api.anthropic.com")

			// Check OpenAI
			openaiStatus := "not configured"
			if apiKey := auth.ResolveAPIKey(authStore, "openai", cfg.Providers["openai"].APIKeyEnv); apiKey != "" {
				openaiStatus = "ready"
			}
			endpoint := cfg.Providers["openai"].Endpoint
			if endpoint == "" {
				endpoint = "api.openai.com"
			}
			fmt.Fprintf(w, "openai\t%s\t%s\t%s\n",
				openaiStatus,
				cfg.Providers["openai"].DefaultModel,
				endpoint)

			// Check Ollama
			ollamaStatus := "checking..."
			ollamaEndpoint := cfg.Providers["ollama"].Endpoint
			if ollamaEndpoint == "" {
				ollamaEndpoint = "http://localhost:11434"
			}
			client, err := ollama.New(ollamaEndpoint)
			if err == nil {
				ctx, cancel := context.WithTimeout(cmd.Context(), providers.DefaultTimeout)
				defer cancel()
				if _, err := client.ListModels(ctx); err == nil {
					ollamaStatus = "ready"
				} else {
					ollamaStatus = "unreachable"
				}
			} else {
				ollamaStatus = "error"
			}
			fmt.Fprintf(w, "ollama\t%s\t%s\t%s\n",
				ollamaStatus,
				cfg.Providers["ollama"].DefaultModel,
				ollamaEndpoint)

			w.Flush()
			return nil
		},
	}
}

func newProvidersTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test [provider-name]",
		Short: "Test provider connection",
		Long: `Test connection to a specific provider.

Examples:
  potus providers test anthropic
  potus providers test openai
  potus providers test ollama`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			providerName := args[0]

			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			authStore := auth.NewStore()
			var provider providers.Provider

			switch providerName {
			case "anthropic":
				apiKey := auth.ResolveAPIKey(authStore, "anthropic", cfg.Providers["anthropic"].APIKeyEnv)
				if apiKey == "" {
					return fmt.Errorf("API key not configured for anthropic (run 'potus auth login')")
				}
				provider, err = anthropic.New(apiKey)
				if err != nil {
					return fmt.Errorf("failed to create anthropic client: %w", err)
				}

			case "openai":
				apiKey := auth.ResolveAPIKey(authStore, "openai", cfg.Providers["openai"].APIKeyEnv)
				if apiKey == "" {
					return fmt.Errorf("API key not configured for openai (run 'potus auth login')")
				}
				provider, err = openai.New(apiKey, cfg.Providers["openai"].Organization)
				if err != nil {
					return fmt.Errorf("failed to create openai client: %w", err)
				}

			case "ollama":
				endpoint := cfg.Providers["ollama"].Endpoint
				if endpoint == "" {
					endpoint = "http://localhost:11434"
				}
				provider, err = ollama.New(endpoint)
				if err != nil {
					return fmt.Errorf("failed to create ollama client: %w", err)
				}

			default:
				return fmt.Errorf("unknown provider: %s (available: anthropic, openai, ollama)", providerName)
			}

			fmt.Printf("Testing connection to %s...\n", providerName)

			ctx, cancel := context.WithTimeout(cmd.Context(), providers.DefaultTimeout)
			defer cancel()

			models, err := provider.ListModels(ctx)
			if err != nil {
				return fmt.Errorf("connection failed: %w", err)
			}

			fmt.Printf("✓ Connection successful! Found %d models.\n", len(models))
			return nil
		},
	}
}

func newProvidersModelsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "models [provider-name]",
		Short: "List available models for provider",
		Long: `List all available models for a specific provider.

Examples:
  potus providers models anthropic
  potus providers models openai
  potus providers models ollama`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			providerName := args[0]

			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			authStore := auth.NewStore()
			var provider providers.Provider

			switch providerName {
			case "anthropic":
				apiKey := auth.ResolveAPIKey(authStore, "anthropic", cfg.Providers["anthropic"].APIKeyEnv)
				if apiKey == "" {
					return fmt.Errorf("API key not configured for anthropic (run 'potus auth login')")
				}
				provider, err = anthropic.New(apiKey)
				if err != nil {
					return fmt.Errorf("failed to create anthropic client: %w", err)
				}

			case "openai":
				apiKey := auth.ResolveAPIKey(authStore, "openai", cfg.Providers["openai"].APIKeyEnv)
				if apiKey == "" {
					return fmt.Errorf("API key not configured for openai (run 'potus auth login')")
				}
				provider, err = openai.New(apiKey, cfg.Providers["openai"].Organization)
				if err != nil {
					return fmt.Errorf("failed to create openai client: %w", err)
				}

			case "ollama":
				endpoint := cfg.Providers["ollama"].Endpoint
				if endpoint == "" {
					endpoint = "http://localhost:11434"
				}
				provider, err = ollama.New(endpoint)
				if err != nil {
					return fmt.Errorf("failed to create ollama client: %w", err)
				}

			default:
				return fmt.Errorf("unknown provider: %s (available: anthropic, openai, ollama)", providerName)
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), providers.DefaultTimeout)
			defer cancel()

			models, err := provider.ListModels(ctx)
			if err != nil {
				return fmt.Errorf("failed to list models: %w", err)
			}

			if len(models) == 0 {
				fmt.Println("No models available.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "MODEL ID\tNAME\tCONTEXT SIZE\tINPUT $/1M\tOUTPUT $/1M")
			fmt.Fprintln(w, "--------\t----\t------------\t----------\t-----------")

			for _, m := range models {
				contextSize := "-"
				if m.ContextSize > 0 {
					contextSize = fmt.Sprintf("%dk", m.ContextSize/1000)
				}
				inputPrice := "-"
				if m.Pricing.InputPer1M > 0 {
					inputPrice = fmt.Sprintf("$%.2f", m.Pricing.InputPer1M)
				}
				outputPrice := "-"
				if m.Pricing.OutputPer1M > 0 {
					outputPrice = fmt.Sprintf("$%.2f", m.Pricing.OutputPer1M)
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					m.ID, m.Name, contextSize, inputPrice, outputPrice)
			}

			w.Flush()
			return nil
		},
	}
}
