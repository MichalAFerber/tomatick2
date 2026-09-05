package assets

import "testing"

func TestIdleAndShakeFrames(t *testing.T) {
	for _, theme := range []string{"red", "white", "black"} {
		if len(IdleIcon(theme)) == 0 {
			t.Fatalf("missing idle icon for %s", theme)
		}
		frames := ShakeFrames(theme)
		if len(frames) == 0 {
			t.Fatalf("missing shake frames for %s", theme)
		}
	}
	if len(DingWAV()) == 0 {
		t.Fatal("missing ding.wav")
	}
	if len(PNG("about.png")) == 0 {
		t.Fatal("missing about.png")
	}
}
