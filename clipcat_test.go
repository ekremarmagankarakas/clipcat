package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Unit Tests

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

func TestExcludeMatcherShouldExclude_GlobPatterns(t *testing.T) {
	matcher := &ExcludeMatcher{
		globPatterns: []string{"*.log", "temp/", "__pycache__/"}, // note trailing slash for dir
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"matches log extension", "debug.log", true},
		{"matches log in subdir", "src/debug.log", true},
		{"doesn't match different extension", "debug.txt", false},
		{"matches temp directory", "temp/file.txt", true},
		{"matches pycache anywhere", "src/__pycache__/file.pyc", true},
		{"doesn't match partial", "src/temporary/file.txt", false},
		{"matches at root", "__pycache__/", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.ShouldExclude(tt.path)
			if result != tt.expected {
				t.Errorf("ShouldExclude(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestReadPatternsFromFile(t *testing.T) {
	// Create a temporary file with patterns
	tmpfile, err := os.CreateTemp("", "patterns-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := `# This is a comment
*.log
node_modules/

# Another comment
*.tmp
`
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	patterns, err := readPatternsFromFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("readPatternsFromFile failed: %v", err)
	}

	// Should include everything including comments (gitignore library handles them)
	expected := []string{
		"# This is a comment",
		"*.log",
		"node_modules/",
		"",
		"# Another comment",
		"*.tmp",
	}

	if len(patterns) != len(expected) {
		t.Errorf("got %d patterns, want %d", len(patterns), len(expected))
	}

	for i, pattern := range patterns {
		if i < len(expected) && pattern != expected[i] {
			t.Errorf("pattern[%d] = %q, want %q", i, pattern, expected[i])
		}
	}
}

func TestReadPatternsFromFile_NonExistent(t *testing.T) {
	_, err := readPatternsFromFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
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

func TestBuildExcludeMatcher_EmptyPatterns(t *testing.T) {
	matcher, err := buildExcludeMatcher([]string{}, []string{}, false)
	if err != nil {
		t.Fatalf("buildExcludeMatcher failed: %v", err)
	}

	if matcher == nil {
		t.Error("expected non-nil matcher")
	}

	if matcher.gitignoreMatcher != nil {
		t.Error("expected nil gitignoreMatcher for empty patterns")
	}
}

func TestBuildExcludeMatcher_WithGlobPatterns(t *testing.T) {
	patterns := []string{"*.log", "*.tmp"}
	matcher, err := buildExcludeMatcher([]string{}, patterns, false)
	if err != nil {
		t.Fatalf("buildExcludeMatcher failed: %v", err)
	}

	if len(matcher.globPatterns) != 2 {
		t.Errorf("expected 2 glob patterns, got %d", len(matcher.globPatterns))
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

	matcher, err := buildExcludeMatcher([]string{tmpfile.Name()}, []string{}, false)
	if err != nil {
		t.Fatalf("buildExcludeMatcher failed: %v", err)
	}

	if matcher.gitignoreMatcher == nil {
		t.Error("expected non-nil gitignoreMatcher")
	}

	// Test that it excludes correctly
	if !matcher.ShouldExclude("test.log") {
		t.Error("expected test.log to be excluded")
	}
}

func TestBuildExcludeMatcher_NonExistentFile(t *testing.T) {
	_, err := buildExcludeMatcher([]string{"/nonexistent/file.txt"}, []string{}, false)
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

	matcher, err := buildExcludeMatcher([]string{tmpfile.Name()}, []string{}, false)
	if err != nil {
		t.Fatalf("buildExcludeMatcher failed: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"excludes log files", "debug.log", true},
		{"excludes node_modules", "node_modules/package.json", true},
		{"excludes root txt", "root.txt", true},
		{"doesn't exclude subdir txt", "src/root.txt", false},
		{"includes negated important.log", "important.log", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.ShouldExclude(tt.path)
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
	matcher, err := buildExcludeMatcher([]string{tmpfile.Name()}, []string{"*.tmp", "build/"}, false)
	if err != nil {
		t.Fatalf("buildExcludeMatcher failed: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"excludes log (gitignore)", "debug.log", true},
		{"excludes tmp (glob)", "test.tmp", true},
		{"excludes build (glob)", "build/output.txt", true},
		{"includes regular file", "main.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.ShouldExclude(tt.path)
			if result != tt.expected {
				t.Errorf("ShouldExclude(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

