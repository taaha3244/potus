package permissions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Settings struct {
	path        string
	Permissions map[string]string `json:"permissions"`
}

func LoadSettings(workDir string) *Settings {
	path := filepath.Join(workDir, ".potus", "settings.json")
	s := &Settings{
		path:        path,
		Permissions: make(map[string]string),
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}

	json.Unmarshal(data, s)
	if s.Permissions == nil {
		s.Permissions = make(map[string]string)
	}

	return s
}

func (s *Settings) Save() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0644)
}

func (s *Settings) IsAllowed(tool string) bool {
	return s.Permissions[strings.ToLower(tool)] == "allow"
}

func (s *Settings) SetAllow(tool string) {
	s.Permissions[strings.ToLower(tool)] = "allow"
}

func (s *Settings) Path() string {
	return s.path
}
