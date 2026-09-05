// Package autostart enables or disables launch-at-login.
package autostart

import (
	"os"
	"path/filepath"
	"strings"
)

// SetEnabled installs or removes the platform autostart entry.
func SetEnabled(enabled bool) error {
	if enabled {
		return install()
	}
	return uninstall()
}

func executablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

// bundlePath returns the .app path when running from a macOS app bundle.
func bundlePath(exe string) string {
	p := exe
	for {
		if strings.HasSuffix(p, ".app") {
			return p
		}
		next := filepath.Dir(p)
		if next == p {
			return ""
		}
		p = next
	}
}
