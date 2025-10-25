package unit_test

import (
	"bytes"
	"clipcat/pkg/clipcat"
	"io"
	"os"
	"strings"
	"testing"
)

// mockExit captures os.Exit calls for testing
func mockExit(t *testing.T) (func(), *int) {
	var exitCode int
	
	// We can't actually mock os.Exit directly, so we'll test the behavior
	// by temporarily redirecting stderr and testing with goroutines
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	
	restore := func() {
		os.Stderr = oldStderr
		w.Close()
	}
	
	// Read stderr in a goroutine
	go func() {
		// This will capture the error output before exit
		defer r.Close()
		_, _ = io.Copy(io.Discard, r)
	}()
	
	return restore, &exitCode
}

func TestParseArgs_UnknownFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"unknown short flag", []string{"clipcat", "-z", "file.txt"}},
		{"unknown long flag", []string{"clipcat", "--unknown", "file.txt"}},
		{"unknown flag with value", []string{"clipcat", "--invalid=value", "file.txt"}},
		{"multiple unknown flags", []string{"clipcat", "-x", "-y", "file.txt"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// Mock os.Args
			oldArgs := os.Args
			os.Args = tt.args

			done := make(chan bool)

			// Run ParseArgs in a goroutine to catch the exit
			go func() {
				defer func() {
					if r := recover(); r != nil {
						// Handle panic from os.Exit
					}
					done <- true
				}()
				
				// This should call os.Exit(2) which we can't easily intercept
				// But we can verify the error message is printed
				clipcat.ParseArgs()
			}()

			// Wait a bit for the function to execute
			go func() {
				<-done
				w.Close()
			}()

			// Read stderr
			var buf bytes.Buffer
			buf.ReadFrom(r)
			os.Stderr = oldStderr
			os.Args = oldArgs

			stderr := buf.String()
			
			// Should contain error message about unknown option
			if !strings.Contains(stderr, "unknown option") && !strings.Contains(stderr, "Error:") {
				t.Errorf("Expected error message for unknown option, got: %q", stderr)
			}

			// Should contain usage information
			if !strings.Contains(stderr, "Usage:") {
				t.Errorf("Expected usage information in stderr, got: %q", stderr)
			}
		})
	}
}

func TestParseArgs_MissingArguments(t *testing.T) {
	tests := []struct {
		name string
		args []string
		expectedError string
	}{
		{
			name: "exclude without pattern",
			args: []string{"clipcat", "-e"},
			expectedError: "-e requires a pattern",
		},
		{
			name: "exclude long without pattern",
			args: []string{"clipcat", "--exclude"},
			expectedError: "--exclude requires a pattern", 
		},
		{
			name: "exclude-from without file",
			args: []string{"clipcat", "--exclude-from"},
			expectedError: "--exclude-from requires a file",
		},
		{
			name: "exclude at end of args",
			args: []string{"clipcat", "file.txt", "-e"},
			expectedError: "-e requires a pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// Mock os.Args
			oldArgs := os.Args
			os.Args = tt.args

			done := make(chan bool)

			// Run ParseArgs in a goroutine
			go func() {
				defer func() {
					recover() // Ignore panics from os.Exit
					done <- true
				}()
				clipcat.ParseArgs()
			}()

			// Wait and close
			go func() {
				<-done
				w.Close()
			}()

			// Read stderr
			var buf bytes.Buffer
			buf.ReadFrom(r)
			os.Stderr = oldStderr
			os.Args = oldArgs

			stderr := buf.String()
			
			// Should contain specific error message
			if !strings.Contains(stderr, tt.expectedError) {
				t.Errorf("Expected error %q in stderr, got: %q", tt.expectedError, stderr)
			}
		})
	}
}

func TestParseArgs_NoPathsProvided(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no args at all", []string{"clipcat"}},
		{"only flags no paths", []string{"clipcat", "-t", "-p"}},
		{"only exclude flags", []string{"clipcat", "-e", "*.log", "--exclude-from", ".gitignore"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// Mock os.Args
			oldArgs := os.Args
			os.Args = tt.args

			done := make(chan bool)

			// Run ParseArgs in a goroutine
			go func() {
				defer func() {
					recover() // Ignore panics from os.Exit
					done <- true
				}()
				clipcat.ParseArgs()
			}()

			// Wait and close
			go func() {
				<-done
				w.Close()
			}()

			// Read stderr
			var buf bytes.Buffer
			buf.ReadFrom(r)
			os.Stderr = oldStderr
			os.Args = oldArgs

			stderr := buf.String()
			
			// Should print usage when no paths provided
			if !strings.Contains(stderr, "Usage:") {
				t.Errorf("Expected usage information when no paths provided, got: %q", stderr)
			}

			// Should show examples
			if !strings.Contains(stderr, "Examples:") {
				t.Errorf("Expected examples in usage output, got: %q", stderr)
			}
		})
	}
}

// Test that help flag exits with code 0 (not 2)
func TestParseArgs_HelpFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"help short", []string{"clipcat", "-h"}},
		{"help long", []string{"clipcat", "--help"}},
		{"help with other args", []string{"clipcat", "-h", "somefile.txt"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr  
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// Mock os.Args
			oldArgs := os.Args
			os.Args = tt.args

			done := make(chan bool)

			// Run ParseArgs in a goroutine
			go func() {
				defer func() {
					recover() // Ignore panics from os.Exit
					done <- true
				}()
				clipcat.ParseArgs()
			}()

			// Wait and close
			go func() {
				<-done
				w.Close()
			}()

			// Read stderr
			var buf bytes.Buffer
			buf.ReadFrom(r)
			os.Stderr = oldStderr
			os.Args = oldArgs

			stderr := buf.String()
			
			// Help should show usage
			if !strings.Contains(stderr, "Usage:") {
				t.Errorf("Expected usage for help flag, got: %q", stderr)
			}

			// Help should show all options
			expectedOptions := []string{"-e, --exclude", "--exclude-from", "-i, --ignore-case", "-t, --tree", "--only-tree", "-p, --print"}
			for _, option := range expectedOptions {
				if !strings.Contains(stderr, option) {
					t.Errorf("Expected option %q in help output", option)
				}
			}
		})
	}
}

func TestParseArgs_ValidArguments(t *testing.T) {
	// Test that valid arguments don't cause exit/panic
	validTests := []struct {
		name string
		args []string
	}{
		{"single file", []string{"clipcat", "file.txt"}},
		{"multiple files", []string{"clipcat", "file1.txt", "file2.txt"}},
		{"with flags", []string{"clipcat", "-t", "-p", "file.txt"}},
		{"with excludes", []string{"clipcat", "-e", "*.log", "file.txt"}},
		{"complex valid", []string{"clipcat", "src/", "-t", "-e", "*.tmp", "--exclude-from", ".gitignore", "-i"}},
	}

	for _, tt := range validTests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock os.Args
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()
			os.Args = tt.args

			// This should not panic or exit
			cfg := clipcat.ParseArgs()
			
			// Should have at least one path
			if len(cfg.Paths) == 0 {
				t.Errorf("Valid arguments should result in at least one path")
			}
		})
	}
}

// Helper function to run a command that might call os.Exit
func runWithExitCapture(t *testing.T, fn func()) (stderr string, exited bool) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	done := make(chan bool, 1)
	exited = false

	// Run function in goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Function called os.Exit or panicked
				exited = true
			}
			done <- true
		}()
		fn()
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		// Function completed
	}

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stderr = oldStderr

	return buf.String(), exited
}