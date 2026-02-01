package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Providers["anthropic"].DefaultModel != "claude-sonnet-4-5-20250929" {
		t.Errorf("expected default anthropic model, got %s", cfg.Providers["anthropic"].DefaultModel)
	}

	if cfg.Permissions.FileRead != PermissionAllow {
		t.Errorf("expected file_read = allow, got %s", cfg.Permissions.FileRead)
	}

	if cfg.Context.MaxTokens != 100000 {
		t.Errorf("expected max_tokens = 100000, got %d", cfg.Context.MaxTokens)
	}
}

func TestLoad_CustomConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfgFile := filepath.Join(tmpDir, "config.yaml")

	configContent := `
providers:
  anthropic:
    default_model: claude-opus-4-5
    max_tokens: 4096

permissions:
  file_write: allow
  bash: deny

context:
  max_tokens: 50000
`
	if err := os.WriteFile(cfgFile, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Providers["anthropic"].DefaultModel != "claude-opus-4-5" {
		t.Errorf("expected claude-opus-4-5, got %s", cfg.Providers["anthropic"].DefaultModel)
	}

	if cfg.Providers["anthropic"].MaxTokens != 4096 {
		t.Errorf("expected max_tokens = 4096, got %d", cfg.Providers["anthropic"].MaxTokens)
	}

	if cfg.Permissions.FileWrite != PermissionAllow {
		t.Errorf("expected file_write = allow, got %s", cfg.Permissions.FileWrite)
	}

	if cfg.Permissions.Bash != PermissionDeny {
		t.Errorf("expected bash = deny, got %s", cfg.Permissions.Bash)
	}

	if cfg.Context.MaxTokens != 50000 {
		t.Errorf("expected max_tokens = 50000, got %d", cfg.Context.MaxTokens)
	}
}

func TestPermission_Values(t *testing.T) {
	tests := []struct {
		name  string
		perm  Permission
		valid bool
	}{
		{"ask", PermissionAsk, true},
		{"allow", PermissionAllow, true},
		{"deny", PermissionDeny, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.perm == "" && tt.valid {
				t.Error("expected non-empty permission")
			}
		})
	}
}
