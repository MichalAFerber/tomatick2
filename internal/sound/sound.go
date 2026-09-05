// Package sound plays named cues and looping alarm rings.
package sound

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/MichalAFerber/tomatick2/internal/assets"
)

// Player plays a named sound, optionally looping until Stop.
type Player struct {
	mu   sync.Mutex
	stop chan struct{}
	cmd  *exec.Cmd
}

// Play starts a named sound. If loop is true it repeats until Stop.
func (p *Player) Play(name string, loop bool) {
	p.Stop()
	p.mu.Lock()
	p.stop = make(chan struct{})
	stop := p.stop
	p.mu.Unlock()

	go func() {
		for {
			p.playOnce(name)
			if !loop {
				return
			}
			select {
			case <-stop:
				return
			default:
			}
		}
	}()
}

// Stop ends any in-flight playback.
func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stop != nil {
		close(p.stop)
		p.stop = nil
	}
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
		p.cmd = nil
	}
}

func (p *Player) playOnce(name string) {
	path := resolve(name)
	cmd, err := playCommand(path)
	if err != nil || cmd == nil {
		return
	}
	p.mu.Lock()
	p.cmd = cmd
	p.mu.Unlock()
	_ = cmd.Run()
	p.mu.Lock()
	if p.cmd == cmd {
		p.cmd = nil
	}
	p.mu.Unlock()
}

func resolve(name string) string {
	if name == "" {
		name = "Glass"
	}
	if p := platformSoundPath(name); p != "" {
		return p
	}
	return bundledDing()
}

var dingOnce sync.Once
var dingPath string

func bundledDing() string {
	dingOnce.Do(func() {
		dir, err := os.MkdirTemp("", "tomatick-sound")
		if err != nil {
			return
		}
		path := filepath.Join(dir, "ding.wav")
		if err := os.WriteFile(path, assets.DingWAV(), 0o644); err != nil {
			return
		}
		dingPath = path
	})
	return dingPath
}

// Available returns the names the settings UI can offer.
func Available() []string {
	if names := platformSoundNames(); len(names) > 0 {
		return names
	}
	return []string{"Ding", "Glass", "Ping", "Basso", "Blow", "Bottle", "Funk", "Hero",
		"Morse", "Pop", "Purr", "Sosumi", "Submarine", "Tink"}
}
