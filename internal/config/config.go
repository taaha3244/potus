package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Providers   map[string]ProviderConfig  `mapstructure:"providers"`
	Agents      map[string]AgentConfig     `mapstructure:"agents"`
	Permissions PermissionConfig           `mapstructure:"permissions"`
	MCPServers  map[string]MCPServerConfig `mapstructure:"mcp_servers"`
	LSP         map[string]LSPConfig       `mapstructure:"lsp"`
	Context     ContextConfig              `mapstructure:"context"`
	UI          UIConfig                   `mapstructure:"ui"`
	Safety      SafetyConfig               `mapstructure:"safety"`
	Network     NetworkConfig              `mapstructure:"network"`
}

type ProviderConfig struct {
	APIKeyEnv    string `mapstructure:"api_key_env"`
	DefaultModel string `mapstructure:"default_model"`
	MaxTokens    int    `mapstructure:"max_tokens"`
	Endpoint     string `mapstructure:"endpoint"`
	Organization string `mapstructure:"organization"`
	Region       string `mapstructure:"region"`
	Deployment   string `mapstructure:"deployment"`
	APIVersion   string `mapstructure:"api_version"`
}

type AgentConfig struct {
	Model        string   `mapstructure:"model"`
	MaxTokens    int      `mapstructure:"max_tokens"`
	Temperature  float64  `mapstructure:"temperature"`
	SystemPrompt string   `mapstructure:"system_prompt"`
	Tools        []string `mapstructure:"tools"`
}

type PermissionConfig struct {
	FileRead      Permission `mapstructure:"file_read"`
	FileWrite     Permission `mapstructure:"file_write"`
	FileEdit      Permission `mapstructure:"file_edit"`
	FileDelete    Permission `mapstructure:"file_delete"`
	Bash          Permission `mapstructure:"bash"`
	Git           Permission `mapstructure:"git"`
	WebFetch      Permission `mapstructure:"web_fetch"`
	WebSearch     Permission `mapstructure:"web_search"`
	MCP           Permission `mapstructure:"mcp"`
	BashAllowlist []string   `mapstructure:"bash_allowlist"`
	BashBlocklist []string   `mapstructure:"bash_blocklist"`
}

type Permission string

const (
	PermissionAsk   Permission = "ask"
	PermissionAllow Permission = "allow"
	PermissionDeny  Permission = "deny"
)

type MCPServerConfig struct {
	Command string            `mapstructure:"command"`
	Args    []string          `mapstructure:"args"`
	Env     map[string]string `mapstructure:"env"`
	URL     string            `mapstructure:"url"`
	Enabled bool              `mapstructure:"enabled"`
}

type LSPConfig struct {
	Command string   `mapstructure:"command"`
	Args    []string `mapstructure:"args"`
	Enabled bool     `mapstructure:"enabled"`
}

type ContextConfig struct {
	MaxTokens               int      `mapstructure:"max_tokens"`
	ReserveForResponse      int      `mapstructure:"reserve_for_response"`
	AutoCompact             bool     `mapstructure:"auto_compact"`
	CompactThreshold        float64  `mapstructure:"compact_threshold"`
	WarnThreshold           float64  `mapstructure:"warn_threshold"`
	ProtectedTokens         int      `mapstructure:"protected_tokens"`
	AutoPrune               bool     `mapstructure:"auto_prune"`
	ProtectedTools          []string `mapstructure:"protected_tools"`
	LoadProjectContext      bool     `mapstructure:"load_project_context"`
	ProjectContextFiles     []string `mapstructure:"project_context_files"`
	MaxProjectContextTokens int      `mapstructure:"max_project_context_tokens"`
	SemanticSearch          bool     `mapstructure:"semantic_search"`
	EmbeddingModel          string   `mapstructure:"embedding_model"`
	RepoMap                 bool     `mapstructure:"repo_map"`
	RepoMapTokens           int      `mapstructure:"repo_map_tokens"`
	IncludeGitChanges       bool     `mapstructure:"include_git_changes"`
}

type UIConfig struct {
	Theme          string `mapstructure:"theme"`
	ShowTokens     bool   `mapstructure:"show_tokens"`
	ShowCost       bool   `mapstructure:"show_cost"`
	HighlightStyle string `mapstructure:"highlight_style"`
	VimMode        bool   `mapstructure:"vim_mode"`
	AutoScroll     bool   `mapstructure:"auto_scroll"`
	WordWrap       bool   `mapstructure:"word_wrap"`
	MaxWidth       int    `mapstructure:"max_width"`
}

type SafetyConfig struct {
	GitCheckpoint    bool   `mapstructure:"git_checkpoint"`
	MaxUndo          int    `mapstructure:"max_undo"`
	SecretsDetection bool   `mapstructure:"secrets_detection"`
	SecretsAction    string `mapstructure:"secrets_action"`
	AuditLog         bool   `mapstructure:"audit_log"`
	AuditLogPath     string `mapstructure:"audit_log_path"`
}

type NetworkConfig struct {
	Timeout  string `mapstructure:"timeout"`
	Proxy    string `mapstructure:"proxy"`
	Insecure bool   `mapstructure:"insecure"`
}

func Load(cfgFile string) (*Config, error) {
	v := viper.New()

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("could not get home directory: %w", err)
		}

		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(filepath.Join(home, ".config", "potus"))
		v.AddConfigPath(".potus")
		v.AddConfigPath(".")
	}

	v.SetEnvPrefix("POTUS")
	v.AutomaticEnv()

	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
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
	v.SetDefault("context.protected_tokens", 20000)
	v.SetDefault("context.auto_prune", true)
	v.SetDefault("context.protected_tools", []string{"file_read", "read_file", "search_content", "grep", "glob"})
	v.SetDefault("context.load_project_context", true)
	v.SetDefault("context.project_context_files", []string{"POTUS.md", "CLAUDE.md", "AGENTS.md", "CONTEXT.md"})
	v.SetDefault("context.max_project_context_tokens", 10000)
	v.SetDefault("context.semantic_search", false)
	v.SetDefault("context.repo_map", true)
	v.SetDefault("context.repo_map_tokens", 4000)
	v.SetDefault("context.include_git_changes", true)

	v.SetDefault("ui.theme", "default")
	v.SetDefault("ui.show_tokens", true)
	v.SetDefault("ui.show_cost", true)
	v.SetDefault("ui.highlight_style", "monokai")
	v.SetDefault("ui.vim_mode", false)
	v.SetDefault("ui.auto_scroll", true)
	v.SetDefault("ui.word_wrap", true)
	v.SetDefault("ui.max_width", 120)

	v.SetDefault("safety.git_checkpoint", true)
	v.SetDefault("safety.max_undo", 50)
	v.SetDefault("safety.secrets_detection", true)
	v.SetDefault("safety.secrets_action", "warn")
	v.SetDefault("safety.audit_log", true)

	if home, err := os.UserHomeDir(); err == nil {
		v.SetDefault("safety.audit_log_path", filepath.Join(home, ".local", "share", "potus", "audit.log"))
	}

	v.SetDefault("network.timeout", "60s")
}
