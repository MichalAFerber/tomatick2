package assets

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed *.png *.wav tomatick.icns
var FS embed.FS

// PNG returns the named PNG from the embedded asset set, or nil if missing.
func PNG(name string) []byte {
	b, err := FS.ReadFile(name)
	if err != nil {
		return nil
	}
	return b
}

// DingWAV is a short bundled chime used when no system sound is available.
func DingWAV() []byte {
	b, err := FS.ReadFile("ding.wav")
	if err != nil {
		return nil
	}
	return b
}

// IdleIcon is the static menu-bar / tray icon for a theme (red, white, black).
func IdleIcon(theme string) []byte {
	if b := PNG("mb_" + theme + ".png"); len(b) > 0 {
		return b
	}
	return PNG("mb_red.png")
}

// ShakeFrames returns the alarm-animation frames for a theme, in order.
func ShakeFrames(theme string) [][]byte {
	var frames [][]byte
	for i := 0; ; i++ {
		b := PNG(fmt.Sprintf("mb_%s_%d.png", theme, i))
		if len(b) == 0 {
			break
		}
		frames = append(frames, b)
	}
	if len(frames) == 0 && theme != "red" {
		return ShakeFrames("red")
	}
	return frames
}

// Names lists embedded file names (for tests / debugging).
func Names() []string {
	var names []string
	_ = fs.WalkDir(FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		names = append(names, path)
		return nil
	})
	return names
}
