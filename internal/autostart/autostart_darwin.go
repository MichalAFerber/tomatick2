//go:build darwin

package autostart

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/MichalAFerber/tomatick2/internal/version"
)

func plistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, version.BundleID+".plist"), nil
}

func programArguments() ([]string, error) {
	exe, err := executablePath()
	if err != nil {
		return nil, err
	}
	if app := bundlePath(exe); app != "" {
		return []string{"/usr/bin/open", "-g", app}, nil
	}
	return []string{exe}, nil
}

func install() error {
	args, err := programArguments()
	if err != nil {
		return err
	}
	path, err := plistPath()
	if err != nil {
		return err
	}
	var argXML strings.Builder
	for _, a := range args {
		argXML.WriteString("    <string>")
		argXML.WriteString(xmlEscape(a))
		argXML.WriteString("</string>\n")
	}
	body := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>%s</string>
  <key>ProgramArguments</key>
  <array>
%s  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <false/>
  <key>ProcessType</key>
  <string>Interactive</string>
</dict>
</plist>
`, xmlEscape(version.BundleID), argXML.String())
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return err
	}
	_ = exec.Command("launchctl", "load", path).Run()
	return nil
}

func uninstall() error {
	path, err := plistPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		_ = exec.Command("launchctl", "unload", path).Run()
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove launch agent: %w", err)
		}
	}
	return nil
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
