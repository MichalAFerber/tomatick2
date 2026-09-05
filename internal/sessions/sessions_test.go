package sessions

import (
	"testing"
)

func TestParseDurationOK(t *testing.T) {
	cases := []struct {
		text string
		want int
	}{
		{"25m", 1500},
		{"1h30m", 5400},
		{"90s", 90},
		{"1h", 3600},
		{"45", 2700},
		{"2:30", 150},
		{"1:00:00", 3600},
		{"1h 30m 15s", 5415},
	}
	for _, c := range cases {
		got, err := ParseDuration(c.text)
		if err != nil {
			t.Fatalf("ParseDuration(%q): %v", c.text, err)
		}
		if got != c.want {
			t.Errorf("ParseDuration(%q) = %d, want %d", c.text, got, c.want)
		}
	}
}

func TestParseDurationBad(t *testing.T) {
	for _, text := range []string{"", "abc", "h", "1x"} {
		if _, err := ParseDuration(text); err == nil {
			t.Errorf("ParseDuration(%q) expected error", text)
		}
	}
}

func TestFormatClock(t *testing.T) {
	if FormatClock(5) != "0:05" {
		t.Errorf("got %q", FormatClock(5))
	}
	if FormatClock(75) != "1:15" {
		t.Errorf("got %q", FormatClock(75))
	}
	if FormatClock(3661) != "1:01:01" {
		t.Errorf("got %q", FormatClock(3661))
	}
	if FormatClock(-5) != "0:00" {
		t.Errorf("got %q", FormatClock(-5))
	}
}

func TestTimerCountsDownAndCompletes(t *testing.T) {
	tm := NewTimer(3, "")
	if ev := tm.Tick(1); len(ev) != 0 {
		t.Fatalf("unexpected events: %v", ev)
	}
	if tm.Remaining != 2 {
		t.Fatalf("remaining = %d", tm.Remaining)
	}
	tm.Tick(1)
	events := tm.Tick(1)
	if tm.Remaining != 0 || tm.State() != Done {
		t.Fatalf("remaining=%d state=%s", tm.Remaining, tm.State())
	}
	if len(events) == 0 || events[0].Action != "completed" {
		t.Fatalf("events = %+v", events)
	}
	if events[0].DurationS == nil || *events[0].DurationS != 3 {
		t.Fatalf("duration = %v", events[0].DurationS)
	}
}

func TestTimerPauseBlocksTick(t *testing.T) {
	tm := NewTimer(10, "")
	tm.TogglePause()
	if tm.State() != Paused {
		t.Fatalf("state = %s", tm.State())
	}
	tm.Tick(1)
	if tm.Remaining != 10 {
		t.Fatalf("remaining changed while paused: %d", tm.Remaining)
	}
	tm.TogglePause()
	tm.Tick(1)
	if tm.Remaining != 9 {
		t.Fatalf("remaining = %d", tm.Remaining)
	}
}

func TestTimerReset(t *testing.T) {
	tm := NewTimer(10, "")
	for i := 0; i < 5; i++ {
		tm.Tick(1)
	}
	if tm.Remaining != 5 {
		t.Fatalf("remaining = %d", tm.Remaining)
	}
	tm.Reset()
	if tm.Remaining != 10 || tm.State() != Running {
		t.Fatalf("after reset remaining=%d state=%s", tm.Remaining, tm.State())
	}
}

func TestStopwatchCountsUpAndLaps(t *testing.T) {
	s := NewStopwatch("")
	for i := 0; i < 5; i++ {
		s.Tick(1)
	}
	if s.Elapsed != 5 {
		t.Fatalf("elapsed = %d", s.Elapsed)
	}
	if s.Lap() != 5 {
		t.Fatalf("lap 1")
	}
	s.Tick(1)
	if s.Lap() != 6 {
		t.Fatalf("lap 2")
	}
	if len(s.Laps) != 2 || s.Laps[0] != 5 || s.Laps[1] != 6 {
		t.Fatalf("laps = %v", s.Laps)
	}
}

var pomoConfig = PomodoroConfig{
	WorkMinutes:           25,
	ShortBreakMinutes:     5,
	LongBreakMinutes:      15,
	CyclesBeforeLongBreak: 4,
	AutoStartNext:         true,
}

func drain(s Session, seconds int) []Event {
	var out []Event
	for i := 0; i < seconds; i++ {
		out = append(out, s.Tick(1)...)
	}
	return out
}

func TestPomodoroWorkThenShortBreak(t *testing.T) {
	p := NewPomodoro(pomoConfig, "")
	if p.Phase != Work {
		t.Fatalf("phase = %s", p.Phase)
	}
	events := drain(p, 25*60)
	found := false
	for _, e := range events {
		if e.Action == "phase_change" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected phase_change")
	}
	if p.Phase != ShortBreak {
		t.Fatalf("phase = %s", p.Phase)
	}
	if p.CompletedWork != 1 {
		t.Fatalf("completed = %d", p.CompletedWork)
	}
}

func TestPomodoroLongBreakAfterFourCycles(t *testing.T) {
	p := NewPomodoro(pomoConfig, "")
	phases := []string{p.Phase}
	for i := 0; i < 4; i++ {
		drain(p, p.Remaining)
		phases = append(phases, p.Phase)
		drain(p, p.Remaining)
		phases = append(phases, p.Phase)
	}
	found := false
	for _, ph := range phases {
		if ph == LongBreak {
			found = true
		}
	}
	if !found {
		t.Fatalf("long break not in %v", phases)
	}
}

func TestPomodoroSkipPhase(t *testing.T) {
	p := NewPomodoro(pomoConfig, "")
	events := p.SkipPhase()
	if events[0].Action != "phase_change" {
		t.Fatalf("action = %s", events[0].Action)
	}
	if events[0].Details["skipped"] != true {
		t.Fatal("expected skipped")
	}
	if p.Phase != ShortBreak {
		t.Fatalf("phase = %s", p.Phase)
	}
}

func TestPomodoroNoAutostartPausesAfterPhase(t *testing.T) {
	cfg := pomoConfig
	cfg.AutoStartNext = false
	p := NewPomodoro(cfg, "")
	drain(p, p.Remaining)
	if p.Phase != ShortBreak {
		t.Fatalf("phase = %s", p.Phase)
	}
	if p.State() != Paused {
		t.Fatalf("state = %s", p.State())
	}
}
