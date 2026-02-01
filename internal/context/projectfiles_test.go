package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewProjectFiles(t *testing.T) {
	t.Run("with defaults", func(t *testing.T) {
		pf := NewProjectFiles(ProjectFilesConfig{})

		if pf == nil {
			t.Fatal("NewProjectFiles returned nil")
		}

		// Should have default file names
		if len(pf.contextFileNames) == 0 {
			t.Error("Should have default context file names")
		}
	})

	t.Run("with custom file names", func(t *testing.T) {
		pf := NewProjectFiles(ProjectFilesConfig{
			ContextFileNames: []string{"CUSTOM.md", "PROJECT.md"},
		})

		if len(pf.contextFileNames) != 2 {
			t.Errorf("Should have 2 context file names, got %d", len(pf.contextFileNames))
		}
	})

	t.Run("with max tokens", func(t *testing.T) {
		pf := NewProjectFiles(ProjectFilesConfig{
			MaxTokens: 5000,
		})

		if pf.maxTokens != 5000 {
			t.Errorf("MaxTokens = %d, want 5000", pf.maxTokens)
		}
	})
}

func TestProjectFiles_Load(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()

	// Create a POTUS.md file
	potusContent := "# Project Context\n\nThis is a test project."
	potusPath := filepath.Join(tmpDir, "POTUS.md")
	if err := os.WriteFile(potusPath, []byte(potusContent), 0644); err != nil {
		t.Fatal(err)
	}

	pf := NewProjectFiles(ProjectFilesConfig{})
	estimator := NewSimpleEstimator()

	ctx, err := pf.Load(tmpDir, estimator)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(ctx.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(ctx.Files))
	}

	if ctx.Files[0].Name != "POTUS.md" {
		t.Errorf("Expected POTUS.md, got %s", ctx.Files[0].Name)
	}

	if ctx.Files[0].Content != potusContent {
		t.Errorf("Content mismatch")
	}

	if ctx.TotalTokens <= 0 {
		t.Error("TotalTokens should be > 0")
	}
}

func TestProjectFiles_Load_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple context files
	files := map[string]string{
		"POTUS.md":  "# POTUS\nProject context",
		"CLAUDE.md": "# Claude\nMore context",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	pf := NewProjectFiles(ProjectFilesConfig{})
	estimator := NewSimpleEstimator()

	ctx, err := pf.Load(tmpDir, estimator)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(ctx.Files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(ctx.Files))
	}
}

func TestProjectFiles_Load_Hierarchy(t *testing.T) {
	// Create directory hierarchy
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subproject")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Parent has POTUS.md
	parentContent := "# Parent Project"
	if err := os.WriteFile(filepath.Join(tmpDir, "POTUS.md"), []byte(parentContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Subdir has CLAUDE.md (different file, should also be loaded)
	childContent := "# Child Project"
	if err := os.WriteFile(filepath.Join(subDir, "CLAUDE.md"), []byte(childContent), 0644); err != nil {
		t.Fatal(err)
	}

	pf := NewProjectFiles(ProjectFilesConfig{})
	estimator := NewSimpleEstimator()

	// Load from subdir
	ctx, err := pf.Load(subDir, estimator)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Should find both files
	if len(ctx.Files) != 2 {
		t.Errorf("Expected 2 files (from child and parent), got %d", len(ctx.Files))
	}

	// CLAUDE.md from child should be first (closer to workdir)
	foundClaude := false
	foundPotus := false
	for _, f := range ctx.Files {
		if f.Name == "CLAUDE.md" {
			foundClaude = true
		}
		if f.Name == "POTUS.md" {
			foundPotus = true
		}
	}

	if !foundClaude || !foundPotus {
		t.Error("Should find both CLAUDE.md and POTUS.md")
	}
}

func TestProjectFiles_Load_FirstFoundWins(t *testing.T) {
	// Create directory hierarchy with same file in both
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subproject")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Both have POTUS.md
	if err := os.WriteFile(filepath.Join(tmpDir, "POTUS.md"), []byte("Parent"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "POTUS.md"), []byte("Child"), 0644); err != nil {
		t.Fatal(err)
	}

	pf := NewProjectFiles(ProjectFilesConfig{})
	estimator := NewSimpleEstimator()

	ctx, err := pf.Load(subDir, estimator)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Should only have one POTUS.md (from child, first found)
	if len(ctx.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(ctx.Files))
	}

	if ctx.Files[0].Content != "Child" {
		t.Error("Should load child POTUS.md (first found wins)")
	}
}

func TestProjectFiles_Load_MaxTokens(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a large file
	largeContent := strings.Repeat("This is a test. ", 1000)
	if err := os.WriteFile(filepath.Join(tmpDir, "POTUS.md"), []byte(largeContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a small file
	smallContent := "Small file"
	if err := os.WriteFile(filepath.Join(tmpDir, "CLAUDE.md"), []byte(smallContent), 0644); err != nil {
		t.Fatal(err)
	}

	pf := NewProjectFiles(ProjectFilesConfig{
		MaxTokens: 100, // Very low limit
	})
	estimator := NewSimpleEstimator()

	ctx, err := pf.Load(tmpDir, estimator)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Should not exceed max tokens
	if ctx.TotalTokens > 100 {
		t.Errorf("TotalTokens %d exceeds max 100", ctx.TotalTokens)
	}
}

func TestProjectFiles_Load_NoFiles(t *testing.T) {
	tmpDir := t.TempDir()

	pf := NewProjectFiles(ProjectFilesConfig{})
	estimator := NewSimpleEstimator()

	ctx, err := pf.Load(tmpDir, estimator)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(ctx.Files) != 0 {
		t.Errorf("Expected 0 files, got %d", len(ctx.Files))
	}

	if ctx.TotalTokens != 0 {
		t.Errorf("TotalTokens should be 0, got %d", ctx.TotalTokens)
	}
}

func TestProjectFiles_FormatForSystemPrompt(t *testing.T) {
	pf := NewProjectFiles(ProjectFilesConfig{})

	t.Run("nil context", func(t *testing.T) {
		result := pf.FormatForSystemPrompt(nil)
		if result != "" {
			t.Error("Should return empty string for nil context")
		}
	})

	t.Run("empty files", func(t *testing.T) {
		ctx := &ProjectContext{Files: []ContextFile{}}
		result := pf.FormatForSystemPrompt(ctx)
		if result != "" {
			t.Error("Should return empty string for empty files")
		}
	})

	t.Run("with files", func(t *testing.T) {
		ctx := &ProjectContext{
			Files: []ContextFile{
				{
					Path:    "/project/POTUS.md",
					Name:    "POTUS.md",
					Content: "# Project\nTest content",
					Tokens:  10,
				},
			},
			TotalTokens: 10,
		}

		result := pf.FormatForSystemPrompt(ctx)

		if !strings.Contains(result, "## Project Context") {
			t.Error("Should contain Project Context header")
		}

		if !strings.Contains(result, "POTUS.md") {
			t.Error("Should contain file name")
		}

		if !strings.Contains(result, "# Project") {
			t.Error("Should contain file content")
		}

		if !strings.Contains(result, "Test content") {
			t.Error("Should contain file content")
		}
	})
}

func TestProjectFiles_GetLoadedFiles(t *testing.T) {
	pf := NewProjectFiles(ProjectFilesConfig{})

	t.Run("nil context", func(t *testing.T) {
		files := pf.GetLoadedFiles(nil)
		if files != nil {
			t.Error("Should return nil for nil context")
		}
	})

	t.Run("with files", func(t *testing.T) {
		ctx := &ProjectContext{
			Files: []ContextFile{
				{Path: "/a/POTUS.md"},
				{Path: "/b/CLAUDE.md"},
			},
		}

		files := pf.GetLoadedFiles(ctx)

		if len(files) != 2 {
			t.Errorf("Expected 2 files, got %d", len(files))
		}

		if files[0] != "/a/POTUS.md" {
			t.Errorf("First file = %s, want /a/POTUS.md", files[0])
		}
	})
}

func TestProjectFiles_FindContextFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a context file
	if err := os.WriteFile(filepath.Join(tmpDir, "POTUS.md"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	pf := NewProjectFiles(ProjectFilesConfig{})

	t.Run("file exists", func(t *testing.T) {
		path, err := pf.FindContextFile(tmpDir, "POTUS.md")
		if err != nil {
			t.Fatalf("FindContextFile failed: %v", err)
		}

		if !strings.HasSuffix(path, "POTUS.md") {
			t.Errorf("Path should end with POTUS.md, got %s", path)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := pf.FindContextFile(tmpDir, "NONEXISTENT.md")
		if err == nil {
			t.Error("Should return error for nonexistent file")
		}
	})
}

func TestProjectFiles_BuildSearchPaths(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "a", "b", "c")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	pf := NewProjectFiles(ProjectFilesConfig{})
	paths := pf.buildSearchPaths(subDir)

	// Should include subDir
	found := false
	for _, p := range paths {
		if p == subDir {
			found = true
			break
		}
	}
	if !found {
		t.Error("Should include the work directory itself")
	}

	// Should include parents
	parentFound := false
	for _, p := range paths {
		if p == tmpDir {
			parentFound = true
			break
		}
	}
	if !parentFound {
		t.Error("Should include parent directories")
	}

	// Should have multiple paths (work dir + parents + config dirs)
	if len(paths) < 3 {
		t.Errorf("Expected at least 3 paths, got %d", len(paths))
	}
}

func TestDefaultContextFileNames(t *testing.T) {
	// Verify expected defaults
	expectedNames := []string{"POTUS.md", "CLAUDE.md"}

	for _, expected := range expectedNames {
		found := false
		for _, name := range DefaultContextFileNames {
			if name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected %s in default context file names", expected)
		}
	}
}
