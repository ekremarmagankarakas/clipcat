package exclude_test

import (
	"clipcat/pkg/exclude"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsGlobPattern(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"asterisk", "*test*", true},
		{"question mark", "file?.txt", true},
		{"brackets", "file[0-9].txt", true},
		{"no glob", "regular.txt", false},
		{"path without glob", "src/main.go", false},
		{"empty string", "", false},
		{"multiple globs", "**/*.go", true}, // contains '*'
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGlobPattern(tt.path)
			if result != tt.expected {
				t.Errorf("isGlobPattern(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func isGlobPattern(path string) bool {
	return strings.ContainsAny(path, "*?[")
}

func TestExcludeMatcherShouldExclude_GlobPatterns(t *testing.T) {
	matcher, _ := exclude.BuildMatcher([]string{}, []string{"*.log", "temp/", "__pycache__/"}, false)

	tests := []struct {
		name     string
		path     string
		isDir    bool
		expected bool
	}{
		{"matches log extension", "debug.log", false, true},
		{"matches log in subdir", "src/debug.log", false, true},
		{"doesn't match different extension", "debug.txt", false, false},
		{"matches temp directory", "temp/file.txt", false, true},
		{"matches pycache anywhere", "src/__pycache__/file.pyc", false, true},
		{"doesn't match partial", "src/temporary/file.txt", false, false},
		{"matches at root", "__pycache__", true, true}, // directory itself
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.ShouldExclude(tt.path, tt.isDir)
			if result != tt.expected {
				t.Errorf("ShouldExclude(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestBuildExcludeMatcher_EmptyPatterns(t *testing.T) {
	matcher, err := exclude.BuildMatcher([]string{}, []string{}, false)
	if err != nil {
		t.Fatalf("BuildMatcher failed: %v", err)
	}

	if matcher == nil {
		t.Error("expected non-nil matcher")
	}
}

func TestBuildExcludeMatcher_WithGlobPatterns(t *testing.T) {
	patterns := []string{"*.log", "*.tmp"}
	matcher, err := exclude.BuildMatcher([]string{}, patterns, false)
	if err != nil {
		t.Fatalf("BuildMatcher failed: %v", err)
	}

	if matcher == nil {
		t.Error("expected non-nil matcher")
	}
}

func TestBuildExcludeMatcher_WithFile(t *testing.T) {
	// Create a temporary exclude file
	tmpfile, err := os.CreateTemp("", "exclude-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := "*.log\nnode_modules/\n"
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	matcher, err := exclude.BuildMatcher([]string{tmpfile.Name()}, []string{}, false)
	if err != nil {
		t.Fatalf("BuildMatcher failed: %v", err)
	}

	// Test that it excludes correctly
	if !matcher.ShouldExclude("test.log", false) {
		t.Error("expected test.log to be excluded")
	}
}

func TestBuildExcludeMatcher_NonExistentFile(t *testing.T) {
	_, err := exclude.BuildMatcher([]string{"/nonexistent/file.txt"}, []string{}, false)
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestExcludeMatcherShouldExclude_GitignorePatterns(t *testing.T) {
	// Create a temporary gitignore file
	tmpfile, err := os.CreateTemp("", "gitignore-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := `# Test gitignore
*.log
node_modules/
/root.txt
!important.log
`
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	matcher, err := exclude.BuildMatcher([]string{tmpfile.Name()}, []string{}, false)
	if err != nil {
		t.Fatalf("BuildMatcher failed: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		isDir    bool
		expected bool
	}{
		{"excludes log files", "debug.log", false, true},
		{"excludes node_modules", "node_modules/package.json", false, true},
		{"excludes root txt", "root.txt", false, true},
		{"doesn't exclude subdir txt", "src/root.txt", false, false},
		{"includes negated important.log", "important.log", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.ShouldExclude(tt.path, tt.isDir)
			if result != tt.expected {
				t.Errorf("ShouldExclude(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestExcludeMatcherShouldExclude_MixedPatterns(t *testing.T) {
	// Create a temporary gitignore file
	tmpfile, err := os.CreateTemp("", "gitignore-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := "*.log\n"
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Mix gitignore patterns and glob patterns
	matcher, err := exclude.BuildMatcher([]string{tmpfile.Name()}, []string{"*.tmp", "build/"}, false)
	if err != nil {
		t.Fatalf("BuildMatcher failed: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		isDir    bool
		expected bool
	}{
		{"excludes log (gitignore)", "debug.log", false, true},
		{"excludes tmp (glob)", "test.tmp", false, true},
		{"excludes build (glob)", "build/output.txt", false, true},
		{"includes regular file", "main.go", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.ShouldExclude(tt.path, tt.isDir)
			if result != tt.expected {
				t.Errorf("ShouldExclude(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestGetRelativePath(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "clipcat-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(srcDir, "main.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	absTestFile, _ := filepath.Abs(testFile)
	roots := []string{srcDir}

	result := getRelativePath(absTestFile, roots)

	if !strings.HasSuffix(result, ":main.go") {
		t.Errorf("getRelativePath result %q doesn't end with :main.go", result)
	}
}

func getRelativePath(file string, roots []string) string {
	// Find the best matching root
	var bestRoot string
	var bestLabel string

	for _, root := range roots {
		if isGlobPattern(root) {
			continue
		}

		absRoot, _ := filepath.Abs(root)
		if strings.HasPrefix(file, absRoot) && len(absRoot) > len(bestRoot) {
			bestRoot = absRoot
			bestLabel = root
		}
	}

	if bestRoot != "" {
		rel, _ := filepath.Rel(bestRoot, file)
		return bestLabel + ":" + rel
	}

	rel, _ := filepath.Rel(".", file)
	return ".:" + rel
}

func TestGetRelativePath_WithGlobInRoots(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clipcat-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	absTestFile, _ := filepath.Abs(testFile)

	// Glob patterns should be skipped in root matching
	roots := []string{"*test*", tmpDir}

	result := getRelativePath(absTestFile, roots)

	// Should match tmpDir, not the glob pattern
	if !strings.Contains(result, ":test.go") {
		t.Errorf("getRelativePath result %q doesn't contain :test.go", result)
	}
}