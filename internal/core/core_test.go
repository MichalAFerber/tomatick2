package core

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/MichalAFerber/tomatick2/internal/alarms"
	"github.com/MichalAFerber/tomatick2/internal/history"
	"github.com/MichalAFerber/tomatick2/internal/sessions"
	"github.com/MichalAFerber/tomatick2/internal/settings"
)

func testApp(t *testing.T) *App {
	t.Helper()
	t.Setenv("TOMATICK_SUPPORT_DIR", t.TempDir())
	s := &settings.Settings{Data: settings.Defaults()}
	h, err := history.Open(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = h.Close() })
	now := time.Date(2026, 6, 26, 8, 0, 0, 0, time.Local)
	return &App{
		Settings: s,
		History:  h,
		now:      func() time.Time { return now },
	}
}

func TestTimerCompletesAndRings(t *testing.T) {
	a := testApp(t)
	var notified string
	a.Hooks.Notify = func(title, _, _ string) { notified = title }
	a.Hooks.PlayAlarm = func(string, bool) {}
	a.StartTimer(2, "tea")
	a.Tick()
	if a.Alarming() {
		t.Fatal("should not ring yet")
	}
	res := a.Tick()
	if !res.Structural {
		t.Fatal("expected structural change")
	}
	if len(a.Sessions) != 0 {
		t.Fatalf("sessions left: %d", len(a.Sessions))
	}
	if !a.Alarming() {
		t.Fatal("expected ringing")
	}
	if notified != "Timer done" {
		t.Fatalf("notify = %q", notified)
	}
}

func TestPrimaryPinnedThenFallsBack(t *testing.T) {
	a := testApp(t)
	a.StartTimer(100, "a")
	a.StartTimer(100, "b")
	if a.Primary().Label() != "b" {
		t.Fatalf("primary = %s", a.Primary().Label())
	}
	a.Pin(a.Sessions[0].ID())
	if a.Primary().Label() != "a" {
		t.Fatalf("pinned = %s", a.Primary().Label())
	}
	a.StopSession(a.Sessions[0].ID())
	if a.Primary() == nil || a.Primary().Label() != "b" {
		t.Fatalf("fallback primary = %v", a.Primary())
	}
}

func TestAlarmFiresWhenDue(t *testing.T) {
	a := testApp(t)
	var titles []string
	a.Hooks.Notify = func(title, _, _ string) { titles = append(titles, title) }
	a.Hooks.PlayAlarm = func(string, bool) {}
	now := time.Date(2026, 6, 26, 8, 59, 0, 0, time.Local)
	a.now = func() time.Time { return now }

	al := &alarms.Alarm{
		ID: 1, Kind: alarms.Recurring, TimeStr: "09:00",
		DaysOfWeek: []int{4}, Enabled: true, Label: "standup",
	}
	al.ComputeNextFire(now)
	a.Alarms = []*alarms.Alarm{al}

	a.Tick()
	if a.Alarming() {
		t.Fatal("too early")
	}
	now = time.Date(2026, 6, 26, 9, 0, 0, 0, time.Local)
	a.Tick()
	if !a.Alarming() {
		t.Fatal("should be ringing")
	}
	if len(titles) == 0 || titles[0] != "Alarm" {
		t.Fatalf("titles = %v", titles)
	}
}

func TestSnoozeThenRefire(t *testing.T) {
	a := testApp(t)
	a.Hooks.PlayAlarm = func(string, bool) {}
	a.Hooks.StopAlarm = func() {}
	a.Hooks.Notify = func(string, string, string) {}
	a.Settings.Data["snooze_minutes"] = 1
	now := time.Date(2026, 6, 26, 9, 0, 0, 0, time.Local)
	a.now = func() time.Time { return now }

	al := &alarms.Alarm{ID: 1, Kind: alarms.OneShot, TimeStr: "09:00", DateStr: "2026-06-26", Enabled: true, Label: "x"}
	al.ComputeNextFire(now)
	a.Alarms = []*alarms.Alarm{al}
	a.Tick()
	if !a.Alarming() {
		t.Fatal("expected fire")
	}
	ringing := a.Firing[0].Alarm
	a.SnoozeAlarm(ringing)
	if a.Alarming() {
		t.Fatal("should be quiet after snooze")
	}
	if len(a.Snoozes) != 1 {
		t.Fatal("expected snooze")
	}
	now = now.Add(time.Minute)
	a.Tick()
	if !a.Alarming() {
		t.Fatal("snooze should have re-fired")
	}
}

func TestFocusDesiredDuringWork(t *testing.T) {
	a := testApp(t)
	var ran []string
	a.Hooks.RunFocus = func(name string) { ran = append(ran, name) }
	a.Settings.Data["focus_shortcut_on"] = "Focus On"
	a.Settings.Data["focus_shortcut_off"] = "Focus Off"
	a.Settings.Data["focus_during_work"] = true
	a.StartPomodoro("")
	if !a.FocusActive {
		t.Fatal("expected focus on")
	}
	if len(ran) != 1 || ran[0] != "Focus On" {
		t.Fatalf("ran = %v", ran)
	}
	p := a.Sessions[0].(*sessions.Pomodoro)
	p.SkipPhase()
	a.SyncFocus()
	if a.FocusActive {
		t.Fatal("focus should drop on break")
	}
	if ran[len(ran)-1] != "Focus Off" {
		t.Fatalf("ran = %v", ran)
	}
}

func TestImportExportSettings(t *testing.T) {
	a := testApp(t)
	a.Settings.Data["default_sound"] = "Ping"
	path := filepath.Join(t.TempDir(), "s.json")
	if err := a.ExportSettings(path); err != nil {
		t.Fatal(err)
	}
	a.Settings.Data["default_sound"] = "Glass"
	n, err := a.ImportSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("nothing applied")
	}
	if a.Settings.DefaultSound() != "Ping" {
		t.Fatalf("sound = %s", a.Settings.DefaultSound())
	}
}
