package output

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func WriteHeader(w io.Writer, path string) {
	bar := strings.Repeat("=", len(path))
	fmt.Fprintf(w, "%s\n%s\n%s\n\n", bar, path, bar)
}

func WriteFileContent(w io.Writer, path string) error {
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

func isGlobPattern(path string) bool {
	return strings.ContainsAny(path, "*?[")
}

func WriteTree(w io.Writer, roots []string, files []string) {
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