package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const authFileName = "auth.json"

type Store struct {
	path string
}

type AuthEntry struct {
	Type string `json:"type"`
	Key  string `json:"key"`
}

func NewStore() *Store {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return &Store{
		path: filepath.Join(home, ".config", "potus", authFileName),
	}
}

func NewStoreWithPath(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Get(provider string) (string, error) {
	entries, err := s.load()
	if err != nil {
		return "", err
	}

	entry, ok := entries[strings.ToLower(provider)]
	if !ok {
		return "", fmt.Errorf("no key stored for provider: %s", provider)
	}

	return entry.Key, nil
}

func (s *Store) Set(provider, key string) error {
	entries, err := s.load()
	if err != nil {
		entries = make(map[string]AuthEntry)
	}

	entries[strings.ToLower(provider)] = AuthEntry{
		Type: "api",
		Key:  key,
	}

	return s.save(entries)
}

func (s *Store) Delete(provider string) error {
	entries, err := s.load()
	if err != nil {
		return err
	}

	name := strings.ToLower(provider)
	if _, ok := entries[name]; !ok {
		return fmt.Errorf("no key stored for provider: %s", provider)
	}

	delete(entries, name)
	return s.save(entries)
}

func (s *Store) List() (map[string]AuthEntry, error) {
	return s.load()
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) load() (map[string]AuthEntry, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]AuthEntry), nil
		}
		return nil, fmt.Errorf("failed to read auth file: %w", err)
	}

	var entries map[string]AuthEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse auth file: %w", err)
	}

	return entries, nil
}

func (s *Store) save(entries map[string]AuthEntry) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create auth directory: %w", err)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal auth data: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0600); err != nil {
		return fmt.Errorf("failed to write auth file: %w", err)
	}

	return nil
}

func MaskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:6] + "..." + strings.Repeat("*", 4)
}
