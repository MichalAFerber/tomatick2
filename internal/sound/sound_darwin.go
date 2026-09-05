//go:build darwin

package sound

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const systemSounds = "/System/Library/Sounds"

func platformSoundPath(name string) string {
	candidate := filepath.Join(systemSounds, name+".aiff")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}

func platformSoundNames() []string {
	entries, err := os.ReadDir(systemSounds)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.HasSuffix(strings.ToLower(n), ".aiff") {
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
	return exec.Command("afplay", path), nil
}
