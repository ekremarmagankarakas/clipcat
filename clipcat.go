package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
)

type Config struct {
	Paths        []string
	Excludes     []string
	ExcludeFiles []string
	ShowTree     bool
	OnlyTree     bool
	PrintOut     bool
	IgnoreCase   bool
}

type ExcludeMatcher struct {
	gitignoreMatcher *gitignore.GitIgnore
	globPatterns     []string
	ignoreCase       bool
}

func main() {
	cfg := parseArgs()

	// Build exclude matcher
	matcher, err := buildExcludeMatcher(cfg.ExcludeFiles, cfg.Excludes, cfg.IgnoreCase)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading exclude patterns: %v\n", err)
		os.Exit(1)
	}

	// Collect all files
	files, err := collectFiles(cfg.Paths, matcher, cfg.IgnoreCase)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error collecting files: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "No files matched after applying excludes.\n")
		os.Exit(1)
	}

	// Sort for consistent output
	sort.Strings(files)

	// Build output
	var output bytes.Buffer

	if cfg.ShowTree {
		writeHeader(&output, "FILE HIERARCHY")
		writeTree(&output, cfg.Paths, files)
		output.WriteString("\n")
	}

	if !cfg.OnlyTree {
		for _, file := range files {
			writeHeader(&output, file)
			if err := writeFileContent(&output, file); err != nil {
				output.WriteString("[unreadable]\n")
			}
			output.WriteString("\n")
		}
	}

	// Copy to clipboard
	if err := copyToClipboard(output.Bytes()); err != nil {
		fmt.Fprintf(os.Stderr, "Error copying to clipboard: %v\n", err)
		os.Exit(1)
	}

	// Optionally print to stdout
	if cfg.PrintOut {
		os.Stdout.Write(output.Bytes())
	}

	// Success message
	if cfg.OnlyTree {
		fmt.Printf("Copied file hierarchy for %d files to clipboard.\n", len(files))
	} else {
		fmt.Printf("Copied %d files to clipboard.\n", len(files))
	}
}

func parseArgs() *Config {
	cfg := &Config{}

	// Manual argument parsing to allow intermixed flags and paths
	args := os.Args[1:]

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch arg {
		case "-h", "--help":
			printUsage()
			os.Exit(0)
		case "-e", "--exclude":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "Error: %s requires a pattern\n", arg)
				os.Exit(2)
			}
			cfg.Excludes = append(cfg.Excludes, args[i+1])
			i++
		case "--exclude-from":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "Error: --exclude-from requires a file\n")
				os.Exit(2)
			}
			cfg.ExcludeFiles = append(cfg.ExcludeFiles, args[i+1])
			i++
		case "-t", "--tree":
			cfg.ShowTree = true
		case "--only-tree":
			cfg.ShowTree = true
			cfg.OnlyTree = true
		case "-p", "--print":
			cfg.PrintOut = true
		case "-i", "--ignore-case":
			cfg.IgnoreCase = true
		default:
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(os.Stderr, "Error: unknown option: %s\n", arg)
				printUsage()
				os.Exit(2)
			}
			cfg.Paths = append(cfg.Paths, arg)
		}
	}

	if len(cfg.Paths) == 0 {
		printUsage()
		os.Exit(2)
	}

	return cfg
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: clipcat [OPTIONS] <path1> [<path2> ...]

Description:
  - If a path is a file: include that file.
  - If a path is a directory: include ALL files recursively.
  - If a path contains glob patterns (* ? [) and doesn't exist as a literal path,
    it will be treated as a recursive search pattern.
  - Output is a single stream: each file is preceded by a header with its path.
  - The final stream is copied to the clipboard.

Options:
  -e, --exclude PATTERN     Exclude glob pattern (repeatable)
      --exclude-from FILE   Read patterns from FILE with full .gitignore semantics (repeatable)
  -i, --ignore-case         Make glob pattern matching case-insensitive
  -t, --tree                Prepend a FILE HIERARCHY section
      --only-tree           Copy only the FILE HIERARCHY (no file contents)
  -p, --print               Also print to stdout
  -h, --help                Show help

Examples:
  clipcat README.md src/
  clipcat src/ -t
  clipcat . -e go.mod -e go.sum
  clipcat '*checkin*' -i --exclude-from .gitignore
  clipcat '*TEST*' --ignore-case --exclude 'frontend/assets/'
`)
}

// buildExcludeMatcher builds an ExcludeMatcher from ignore files and -e patterns.
func buildExcludeMatcher(files []string, globPatterns []string, ignoreCase bool) (*ExcludeMatcher, error) {
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

func (m *ExcludeMatcher) ShouldExclude(path string) bool {
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

		// Directory patterns ending with separator
		if strings.HasSuffix(patCmp, osSep) {
			dirPat := strings.TrimSuffix(patCmp, osSep)

			// Simple directory name (no globs, no sep) like "__pycache__/"
			if !hasGlobChars(dirPat) && !strings.Contains(dirPat, osSep) {
				// Dir itself
				if relCmp == dirPat || relCmp == dirPat+osSep {
					return true
				}
				// At root
				if strings.HasPrefix(relCmp, dirPat+osSep) {
					return true
				}
				// Nested segment anywhere
				if strings.Contains(relCmp, osSep+dirPat+osSep) {
					return true
				}
				continue
			}

			// Complex dir pattern: treat as prefix of any content under it
			dirAny := dirPat + osSep + "*"
			if matchPath(dirAny, relCmp) {
				return true
			}
			continue
		}

		// If pattern contains a path separator, match against full relative path
		if strings.Contains(patCmp, osSep) {
			if matchPath(patCmp, relCmp) {
				return true
			}
			continue
		}

		// Otherwise match against basename
		if matchPath(patCmp, baseCmp) {
			return true
		}
	}

	return false
}

func matchPath(pattern, target string) bool {
	ok, _ := filepath.Match(pattern, target)
	return ok
}

func isGlobPattern(path string) bool {
	return strings.ContainsAny(path, "*?[")
}

func containsAnySep(s string) bool {
	return strings.Contains(s, "/") || strings.Contains(s, string(filepath.Separator))
}

func collectFiles(paths []string, matcher *ExcludeMatcher, ignoreCase bool) ([]string, error) {
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
					if matcher.ShouldExclude(absPath) {
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
				if !matcher.ShouldExclude(absPath) && !seen[absPath] {
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
				if matcher.ShouldExclude(absPath) {
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

func writeHeader(w io.Writer, path string) {
	bar := strings.Repeat("=", len(path))
	fmt.Fprintf(w, "%s\n%s\n%s\n\n", bar, path, bar)
}

func writeFileContent(w io.Writer, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	return err
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

func writeTree(w io.Writer, roots []string, files []string) {
	// Group files by root
	type rootGroup struct {
		label string
		files []string
	}

	groups := make(map[string]*rootGroup)
	order := []string{}

	for _, file := range files {
		relPath := getRelativePath(file, roots)
		parts := strings.SplitN(relPath, ":", 2)
		root := parts[0]
		rel := parts[1]

		if _, exists := groups[root]; !exists {
			groups[root] = &rootGroup{label: root, files: []string{}}
			order = append(order, root)
		}
		groups[root].files = append(groups[root].files, rel)
	}

	// Print tree for each root
	for i, rootKey := range order {
		if i > 0 {
			fmt.Fprintln(w)
		}

		group := groups[rootKey]
		label := filepath.Base(group.label)
		if group.label == "." {
			label = "."
		}
		fmt.Fprintf(w, "%s/\n", label)

		seenDirs := make(map[string]bool)

		for _, relPath := range group.files {
			// Print directory hierarchy
			parts := strings.Split(relPath, string(filepath.Separator))
			accum := ""
			for i := 0; i < len(parts)-1; i++ {
				if accum != "" {
					accum += string(filepath.Separator)
				}
				accum += parts[i]

				if !seenDirs[accum] {
					seenDirs[accum] = true
					depth := i + 1
					fmt.Fprintf(w, "%s%s/\n", strings.Repeat("-", depth), parts[i])
				}
			}

			// Print file
			depth := len(parts)
			fmt.Fprintf(w, "%s%s\n", strings.Repeat("-", depth), parts[len(parts)-1])
		}
	}
}

func copyToClipboard(data []byte) error {
	// Detect clipboard command based on platform
	var cmd *exec.Cmd

	// Try xclip (Linux X11)
	if _, err := exec.LookPath("xclip"); err == nil {
		cmd = exec.Command("xclip", "-selection", "clipboard")
	} else if _, err := exec.LookPath("pbcopy"); err == nil {
		// macOS
		cmd = exec.Command("pbcopy")
	} else if _, err := exec.LookPath("clip.exe"); err == nil {
		// Windows
		cmd = exec.Command("clip.exe")
	} else if _, err := exec.LookPath("wl-copy"); err == nil {
		// Wayland
		cmd = exec.Command("wl-copy")
	} else {
		return fmt.Errorf("no clipboard command found (tried xclip, wl-copy, pbcopy, clip.exe)")
	}

	cmd.Stdin = bytes.NewReader(data)
	return cmd.Run()
}

