package exclude

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
)

type ExcludeMatcher struct {
	gitignoreMatcher *gitignore.GitIgnore
	globPatterns     []string
	ignoreCase       bool
}

func BuildMatcher(files []string, globPatterns []string, ignoreCase bool) (*ExcludeMatcher, error) {
	matcher := &ExcludeMatcher{
		globPatterns: globPatterns,
		ignoreCase:   ignoreCase,
	}

	// Collect all patterns from files
	var allPatterns []string

	for _, file := range files {
		patterns, err := readPatternsFromFile(file)
		if err != nil {
			return nil, fmt.Errorf("cannot read exclude file %s: %w", file, err)
		}
		allPatterns = append(allPatterns, patterns...)
	}

	// Build gitignore matcher if we have patterns
	if len(allPatterns) > 0 {
		matcher.gitignoreMatcher = gitignore.CompileIgnoreLines(allPatterns...)
	}

	return matcher, nil
}

func readPatternsFromFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var patterns []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// Keep comments/empties; gitignore lib will handle them
		patterns = append(patterns, line)
	}

	return patterns, scanner.Err()
}

func hasGlobChars(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

func (m *ExcludeMatcher) ShouldExclude(path string, isDir bool) bool {
	// Convert to relative path for gitignore matching
	relPath, err := filepath.Rel(".", path)
	if err != nil {
		relPath = path
	}

	// Normalize separators for robust matching
	osSep := string(filepath.Separator)
	relNorm := strings.ReplaceAll(relPath, "/", osSep)
	base := filepath.Base(relNorm)

	lower := func(s string) string {
		if m.ignoreCase {
			return strings.ToLower(s)
		}
		return s
	}
	relCmp := lower(relNorm)
	baseCmp := lower(base)

	// 1) Check gitignore matcher (if any)
	if m.gitignoreMatcher != nil && m.gitignoreMatcher.MatchesPath(relNorm) {
		return true
	}

	// 2) Check our -e/--exclude glob patterns
	for _, raw := range m.globPatterns {
		pat := strings.TrimSpace(raw)
		if pat == "" {
			continue
		}

		// Normalize separators in the pattern so user-written "/" also works on Windows
		pat = strings.ReplaceAll(pat, "/", osSep)
		patCmp := lower(pat)

		// Directory patterns MUST end with a separator to affect directories.
		if strings.HasSuffix(patCmp, osSep) {
			dirPat := strings.TrimSuffix(patCmp, osSep)

			// Simple dir name (no globs/seps) like "__pycache__/"
			if !hasGlobChars(dirPat) && !strings.Contains(dirPat, osSep) {
				// Directory itself
				if isDir && (relCmp == dirPat || relCmp == dirPat+osSep) {
					return true
				}
				// Any content at root under that dir
				if strings.HasPrefix(relCmp, dirPat+osSep) {
					return true
				}
				// Nested segment anywhere
				if strings.Contains(relCmp, osSep+dirPat+osSep) {
					return true
				}
				continue
			}

			// Complex dir pattern (globs or seps): treat as prefix for anything under it
			dirAny := dirPat + osSep + "*"
			if matchPath(dirAny, relCmp) {
				return true
			}
			continue
		}

		// Non-slash patterns WITHOUT trailing slash:
		// - If they contain a separator → path-aware file match on full rel path
		// - If they do NOT contain a separator → match FILE BASENAME ONLY
		if strings.Contains(patCmp, osSep) {
			// Path-aware pattern; only meaningful for files (but matching against full path is fine)
			if matchPath(patCmp, relCmp) {
				// If the path matches and we're visiting a directory, don't exclude the directory
				// (these patterns are intended for files). For directories, keep walking.
				if !isDir {
					return true
				}
			}
			continue
		}

		// Basename-only pattern: applies to FILES only (require '/' for directories)
		if !isDir && matchPath(patCmp, baseCmp) {
			return true
		}
	}

	return false
}

func matchPath(pattern, target string) bool {
	ok, _ := filepath.Match(pattern, target)
	return ok
}