package file

import (
	"path/filepath"
	"strings"
)

func resolvePath(workDir, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(workDir, path)
}

func sanitizePath(path string) error {
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return ErrPathTraversal
	}
	return nil
}

var ErrPathTraversal = filepath.ErrBadPattern
