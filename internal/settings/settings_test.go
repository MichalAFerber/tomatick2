package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultsPresent(t *testing.T) {
	s := &Settings{Data: Defaults()}
	if s.Pomodoro().WorkMinutes != 25 {
		t.Fatalf("work = %d", s.Pomodoro().WorkMinutes)
	}
	if s.SnoozeMinutes() != 9 {
		t.Fatal("snooze")
	}
	if s.LaunchAtLogin() {
		t.Fatal("launch")
	}
}

func TestDeepMergeBackfillsNewDefaults(t *testing.T) {
	s := &Settings{Data: deepMerge(Defaults(), map[string]any{
		"snooze_minutes": 5,
		"pomodoro":       map[string]any{"work_minutes": 50},
	})}
	if s.SnoozeMinutes() != 5 {
		t.Fatalf("snooze = %d", s.SnoozeMinutes())
	}
	if s.Pomodoro().WorkMinutes != 50 {
		t.Fatalf("work = %d", s.Pomodoro().WorkMinutes)
	}
	if s.Pomodoro().ShortBreakMinutes != 5 {
		t.Fatalf("short = %d", s.Pomodoro().ShortBreakMinutes)
	}
}

func TestSaveAndLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TOMATICK_SUPPORT_DIR", dir)
	s, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	s.Data["snooze_minutes"] = 3
	if err := s.Set("default_sound", "Ping"); err != nil {
		t.Fatal(err)
	}
	reloaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.SnoozeMinutes() != 3 {
		t.Fatalf("snooze = %d", reloaded.SnoozeMinutes())
	}
	if reloaded.DefaultSound() != "Ping" {
		t.Fatalf("sound = %s", reloaded.DefaultSound())
	}
	b, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]any
	if err := json.Unmarshal(b, &saved); err != nil {
		t.Fatal(err)
	}
	if saved["default_sound"] != "Ping" {
		t.Fatalf("file = %v", saved["default_sound"])
	}
}
