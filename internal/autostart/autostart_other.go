//go:build !darwin && !windows

package autostart

import (
	"os"
	"path/filepath"
)

func desktopPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "autostart")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "tomatick.desktop"), nil
}

func install() error {
	exe, err := executablePath()
	if err != nil {
		return err
	}
	path, err := desktopPath()
	if err != nil {
		return err
	}
	body := `[Desktop Entry]
Type=Application
Name=Tomatick
Comment=Menu bar timer, stopwatch, alarm and pomodoro
Exec=` + exe + `
X-GNOME-Autostart-enabled=true
Terminal=false
Categories=Utility;
`
	return os.WriteFile(path, []byte(body), 0o644)
}

func uninstall() error {
	path, err := desktopPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
