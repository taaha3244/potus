package file

import (
	"path/filepath"
	"testing"
)

func TestResolvePath(t *testing.T) {
	tests := []struct {
		name     string
		workDir  string
		path     string
		expected string
	}{
		{
			name:     "relative path",
			workDir:  "/home/user/project",
			path:     "src/main.go",
			expected: "/home/user/project/src/main.go",
		},
		{
			name:     "absolute path",
			workDir:  "/home/user/project",
			path:     "/tmp/test.txt",
			expected: "/tmp/test.txt",
		},
		{
			name:     "path with dots",
			workDir:  "/home/user/project",
			path:     "./src/main.go",
			expected: "/home/user/project/src/main.go",
		},
		{
			name:     "empty workDir with relative path",
			workDir:  "",
			path:     "test.txt",
			expected: "test.txt",
		},
		{
			name:     "path with parent directory",
			workDir:  "/home/user/project/src",
			path:     "../config.yaml",
			expected: "/home/user/project/config.yaml",
		},
		{
			name:     "clean absolute path with redundant slashes",
			workDir:  "/work",
			path:     "/tmp//test.txt",
			expected: "/tmp/test.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolvePath(tt.workDir, tt.path)
			// Use filepath.Clean on expected for cross-platform compatibility
			expected := filepath.Clean(tt.expected)
			if got != expected {
				t.Errorf("resolvePath(%q, %q) = %q, want %q", tt.workDir, tt.path, got, expected)
			}
		})
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		shouldErr bool
	}{
		{
			name:      "simple path",
			path:      "src/main.go",
			shouldErr: false,
		},
		{
			name:      "absolute path",
			path:      "/home/user/file.txt",
			shouldErr: false,
		},
		{
			name:      "path with single dot",
			path:      "./test.txt",
			shouldErr: false,
		},
		{
			name:      "path traversal attempt",
			path:      "../../../etc/passwd",
			shouldErr: true,
		},
		{
			name:      "embedded path traversal",
			path:      "src/../../../etc/passwd",
			shouldErr: true,
		},
		{
			name:      "double dot in filename triggers error",
			path:      "file..txt",
			shouldErr: true, // sanitizePath uses strings.Contains for ".." which catches this
		},
		{
			name:      "multiple parent directories",
			path:      "../../file.txt",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sanitizePath(tt.path)
			if tt.shouldErr && err == nil {
				t.Errorf("sanitizePath(%q) should return error", tt.path)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("sanitizePath(%q) returned unexpected error: %v", tt.path, err)
			}
		})
	}
}

func TestErrPathTraversal(t *testing.T) {
	// ErrPathTraversal should be the same as filepath.ErrBadPattern
	if ErrPathTraversal != filepath.ErrBadPattern {
		t.Error("ErrPathTraversal should equal filepath.ErrBadPattern")
	}
}
