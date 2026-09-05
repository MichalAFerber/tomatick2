// Package settings loads and persists Tomatick's JSON config.
//
// The schema matches the Python app so config.json files round-trip.
package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"

	"github.com/MichalAFerber/tomatick2/internal/sessions"
	"github.com/MichalAFerber/tomatick2/internal/version"
)

const appName = version.AppName

// ShareableKeys are safe to export/import across machines (excludes
// launch_at_login, which is per-machine and tied to the installed app path).
var ShareableKeys = []string{
	"pomodoro", "snooze_minutes", "default_sound", "alarms", "presets",
	"focus_shortcut_on", "focus_shortcut_off", "focus_during_work",
	"hotkey_action", "hotkey_key", "icon_theme",
}

// Defaults is the factory configuration.
func Defaults() map[string]any {
	return map[string]any{
		"pomodoro": map[string]any{
			"work_minutes":             25,
			"short_break_minutes":      5,
			"long_break_minutes":       15,
			"cycles_before_long_break": 4,
			"auto_start_next":          true,
		},
		"snooze_minutes":     9,
		"default_sound":      "Glass",
		"launch_at_login":    false,
		"alarms":             []any{},
		"presets": []any{
			map[string]any{"label": "Focus", "seconds": 1500},
			map[string]any{"label": "Quick break", "seconds": 300},
		},
		"focus_shortcut_on":  "",
		"focus_shortcut_off": "",
		"focus_during_work":  true,
		"hotkey_action":      "none",
		"hotkey_key":         "",
		"icon_theme":         "red",
	}
}

// Settings loads, exposes, and persists the JSON config, backfilling defaults.
type Settings struct {
	Data map[string]any
}

// SupportDir is the per-user data directory, created if needed.
// Honors TOMATICK_SUPPORT_DIR so tests can redirect storage.
func SupportDir() (string, error) {
	if override := os.Getenv("TOMATICK_SUPPORT_DIR"); override != "" {
		return mkdir(override)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	// UserConfigDir is already Application Support on macOS, AppData on
	// Windows, and ~/.config on Linux. Keep a stable "Tomatick" leaf so the
	// Python app's config can be reused on macOS.
	if runtime.GOOS == "linux" {
		// Prefer XDG data for the sqlite db + config together, matching
		// "one folder for everything" from the original.
		if data := os.Getenv("XDG_DATA_HOME"); data != "" {
			return mkdir(filepath.Join(data, appName))
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return mkdir(filepath.Join(home, ".local", "share", appName))
	}
	return mkdir(filepath.Join(base, appName))
}

func mkdir(path string) (string, error) {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", err
	}
	return path, nil
}

// ConfigPath is supportDir/config.json.
func ConfigPath() (string, error) {
	dir, err := SupportDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// HistoryPath is supportDir/history.db.
func HistoryPath() (string, error) {
	dir, err := SupportDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "history.db"), nil
}

// Load reads config.json, or returns defaults if missing/corrupt.
func Load() (*Settings, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	raw := map[string]any{}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &raw)
	}
	return &Settings{Data: deepMerge(Defaults(), raw)}, nil
}

// Save writes config.json.
func (s *Settings) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.Data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// Normalize backfills any missing defaults (e.g. after importing a partial config).
func (s *Settings) Normalize() {
	s.Data = deepMerge(Defaults(), s.Data)
}

// Get returns a top-level key.
func (s *Settings) Get(key string) any {
	return s.Data[key]
}

// GetString returns a string key, or def if missing/wrong type.
func (s *Settings) GetString(key, def string) string {
	if v, ok := s.Data[key].(string); ok {
		return v
	}
	return def
}

// GetBool returns a bool key, or def if missing/wrong type.
func (s *Settings) GetBool(key string, def bool) bool {
	if v, ok := s.Data[key].(bool); ok {
		return v
	}
	return def
}

// Set stores a key and saves.
func (s *Settings) Set(key string, value any) error {
	s.Data[key] = value
	return s.Save()
}

// Pomodoro returns the pomodoro config as a typed struct.
func (s *Settings) Pomodoro() sessions.PomodoroConfig {
	m, _ := s.Data["pomodoro"].(map[string]any)
	return sessions.PomodoroConfig{
		WorkMinutes:           asInt(m, "work_minutes", 25),
		ShortBreakMinutes:     asInt(m, "short_break_minutes", 5),
		LongBreakMinutes:      asInt(m, "long_break_minutes", 15),
		CyclesBeforeLongBreak: asInt(m, "cycles_before_long_break", 4),
		AutoStartNext:         asBool(m, "auto_start_next", true),
	}
}

// SetPomodoro writes a typed pomodoro config back into Data (does not save).
func (s *Settings) SetPomodoro(cfg sessions.PomodoroConfig) {
	s.Data["pomodoro"] = map[string]any{
		"work_minutes":             cfg.WorkMinutes,
		"short_break_minutes":      cfg.ShortBreakMinutes,
		"long_break_minutes":       cfg.LongBreakMinutes,
		"cycles_before_long_break": cfg.CyclesBeforeLongBreak,
		"auto_start_next":          cfg.AutoStartNext,
	}
}

// SnoozeMinutes is the alarm snooze length.
func (s *Settings) SnoozeMinutes() int {
	return asInt(s.Data, "snooze_minutes", 9)
}

// DefaultSound is the named cue/alarm sound.
func (s *Settings) DefaultSound() string {
	return s.GetString("default_sound", "Glass")
}

// LaunchAtLogin is the persisted autostart flag.
func (s *Settings) LaunchAtLogin() bool {
	return s.GetBool("launch_at_login", false)
}

// Presets returns the quick-start timer presets.
func (s *Settings) Presets() []Preset {
	raw, _ := s.Data["presets"].([]any)
	var out []Preset
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, Preset{
			Label:   asString(m, "label", ""),
			Seconds: asInt(m, "seconds", 0),
		})
	}
	return out
}

// SetPresets replaces the preset list (does not save).
func (s *Settings) SetPresets(presets []Preset) {
	raw := make([]any, 0, len(presets))
	for _, p := range presets {
		raw = append(raw, map[string]any{"label": p.Label, "seconds": p.Seconds})
	}
	s.Data["presets"] = raw
}

// AlarmsRaw is the serialized alarm list.
func (s *Settings) AlarmsRaw() []any {
	raw, _ := s.Data["alarms"].([]any)
	if raw == nil {
		return []any{}
	}
	return raw
}

// Preset is a named timer duration.
type Preset struct {
	Label   string
	Seconds int
}

func deepMerge(base, override map[string]any) map[string]any {
	result := map[string]any{}
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		if bv, ok := result[k].(map[string]any); ok {
			if ov, ok := v.(map[string]any); ok {
				result[k] = deepMerge(bv, ov)
				continue
			}
		}
		result[k] = v
	}
	return result
}

func asInt(m map[string]any, key string, def int) int {
	if m == nil {
		return def
	}
	switch n := m[key].(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return def
}

func asBool(m map[string]any, key string, def bool) bool {
	if m == nil {
		return def
	}
	if v, ok := m[key].(bool); ok {
		return v
	}
	return def
}

func asString(m map[string]any, key, def string) string {
	if m == nil {
		return def
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return def
}
