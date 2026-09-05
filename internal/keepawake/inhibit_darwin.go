//go:build darwin

package keepawake

import (
	"os"
	"os/exec"
	"strconv"
)

func inhibit() (func(), error) {
	cmd := exec.Command("caffeinate", "-d", "-i", "-s", "-w", strconv.Itoa(os.Getpid()))
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}, nil
}
