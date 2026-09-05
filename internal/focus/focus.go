// Package focus triggers a named macOS Shortcut for Focus / DND.
// Off macOS (and when the name is empty) it is a no-op.
package focus

import (
	"os/exec"
	"runtime"
	"strings"
)

// Run launches a Shortcut by name. Returns true if it was started.
func Run(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || runtime.GOOS != "darwin" {
		return false
	}
	if _, err := exec.LookPath("shortcuts"); err != nil {
		return false
	}
	cmd := exec.Command("shortcuts", "run", name)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return false
	}
	go func() { _ = cmd.Wait() }()
	return true
}
