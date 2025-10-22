# ClipCat Test Suite

Comprehensive test suite for clipcat with both unit and integration tests.

## Test Structure

```
clipcat.go                      # Main program
clipcat_test.go                 # Unit tests
clipcat_integration_test.go     # Integration tests
```

## Running Tests

### Run all tests
```bash
go test -v
```

### Run only unit tests
```bash
go test -v -run TestIsGlobPattern
go test -v -run TestExcludeMatcher
go test -v -run TestReadPatternsFromFile
go test -v -run TestGetRelativePath
go test -v -run TestBuildExcludeMatcher
```

### Run only integration tests
```bash
go test -v -run TestCollectFiles
go test -v -run TestWriteTree
go test -v -run TestEndToEnd
```

### Run with coverage
```bash
go test -cover
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Run with race detection
```bash
go test -race
```

### Run specific test
```bash
go test -v -run TestCollectFiles_Directory
```

## Test Coverage

### Unit Tests (`clipcat_test.go`)

**Pattern Matching:**
- ✅ `TestIsGlobPattern` - Detects glob characters in paths
- ✅ `TestExcludeMatcherShouldExclude_GlobPatterns` - Glob pattern exclusion
- ✅ `TestExcludeMatcherShouldExclude_GitignorePatterns` - Gitignore pattern exclusion
- ✅ `TestExcludeMatcherShouldExclude_MixedPatterns` - Combined glob + gitignore

**File Reading:**
- ✅ `TestReadPatternsFromFile` - Reads patterns from files correctly
- ✅ `TestReadPatternsFromFile_NonExistent` - Handles missing files

**Path Handling:**
- ✅ `TestGetRelativePath` - Computes relative paths from roots
- ✅ `TestGetRelativePath_WithGlobInRoots` - Skips glob patterns in root calculation

**Matcher Building:**
- ✅ `TestBuildExcludeMatcher_EmptyPatterns` - Handles no patterns
- ✅ `TestBuildExcludeMatcher_WithGlobPatterns` - Glob pattern matcher
- ✅ `TestBuildExcludeMatcher_WithFile` - Gitignore file matcher
- ✅ `TestBuildExcludeMatcher_NonExistentFile` - Error handling

### Integration Tests (`clipcat_integration_test.go`)

**File Collection:**
- ✅ `TestCollectFiles_SingleFile` - Single file input
- ✅ `TestCollectFiles_Directory` - Recursive directory traversal
- ✅ `TestCollectFiles_DirectoryWithExclusions` - Respects exclusion patterns
- ✅ `TestCollectFiles_GlobPattern` - Glob pattern matching
- ✅ `TestCollectFiles_MultipleInputs` - Multiple paths
- ✅ `TestCollectFiles_Deduplication` - Removes duplicate files
- ✅ `TestCollectFiles_NonExistentPath` - Handles missing paths
- ✅ `TestCollectFiles_SortedOutput` - Files are sorted

**Tree Generation:**
- ✅ `TestWriteTree_SingleRoot` - Tree for single directory
- ✅ `TestWriteTree_MultipleRoots` - Tree for multiple directories

**Output Formatting:**
- ✅ `TestWriteHeader` - File header formatting
- ✅ `TestWriteFileContent` - File content reading
- ✅ `TestWriteFileContent_Unreadable` - Unreadable file handling

**End-to-End:**
- ✅ `TestEndToEnd_BasicUsage` - Full workflow without exclusions
- ✅ `TestEndToEnd_WithTreeAndExclusions` - Full workflow with all features

## Test Data

Integration tests use `setupTestDirectory()` which creates:

```
tmpDir/
├── README.md
├── main.go
├── main_test.go
├── debug.log
├── error.log
├── .gitignore
├── src/
│   ├── app.go
│   ├── components/
│   │   └── button.go
│   └── utils/
│       └── format.go
├── tests/
│   └── integration_test.go
├── build/
│   └── output.txt
└── node_modules/
    └── pkg.json
```

## Writing New Tests

### Unit Test Template
```go
func TestNewFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"case 1", "input1", "output1"},
        {"case 2", "input2", "output2"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := yourFunction(tt.input)
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

### Integration Test Template
```go
func TestNewIntegration(t *testing.T) {
    tmpDir := setupTestDirectory(t)
    defer os.RemoveAll(tmpDir)

    // Your test logic here
    
    if /* condition */ {
        t.Error("test failed")
    }
}
```

## Continuous Integration

Add to your CI pipeline:

```yaml
# .github/workflows/test.yml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: go test -v -race -coverprofile=coverage.out
      - run: go tool cover -func=coverage.out
```

## Benchmarking

To add benchmarks:

```go
func BenchmarkCollectFiles(b *testing.B) {
    tmpDir := setupTestDirectory(b)
    defer os.RemoveAll(tmpDir)
    
    matcher := &ExcludeMatcher{}
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        collectFiles([]string{tmpDir}, matcher)
    }
}
```

Run benchmarks:
```bash
go test -bench=.
go test -bench=BenchmarkCollectFiles -benchmem
```

## Test Maintenance

- Keep tests independent - each test should set up its own environment
- Use `t.Parallel()` for tests that can run concurrently
- Clean up temporary files with `defer os.RemoveAll()`
- Use descriptive test names that explain what's being tested
- Group related tests with subtests using `t.Run()`

## Coverage Goals

- **Target**: >80% code coverage
- **Current**: Run `go test -cover` to check
- **Critical paths**: 100% coverage for exclusion logic and file collection

## Known Limitations

- Clipboard operations are not tested (require system clipboard)
- Command-line flag parsing is tested indirectly through integration tests
- Real-world .gitignore complexity may exceed test scenarios

## Troubleshooting

**Tests fail with "permission denied":**
```bash
# Check file permissions
ls -la /tmp/clipcat-*
```

**Tests hang:**
```bash
# Run with timeout
go test -timeout 30s
```

**Race conditions detected:**
```bash
# Run with race detector
go test -race
```
