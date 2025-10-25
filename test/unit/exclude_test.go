package unit_test

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

// Advanced exclusion pattern tests

func TestExcludeMatcherShouldExclude_ComplexGlobs(t *testing.T) {
	matcher, _ := exclude.BuildMatcher([]string{}, []string{
		"**/node_modules/**",
		"**/*.{tmp,log,cache}", 
		"build/**/output.*",
		"src/**/test_*.go",
		"[Tt]emp*/",
		"*.{js,ts}.*", // compiled JS/TS files
	}, false)

	tests := []struct {
		name     string
		path     string
		isDir    bool
		expected bool
	}{
		{"deep node_modules", "project/lib/node_modules/pkg/index.js", false, true},
		{"tmp extension", "debug.tmp", false, true},
		{"log extension", "app.log", false, true}, 
		{"cache extension", "webpack.cache", false, true},
		{"build output", "build/dist/output.js", false, true},
		{"test file pattern", "src/utils/test_helper.go", false, true},
		{"Temp directory", "Temp/file.txt", false, true},
		{"temp directory lowercase", "temp/file.txt", false, true},
		{"compiled JS", "main.js.map", false, true},
		{"compiled TS", "app.ts.build", false, true},
		{"regular JS file", "script.js", false, false}, // doesn't match *.js.*
		{"regular source file", "src/main.go", false, false},
		{"non-test file", "src/helper.go", false, false},
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

func TestExcludeMatcherShouldExclude_DirectoryTrailingSlash(t *testing.T) {
	// Test the smart directory exclusion behavior
	matcher, _ := exclude.BuildMatcher([]string{}, []string{
		"dist/",      // Directory pattern
		"*.tmp",     // File pattern  
		"logs/",     // Directory pattern
		"cache/",    // Directory pattern
	}, false)

	tests := []struct {
		name     string
		path     string
		isDir    bool
		expected bool
	}{
		// Directory exclusions should work
		{"dist directory itself", "dist", true, true},
		{"file in dist", "dist/main.js", false, true},
		{"nested in dist", "dist/assets/style.css", false, true},
		{"logs directory", "logs", true, true},
		{"file in logs", "logs/app.log", false, true},
		{"cache directory", "cache", true, true},
		{"nested cache", "src/cache/file.dat", false, true},
		
		// File patterns should not affect directories with similar names
		{"tmp directory", "tmp", true, false}, // *.tmp doesn't match directories
		{"tmp file", "debug.tmp", false, true},
		
		// Should not match partial names
		{"distribution", "distribution", true, false}, // doesn't match "dist/"
		{"dist_old", "dist_old/file.js", false, false},
		{"mydist", "mydist/build.js", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.ShouldExclude(tt.path, tt.isDir)
			if result != tt.expected {
				t.Errorf("ShouldExclude(%q, isDir=%v) = %v, want %v", tt.path, tt.isDir, result, tt.expected)
			}
		})
	}
}

func TestExcludeMatcherShouldExclude_CaseInsensitivity(t *testing.T) {
	// Test case-insensitive exclusions
	matcherSensitive, _ := exclude.BuildMatcher([]string{}, []string{"*.LOG", "TeMp/"}, false)
	matcherInsensitive, _ := exclude.BuildMatcher([]string{}, []string{"*.LOG", "TeMp/"}, true)

	tests := []struct {
		name       string
		path       string
		isDir      bool
		sensitive  bool   // expected for case-sensitive
		insensitive bool  // expected for case-insensitive
	}{
		{"uppercase LOG", "debug.LOG", false, true, true},
		{"lowercase log", "debug.log", false, false, true},
		{"mixed case log", "debug.Log", false, false, true},
		{"exact TeMp dir", "TeMp/file.txt", false, true, true},
		{"lowercase temp", "temp/file.txt", false, false, true},
		{"uppercase TEMP", "TEMP/file.txt", false, false, true},
		{"mixed case Temp", "Temp/file.txt", false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_sensitive", func(t *testing.T) {
			result := matcherSensitive.ShouldExclude(tt.path, tt.isDir)
			if result != tt.sensitive {
				t.Errorf("ShouldExclude(case-sensitive, %q) = %v, want %v", tt.path, result, tt.sensitive)
			}
		})
		t.Run(tt.name+"_insensitive", func(t *testing.T) {
			result := matcherInsensitive.ShouldExclude(tt.path, tt.isDir)
			if result != tt.insensitive {
				t.Errorf("ShouldExclude(case-insensitive, %q) = %v, want %v", tt.path, result, tt.insensitive)
			}
		})
	}
}

func TestExcludeMatcherShouldExclude_PathVsBasename(t *testing.T) {
	// Test path-aware vs basename-only patterns
	matcher, _ := exclude.BuildMatcher([]string{}, []string{
		"test.go",        // basename only
		"src/test.go",    // path-aware
		"*/temp.txt",     // path pattern
		"build/",         // directory
	}, false)

	tests := []struct {
		name     string
		path     string
		isDir    bool
		expected bool
	}{
		// Basename-only pattern should match any file with that name
		{"test.go in root", "test.go", false, true},
		{"test.go in src", "src/test.go", false, true},
		{"test.go deep", "project/pkg/test.go", false, true},
		
		// Path-aware pattern should only match exact path
		{"exact src/test.go", "src/test.go", false, true},
		{"test.go in root (path)", "test.go", false, false}, // doesn't match src/test.go
		{"test.go in lib", "lib/test.go", false, false},   // doesn't match src/test.go
		
		// Wildcard path pattern
		{"temp.txt in any subdir", "cache/temp.txt", false, true},
		{"temp.txt in deep path", "build/output/temp.txt", false, true},
		{"temp.txt in root", "temp.txt", false, false}, // doesn't match */temp.txt
		
		// Directory patterns
		{"build directory", "build/file.js", false, true},
		{"file in build", "build/dist/main.js", false, true},
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

func TestExcludeMatcherShouldExclude_EdgeCases(t *testing.T) {
	// Test edge cases and corner cases
	matcher, _ := exclude.BuildMatcher([]string{}, []string{
		"",           // empty pattern
		"  ",        // whitespace only
		"*",         // match everything
		"?",         // single char
		"[abc]",     // character class
		"file[0-9]*", // complex pattern
	}, false)

	tests := []struct {
		name     string
		path     string
		isDir    bool
		expected bool
	}{
		// Empty pattern should not match anything
		{"empty pattern vs file", "any.file", false, false},
		
		// Whitespace pattern should not match anything
		{"whitespace pattern vs file", "any.file", false, false},
		
		// * should match any file basename
		{"star matches file", "test.go", false, true},
		{"star matches file in subdir", "src/test.go", false, true},
		
		// ? should match single character
		{"single char match", "a", false, true},
		{"single char no match", "ab", false, false},
		
		// Character class
		{"char class match a", "a", false, true},
		{"char class match b", "b", false, true}, 
		{"char class match c", "c", false, true},
		{"char class no match d", "d", false, false},
		
		// Complex pattern
		{"complex pattern match", "file1.txt", false, true},
		{"complex pattern match 2", "file99", false, true},
		{"complex pattern no match", "file.txt", false, false}, // no digit
		{"complex pattern no match 2", "test1.txt", false, false}, // doesn't start with "file"
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

func TestExcludeMatcherShouldExclude_GitignoreAdvanced(t *testing.T) {
	// Test advanced gitignore patterns
	tmpfile, err := os.CreateTemp("", "advanced-gitignore-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := `# Advanced gitignore patterns
# Negation patterns
*.log
!important.log
!critical/*.log

# Directory vs file patterns  
logs/
config.json
/root-only.txt

# Wildcard patterns
**/*.tmp
**/build/**
node_modules/

# Character classes
*.[oa]
temp[0-9]/

# Comments and empty lines

# This is a comment

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
		// Negation patterns
		{"regular log excluded", "debug.log", false, true},
		{"important log included", "important.log", false, false}, // negated
		{"critical log included", "critical/error.log", false, false}, // negated
		{"other critical log excluded", "critical/debug.log", false, true}, // not negated
		
		// Directory vs file
		{"logs directory", "logs/app.log", false, true},
		{"config file", "config.json", false, true},
		{"root-only file at root", "root-only.txt", false, true},
		{"root-only file in subdir", "src/root-only.txt", false, false}, // leading / means root only
		
		// Wildcards
		{"deep tmp file", "project/cache/temp.tmp", false, true},
		{"build anywhere", "frontend/build/main.js", false, true},
		{"node_modules", "node_modules/package.json", false, true},
		
		// Character classes
		{"object file", "main.o", false, true},
		{"archive file", "lib.a", false, true},
		{"source file not matched", "main.c", false, false}, // not in [oa]
		{"temp dir with number", "temp1/file.txt", false, true},
		{"temp dir no number", "temp/file.txt", false, false}, // doesn't match temp[0-9]/
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