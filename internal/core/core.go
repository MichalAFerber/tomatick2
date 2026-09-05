// Package core is the platform-agnostic Tomatick runtime: sessions, alarms,
// history, and settings. The UI layer drives it with a 1-second Tick and
// user actions, and implements Hooks for sound, notifications, and focus.
package core

import (
	"encoding/json"
	"os"
	"time"

	"github.com/MichalAFerber/tomatick2/internal/alarms"
	"github.com/MichalAFerber/tomatick2/internal/history"
	"github.com/MichalAFerber/tomatick2/internal/sessions"
	"github.com/MichalAFerber/tomatick2/internal/settings"
)

// Firing is an alarm (or timer-done stand-in) currently ringing.
type Firing struct {
	Alarm *alarms.Alarm
	Title string // optional override, e.g. "⏱ tea · done"
}

// Snooze is a postponed re-fire.
type Snooze struct {
	FireAt time.Time
	Label  string
	Sound  string
}

// Hooks are side effects the UI / platform layer provides.
type Hooks struct {
	Notify    func(title, subtitle, message string)
	PlayAlarm func(sound string, loop bool)
	StopAlarm func()
	PlayCue   func(sound string)
	RunFocus  func(name string)
}

func (h Hooks) notify(title, subtitle, message string) {
	if h.Notify != nil {
		h.Notify(title, subtitle, message)
	}
}

func (h Hooks) playAlarm(sound string, loop bool) {
	if h.PlayAlarm != nil {
		h.PlayAlarm(sound, loop)
	}
}

func (h Hooks) stopAlarm() {
	if h.StopAlarm != nil {
		h.StopAlarm()
	}
}

func (h Hooks) playCue(sound string) {
	if h.PlayCue != nil {
		h.PlayCue(sound)
	}
}

func (h Hooks) runFocus(name string) {
	if h.RunFocus != nil {
		h.RunFocus(name)
	}
}

// TickResult tells the UI what changed this second.
type TickResult struct {
	Structural bool // menu should be rebuilt
	TitleDirty bool
}

// App is the live Tomatick state.
type App struct {
	Settings *settings.Settings
	History  *history.History
	Alarms   []*alarms.Alarm
	Sessions []sessions.Session
	Firing   []Firing
	Snoozes  []Snooze
	PinnedID int
	Hooks    Hooks

	KeepAwakeActive bool
	FocusActive     bool

	now func() time.Time
}

// Open loads settings and history from the support dir.
func Open() (*App, error) {
	s, err := settings.Load()
	if err != nil {
		return nil, err
	}
	hp, err := settings.HistoryPath()
	if err != nil {
		return nil, err
	}
	h, err := history.Open(hp)
	if err != nil {
		return nil, err
	}
	a := &App{
		Settings: s,
		History:  h,
		now:      time.Now,
	}
	a.Alarms = alarms.Load(s.AlarmsRaw(), a.now())
	return a, nil
}

// Close releases the history database.
func (a *App) Close() error {
	if a.History != nil {
		return a.History.Close()
	}
	return nil
}

func (a *App) clock() time.Time {
	if a.now != nil {
		return a.now()
	}
	return time.Now()
}

func (a *App) log(kind, action, label string, details map[string]any, durationS *int) {
	if a.History == nil {
		return
	}
	_, _ = a.History.LogEvent(kind, action, label, details, durationS, time.Time{})
}

func (a *App) persistAlarms() {
	raw := make([]any, 0, len(a.Alarms))
	for _, al := range a.Alarms {
		raw = append(raw, al.ToMap())
	}
	a.Settings.Data["alarms"] = raw
	_ = a.Settings.Save()
}

// AddSession inserts a session and pins it as primary.
func (a *App) AddSession(sess sessions.Session) {
	a.Sessions = append(a.Sessions, sess)
	a.PinnedID = sess.ID()
	a.SyncFocus()
}

// SessionByID looks up a live session.
func (a *App) SessionByID(id int) sessions.Session {
	for _, s := range a.Sessions {
		if s.ID() == id {
			return s
		}
	}
	return nil
}

// StartTimer starts a countdown.
func (a *App) StartTimer(seconds int, label string) {
	if seconds <= 0 {
		return
	}
	sess := sessions.NewTimer(seconds, label)
	a.AddSession(sess)
	a.log("timer", "started", label, nil, &seconds)
}

// StartStopwatch starts a count-up.
func (a *App) StartStopwatch(label string) {
	sess := sessions.NewStopwatch(label)
	a.AddSession(sess)
	a.log("stopwatch", "started", label, nil, nil)
}

// StartPomodoro starts a pomodoro using current settings.
func (a *App) StartPomodoro(label string) {
	sess := sessions.NewPomodoro(a.Settings.Pomodoro(), label)
	a.AddSession(sess)
	a.log("pomodoro", "started", label, nil, nil)
}

// StartPreset starts a timer from a named preset.
func (a *App) StartPreset(p settings.Preset) {
	a.StartTimer(p.Seconds, p.Label)
}

// TogglePause pauses or resumes a session.
func (a *App) TogglePause(id int) {
	s := a.SessionByID(id)
	if s == nil {
		return
	}
	if action := s.TogglePause(); action != "" {
		a.log(s.Kind(), action, s.Label(), nil, nil)
	}
	a.SyncFocus()
}

// ResetSession resets a timer/stopwatch.
func (a *App) ResetSession(id int) {
	s := a.SessionByID(id)
	if s == nil {
		return
	}
	s.Reset()
	a.log(s.Kind(), "reset", s.Label(), nil, nil)
	a.SyncFocus()
}

// SkipPhase advances a pomodoro.
func (a *App) SkipPhase(id int) {
	s := a.SessionByID(id)
	p, ok := s.(*sessions.Pomodoro)
	if !ok {
		return
	}
	for _, ev := range p.SkipPhase() {
		a.log(ev.Kind, ev.Action, ev.Label, ev.Details, ev.DurationS)
	}
	a.notifyPhase(p)
	a.SyncFocus()
}

// Lap records a stopwatch lap.
func (a *App) Lap(id int) {
	s := a.SessionByID(id)
	sw, ok := s.(*sessions.Stopwatch)
	if !ok {
		return
	}
	elapsed := sw.Lap()
	a.log(sw.Kind(), "lap", sw.Label(), map[string]any{"elapsed_s": elapsed}, nil)
}

// StopSession removes a session.
func (a *App) StopSession(id int) {
	s := a.SessionByID(id)
	if s == nil {
		return
	}
	a.log(s.Kind(), "stopped", s.Label(), nil, nil)
	a.removeSession(id)
}

// Pin makes a session the primary menu-bar display.
func (a *App) Pin(id int) {
	if a.SessionByID(id) != nil {
		a.PinnedID = id
	}
}

func (a *App) removeSession(id int) {
	out := a.Sessions[:0]
	for _, s := range a.Sessions {
		if s.ID() != id {
			out = append(out, s)
		}
	}
	a.Sessions = out
	if a.PinnedID == id {
		a.PinnedID = 0
		if n := len(a.Sessions); n > 0 {
			a.PinnedID = a.Sessions[n-1].ID()
		}
	}
	a.SyncFocus()
}

// Tick advances every session by one second and fires due alarms.
func (a *App) Tick() TickResult {
	var res TickResult
	var completed []int

	for _, sess := range a.Sessions {
		for _, ev := range sess.Tick(1) {
			a.log(ev.Kind, ev.Action, ev.Label, ev.Details, ev.DurationS)
			switch ev.Action {
			case "completed":
				completed = append(completed, sess.ID())
				a.notifyComplete(sess)
			case "phase_change":
				if p, ok := sess.(*sessions.Pomodoro); ok {
					a.notifyPhase(p)
				}
				res.Structural = true
			}
		}
	}
	for _, id := range completed {
		a.removeSession(id)
		res.Structural = true
	}
	if a.checkAlarms() {
		res.Structural = true
	}
	res.TitleDirty = true
	a.SyncFocus()
	return res
}

func (a *App) checkAlarms() bool {
	now := a.clock()
	changed := false
	for _, al := range a.Alarms {
		if al.Enabled && al.NextFireTime() != nil && !al.NextFire.After(now) && !a.isFiring(al) {
			a.fireAlarm(al, now)
			changed = true
		}
	}
	still := a.Snoozes[:0]
	for _, snz := range a.Snoozes {
		if !snz.FireAt.After(now) {
			a.fireSnooze(snz)
			changed = true
		} else {
			still = append(still, snz)
		}
	}
	a.Snoozes = still
	return changed
}

func (a *App) isFiring(al *alarms.Alarm) bool {
	for _, f := range a.Firing {
		if f.Alarm == al {
			return true
		}
	}
	return false
}

func (a *App) fireAlarm(al *alarms.Alarm, now time.Time) {
	a.Firing = append(a.Firing, Firing{Alarm: al})
	a.log("alarm", "alarm_fired", al.Label, nil, nil)
	a.Hooks.notify("Alarm", al.Label, al.Describe())
	sound := al.Sound
	if sound == "" {
		sound = a.Settings.DefaultSound()
	}
	a.Hooks.playAlarm(sound, true)
	al.MarkFired(now)
	a.persistAlarms()
}

func (a *App) fireSnooze(snz Snooze) {
	transient := alarms.New()
	transient.Label = snz.Label
	transient.Sound = snz.Sound
	transient.Enabled = false
	a.Firing = append(a.Firing, Firing{Alarm: transient})
	a.log("alarm", "alarm_fired", snz.Label, map[string]any{"snoozed": true}, nil)
	a.Hooks.notify("Alarm (snoozed)", snz.Label, "")
	sound := snz.Sound
	if sound == "" {
		sound = a.Settings.DefaultSound()
	}
	a.Hooks.playAlarm(sound, true)
}

// DismissAlarm stops a ringing alarm.
func (a *App) DismissAlarm(al *alarms.Alarm) {
	out := a.Firing[:0]
	for _, f := range a.Firing {
		if f.Alarm != al {
			out = append(out, f)
		}
	}
	a.Firing = out
	a.log("alarm", "alarm_dismissed", al.Label, nil, nil)
	if len(a.Firing) == 0 {
		a.Hooks.stopAlarm()
	}
}

// SnoozeAlarm dismisses and re-fires later.
func (a *App) SnoozeAlarm(al *alarms.Alarm) {
	mins := a.Settings.SnoozeMinutes()
	out := a.Firing[:0]
	for _, f := range a.Firing {
		if f.Alarm != al {
			out = append(out, f)
		}
	}
	a.Firing = out
	a.Snoozes = append(a.Snoozes, Snooze{
		FireAt: a.clock().Add(time.Duration(mins) * time.Minute),
		Label:  al.Label,
		Sound:  al.Sound,
	})
	a.log("alarm", "snoozed", al.Label, map[string]any{"minutes": mins}, nil)
	if len(a.Firing) == 0 {
		a.Hooks.stopAlarm()
	}
}

func (a *App) notifyComplete(sess sessions.Session) {
	a.Hooks.notify("Timer done", sess.Label(), "Time's up!")
	sound := a.Settings.DefaultSound()
	transient := alarms.New()
	transient.Label = sess.Label()
	if transient.Label == "" {
		transient.Label = "Timer"
	}
	transient.Sound = sound
	transient.Enabled = false
	title := "⏱ " + transient.Label + " · done"
	a.Firing = append(a.Firing, Firing{Alarm: transient, Title: title})
	a.Hooks.playAlarm(sound, true)
}

func (a *App) notifyPhase(p *sessions.Pomodoro) {
	phase := p.Phase
	switch phase {
	case sessions.ShortBreak:
		phase = "short break"
	case sessions.LongBreak:
		phase = "long break"
	}
	a.Hooks.notify("Pomodoro", p.Label(), "Now: "+phase)
	a.Hooks.playCue(a.Settings.DefaultSound())
}

// Primary is the session shown in the tray title.
func (a *App) Primary() sessions.Session {
	if s := a.SessionByID(a.PinnedID); s != nil {
		return s
	}
	for i := len(a.Sessions) - 1; i >= 0; i-- {
		if a.Sessions[i].State() == sessions.Running {
			return a.Sessions[i]
		}
	}
	if n := len(a.Sessions); n > 0 {
		return a.Sessions[n-1]
	}
	return nil
}

// TitleText is the live countdown/up string for the tray, including a
// leading space when a session is showing (matches the Python app).
func (a *App) TitleText() string {
	if p := a.Primary(); p != nil {
		return " " + p.TitleText()
	}
	return ""
}

// Alarming is true while something is ringing.
func (a *App) Alarming() bool {
	return len(a.Firing) > 0
}

// FiringTitle is the menu label for a ringing entry.
func FiringTitle(f Firing) string {
	if f.Title != "" {
		return f.Title
	}
	name := f.Alarm.Label
	if name == "" {
		name = "Alarm"
	}
	return "🔔 " + name + " · ringing"
}

// AddAlarm appends and persists.
func (a *App) AddAlarm(al *alarms.Alarm) {
	a.Alarms = append(a.Alarms, al)
	a.persistAlarms()
}

// UpdateAlarm replaces a same-id alarm and recomputes its schedule.
func (a *App) UpdateAlarm(al *alarms.Alarm) {
	al.ComputeNextFire(a.clock())
	for i, existing := range a.Alarms {
		if existing.ID == al.ID {
			a.Alarms[i] = al
			break
		}
	}
	a.persistAlarms()
}

// DeleteAlarm removes by id.
func (a *App) DeleteAlarm(id int) {
	out := a.Alarms[:0]
	for _, al := range a.Alarms {
		if al.ID != id {
			out = append(out, al)
		}
	}
	a.Alarms = out
	a.persistAlarms()
}

// AddPreset appends a quick-start timer.
func (a *App) AddPreset(p settings.Preset) {
	presets := a.Settings.Presets()
	presets = append(presets, p)
	a.Settings.SetPresets(presets)
	_ = a.Settings.Save()
}

// UpdatePreset replaces a preset by index.
func (a *App) UpdatePreset(index int, p settings.Preset) {
	presets := a.Settings.Presets()
	if index < 0 || index >= len(presets) {
		return
	}
	presets[index] = p
	a.Settings.SetPresets(presets)
	_ = a.Settings.Save()
}

// DeletePreset removes a preset by index.
func (a *App) DeletePreset(index int) {
	presets := a.Settings.Presets()
	if index < 0 || index >= len(presets) {
		return
	}
	presets = append(presets[:index], presets[index+1:]...)
	a.Settings.SetPresets(presets)
	_ = a.Settings.Save()
}

// ToggleKeepAwake flips the caffeine-style flag (platform layer does the work).
func (a *App) ToggleKeepAwake() bool {
	a.KeepAwakeActive = !a.KeepAwakeActive
	action := "disabled"
	if a.KeepAwakeActive {
		action = "enabled"
	}
	a.log("keepawake", action, "", nil, nil)
	return a.KeepAwakeActive
}

// SyncFocus runs the Focus on/off shortcut as work-phase activity changes.
func (a *App) SyncFocus() {
	desired := false
	if a.Settings.GetBool("focus_during_work", true) {
		for _, s := range a.Sessions {
			if p, ok := s.(*sessions.Pomodoro); ok && p.Phase == sessions.Work && p.State() == sessions.Running {
				desired = true
				break
			}
		}
	}
	if desired == a.FocusActive {
		return
	}
	a.FocusActive = desired
	key := "focus_shortcut_off"
	if desired {
		key = "focus_shortcut_on"
	}
	a.Hooks.runFocus(a.Settings.GetString(key, ""))
}

// ClearFocus runs the off-shortcut if Focus is currently engaged.
func (a *App) ClearFocus() {
	if a.FocusActive {
		a.FocusActive = false
		a.Hooks.runFocus(a.Settings.GetString("focus_shortcut_off", ""))
	}
}

// ExportHistoryCSV writes the full log.
func (a *App) ExportHistoryCSV(path string) error {
	return a.History.ExportCSV(path)
}

// ExportHistoryJSON writes the full log.
func (a *App) ExportHistoryJSON(path string) error {
	return a.History.ExportJSON(path)
}

// ClearHistory deletes all events after the caller has confirmed.
func (a *App) ClearHistory() error {
	return a.History.Clear()
}

// ExportSettings writes the shareable keys to path.
func (a *App) ExportSettings(path string) error {
	data := map[string]any{}
	for _, k := range settings.ShareableKeys {
		if v, ok := a.Settings.Data[k]; ok {
			data[k] = v
		}
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// ImportSettings merges shareable keys from path. Returns how many groups applied.
func (a *App) ImportSettings(path string) (int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var incoming map[string]any
	if err := json.Unmarshal(b, &incoming); err != nil {
		return 0, err
	}
	applied := 0
	for _, k := range settings.ShareableKeys {
		if v, ok := incoming[k]; ok {
			a.Settings.Data[k] = v
			applied++
		}
	}
	a.Settings.Normalize()
	a.Alarms = alarms.Load(a.Settings.AlarmsRaw(), a.clock())
	if err := a.Settings.Save(); err != nil {
		return applied, err
	}
	return applied, nil
}

// SetLaunchAtLogin persists the flag; the platform layer performs the install.
func (a *App) SetLaunchAtLogin(enabled bool) error {
	a.Settings.Data["launch_at_login"] = enabled
	return a.Settings.Save()
}
