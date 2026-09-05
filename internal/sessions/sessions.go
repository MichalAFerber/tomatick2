// Package sessions holds the Timer, Stopwatch, and Pomodoro models.
//
// All timekeeping is driven by an external 1-second tick so these types
// contain no timers or threads of their own and are fully unit-testable.
// Tick advances a session by seconds (default 1) and returns events the
// app layer translates into history rows, notifications, and sounds.
package sessions

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// Session states.
const (
	Running = "running"
	Paused  = "paused"
	Done    = "done"
)

// Pomodoro phases.
const (
	Work       = "work"
	ShortBreak = "short_break"
	LongBreak  = "long_break"
)

var idCounter atomic.Int64

func nextID() int {
	return int(idCounter.Add(1))
}

// Event is something noteworthy that happened during a tick or user action.
type Event struct {
	Action    string
	Kind      string
	Label     string
	Details   map[string]any
	DurationS *int
}

// Session is a tracked, tickable timer-like object.
type Session interface {
	ID() int
	Kind() string
	Icon() string
	Label() string
	State() string
	StartedAt() time.Time
	TogglePause() string
	Reset()
	Tick(seconds int) []Event
	TitleText() string
	MenuText() string
}

// PomodoroConfig is the duration/cycle settings for a Pomodoro session.
type PomodoroConfig struct {
	WorkMinutes           int  `json:"work_minutes"`
	ShortBreakMinutes     int  `json:"short_break_minutes"`
	LongBreakMinutes      int  `json:"long_break_minutes"`
	CyclesBeforeLongBreak int  `json:"cycles_before_long_break"`
	AutoStartNext         bool `json:"auto_start_next"`
}

var (
	bareInt   = regexp.MustCompile(`^\d+$`)
	unitParts = regexp.MustCompile(`(\d+)\s*([hms])`)
)

// ParseDuration parses a natural-language duration into seconds.
//
// Accepts forms like 25m, 1h30m, 90s, 1h, 45 (bare number means minutes),
// and mm:ss / hh:mm:ss.
func ParseDuration(text string) (int, error) {
	text = strings.TrimSpace(strings.ToLower(text))
	if text == "" {
		return 0, fmt.Errorf("empty duration")
	}

	if strings.Contains(text, ":") {
		parts := strings.Split(text, ":")
		nums := make([]int, len(parts))
		for i, p := range parts {
			n, err := strconv.Atoi(p)
			if err != nil {
				return 0, fmt.Errorf("bad time format: %q", text)
			}
			nums[i] = n
		}
		switch len(nums) {
		case 2:
			return nums[0]*60 + nums[1], nil
		case 3:
			return nums[0]*3600 + nums[1]*60 + nums[2], nil
		default:
			return 0, fmt.Errorf("bad time format: %q", text)
		}
	}

	if bareInt.MatchString(text) {
		n, _ := strconv.Atoi(text)
		return n * 60, nil
	}

	matches := unitParts.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return 0, fmt.Errorf("cannot parse duration: %q", text)
	}
	total := 0
	mult := map[string]int{"h": 3600, "m": 60, "s": 1}
	for _, m := range matches {
		n, _ := strconv.Atoi(m[1])
		total += n * mult[m[2]]
	}
	return total, nil
}

// FormatClock formats seconds as M:SS, or H:MM:SS once an hour is reached.
func FormatClock(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}
	h := seconds / 3600
	rem := seconds % 3600
	m := rem / 60
	s := rem % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

type base struct {
	id        int
	label     string
	state     string
	startedAt time.Time
}

func newBase(label string) base {
	return base{
		id:        nextID(),
		label:     label,
		state:     Running,
		startedAt: time.Now(),
	}
}

func (b *base) ID() int              { return b.id }
func (b *base) Label() string        { return b.label }
func (b *base) State() string        { return b.state }
func (b *base) StartedAt() time.Time { return b.startedAt }

func (b *base) pause() {
	if b.state == Running {
		b.state = Paused
	}
}

func (b *base) resume() {
	if b.state == Paused {
		b.state = Running
	}
}

func (b *base) TogglePause() string {
	switch b.state {
	case Running:
		b.pause()
		return "paused"
	case Paused:
		b.resume()
		return "resumed"
	}
	return ""
}

func menuText(icon, name, title, state string) string {
	suffix := ""
	if state == Paused {
		suffix = " (paused)"
	}
	return fmt.Sprintf("%s %s  %s%s", icon, name, title, suffix)
}

// Timer counts down from Duration seconds to zero.
type Timer struct {
	base
	Duration  int
	Remaining int
}

func NewTimer(duration int, label string) *Timer {
	return &Timer{
		base:      newBase(label),
		Duration:  duration,
		Remaining: duration,
	}
}

func (t *Timer) Kind() string { return "timer" }
func (t *Timer) Icon() string { return "⏱" }

func (t *Timer) Reset() {
	t.Remaining = t.Duration
	t.state = Running
}

func (t *Timer) Tick(seconds int) []Event {
	if t.state != Running {
		return nil
	}
	t.Remaining -= seconds
	if t.Remaining < 0 {
		t.Remaining = 0
	}
	if t.Remaining == 0 {
		t.state = Done
		d := t.Duration
		return []Event{{
			Action:    "completed",
			Kind:      t.Kind(),
			Label:     t.label,
			DurationS: &d,
		}}
	}
	return nil
}

func (t *Timer) TitleText() string { return FormatClock(t.Remaining) }

func (t *Timer) MenuText() string {
	name := t.label
	if name == "" {
		name = "Timer"
	}
	return menuText(t.Icon(), name, t.TitleText(), t.state)
}

// Stopwatch counts up indefinitely.
type Stopwatch struct {
	base
	Elapsed int
	Laps    []int
}

func NewStopwatch(label string) *Stopwatch {
	return &Stopwatch{base: newBase(label)}
}

func (s *Stopwatch) Kind() string { return "stopwatch" }
func (s *Stopwatch) Icon() string { return "⏲" }

func (s *Stopwatch) Reset() {
	s.Elapsed = 0
	s.Laps = nil
	s.state = Running
}

func (s *Stopwatch) Lap() int {
	s.Laps = append(s.Laps, s.Elapsed)
	return s.Elapsed
}

func (s *Stopwatch) Tick(seconds int) []Event {
	if s.state != Running {
		return nil
	}
	s.Elapsed += seconds
	return nil
}

func (s *Stopwatch) TitleText() string { return FormatClock(s.Elapsed) }

func (s *Stopwatch) MenuText() string {
	name := s.label
	if name == "" {
		name = "Stopwatch"
	}
	return menuText(s.Icon(), name, s.TitleText(), s.state)
}

// Pomodoro cycles through work / short break / long break phases.
type Pomodoro struct {
	base
	workS             int
	shortS            int
	longS             int
	cyclesBeforeLong  int
	autoStartNext     bool
	Phase             string
	CompletedWork     int
	Remaining         int
}

func NewPomodoro(cfg PomodoroConfig, label string) *Pomodoro {
	p := &Pomodoro{
		base:             newBase(label),
		workS:            cfg.WorkMinutes * 60,
		shortS:           cfg.ShortBreakMinutes * 60,
		longS:            cfg.LongBreakMinutes * 60,
		cyclesBeforeLong: cfg.CyclesBeforeLongBreak,
		autoStartNext:    cfg.AutoStartNext,
		Phase:            Work,
	}
	if p.cyclesBeforeLong < 1 {
		p.cyclesBeforeLong = 1
	}
	p.Remaining = p.workS
	return p
}

func (p *Pomodoro) Kind() string { return "pomodoro" }

func (p *Pomodoro) Icon() string {
	if p.Phase == Work {
		return "🍅"
	}
	return "☕"
}

func (p *Pomodoro) phaseDuration(phase string) int {
	switch phase {
	case Work:
		return p.workS
	case ShortBreak:
		return p.shortS
	case LongBreak:
		return p.longS
	}
	return p.workS
}

func (p *Pomodoro) nextPhase() string {
	if p.Phase == Work {
		if p.cyclesBeforeLong > 0 && p.CompletedWork%p.cyclesBeforeLong == 0 {
			return LongBreak
		}
		return ShortBreak
	}
	return Work
}

func (p *Pomodoro) enterPhase(phase string) {
	p.Phase = phase
	p.Remaining = p.phaseDuration(phase)
	if p.autoStartNext {
		p.state = Running
	} else {
		p.state = Paused
	}
}

func (p *Pomodoro) advance(skipped bool) []Event {
	finishing := p.Phase
	if finishing == Work {
		p.CompletedWork++
	}
	nxt := p.nextPhase()
	p.enterPhase(nxt)
	return []Event{{
		Action: "phase_change",
		Kind:   p.Kind(),
		Label:  p.label,
		Details: map[string]any{
			"from":                  finishing,
			"to":                    nxt,
			"skipped":               skipped,
			"completed_work_cycles": p.CompletedWork,
		},
	}}
}

// SkipPhase force-advances to the next phase.
func (p *Pomodoro) SkipPhase() []Event {
	return p.advance(true)
}

func (p *Pomodoro) Reset() {
	p.Phase = Work
	p.CompletedWork = 0
	p.Remaining = p.workS
	p.state = Running
}

func (p *Pomodoro) Tick(seconds int) []Event {
	if p.state != Running {
		return nil
	}
	p.Remaining -= seconds
	if p.Remaining < 0 {
		p.Remaining = 0
	}
	if p.Remaining == 0 {
		return p.advance(false)
	}
	return nil
}

func (p *Pomodoro) TitleText() string { return FormatClock(p.Remaining) }

func (p *Pomodoro) MenuText() string {
	name := p.label
	if name == "" {
		name = "Pomodoro"
	}
	phaseLabel := map[string]string{
		Work:       "work",
		ShortBreak: "break",
		LongBreak:  "long break",
	}[p.Phase]
	suffix := ""
	if p.state == Paused {
		suffix = " (paused)"
	}
	return fmt.Sprintf("%s %s · %s  %s%s", p.Icon(), name, phaseLabel, p.TitleText(), suffix)
}
