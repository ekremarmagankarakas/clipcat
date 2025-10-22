# ClipCat 📋

A powerful command-line tool to concatenate files with headers and copy to clipboard. Perfect for sharing code context with AI assistants, creating documentation, or compiling project overviews.

## ✨ Features

* 📁 **Flexible Input**: Single files, directories (recursive), or glob patterns
* 🚫 **Smart Exclusions**: Full `.gitignore` semantics + custom glob patterns
  *(now: **directory excludes require a trailing `/`**)*
* 🌲 **Tree View**: Optional file hierarchy visualization
* 🧠 **Case-insensitive matching**: `-i/--ignore-case` for patterns and globs
* 📋 **Cross-Platform Clipboard**: Auto-detects `xclip`, `wl-copy`, `pbcopy`, or `clip.exe`
* ⚡ **Fast**: Single binary with no runtime dependencies
* 🎯 **Zero Config**: Works out of the box

## 📦 Installation

### Pre-built Binaries

Download from Releases:

```bash
# Linux
wget https://github.com/ekremarmagankarakas/clipcat/releases/latest/download/clipcat-linux-amd64
chmod +x clipcat-linux-amd64
sudo mv clipcat-linux-amd64 /usr/local/bin/clipcat

# macOS (Intel)
wget https://github.com/ekremarmagankarakas/clipcat/releases/latest/download/clipcat-darwin-amd64
chmod +x clipcat-darwin-amd64
sudo mv clipcat-darwin-amd64 /usr/local/bin/clipcat

# macOS (Apple Silicon)
wget https://github.com/ekremarmagankarakas/clipcat/releases/latest/download/clipcat-darwin-arm64
chmod +x clipcat-darwin-arm64
sudo mv clipcat-darwin-arm64 /usr/local/bin/clipcat
```

### From Source

Requires Go 1.21 or later:

```bash
go install github.com/ekremarmagankarakas/clipcat@latest
```

Or build manually:

```bash
git clone https://github.com/ekremarmagankarakas/clipcat.git
cd clipcat
make build
make install
```

### System Requirements

**One** of the following clipboard commands:

* Linux X11: `xclip` (`sudo apt install xclip`)
* Linux Wayland: `wl-copy` (`sudo apt install wl-clipboard`)
* macOS: `pbcopy` (built-in)
* Windows: `clip.exe` (built-in)

## 🚀 Quick Start

```bash
# Copy a single file
clipcat README.md

# Copy entire directory recursively
clipcat src/

# Find files matching a pattern (quote to avoid shell expansion)
clipcat '*test*.go'

# Respect .gitignore (negations supported)
clipcat . --exclude-from .gitignore

# Show a tree first
clipcat src/ -t

# Multiple sources with custom exclusions
clipcat frontend/ backend/ -e 'node_modules/' -e '*.log' -t
```

## 📖 Usage

```
clipcat [OPTIONS] <path1> [<path2> ...]

Options:
  -e, --exclude PATTERN     Exclude glob pattern (repeatable)
      --exclude-from FILE   Read patterns from FILE with full .gitignore semantics (repeatable)
  -i, --ignore-case         Make glob pattern matching case-insensitive
  -t, --tree                Prepend a FILE HIERARCHY section
      --only-tree           Copy only the FILE HIERARCHY (no file contents)
  -p, --print               Also print to stdout
  -h, --help                Show help
```

### Input Types

1. **Single file**: `clipcat main.go`
2. **Directory** (recursive): `clipcat src/`
3. **Glob pattern**: `clipcat '*checkin*'` (searches the whole tree)
4. **Mixed**: `clipcat README.md src/ '*test*'`

### Pattern Matching Semantics (important!)

* **Path-aware vs basename-only**

  * If your pattern **contains a path separator** (`/` or `\`), it matches against the **relative path** (e.g., `src/*.go`).
  * If your pattern **has no separator**, it matches the **basename** only (e.g., `*.go`, `README.md`).

* **Directory excludes must end with `/`**

  * `-e node_modules/` → excludes any directory named `node_modules` (and all its contents)
  * `-e build/` → excludes `build` directories
  * `-e clipcat/` → excludes directories named `clipcat`
  * `-e clipcat` (no slash) → **only files** named `clipcat` (e.g., built binary), **not** the directory

* **Case-insensitive option**

  * Add `-i` / `--ignore-case` to make exclude/collect globs case-insensitive:

    ```bash
    clipcat . -i -e '*.MD' -e 'docs/'   # matches README.MD, Docs/, etc.
    ```

* **.gitignore support**

  * `--exclude-from FILE` uses full `.gitignore` semantics:

    * Negation: `!keep.txt`
    * Root anchored: `/dist` vs `dist`
    * Directory markers: `node_modules/`
    * Deep wildcards: `**/tests/**/*.go`
    * Comments & blanks are handled by the library

**Combine both:**

```bash
clipcat . --exclude-from .gitignore -e '*.bak' -e 'temp/'
```

### Tree View

Show a file hierarchy before file contents:

```bash
clipcat src/ -t
```

Output (example):

```
==============
FILE HIERARCHY
==============

src/
-components/
--Button.tsx
--Input.tsx
-utils/
--format.ts

==================
/path/to/src/components/Button.tsx
==================

[file contents...]
```

## 💡 Common Use Cases

### Share Code with AI

```bash
# Copy all Python files except tests and caches
clipcat '*.py' -e '*test*.py' -e '__pycache__/' -t
```

### Project Overview

```bash
# Show only the file hierarchy, respecting .gitignore
clipcat . --exclude-from .gitignore --only-tree -p
```

### Find Specific Files

```bash
# All files with "config" in the name (any depth)
clipcat '*config*' --exclude-from .gitignore -t
```

### Documentation

```bash
# All markdown files
clipcat '*.md' -t -p > documentation.txt
```

### Code Review Prep

```bash
# All files in feature dirs, skip build artifacts
clipcat src/feature/ tests/feature/ -e 'dist/' -e 'build/' -t
```

## 🛠️ Development

### Prerequisites

* Go 1.21+
* Make (optional)

### Setup

```bash
git clone https://github.com/ekremarmagankarakas/clipcat.git
cd clipcat
go mod download
```

### Build

```bash
# Using Make
make build

# Or directly
go build -o clipcat clipcat.go
```

### Test

```bash
# All tests
make test

# With coverage
make test-coverage

# Watch mode (requires entr)
make watch

# Race detection
make test-race
```

### Makefile Commands

```bash
make build              # Build binary
make test               # Run all tests
make test-unit          # Unit tests only
make test-integration   # Integration tests only
make test-coverage      # Generate coverage report
make install            # Install to ~/.local/bin
make clean              # Remove build artifacts
make fmt                # Format code
make lint               # Run linter
make help               # Show all commands
```

## 📊 Test Coverage

Current coverage: **~62%**

```bash
# Generate HTML coverage report
make test-coverage
# Opens coverage.html in browser
```

## 📝 Output Format

Files are formatted with headers:

```
====================
/absolute/path/to/file.go
====================

[file contents]

====================
/absolute/path/to/next.go
====================

[file contents]
```

Unreadable files show `[unreadable]` instead of contents.

## 🔧 Troubleshooting

### “no clipboard command found”

Install a clipboard tool:

* **Linux X11**: `sudo apt install xclip`
* **Linux Wayland**: `sudo apt install wl-clipboard`
* **macOS/Windows**: Built-in

### “No files matched after applying excludes”

* Remember: **directory excludes must end with `/`**.

  * `-e clipcat/` excludes directories named `clipcat`;
  * `-e clipcat` excludes only files named `clipcat`.
* If running from a **parent directory**, `-e clipcat/` will exclude the repo folder itself.
  Run inside the repo (`cd clipcat && clipcat . ...`) or scope the pattern (e.g., `-e '*/clipcat/'`).

### Exclusions not working

* `.gitignore` patterns are resolved relative to your **current working directory**.
* Quote globs to prevent the shell from expanding them first:

  ```bash
  clipcat '*test*'   # ✅
  clipcat *test*     # ❌ shell expands before clipcat sees it
  ```

## 🤝 Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for new functionality
4. Ensure all tests pass (`make test`)
5. Format code (`make fmt`)
6. Commit changes (`git commit -m 'Add amazing feature'`)
7. Push to branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

### Guidelines

* Write tests for new features
* Maintain or improve code coverage
* Follow Go conventions
* Update README if adding user-facing features

## 🙏 Acknowledgments

* [go-gitignore](https://github.com/sabhiram/go-gitignore) — gitignore pattern matching
* Inspired by the need to share code context with AI assistants

## 📮 Contact

* **Issues**: [https://github.com/ekremarmagankarakas/clipcat/issues](https://github.com/ekremarmagankarakas/clipcat/issues)
* **Discussions**: [https://github.com/ekremarmagankarakas/clipcat/discussions](https://github.com/ekremarmagankarakas/clipcat/discussions)

---

Made with ❤️ for developers who love efficiency

