package clipcat

import (
	"fmt"
	"os"
	"strings"
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

func ParseArgs() *Config {
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