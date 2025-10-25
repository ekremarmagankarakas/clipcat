package unit_test

import (
	"clipcat/internal/clipboard"
	"os"
	"strings"
	"testing"
)

// Test clipboard functionality that doesn't require mocking system commands
func TestCopyToClipboard_Basic(t *testing.T) {
	// Skip if no clipboard is available (common in CI)
	if os.Getenv("CLIPCAT_INTEGRATION_TEST") != "1" {
		t.Skip("Set CLIPCAT_INTEGRATION_TEST=1 to test actual clipboard functionality")
	}
	
	testData := []byte("ClipCat test data")
	err := clipboard.CopyToClipboard(testData)
	
	// We can't easily verify the clipboard contents, but we can test it doesn't crash
	if err != nil && !isKnownClipboardError(err) {
		t.Errorf("Unexpected clipboard error: %v", err)
	}
}

func TestCopyToClipboard_EmptyData(t *testing.T) {
	if os.Getenv("CLIPCAT_INTEGRATION_TEST") != "1" {
		t.Skip("Set CLIPCAT_INTEGRATION_TEST=1 to test actual clipboard functionality")
	}
	
	testData := []byte("")
	err := clipboard.CopyToClipboard(testData)
	
	if err != nil && !isKnownClipboardError(err) {
		t.Errorf("Unexpected clipboard error with empty data: %v", err)
	}
}

func TestCopyToClipboard_LargeData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large data test in short mode")
	}
	if os.Getenv("CLIPCAT_INTEGRATION_TEST") != "1" {
		t.Skip("Set CLIPCAT_INTEGRATION_TEST=1 to test actual clipboard functionality")
	}
	
	// Create large test data (10KB)
	largeData := make([]byte, 10240)
	for i := range largeData {
		largeData[i] = byte('A' + (i % 26))
	}
	
	err := clipboard.CopyToClipboard(largeData)
	
	if err != nil && !isKnownClipboardError(err) {
		t.Errorf("Unexpected clipboard error with large data: %v", err)
	}
}

// Helper to identify known/expected clipboard errors
func isKnownClipboardError(err error) bool {
	errStr := err.Error()
	knownErrors := []string{
		"no clipboard command found",
		"exit status",
		"not found",
		"permission denied",
		"no display",
	}
	
	for _, known := range knownErrors {
		if strings.Contains(errStr, known) {
			return true
		}
	}
	return false
}