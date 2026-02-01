package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStore(t *testing.T) {
	dir := t.TempDir()
	store := NewStoreWithPath(filepath.Join(dir, "auth.json"))

	t.Run("set and get", func(t *testing.T) {
		if err := store.Set("anthropic", "sk-ant-test123"); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		key, err := store.Get("anthropic")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if key != "sk-ant-test123" {
			t.Errorf("Get() = %s, want sk-ant-test123", key)
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		key, err := store.Get("Anthropic")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if key != "sk-ant-test123" {
			t.Errorf("Get() = %s, want sk-ant-test123", key)
		}
	})

	t.Run("get missing provider", func(t *testing.T) {
		_, err := store.Get("nonexistent")
		if err == nil {
			t.Error("expected error for missing provider")
		}
	})

	t.Run("list", func(t *testing.T) {
		store.Set("openai", "sk-openai-test")

		entries, err := store.List()
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(entries) != 2 {
			t.Errorf("List() returned %d entries, want 2", len(entries))
		}
	})

	t.Run("delete", func(t *testing.T) {
		if err := store.Delete("openai"); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		_, err := store.Get("openai")
		if err == nil {
			t.Error("expected error after delete")
		}
	})

	t.Run("delete missing", func(t *testing.T) {
		err := store.Delete("nonexistent")
		if err == nil {
			t.Error("expected error for deleting missing provider")
		}
	})

	t.Run("file permissions", func(t *testing.T) {
		info, err := os.Stat(store.Path())
		if err != nil {
			t.Fatalf("Stat() error = %v", err)
		}
		perm := info.Mode().Perm()
		if perm != 0600 {
			t.Errorf("file permissions = %o, want 0600", perm)
		}
	})
}

func TestMaskKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"short", "****"},
		{"sk-ant-api03-longkey123", "sk-ant...****"},
	}

	for _, tt := range tests {
		got := MaskKey(tt.input)
		if got != tt.expected {
			t.Errorf("MaskKey(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestResolveAPIKey(t *testing.T) {
	dir := t.TempDir()
	store := NewStoreWithPath(filepath.Join(dir, "auth.json"))

	t.Run("from store", func(t *testing.T) {
		store.Set("anthropic", "stored-key")
		key := ResolveAPIKey(store, "anthropic", "NONEXISTENT_ENV_VAR")
		if key != "stored-key" {
			t.Errorf("ResolveAPIKey() = %s, want stored-key", key)
		}
	})

	t.Run("from env var", func(t *testing.T) {
		os.Setenv("TEST_POTUS_KEY", "env-key")
		defer os.Unsetenv("TEST_POTUS_KEY")

		key := ResolveAPIKey(store, "unknown-provider", "TEST_POTUS_KEY")
		if key != "env-key" {
			t.Errorf("ResolveAPIKey() = %s, want env-key", key)
		}
	})

	t.Run("store takes priority over env", func(t *testing.T) {
		os.Setenv("TEST_POTUS_KEY2", "env-key")
		defer os.Unsetenv("TEST_POTUS_KEY2")

		store.Set("priority-test", "store-key")
		key := ResolveAPIKey(store, "priority-test", "TEST_POTUS_KEY2")
		if key != "store-key" {
			t.Errorf("ResolveAPIKey() = %s, want store-key (store should take priority)", key)
		}
	})

	t.Run("returns empty when neither", func(t *testing.T) {
		key := ResolveAPIKey(store, "missing", "MISSING_ENV")
		if key != "" {
			t.Errorf("ResolveAPIKey() = %s, want empty", key)
		}
	})

	t.Run("nil store falls back to env", func(t *testing.T) {
		os.Setenv("TEST_POTUS_NIL", "nil-env-key")
		defer os.Unsetenv("TEST_POTUS_NIL")

		key := ResolveAPIKey(nil, "anything", "TEST_POTUS_NIL")
		if key != "nil-env-key" {
			t.Errorf("ResolveAPIKey() = %s, want nil-env-key", key)
		}
	})
}
