package integration_test

import (
	"clipcat/pkg/clipcat"
	"os"
	"path/filepath"
	"testing"
)

// setupComplexTestDirectory creates a more comprehensive test structure
func setupComplexTestDirectory(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "clipcat-scenarios-")
	if err != nil {
		t.Fatal(err)
	}

	// Create complex directory structure
	dirs := []string{
		"src", "src/components", "src/utils",
		"tests", "docs", "config",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create comprehensive file set
	files := map[string]string{
		"README.md":     "# Test Project",
		"package.json":  `{"name": "test-project"}`,
		".gitignore":    "*.log\nnode_modules/",
		"src/main.go":   "package main",
		"config/app.json": `{"debug": true}`,
		"config/database.yaml": "host: localhost",
		"config/prod.toml": "[server]\nport = 8080",
		"docs/api.md":   "# API Docs",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	return tmpDir
}

func TestScenario_ConfigurationFilesOnlyClean(t *testing.T) {
	// Example: clipcat "*.json" "*.yaml" "*.toml" -i
	tmpDir := setupComplexTestDirectory(t)
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"clipcat", "*.json", "*.yaml", "*.toml", "-i"}

	cfg := clipcat.ParseArgs()

	if !cfg.IgnoreCase {
		t.Error("Expected IgnoreCase to be true")
	}
	if len(cfg.Paths) != 3 {
		t.Errorf("Expected 3 glob patterns, got %d", len(cfg.Paths))
	}

	expectedPatterns := []string{"*.json", "*.yaml", "*.toml"}
	for i, pattern := range cfg.Paths {
		if pattern != expectedPatterns[i] {
			t.Errorf("Pattern[%d]: got %q, want %q", i, pattern, expectedPatterns[i])
		}
	}
}

func TestScenario_TestFilesAnalysisClean(t *testing.T) {
	// Example: clipcat "*test*" "*spec*" -i --only-tree
	tmpDir := setupComplexTestDirectory(t)
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"clipcat", "*test*", "*spec*", "-i", "--only-tree"}

	cfg := clipcat.ParseArgs()

	if !cfg.IgnoreCase {
		t.Error("Expected IgnoreCase to be true")
	}
	if !cfg.ShowTree {
		t.Error("Expected ShowTree to be true")
	}
	if !cfg.OnlyTree {
		t.Error("Expected OnlyTree to be true")
	}
	if len(cfg.Paths) != 2 {
		t.Errorf("Expected 2 patterns, got %d", len(cfg.Paths))
	}
}

func TestScenario_AllFlagsClean(t *testing.T) {
	// Example: clipcat . -t -p -i -e "*.log" -e "node_modules/" --exclude-from .gitignore --only-tree
	tmpDir := setupComplexTestDirectory(t)
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"clipcat", ".", "-t", "-p", "-i", "-e", "*.log", "-e", "node_modules/", "--exclude-from", ".gitignore", "--only-tree"}

	cfg := clipcat.ParseArgs()

	// Verify all flags are set
	if !cfg.ShowTree {
		t.Error("Expected ShowTree to be true")
	}
	if !cfg.OnlyTree {
		t.Error("Expected OnlyTree to be true")
	}
	if !cfg.PrintOut {
		t.Error("Expected PrintOut to be true")
	}
	if !cfg.IgnoreCase {
		t.Error("Expected IgnoreCase to be true")
	}

	// Verify exclude patterns
	expectedExcludes := []string{"*.log", "node_modules/"}
	if len(cfg.Excludes) != len(expectedExcludes) {
		t.Errorf("Expected %d excludes, got %d", len(expectedExcludes), len(cfg.Excludes))
	} else {
		for i, exclude := range cfg.Excludes {
			if exclude != expectedExcludes[i] {
				t.Errorf("Exclude[%d]: got %q, want %q", i, exclude, expectedExcludes[i])
			}
		}
	}

	// Verify exclude files
	if len(cfg.ExcludeFiles) != 1 || cfg.ExcludeFiles[0] != ".gitignore" {
		t.Errorf("Expected [.gitignore] in ExcludeFiles, got %v", cfg.ExcludeFiles)
	}
}