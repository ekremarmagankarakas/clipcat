package unit_test

import (
	"clipcat/pkg/collector"
	"clipcat/pkg/exclude"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectFiles_EdgeCases(t *testing.T) {
	// Test empty path list
	t.Run("empty_paths", func(t *testing.T) {
		matcher, _ := exclude.BuildMatcher([]string{}, []string{}, false)
		
		files, err := collector.CollectFiles([]string{}, matcher, false)
		
		if err != nil {
			t.Fatalf("Expected no error for empty paths, got: %v", err)
		}
		if len(files) != 0 {
			t.Errorf("Expected 0 files for empty paths, got %d", len(files))
		}
	})

	// Test nonexistent paths
	t.Run("nonexistent_paths", func(t *testing.T) {
		matcher, _ := exclude.BuildMatcher([]string{}, []string{}, false)
		
		files, err := collector.CollectFiles([]string{"/totally/nonexistent/path", "/another/fake/path"}, matcher, false)
		
		if err != nil {
			t.Fatalf("Expected no error for nonexistent paths, got: %v", err)
		}
		if len(files) != 0 {
			t.Errorf("Expected 0 files for nonexistent paths, got %d", len(files))
		}
	})

	// Test mixed existing and nonexistent paths
	t.Run("mixed_paths", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "collector-mixed-")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("content"), 0644)

		matcher, _ := exclude.BuildMatcher([]string{}, []string{}, false)
		
		paths := []string{
			testFile,                    // exists
			"/nonexistent/path",         // doesn't exist
			tmpDir,                      // exists (directory)
			"/another/fake/directory",   // doesn't exist
		}
		
		files, err := collector.CollectFiles(paths, matcher, false)
		
		if err != nil {
			t.Fatalf("Expected no error for mixed paths, got: %v", err)
		}
		
		// Should find the existing file twice (once directly, once through directory)
		if len(files) != 1 { // deduplicated
			t.Errorf("Expected 1 file from mixed paths, got %d", len(files))
		}
		
		if !strings.HasSuffix(files[0], "test.txt") {
			t.Errorf("Expected test.txt, got %s", files[0])
		}
	})
}

func TestCollectFiles_SymbolicLinks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "collector-symlink-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create original file
	originalFile := filepath.Join(tmpDir, "original.txt")
	os.WriteFile(originalFile, []byte("original content"), 0644)

	// Create symbolic link
	linkFile := filepath.Join(tmpDir, "link.txt")
	err = os.Symlink(originalFile, linkFile)
	if err != nil {
		t.Skip("Symbolic links not supported on this system")
	}

	matcher, _ := exclude.BuildMatcher([]string{}, []string{}, false)

	// Test collecting the symlink directly
	t.Run("collect_symlink_directly", func(t *testing.T) {
		files, err := collector.CollectFiles([]string{linkFile}, matcher, false)
		
		if err != nil {
			t.Fatalf("CollectFiles failed: %v", err)
		}
		
		if len(files) != 1 {
			t.Errorf("Expected 1 file, got %d", len(files))
		}
		
		if !strings.HasSuffix(files[0], "link.txt") {
			t.Errorf("Expected link.txt, got %s", files[0])
		}
	})

	// Test collecting directory with symlink
	t.Run("collect_directory_with_symlink", func(t *testing.T) {
		files, err := collector.CollectFiles([]string{tmpDir}, matcher, false)
		
		if err != nil {
			t.Fatalf("CollectFiles failed: %v", err)
		}
		
		// Should find both original and link
		if len(files) != 2 {
			t.Errorf("Expected 2 files, got %d", len(files))
		}
		
		foundOriginal := false
		foundLink := false
		for _, file := range files {
			if strings.HasSuffix(file, "original.txt") {
				foundOriginal = true
			}
			if strings.HasSuffix(file, "link.txt") {
				foundLink = true
			}
		}
		
		if !foundOriginal {
			t.Error("Expected to find original.txt")
		}
		if !foundLink {
			t.Error("Expected to find link.txt")
		}
	})

	// Test broken symlink
	t.Run("broken_symlink", func(t *testing.T) {
		brokenLink := filepath.Join(tmpDir, "broken.txt")
		os.Symlink("/nonexistent/target", brokenLink)
		defer os.Remove(brokenLink)

		files, err := collector.CollectFiles([]string{brokenLink}, matcher, false)
		
		// Broken symlinks might be handled differently by different systems
		// The important thing is that it doesn't crash
		if err != nil {
			t.Logf("Broken symlink error (expected): %v", err)
		}
		
		// Result depends on system behavior with broken symlinks
		t.Logf("Broken symlink result: %d files", len(files))
	})
}

func TestCollectFiles_SpecialFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "collector-special-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	matcher, _ := exclude.BuildMatcher([]string{}, []string{}, false)

	// Test hidden files (starting with .)
	t.Run("hidden_files", func(t *testing.T) {
		hiddenFile := filepath.Join(tmpDir, ".hidden")
		os.WriteFile(hiddenFile, []byte("hidden content"), 0644)

		files, err := collector.CollectFiles([]string{tmpDir}, matcher, false)
		
		if err != nil {
			t.Fatalf("CollectFiles failed: %v", err)
		}
		
		foundHidden := false
		for _, file := range files {
			if strings.HasSuffix(file, ".hidden") {
				foundHidden = true
				break
			}
		}
		
		if !foundHidden {
			t.Error("Expected to find hidden file")
		}
	})

	// Test files with no extension
	t.Run("no_extension", func(t *testing.T) {
		noExtFile := filepath.Join(tmpDir, "README")
		os.WriteFile(noExtFile, []byte("readme content"), 0644)

		files, err := collector.CollectFiles([]string{tmpDir}, matcher, false)
		
		if err != nil {
			t.Fatalf("CollectFiles failed: %v", err)
		}
		
		foundReadme := false
		for _, file := range files {
			if strings.HasSuffix(file, "README") {
				foundReadme = true
				break
			}
		}
		
		if !foundReadme {
			t.Error("Expected to find README file")
		}
	})

	// Test empty directory
	t.Run("empty_directory", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "empty")
		os.Mkdir(emptyDir, 0755)

		files, err := collector.CollectFiles([]string{emptyDir}, matcher, false)
		
		if err != nil {
			t.Fatalf("CollectFiles failed: %v", err)
		}
		
		// Empty directory should yield no files
		if len(files) != 0 {
			t.Errorf("Expected 0 files from empty directory, got %d", len(files))
		}
	})
}

func TestCollectFiles_GlobPatterns_Advanced(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "collector-glob-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create complex directory structure
	dirs := []string{
		"src", "src/components", "src/utils", 
		"test", "test/unit", "test/integration",
		"docs", "build", "node_modules",
	}
	for _, dir := range dirs {
		os.MkdirAll(filepath.Join(tmpDir, dir), 0755)
	}

	// Create various files
	files := map[string]string{
		"README.md":                     "readme",
		"src/main.go":                   "main",
		"src/app.js":                    "app",
		"src/components/Button.tsx":     "button",
		"src/components/Modal.vue":      "modal", 
		"src/utils/helper.go":           "helper",
		"test/main_test.go":             "test",
		"test/unit/helper_test.js":      "test",
		"test/integration/api_test.py":  "test",
		"docs/api.md":                   "docs",
		"build/main.js":                 "build",
		"node_modules/react.js":         "dependency",
		"Dockerfile":                    "docker",
		"docker-compose.yml":            "compose",
		"package.json":                  "package",
		"TestConfig.json":               "config",
		"app.LOG":                       "log",
		"debug.log":                     "log",
	}
	
	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	// Change to test directory for relative glob patterns
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	matcher, _ := exclude.BuildMatcher([]string{}, []string{}, false)

	tests := []struct {
		name           string
		patterns       []string
		ignoreCase     bool
		expectedFiles  []string
		unexpectedFiles []string
	}{
		{
			name:           "all_go_files",
			patterns:       []string{"**/*.go"},
			ignoreCase:     false,
			expectedFiles:  []string{"main.go", "helper.go", "main_test.go"},
			unexpectedFiles: []string{"app.js", "Button.tsx"},
		},
		{
			name:           "test_files_case_insensitive", 
			patterns:       []string{"*test*"},
			ignoreCase:     true,
			expectedFiles:  []string{"main_test.go", "helper_test.js", "api_test.py", "TestConfig.json"},
			unexpectedFiles: []string{"main.go", "app.js"},
		},
		{
			name:           "config_files",
			patterns:       []string{"*.json", "*.yml", "Dockerfile"},
			ignoreCase:     false,
			expectedFiles:  []string{"package.json", "TestConfig.json", "docker-compose.yml", "Dockerfile"},
			unexpectedFiles: []string{"main.go", "README.md"},
		},
		{
			name:           "source_files_multiple_extensions",
			patterns:       []string{"src/**/*.{go,js,tsx,vue}"},
			ignoreCase:     false,
			expectedFiles:  []string{"main.go", "app.js", "Button.tsx", "Modal.vue", "helper.go"},
			unexpectedFiles: []string{"main_test.go", "README.md"},
		},
		{
			name:           "markdown_files_case_insensitive",
			patterns:       []string{"*.MD"},
			ignoreCase:     true,
			expectedFiles:  []string{"README.md", "api.md"},
			unexpectedFiles: []string{"main.go", "package.json"},
		},
		{
			name:           "log_files_mixed_case",
			patterns:       []string{"*.log", "*.LOG"},
			ignoreCase:     false,
			expectedFiles:  []string{"debug.log", "app.LOG"},
			unexpectedFiles: []string{"main.go", "README.md"},
		},
		{
			name:           "deep_pattern_with_specific_dir",
			patterns:       []string{"src/**/test*"},
			ignoreCase:     false,
			expectedFiles:  []string{}, // No files match this pattern in our setup
			unexpectedFiles: []string{"main_test.go", "helper_test.js"}, // these are not under src/
		},
		{
			name:           "single_character_wildcard",
			patterns:       []string{"????.????"},
			ignoreCase:     false,
			expectedFiles:  []string{"main.go"}, // 4 chars + . + 2 chars
			unexpectedFiles: []string{"README.md", "Button.tsx"},
		},
		{
			name:           "character_class",
			patterns:       []string{"*.[jt]s"},
			ignoreCase:     false,
			expectedFiles:  []string{"app.js", "react.js", "main.js"},
			unexpectedFiles: []string{"Button.tsx", "main.go"}, // tsx doesn't match [jt]s
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := collector.CollectFiles(tt.patterns, matcher, tt.ignoreCase)
			
			if err != nil {
				t.Fatalf("CollectFiles failed: %v", err)
			}

			// Check for expected files
			for _, expectedFile := range tt.expectedFiles {
				found := false
				for _, file := range files {
					if strings.HasSuffix(file, expectedFile) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected file %q not found in results: %v", expectedFile, getBasenames(files))
				}
			}

			// Check that unexpected files are not present
			for _, unexpectedFile := range tt.unexpectedFiles {
				for _, file := range files {
					if strings.HasSuffix(file, unexpectedFile) {
						t.Errorf("Unexpected file %q found in results", unexpectedFile)
						break
					}
				}
			}
		})
	}
}

func TestCollectFiles_PathSeparators(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "collector-sep-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create nested structure
	nestedDir := filepath.Join(tmpDir, "level1", "level2", "level3")
	os.MkdirAll(nestedDir, 0755)
	
	testFile := filepath.Join(nestedDir, "deep.txt")
	os.WriteFile(testFile, []byte("deep content"), 0644)

	matcher, _ := exclude.BuildMatcher([]string{}, []string{}, false)

	// Change to test directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Test different path separator styles
	tests := []struct {
		name    string
		pattern string
	}{
		{"unix_separators", "level1/level2/level3/*.txt"},
		{"glob_recursive", "level1/**/deep.txt"},
		{"mixed_pattern", "level1/*/level3/deep.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := collector.CollectFiles([]string{tt.pattern}, matcher, false)
			
			if err != nil {
				t.Fatalf("CollectFiles failed: %v", err)
			}
			
			if len(files) != 1 {
				t.Errorf("Expected 1 file for pattern %q, got %d: %v", tt.pattern, len(files), getBasenames(files))
			}
			
			if len(files) > 0 && !strings.HasSuffix(files[0], "deep.txt") {
				t.Errorf("Expected deep.txt, got %s", files[0])
			}
		})
	}
}

func TestCollectFiles_Unicode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "collector-unicode-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create files with Unicode names
	unicodeFiles := []string{
		"文档.txt",       // Chinese
		"файл.js",        // Russian  
		"ファイル.go",       // Japanese
		"café.json",      // French accent
		"naïve.py",       // Diaeresis
		"🚀rocket.md",    // Emoji
		"résumé.pdf",     // Multiple accents
	}

	for _, name := range unicodeFiles {
		testFile := filepath.Join(tmpDir, name)
		os.WriteFile(testFile, []byte("unicode content"), 0644)
	}

	matcher, _ := exclude.BuildMatcher([]string{}, []string{}, false)

	// Change to test directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Test collecting Unicode files
	t.Run("collect_unicode_files", func(t *testing.T) {
		files, err := collector.CollectFiles([]string{"."}, matcher, false)
		
		if err != nil {
			t.Fatalf("CollectFiles failed: %v", err)
		}
		
		if len(files) != len(unicodeFiles) {
			t.Errorf("Expected %d files, got %d", len(unicodeFiles), len(files))
		}
		
		for _, expectedName := range unicodeFiles {
			found := false
			for _, file := range files {
				if strings.HasSuffix(file, expectedName) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected Unicode file %q not found", expectedName)
			}
		}
	})

	// Test Unicode glob patterns
	t.Run("unicode_glob_patterns", func(t *testing.T) {
		tests := []struct {
			name     string
			pattern  string
			expected []string
		}{
			{"chinese_files", "*文档*", []string{"文档.txt"}},
			{"files_with_accents", "*é*", []string{"café.json", "résumé.pdf"}},
			{"emoji_files", "*🚀*", []string{"🚀rocket.md"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				files, err := collector.CollectFiles([]string{tt.pattern}, matcher, false)
				
				if err != nil {
					t.Fatalf("CollectFiles failed: %v", err)
				}
				
				if len(files) != len(tt.expected) {
					t.Errorf("Pattern %q: expected %d files, got %d", tt.pattern, len(tt.expected), len(files))
				}
				
				for _, expectedFile := range tt.expected {
					found := false
					for _, file := range files {
						if strings.HasSuffix(file, expectedFile) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Pattern %q: expected file %q not found", tt.pattern, expectedFile)
					}
				}
			})
		}
	})
}

func TestCollectFiles_LargeDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large directory test in short mode")
	}

	tmpDir, err := os.MkdirTemp("", "collector-large-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create many files (but not too many to avoid slow tests)
	numFiles := 100
	expectedFiles := make([]string, numFiles)
	
	for i := 0; i < numFiles; i++ {
		fileName := filepath.Join(tmpDir, fmt.Sprintf("file_%03d.txt", i))
		os.WriteFile(fileName, []byte(fmt.Sprintf("content %d", i)), 0644)
		expectedFiles[i] = fileName
	}

	matcher, _ := exclude.BuildMatcher([]string{}, []string{}, false)

	files, err := collector.CollectFiles([]string{tmpDir}, matcher, false)
	
	if err != nil {
		t.Fatalf("CollectFiles failed: %v", err)
	}
	
	if len(files) != numFiles {
		t.Errorf("Expected %d files, got %d", numFiles, len(files))
	}

	// Verify files are sorted (requirement from main function)
	for i := 1; i < len(files); i++ {
		if files[i-1] > files[i] {
			t.Errorf("Files not sorted: %s > %s", files[i-1], files[i])
			break
		}
	}
}

// Helper function to get basenames from full paths for easier assertion debugging
func getBasenames(files []string) []string {
	basenames := make([]string, len(files))
	for i, file := range files {
		basenames[i] = filepath.Base(file)
	}
	return basenames
}

func TestCollectFiles_Deduplication_Advanced(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "collector-dedup-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("content"), 0644)

	matcher, _ := exclude.BuildMatcher([]string{}, []string{}, false)

	// Test various ways the same file could be referenced
	paths := []string{
		testFile,                    // absolute path
		testFile,                    // same absolute path again
		tmpDir,                      // directory containing the file
		tmpDir,                      // same directory again
	}

	files, err := collector.CollectFiles(paths, matcher, false)
	
	if err != nil {
		t.Fatalf("CollectFiles failed: %v", err)
	}
	
	// Should deduplicate to just one file
	if len(files) != 1 {
		t.Errorf("Expected 1 deduplicated file, got %d: %v", len(files), files)
	}
	
	if !strings.HasSuffix(files[0], "test.txt") {
		t.Errorf("Expected test.txt, got %s", files[0])
	}
}