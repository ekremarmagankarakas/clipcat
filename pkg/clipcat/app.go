package clipcat

import (
	"bytes"
	"clipcat/internal/clipboard"
	"clipcat/pkg/collector"
	"clipcat/pkg/exclude"
	"clipcat/pkg/output"
	"fmt"
	"os"
	"sort"
)

func Run(cfg *Config) error {
	// Build exclude matcher
	matcher, err := exclude.BuildMatcher(cfg.ExcludeFiles, cfg.Excludes, cfg.IgnoreCase)
	if err != nil {
		return fmt.Errorf("loading exclude patterns: %w", err)
	}

	// Collect all files
	files, err := collector.CollectFiles(cfg.Paths, matcher, cfg.IgnoreCase)
	if err != nil {
		return fmt.Errorf("collecting files: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no files matched after applying excludes")
	}

	// Sort for consistent output
	sort.Strings(files)

	// Build output
	var outputBuf bytes.Buffer

	if cfg.ShowTree {
		output.WriteHeader(&outputBuf, "FILE HIERARCHY")
		output.WriteTree(&outputBuf, cfg.Paths, files)
		outputBuf.WriteString("\n")
	}

	if !cfg.OnlyTree {
		for _, file := range files {
			output.WriteHeader(&outputBuf, file)
			if err := output.WriteFileContent(&outputBuf, file); err != nil {
				outputBuf.WriteString("[unreadable]\n")
			}
			outputBuf.WriteString("\n")
		}
	}

	// Copy to clipboard
	if err := clipboard.CopyToClipboard(outputBuf.Bytes()); err != nil {
		return fmt.Errorf("copying to clipboard: %w", err)
	}

	// Optionally print to stdout
	if cfg.PrintOut {
		os.Stdout.Write(outputBuf.Bytes())
	}

	// Success message
	if cfg.OnlyTree {
		fmt.Printf("Copied file hierarchy for %d files to clipboard.\n", len(files))
	} else {
		fmt.Printf("Copied %d files to clipboard.\n", len(files))
	}

	return nil
}