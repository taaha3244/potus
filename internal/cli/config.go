package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/taaha3244/potus/internal/config"
	"gopkg.in/yaml.v3"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  "View, edit, and manage POTUS configuration settings.",
	}

	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigEditCmd())
	cmd.AddCommand(newConfigSetCmd())
	cmd.AddCommand(newConfigGetCmd())
	cmd.AddCommand(newConfigPathCmd())

	return cmd
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Long:  "Display the current configuration including all defaults and overrides.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			data, err := yaml.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("failed to marshal config: %w", err)
			}

			fmt.Println(string(data))
			return nil
		},
	}
}

func newConfigEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Edit configuration in default editor",
		Long:  "Open the configuration file in your default editor ($EDITOR or vim).",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := getConfigPath()
			if err != nil {
				return err
			}

			// Ensure config directory exists
			configDir := filepath.Dir(configPath)
			if err := os.MkdirAll(configDir, 0755); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}

			// If config file doesn't exist, create it with defaults
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				cfg, err := config.Load("")
				if err != nil {
					return fmt.Errorf("failed to load defaults: %w", err)
				}
				data, err := yaml.Marshal(cfg)
				if err != nil {
					return fmt.Errorf("failed to marshal config: %w", err)
				}
				if err := os.WriteFile(configPath, data, 0644); err != nil {
					return fmt.Errorf("failed to write config: %w", err)
				}
			}

			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = os.Getenv("VISUAL")
			}
			if editor == "" {
				editor = "vim"
			}

			execCmd := exec.Command(editor, configPath)
			execCmd.Stdin = os.Stdin
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr

			return execCmd.Run()
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set a configuration value",
		Long: `Set a configuration value using dot notation.

Examples:
  potus config set agents.default.model anthropic/claude-opus-4
  potus config set ui.vim_mode true
  potus config set context.max_tokens 150000`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]

			configPath, err := getConfigPath()
			if err != nil {
				return err
			}

			// Ensure config directory exists
			configDir := filepath.Dir(configPath)
			if err := os.MkdirAll(configDir, 0755); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}

			// Load existing config file or create new one
			v := viper.New()
			v.SetConfigFile(configPath)
			v.SetConfigType("yaml")

			if _, err := os.Stat(configPath); err == nil {
				if err := v.ReadInConfig(); err != nil {
					return fmt.Errorf("failed to read config: %w", err)
				}
			}

			// Convert value to appropriate type
			var typedValue interface{}
			switch strings.ToLower(value) {
			case "true":
				typedValue = true
			case "false":
				typedValue = false
			default:
				// Try to parse as number
				var intVal int
				if _, err := fmt.Sscanf(value, "%d", &intVal); err == nil {
					typedValue = intVal
				} else {
					var floatVal float64
					if _, err := fmt.Sscanf(value, "%f", &floatVal); err == nil {
						typedValue = floatVal
					} else {
						typedValue = value
					}
				}
			}

			v.Set(key, typedValue)

			if err := v.WriteConfigAs(configPath); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}

			fmt.Printf("Set %s = %v\n", key, typedValue)
			return nil
		},
	}
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [key]",
		Short: "Get a configuration value",
		Long: `Get a configuration value using dot notation.

Examples:
  potus config get agents.default.model
  potus config get context.max_tokens
  potus config get permissions.bash`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			v := viper.New()

			if cfgFile != "" {
				v.SetConfigFile(cfgFile)
			} else {
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("could not get home directory: %w", err)
				}

				v.SetConfigName("config")
				v.SetConfigType("yaml")
				v.AddConfigPath(filepath.Join(home, ".config", "potus"))
				v.AddConfigPath(".potus")
				v.AddConfigPath(".")
			}

			v.SetEnvPrefix("POTUS")
			v.AutomaticEnv()

			// Set defaults (from config package)
			setConfigDefaults(v)

			if err := v.ReadInConfig(); err != nil {
				if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
					return fmt.Errorf("error reading config: %w", err)
				}
			}

			value := v.Get(key)
			if value == nil {
				return fmt.Errorf("key not found: %s", key)
			}

			// Format output based on type
			switch val := value.(type) {
			case map[string]interface{}:
				data, _ := yaml.Marshal(val)
				fmt.Print(string(data))
			case []interface{}:
				data, _ := yaml.Marshal(val)
				fmt.Print(string(data))
			default:
				fmt.Println(value)
			}

			return nil
		},
	}
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Show configuration file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := getConfigPath()
			if err != nil {
				return err
			}
			fmt.Println(configPath)

			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				fmt.Println("(file does not exist yet)")
			}
			return nil
		},
	}
}

func getConfigPath() (string, error) {
	if cfgFile != "" {
		return cfgFile, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get home directory: %w", err)
	}

	return filepath.Join(home, ".config", "potus", "config.yaml"), nil
}

// setConfigDefaults mirrors the defaults from config.Load
func setConfigDefaults(v *viper.Viper) {
	v.SetDefault("providers.anthropic.api_key_env", "ANTHROPIC_API_KEY")
	v.SetDefault("providers.anthropic.default_model", "claude-sonnet-4-5-20250929")
	v.SetDefault("providers.anthropic.max_tokens", 8192)

	v.SetDefault("providers.openai.api_key_env", "OPENAI_API_KEY")
	v.SetDefault("providers.openai.default_model", "gpt-5.2")
	v.SetDefault("providers.openai.max_tokens", 4096)

	v.SetDefault("providers.ollama.endpoint", "http://localhost:11434")
	v.SetDefault("providers.ollama.default_model", "qwen2.5-coder:32b")
	v.SetDefault("providers.ollama.max_tokens", 4096)

	v.SetDefault("agents.default.model", "anthropic/claude-sonnet-4-5")
	v.SetDefault("agents.default.max_tokens", 8192)
	v.SetDefault("agents.default.temperature", 0.7)

	v.SetDefault("permissions.file_read", "allow")
	v.SetDefault("permissions.file_write", "ask")
	v.SetDefault("permissions.file_edit", "ask")
	v.SetDefault("permissions.file_delete", "ask")
	v.SetDefault("permissions.bash", "ask")
	v.SetDefault("permissions.git", "ask")
	v.SetDefault("permissions.web_fetch", "allow")
	v.SetDefault("permissions.web_search", "allow")
	v.SetDefault("permissions.mcp", "ask")

	v.SetDefault("context.max_tokens", 100000)
	v.SetDefault("context.reserve_for_response", 8192)
	v.SetDefault("context.auto_compact", true)
	v.SetDefault("context.compact_threshold", 0.90)
	v.SetDefault("context.warn_threshold", 0.80)
	v.SetDefault("context.auto_prune", true)
	v.SetDefault("context.load_project_context", true)

	v.SetDefault("ui.theme", "default")
	v.SetDefault("ui.show_tokens", true)
	v.SetDefault("ui.show_cost", true)
	v.SetDefault("ui.vim_mode", false)
	v.SetDefault("ui.auto_scroll", true)
	v.SetDefault("ui.word_wrap", true)
}
