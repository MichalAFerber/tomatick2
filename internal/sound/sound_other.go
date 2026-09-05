//go:build !darwin && !windows

package sound

import (
	"os"
	"os/exec"
)

func platformSoundPath(name string) string {
	return ""
}

func platformSoundNames() []string {
	return nil
}

func playCommand(path string) (*exec.Cmd, error) {
	if path == "" {
		return nil, os.ErrNotExist
	}
	for _, spec := range [][]string{
		{"paplay", path},
		{"aplay", "-q", path},
		{"ffplay", "-nodisp", "-autoexit", "-loglevel", "quiet", path},
		{"pw-play", path},
	} {
		if _, err := exec.LookPath(spec[0]); err == nil {
			return exec.Command(spec[0], spec[1:]...), nil
		}
	}
	return nil, os.ErrNotExist
}
