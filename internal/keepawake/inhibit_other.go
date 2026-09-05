//go:build !darwin && !windows

package keepawake

import (
	"os/exec"
	"time"
)

func inhibit() (func(), error) {
	if _, err := exec.LookPath("systemd-inhibit"); err == nil {
		cmd := exec.Command("systemd-inhibit",
			"--what=idle:sleep",
			"--who=Tomatick",
			"--why=Keep awake",
			"--mode=block",
			"sleep", "infinity",
		)
		if err := cmd.Start(); err != nil {
			return nil, err
		}
		return func() {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}, nil
	}
	// Fallback: xdg-screensaver on X11.
	if _, err := exec.LookPath("xdg-screensaver"); err == nil {
		_ = exec.Command("xdg-screensaver", "suspend", "1").Start()
		return func() {
			_ = exec.Command("xdg-screensaver", "resume", "1").Run()
		}, nil
	}
	_ = time.Second // keep import used if both helpers missing
	return func() {}, nil
}
