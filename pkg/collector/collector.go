package collector

import (
	"clipcat/pkg/exclude"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func isGlobPattern(path string) bool {
	return strings.ContainsAny(path, "*?[")
}

func containsAnySep(s string) bool {
	return strings.Contains(s, "/") || strings.Contains(s, string(filepath.Separator))
}

func matchPath(pattern, target string) bool {
	ok, _ := filepath.Match(pattern, target)
	return ok
}

func CollectFiles(paths []string, matcher *exclude.ExcludeMatcher, ignoreCase bool) ([]string, error) {
	seen := make(map[string]bool)
	var result []string

	for _, path := range paths {
		// Check if it's a literal path
		info, err := os.Stat(path)
		if err == nil {
			// Literal path exists
			if info.IsDir() {
				// Walk directory
				err := filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
					if err != nil {
						return nil // Skip errors
					}

					absPath, _ := filepath.Abs(p)

					// Exclude?
					if matcher.ShouldExclude(absPath, fi.IsDir()) {
						if fi.IsDir() {
							return filepath.SkipDir
						}
						return nil
					}

					if !fi.IsDir() {
						if !seen[absPath] {
							result = append(result, absPath)
							seen[absPath] = true
						}
					}
					return nil
				})
				if err != nil {
					return nil, err
				}
			} else {
				absPath, _ := filepath.Abs(path)
				if !matcher.ShouldExclude(absPath, false) && !seen[absPath] {
					result = append(result, absPath)
					seen[absPath] = true
				}
			}
		} else if isGlobPattern(path) {
			// Glob pattern - search from current directory
			pattern := path
			err := filepath.Walk(".", func(p string, fi os.FileInfo, err error) error {
				if err != nil {
					return nil
				}

				absPath, _ := filepath.Abs(p)

				// Exclude?
				if matcher.ShouldExclude(absPath, fi.IsDir()) {
					if fi.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}

				if fi.IsDir() {
					return nil
				}

				rel, _ := filepath.Rel(".", p)
				sep := string(filepath.Separator)

				// Normalize both sides for matching
				patNorm := strings.ReplaceAll(pattern, "/", sep)
				target := rel

				var matched bool
				if containsAnySep(patNorm) {
					// Match against the relative path when the pattern has a separator
					if ignoreCase {
						matched = matchPath(strings.ToLower(patNorm), strings.ToLower(target))
					} else {
						matched = matchPath(patNorm, target)
					}
				} else {
					// Match against basename when there's no separator
					name := filepath.Base(rel)
					if ignoreCase {
						matched = matchPath(strings.ToLower(patNorm), strings.ToLower(name))
					} else {
						matched = matchPath(patNorm, name)
					}
				}

				if matched {
					if !seen[absPath] {
						result = append(result, absPath)
						seen[absPath] = true
					}
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			fmt.Fprintf(os.Stderr, "Warning: Skipping non-existent path: %s\n", path)
		}
	}

	return result, nil
}