package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/taaha3244/potus/internal/auth"
	"github.com/taaha3244/potus/internal/config"
	"golang.org/x/term"
)

var knownProviders = []struct {
	Name   string
	EnvVar string
}{
	{"anthropic", "ANTHROPIC_API_KEY"},
	{"openai", "OPENAI_API_KEY"},
}

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage API key authentication",
		Long:  "Login, list, and logout provider API keys stored locally.",
	}

	cmd.AddCommand(newAuthLoginCmd())
	cmd.AddCommand(newAuthListCmd())
	cmd.AddCommand(newAuthLogoutCmd())

	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login [provider]",
		Short: "Store an API key for a provider",
		Long: `Interactively store an API key for a provider.

Examples:
  potus auth login
  potus auth login anthropic
  potus auth login openai`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := auth.NewStore()
			reader := bufio.NewReader(os.Stdin)

			var provider string
			if len(args) > 0 {
				provider = strings.ToLower(args[0])
			} else {
				fmt.Println("Select a provider:")
				for i, p := range knownProviders {
					fmt.Printf("  %d. %s\n", i+1, p.Name)
				}
				fmt.Printf("  %d. Other\n", len(knownProviders)+1)
				fmt.Print("\n> ")

				input, _ := reader.ReadString('\n')
				input = strings.TrimSpace(input)

				idx := 0
				fmt.Sscanf(input, "%d", &idx)

				if idx >= 1 && idx <= len(knownProviders) {
					provider = knownProviders[idx-1].Name
				} else if idx == len(knownProviders)+1 {
					fmt.Print("Enter provider name: ")
					provider, _ = reader.ReadString('\n')
					provider = strings.TrimSpace(strings.ToLower(provider))
				} else {
					return fmt.Errorf("invalid selection")
				}
			}

			if provider == "" {
				return fmt.Errorf("provider name is required")
			}

			fmt.Printf("Enter API key for %s: ", provider)
			keyBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("failed to read API key: %w", err)
			}

			key := strings.TrimSpace(string(keyBytes))
			if key == "" {
				return fmt.Errorf("API key cannot be empty")
			}

			if err := store.Set(provider, key); err != nil {
				return fmt.Errorf("failed to save API key: %w", err)
			}

			fmt.Printf("✓ API key saved for %s\n", provider)
			return nil
		},
	}
}

func newAuthListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List stored API keys and their sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			store := auth.NewStore()

			cfg, err := config.Load(cfgFile)
			if err != nil {
				cfg = &config.Config{
					Providers: make(map[string]config.ProviderConfig),
				}
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "PROVIDER\tSOURCE\tSTATUS")
			fmt.Fprintln(w, "--------\t------\t------")

			seen := make(map[string]bool)

			for _, p := range knownProviders {
				envVar := p.EnvVar
				if pc, ok := cfg.Providers[p.Name]; ok && pc.APIKeyEnv != "" {
					envVar = pc.APIKeyEnv
				}

				_, source := auth.ResolveAPIKeySource(store, p.Name, envVar)
				status := string(source)
				if source == auth.SourceAuthStore {
					if key, err := store.Get(p.Name); err == nil {
						status = auth.MaskKey(key)
					}
				} else if source == auth.SourceEnvVar {
					status = "configured"
				}

				fmt.Fprintf(w, "%s\t%s\t%s\n", p.Name, source, status)
				seen[p.Name] = true
			}

			// Show ollama
			endpoint := "http://localhost:11434"
			if pc, ok := cfg.Providers["ollama"]; ok && pc.Endpoint != "" {
				endpoint = pc.Endpoint
			}
			fmt.Fprintf(w, "ollama\tendpoint\t%s\n", endpoint)

			// Show any extra providers from auth store
			if entries, err := store.List(); err == nil {
				for name := range entries {
					if !seen[name] {
						key, _ := store.Get(name)
						fmt.Fprintf(w, "%s\t%s\t%s\n", name, auth.SourceAuthStore, auth.MaskKey(key))
					}
				}
			}

			w.Flush()
			return nil
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout <provider>",
		Short: "Remove a stored API key",
		Long: `Remove a stored API key for a provider.

Examples:
  potus auth logout anthropic
  potus auth logout openai`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := auth.NewStore()
			provider := strings.ToLower(args[0])

			if err := store.Delete(provider); err != nil {
				return fmt.Errorf("failed to remove key: %w", err)
			}

			fmt.Printf("✓ API key removed for %s\n", provider)
			return nil
		},
	}
}
