package clipboard

import (
	"bytes"
	"fmt"
	"os/exec"
)

func CopyToClipboard(data []byte) error {
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