package unit_test

import (
	"clipcat/pkg/collector"
	"clipcat/pkg/exclude"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupDoublestarTestDirectory creates a test structure for doublestar patterns
func setupDoublestarTestDirectory(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "doublestar-test-")
	if err != nil {
		t.Fatal(err)
	}

	// Create comprehensive directory structure for doublestar testing
	dirs := []string{
		"src",
		"src/components",
		"src/components/ui",
		"src/utils",
		"test", 
		"test/unit",
		"test/integration",
		"docs",
		"docs/api",
		"build",
		"build/debug",
		"build/release",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create test files
	files := map[string]string{
		"README.md":                        "# Main readme",
		"main.go":                          "package main",
		"main_test.go":                     "package main_test",
		"src/app.go":                       "package src", 
		"src/app_test.go":                  "package src_test",
		"src/components/button.go":         "package components",
		"src/components/button_test.go":    "package components_test",
		"src/components/ui/modal.go":       "package ui",
		"src/components/ui/modal_test.go":  "package ui_test",
		"src/utils/helper.go":              "package utils",
		"src/utils/helper_test.go":         "package utils_test",
		"test/unit/basic_test.go":          "package unit",
		"test/integration/full_test.go":    "package integration",
		"docs/README.md":                   "# Docs readme",
		"docs/api/spec.md":                 "# API spec",
		"build/debug/app":                  "debug binary",
		"build/release/app":                "release binary",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	return tmpDir
}

func TestDoublestar_BasicRecursivePattern(t *testing.T) {
	tmpDir := setupDoublestarTestDirectory(t)
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	matcher, _ := exclude.BuildMatcher([]string{}, []string{}, false)

	tests := []struct {
		name          string
		pattern       string
		expectedFiles []string // Files that should be found (by basename)
	}{
		{
			name:    "All Go files recursively",
			pattern: "**/*.go",
			expectedFiles: []string{
				"main.go", "app.go", "button.go", "modal.go", "helper.go",
			},
		},
		{
			name:    "All test files recursively", 
			pattern: "**/*_test.go",
			expectedFiles: []string{
				"main_test.go", "app_test.go", "button_test.go", "modal_test.go", "helper_test.go",
				"basic_test.go", "full_test.go",
			},
		},
		{
			name:    "All markdown files recursively",
			pattern: "**/*.md",
			expectedFiles: []string{
				"README.md", "spec.md", // Note: there are two README.md files
			},
		},
		{
			name:    "Files in any components directory",
			pattern: "**/components/*.go", 
			expectedFiles: []string{
				"button.go", "button_test.go",
			},
		},
		{
			name:    "Files in deeply nested ui directory",
			pattern: "**/ui/*.go",
			expectedFiles: []string{
				"modal.go", "modal_test.go", 
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := collector.CollectFiles([]string{tt.pattern}, matcher, false)
			if err != nil {
				t.Fatalf("CollectFiles failed: %v", err)
			}

			// Check that all expected files are found
			for _, expectedFile := range tt.expectedFiles {
				found := false
				for _, file := range files {
					if strings.HasSuffix(file, expectedFile) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected file %q not found in results for pattern %q", expectedFile, tt.pattern)
				}
			}
		})
	}
}

func TestDoublestar_WithExclusions(t *testing.T) {
	tmpDir := setupDoublestarTestDirectory(t)
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	tests := []struct {
		name            string
		globPattern     string
		excludePattern  string
		shouldInclude   []string
		shouldExclude   []string
	}{
		{
			name:           "All Go files but exclude tests",
			globPattern:    "**/*.go",
			excludePattern: "**/*_test.go",
			shouldInclude:  []string{"main.go", "app.go", "button.go", "modal.go", "helper.go"},
			shouldExclude:  []string{"main_test.go", "app_test.go", "button_test.go", "modal_test.go", "helper_test.go"},
		},
		{
			name:           "All files but exclude build directory",
			globPattern:    "**/*",
			excludePattern: "build/",
			shouldInclude:  []string{"main.go", "README.md", "button.go"},
			shouldExclude:  []string{"app"}, // The build binaries
		},
		{
			name:           "Source files but exclude components",
			globPattern:    "src/**/*.go", 
			excludePattern: "**/components/**",
			shouldInclude:  []string{"app.go", "helper.go"},
			shouldExclude:  []string{"button.go", "modal.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := exclude.BuildMatcher([]string{}, []string{tt.excludePattern}, false)
			if err != nil {
				t.Fatalf("BuildMatcher failed: %v", err)
			}

			files, err := collector.CollectFiles([]string{tt.globPattern}, matcher, false)
			if err != nil {
				t.Fatalf("CollectFiles failed: %v", err)
			}

			// Verify included files are present
			for _, shouldInclude := range tt.shouldInclude {
				found := false
				for _, file := range files {
					if strings.HasSuffix(file, shouldInclude) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected file %q to be included but was not found", shouldInclude)
				}
			}

			// Verify excluded files are not present
			for _, shouldExclude := range tt.shouldExclude {
				for _, file := range files {
					if strings.HasSuffix(file, shouldExclude) {
						t.Errorf("Expected file %q to be excluded but was found: %s", shouldExclude, file)
					}
				}
			}
		})
	}
}

func TestDoublestar_ComplexPatterns(t *testing.T) {
	tmpDir := setupDoublestarTestDirectory(t)
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	matcher, _ := exclude.BuildMatcher([]string{}, []string{}, false)

	tests := []struct {
		name          string
		pattern       string
		shouldFind    int // Minimum number of files that should be found
		shouldContain []string // Files that must be included
	}{
		{
			name:          "Any test directory contents",
			pattern:       "**/test/**/*.go",
			shouldFind:    2,
			shouldContain: []string{"basic_test.go", "full_test.go"},
		},
		{
			name:          "Nested UI components",
			pattern:       "**/components/**/*.go",
			shouldFind:    4,
			shouldContain: []string{"button.go", "button_test.go", "modal.go", "modal_test.go"},
		},
		{
			name:          "Any directory named docs",
			pattern:       "**/docs/**",
			shouldFind:    2,
			shouldContain: []string{"README.md", "spec.md"},
		},
		{
			name:          "Root level markdown files",
			pattern:       "*.md",
			shouldFind:    1,
			shouldContain: []string{"README.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := collector.CollectFiles([]string{tt.pattern}, matcher, false)
			if err != nil {
				t.Fatalf("CollectFiles failed: %v", err)
			}

			if len(files) < tt.shouldFind {
				t.Errorf("Expected at least %d files, got %d", tt.shouldFind, len(files))
			}

			for _, shouldContain := range tt.shouldContain {
				found := false
				for _, file := range files {
					if strings.HasSuffix(file, shouldContain) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find file ending with %q in results", shouldContain)
				}
			}
		})
	}
}