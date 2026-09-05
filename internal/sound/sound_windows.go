//go:build windows

package sound

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func mediaDir() string {
	root := os.Getenv("WINDIR")
	if root == "" {
		root = `C:\Windows`
	}
	return filepath.Join(root, "Media")
}

func platformSoundPath(name string) string {
	dir := mediaDir()
	for _, ext := range []string{".wav", ".WAV"} {
		c := filepath.Join(dir, name+ext)
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	// Friendly aliases for the default "Glass" name from the Python app.
	aliases := map[string]string{
		"Glass":    "Windows Notify.wav",
		"Ping":     "Windows Ding.wav",
		"Sosumi":   "Windows Exclamation.wav",
		"Tink":     "Windows Ding.wav",
		"Ding":     "Windows Ding.wav",
		"Hero":     "Windows Notify Calendar.wav",
		"Submarine": "Windows Notify Messaging.wav",
	}
	if alt, ok := aliases[name]; ok {
		c := filepath.Join(dir, alt)
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func platformSoundNames() []string {
	entries, err := os.ReadDir(mediaDir())
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.EqualFold(filepath.Ext(n), ".wav") {
			names = append(names, strings.TrimSuffix(n, filepath.Ext(n)))
		}
	}
	sort.Strings(names)
	return names
}

func playCommand(path string) (*exec.Cmd, error) {
	if path == "" {
		return nil, os.ErrNotExist
	}
	// SoundPlayer.PlaySync blocks until the sound finishes — right for looping.
	ps := `(New-Object Media.SoundPlayer ` + "'" + strings.ReplaceAll(path, "'", "''") + "'" + `).PlaySync()`
	return exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps), nil
}
