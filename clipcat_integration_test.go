package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Integration Tests

// setupTestDirectory creates a test directory structure
func setupTestDirectory(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "clipcat-integration-")
	if err != nil {
		t.Fatal(err)
	}

	// Create directory structure
	dirs := []string{
		"src",
		"src/components",
		"src/utils",
		"tests",
		"build",
		"node_modules",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create files
	files := map[string]string{
		"README.md":                 "# Test Project",
		"main.go":                   "package main\n\nfunc main() {}",
		"main_test.go":              "package main\n\nfunc TestMain(t *testing.T) {}",
		"src/app.go":                "package src",
		"src/components/button.go":  "package components",
		"src/utils/format.go":       "package utils",
		"tests/integration_test.go": "package tests",
		"build/output.txt":          "build output",
		"node_modules/pkg.json":     "{}",
		"debug.log":                 "debug info",
		"error.log":                 "error info",
		".gitignore":                "*.log\nnode_modules/\nbuild/",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	return tmpDir
}

func TestCollectFiles_SingleFile(t *testing.T) {
	tmpDir := setupTestDirectory(t)
	defer os.RemoveAll(tmpDir)

	matcher := &ExcludeMatcher{}
	
	files, err := collectFiles([]string{filepath.Join(tmpDir, "README.md")}, matcher)
	if err != nil {
		t.Fatalf("collectFiles failed: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d", len(files))
	}

	if len(files) > 0 && !strings.HasSuffix(files[0], "README.md") {
		t.Errorf("expected README.md, got %s", files[0])
	}
}

func TestCollectFiles_Directory(t *testing.T) {
	tmpDir := setupTestDirectory(t)
	defer os.RemoveAll(tmpDir)

	matcher := &ExcludeMatcher{}
	srcDir := filepath.Join(tmpDir, "src")
	
	files, err := collectFiles([]string{srcDir}, matcher)
	if err != nil {
		t.Fatalf("collectFiles failed: %v", err)
	}

	// Should find app.go, button.go, format.go
	if len(files) != 3 {
		t.Errorf("expected 3 files in src/, got %d", len(files))
	}

	// Check that all files are from src/
	for _, f := range files {
		if !strings.Contains(f, "/src/") {
			t.Errorf("file %s is not from src/ directory", f)
		}
	}
}

func TestCollectFiles_DirectoryWithExclusions(t *testing.T) {
	tmpDir := setupTestDirectory(t)
	defer os.RemoveAll(tmpDir)

	// Create matcher that excludes *.log and node_modules/
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	matcher, err := buildExcludeMatcher([]string{gitignorePath}, []string{})
	if err != nil {
		t.Fatalf("buildExcludeMatcher failed: %v", err)
	}

	// Change to tmpDir for relative path matching
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	files, err := collectFiles([]string{"."}, matcher)
	if err != nil {
		t.Fatalf("collectFiles failed: %v", err)
	}

	// Should exclude *.log, node_modules/, and build/
	for _, f := range files {
		if strings.Contains(f, ".log") {
			t.Errorf("found excluded .log file: %s", f)
		}
		if strings.Contains(f, "node_modules") {
			t.Errorf("found excluded node_modules file: %s", f)
		}
		if strings.Contains(f, "build") {
			t.Errorf("found excluded build file: %s", f)
		}
	}

	// Should include README.md, main.go, src files, tests
	foundReadme := false
	foundMain := false
	for _, f := range files {
		if strings.HasSuffix(f, "README.md") {
			foundReadme = true
		}
		if strings.HasSuffix(f, "main.go") {
			foundMain = true
		}
	}

	if !foundReadme {
		t.Error("README.md should not be excluded")
	}
	if !foundMain {
		t.Error("main.go should not be excluded")
	}
}

func TestCollectFiles_GlobPattern(t *testing.T) {
	tmpDir := setupTestDirectory(t)
	defer os.RemoveAll(tmpDir)

	// Change to tmpDir for glob matching
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	matcher := &ExcludeMatcher{}
	
	files, err := collectFiles([]string{"*test*"}, matcher)
	if err != nil {
		t.Fatalf("collectFiles failed: %v", err)
	}

	// Should find main_test.go and integration_test.go
	if len(files) < 2 {
		t.Errorf("expected at least 2 files matching *test*, got %d", len(files))
	}

	// All files should have "test" in their basename
	for _, f := range files {
		basename := filepath.Base(f)
		if !strings.Contains(strings.ToLower(basename), "test") {
			t.Errorf("file %s doesn't match *test* pattern", basename)
		}
	}
}

func TestCollectFiles_MultipleInputs(t *testing.T) {
	tmpDir := setupTestDirectory(t)
	defer os.RemoveAll(tmpDir)

	matcher := &ExcludeMatcher{}
	
	inputs := []string{
		filepath.Join(tmpDir, "README.md"),
		filepath.Join(tmpDir, "src"),
	}
	
	files, err := collectFiles(inputs, matcher)
	if err != nil {
		t.Fatalf("collectFiles failed: %v", err)
	}

	// Should find README.md + 3 files in src/
	if len(files) != 4 {
		t.Errorf("expected 4 files, got %d", len(files))
	}
}

func TestCollectFiles_Deduplication(t *testing.T) {
	tmpDir := setupTestDirectory(t)
	defer os.RemoveAll(tmpDir)

	matcher := &ExcludeMatcher{}
	
	readmePath := filepath.Join(tmpDir, "README.md")
	
	// Add the same file twice
	inputs := []string{readmePath, readmePath}
	
	files, err := collectFiles(inputs, matcher)
	if err != nil {
		t.Fatalf("collectFiles failed: %v", err)
	}

	// Should only include the file once
	if len(files) != 1 {
		t.Errorf("expected 1 file (deduplicated), got %d", len(files))
	}
}

func TestWriteTree_SingleRoot(t *testing.T) {
	tmpDir := setupTestDirectory(t)
	defer os.RemoveAll(tmpDir)

	files := []string{
		filepath.Join(tmpDir, "src", "app.go"),
		filepath.Join(tmpDir, "src", "components", "button.go"),
		filepath.Join(tmpDir, "src", "utils", "format.go"),
	}

	var output bytes.Buffer
	writeTree(&output, []string{filepath.Join(tmpDir, "src")}, files)

	result := output.String()

	// Should show the tree structure
	if !strings.Contains(result, "src/") {
		t.Error("tree should contain root directory label")
	}
	if !strings.Contains(result, "app.go") {
		t.Error("tree should contain app.go")
	}
	if !strings.Contains(result, "button.go") {
		t.Error("tree should contain button.go")
	}
	if !strings.Contains(result, "components/") {
		t.Error("tree should contain components directory")
	}

	// Should use dash-depth indicators
	if !strings.Contains(result, "-") {
		t.Error("tree should use dash indicators")
	}
}

func TestWriteTree_MultipleRoots(t *testing.T) {
	tmpDir := setupTestDirectory(t)
	defer os.RemoveAll(tmpDir)

	files := []string{
		filepath.Join(tmpDir, "src", "app.go"),
		filepath.Join(tmpDir, "tests", "integration_test.go"),
	}

	roots := []string{
		filepath.Join(tmpDir, "src"),
		filepath.Join(tmpDir, "tests"),
	}

	var output bytes.Buffer
	writeTree(&output, roots, files)

	result := output.String()

	// Should show both root sections
	if !strings.Contains(result, "src/") {
		t.Error("tree should contain src root")
	}
	if !strings.Contains(result, "tests/") {
		t.Error("tree should contain tests root")
	}

	// Should have blank line between sections
	lines := strings.Split(result, "\n")
	hasBlankLine := false
	for _, line := range lines {
		if line == "" {
			hasBlankLine = true
			break
		}
	}
	if !hasBlankLine {
		t.Error("tree should have blank line between root sections")
	}
}

func TestWriteHeader(t *testing.T) {
	var output bytes.Buffer
	writeHeader(&output, "/path/to/file.go")

	result := output.String()

	// Should have three lines: bar, path, bar
	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}

	// First and last lines should be equal bars
	if lines[0] != lines[2] {
		t.Error("header bars should be equal")
	}

	// Middle line should be the path
	if lines[1] != "/path/to/file.go" {
		t.Errorf("expected path in middle line, got %s", lines[1])
	}

	// Bars should be made of equals signs
	if !strings.Contains(lines[0], "=") {
		t.Error("bars should be made of = characters")
	}

	// Bar length should match path length
	if len(lines[0]) != len(lines[1]) {
		t.Errorf("bar length %d doesn't match path length %d", len(lines[0]), len(lines[1]))
	}
}

func TestWriteFileContent(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "content-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := "Hello, World!\nThis is a test."
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if err := writeFileContent(&output, tmpfile.Name()); err != nil {
		t.Fatalf("writeFileContent failed: %v", err)
	}

	result := output.String()
	if result != content {
		t.Errorf("expected %q, got %q", content, result)
	}
}

func TestWriteFileContent_Unreadable(t *testing.T) {
	var output bytes.Buffer
	err := writeFileContent(&output, "/nonexistent/file.txt")
	
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestEndToEnd_BasicUsage(t *testing.T) {
	tmpDir := setupTestDirectory(t)
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Collect files from src/
	matcher := &ExcludeMatcher{}
	files, err := collectFiles([]string{"src"}, matcher)
	if err != nil {
		t.Fatalf("collectFiles failed: %v", err)
	}

	// Build output with headers
	var output bytes.Buffer
	for _, file := range files {
		writeHeader(&output, file)
		if err := writeFileContent(&output, file); err != nil {
			output.WriteString("[unreadable]\n")
		}
		output.WriteString("\n")
	}

	result := output.String()

	// Should contain headers
	if !strings.Contains(result, "===") {
		t.Error("output should contain header bars")
	}

	// Should contain file paths
	if !strings.Contains(result, "app.go") {
		t.Error("output should contain file paths")
	}

	// Should contain file contents
	if !strings.Contains(result, "package") {
		t.Error("output should contain file contents")
	}
}

func TestEndToEnd_WithTreeAndExclusions(t *testing.T) {
	tmpDir := setupTestDirectory(t)
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Build exclude matcher
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	matcher, err := buildExcludeMatcher([]string{gitignorePath}, []string{})
	if err != nil {
		t.Fatalf("buildExcludeMatcher failed: %v", err)
	}

	// Collect files
	files, err := collectFiles([]string{"."}, matcher)
	if err != nil {
		t.Fatalf("collectFiles failed: %v", err)
	}

	// Verify excluded files are not in the list
	for _, file := range files {
		if strings.HasSuffix(file, ".log") {
			t.Errorf("collected file should not end with .log: %s", file)
		}
		if strings.Contains(file, "node_modules") {
			t.Errorf("collected file should not be from node_modules: %s", file)
		}
		if strings.Contains(file, "build") && strings.Contains(file, "build/output.txt") {
			t.Errorf("collected file should not be from build directory: %s", file)
		}
	}

	// Build output with tree
	var output bytes.Buffer
	
	writeHeader(&output, "FILE HIERARCHY")
	writeTree(&output, []string{"."}, files)
	output.WriteString("\n")

	for _, file := range files {
		writeHeader(&output, file)
		if err := writeFileContent(&output, file); err != nil {
			output.WriteString("[unreadable]\n")
		}
		output.WriteString("\n")
	}

	result := output.String()

	// Should contain tree section
	if !strings.Contains(result, "FILE HIERARCHY") {
		t.Error("output should contain FILE HIERARCHY header")
	}

	// Check for excluded files more precisely - look for actual file names
	// not just the extension (to avoid false positives from .gitignore mention)
	if strings.Contains(result, "debug.log") || strings.Contains(result, "error.log") {
		t.Error("output should not contain debug.log or error.log files")
	}
	if strings.Contains(result, "node_modules/pkg.json") {
		t.Error("output should not contain node_modules files")
	}
	if strings.Contains(result, "build/output.txt") {
		t.Error("output should not contain build directory files")
	}
}

func TestCollectFiles_NonExistentPath(t *testing.T) {
	matcher := &ExcludeMatcher{}
	
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	files, err := collectFiles([]string{"/totally/nonexistent/path"}, matcher)

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	stderr := buf.String()

	// Should not error, but should warn
	if err != nil {
		t.Errorf("collectFiles should not error on nonexistent path, got: %v", err)
	}

	// Should produce empty file list
	if len(files) != 0 {
		t.Errorf("expected 0 files for nonexistent path, got %d", len(files))
	}

	// Should output warning
	if !strings.Contains(stderr, "Warning") {
		t.Error("expected warning for nonexistent path")
	}
}

func TestCollectFiles_SortedOutput(t *testing.T) {
	tmpDir := setupTestDirectory(t)
	defer os.RemoveAll(tmpDir)

	matcher := &ExcludeMatcher{}
	
	files, err := collectFiles([]string{filepath.Join(tmpDir, "src")}, matcher)
	if err != nil {
		t.Fatalf("collectFiles failed: %v", err)
	}

	// Files should be sorted
	for i := 1; i < len(files); i++ {
		if files[i-1] > files[i] {
			t.Errorf("files not sorted: %s comes after %s", files[i-1], files[i])
		}
	}
}

func TestGitignoreWithAbsolutePaths(t *testing.T) {
	tmpDir := setupTestDirectory(t)
	defer os.RemoveAll(tmpDir)

	// Save original directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	
	// Change to tmpDir (this is what the end-to-end test does)
	os.Chdir(tmpDir)

	// Build matcher using absolute path to .gitignore
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	matcher, err := buildExcludeMatcher([]string{gitignorePath}, []string{})
	if err != nil {
		t.Fatalf("buildExcludeMatcher failed: %v", err)
	}

	// Test with absolute paths (what collectFiles produces)
	testCases := []struct {
		name     string
		path     string
		expected bool
	}{
		{"absolute log file", filepath.Join(tmpDir, "debug.log"), true},
		{"absolute node_modules", filepath.Join(tmpDir, "node_modules", "pkg.json"), true},
		{"absolute build", filepath.Join(tmpDir, "build", "output.txt"), true},
		{"absolute source file", filepath.Join(tmpDir, "src", "app.go"), false},
		{"absolute readme", filepath.Join(tmpDir, "README.md"), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := matcher.ShouldExclude(tc.path)
			if result != tc.expected {
				t.Errorf("ShouldExclude(%q) = %v, want %v", tc.path, result, tc.expected)
			}
		})
	}
}
