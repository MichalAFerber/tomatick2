// Package alarms holds the alarm model and scheduling.
//
// An alarm is either one-shot (a specific date + time, fires once) or
// recurring (a time of day on selected weekdays). Scheduling is pure
// datetime arithmetic so it is fully unit-testable.
//
// Weekdays follow Python's datetime.weekday() convention so existing
// Tomatick config.json files round-trip: Monday = 0 ... Sunday = 6.
package alarms

import (
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	OneShot   = "one_shot"
	Recurring = "recurring"
)

// WeekdayNames is Monday-first, matching the stored days_of_week integers.
var WeekdayNames = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

var idCounter atomic.Int64

func nextID() int {
	return int(idCounter.Add(1))
}

func bumpID(id int) {
	for {
		cur := idCounter.Load()
		if int64(id) <= cur {
			return
		}
		if idCounter.CompareAndSwap(cur, int64(id)) {
			return
		}
	}
}

// PythonWeekday converts a Go time.Weekday (Sunday=0) to Python's
// datetime.weekday() (Monday=0 ... Sunday=6).
func PythonWeekday(t time.Time) int {
	return int((t.Weekday() + 6) % 7)
}

// Alarm is a single alarm definition plus its computed next firing time.
type Alarm struct {
	ID         int       `json:"id"`
	Label      string    `json:"label"`
	TimeStr    string    `json:"time_str"`
	Kind       string    `json:"kind"`
	DateStr    string    `json:"date_str,omitempty"`
	DaysOfWeek []int     `json:"days_of_week"`
	Enabled    bool      `json:"enabled"`
	Sound      string    `json:"sound,omitempty"`
	NextFire   time.Time `json:"-"`
	hasNext    bool      `json:"-"`
}

// New returns a recurring 08:00 alarm with a fresh id.
func New() *Alarm {
	return &Alarm{
		ID:      nextID(),
		TimeStr: "08:00",
		Kind:    Recurring,
		Enabled: true,
	}
}

// FromMap reconstructs an Alarm from a serialized config dict.
func FromMap(d map[string]any) *Alarm {
	a := &Alarm{
		Label:   asString(d["label"]),
		TimeStr: asStringDefault(d["time_str"], "08:00"),
		Kind:    asStringDefault(d["kind"], Recurring),
		DateStr: asString(d["date_str"]),
		Enabled: asBoolDefault(d["enabled"], true),
		Sound:   asString(d["sound"]),
	}
	if id, ok := asInt(d["id"]); ok {
		a.ID = id
		bumpID(id)
	} else {
		a.ID = nextID()
	}
	a.DaysOfWeek = uniqueSortedInts(asIntSlice(d["days_of_week"]))
	return a
}

// ToMap serializes the alarm for config.json (matches the Python schema).
func (a *Alarm) ToMap() map[string]any {
	m := map[string]any{
		"id":           a.ID,
		"label":        a.Label,
		"time_str":     a.TimeStr,
		"kind":         a.Kind,
		"days_of_week": a.DaysOfWeek,
		"enabled":      a.Enabled,
	}
	if a.DateStr != "" {
		m["date_str"] = a.DateStr
	} else {
		m["date_str"] = nil
	}
	if a.Sound != "" {
		m["sound"] = a.Sound
	} else {
		m["sound"] = nil
	}
	return m
}

func parseHHMM(value string) (int, int, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("bad time: %q", value)
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, fmt.Errorf("bad time: %q", value)
	}
	return h, m, nil
}

// ParseHHMM validates and normalizes a 24-hour HH:MM string.
func ParseHHMM(value string) (string, error) {
	h, m, err := parseHHMM(value)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%02d:%02d", h, m), nil
}

// NextFireTime returns the computed next fire, or nil if none.
func (a *Alarm) NextFireTime() *time.Time {
	if !a.hasNext {
		return nil
	}
	t := a.NextFire
	return &t
}

// ComputeNextFire returns the next datetime this alarm should fire at.
//
// For a one-shot alarm in the past, returns that past datetime (the app
// treats next_fire <= now as "fire now", then disables it). A recurring
// alarm with no selected days returns nil.
func (a *Alarm) ComputeNextFire(now time.Time) *time.Time {
	a.hasNext = false
	if !a.Enabled {
		return nil
	}
	h, m, err := parseHHMM(a.TimeStr)
	if err != nil {
		return nil
	}

	if a.Kind == OneShot {
		if a.DateStr == "" {
			return nil
		}
		d, err := time.ParseInLocation("2006-01-02", a.DateStr, now.Location())
		if err != nil {
			return nil
		}
		nf := time.Date(d.Year(), d.Month(), d.Day(), h, m, 0, 0, now.Location())
		a.NextFire = nf
		a.hasNext = true
		return &nf
	}

	if len(a.DaysOfWeek) == 0 {
		return nil
	}
	want := map[int]bool{}
	for _, d := range a.DaysOfWeek {
		want[d] = true
	}
	for offset := 0; offset < 8; offset++ {
		candidateDate := now.AddDate(0, 0, offset)
		if want[PythonWeekday(candidateDate)] {
			candidate := time.Date(candidateDate.Year(), candidateDate.Month(), candidateDate.Day(),
				h, m, 0, 0, now.Location())
			if candidate.After(now) {
				a.NextFire = candidate
				a.hasNext = true
				return &candidate
			}
		}
	}
	return nil
}

// MarkFired updates state after firing: disable one-shots, reschedule recurring.
func (a *Alarm) MarkFired(now time.Time) {
	if a.Kind == OneShot {
		a.Enabled = false
		a.hasNext = false
		return
	}
	a.ComputeNextFire(now.Add(time.Second))
}

// Describe is a human-readable schedule summary.
func (a *Alarm) Describe() string {
	if a.Kind == OneShot {
		ds := a.DateStr
		if ds == "" {
			ds = "?"
		}
		return fmt.Sprintf("Once %s %s", ds, a.TimeStr)
	}
	if len(a.DaysOfWeek) == 0 {
		return fmt.Sprintf("(no days) %s", a.TimeStr)
	}
	days := dayPhrase(a.DaysOfWeek)
	return fmt.Sprintf("%s %s", days, a.TimeStr)
}

func dayPhrase(days []int) string {
	if equalInts(days, []int{0, 1, 2, 3, 4}) {
		return "Weekdays"
	}
	if equalInts(days, []int{5, 6}) {
		return "Weekends"
	}
	if len(days) == 7 {
		return "Daily"
	}
	parts := make([]string, 0, len(days))
	for _, d := range days {
		if d >= 0 && d < len(WeekdayNames) {
			parts = append(parts, WeekdayNames[d])
		}
	}
	return strings.Join(parts, ",")
}

// MenuText is the list-row label for this alarm.
func (a *Alarm) MenuText() string {
	name := a.Label
	if name == "" {
		name = "Alarm"
	}
	state := ""
	if !a.Enabled {
		state = "  (off)"
	}
	return fmt.Sprintf("🔔 %s · %s%s", name, a.Describe(), state)
}

// Load builds Alarm objects from serialized settings and computes schedules.
func Load(raw []any, now time.Time) []*Alarm {
	out := make([]*Alarm, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		a := FromMap(m)
		a.ComputeNextFire(now)
		out = append(out, a)
	}
	return out
}

// Due returns the alarms whose next fire has arrived (<= now).
func Due(alarms []*Alarm, now time.Time) []*Alarm {
	var out []*Alarm
	for _, a := range alarms {
		if a.Enabled && a.hasNext && !a.NextFire.After(now) {
			out = append(out, a)
		}
	}
	return out
}

var dayPresets = map[string][]int{
	"daily":    {0, 1, 2, 3, 4, 5, 6},
	"weekdays": {0, 1, 2, 3, 4},
	"weekends": {5, 6},
}

// ParseDays parses "weekdays", "daily", "weekends", or "Mon,Wed,Fri".
func ParseDays(text string) ([]int, error) {
	text = strings.TrimSpace(strings.ToLower(text))
	if text == "" {
		return []int{}, nil
	}
	if preset, ok := dayPresets[text]; ok {
		return append([]int{}, preset...), nil
	}
	names := map[string]int{}
	for i, n := range WeekdayNames {
		names[strings.ToLower(n)] = i
	}
	var out []int
	for _, token := range strings.Split(strings.ReplaceAll(text, " ", ""), ",") {
		if token == "" {
			continue
		}
		d, ok := names[token]
		if !ok {
			return nil, fmt.Errorf("couldn't understand days: %q", text)
		}
		out = append(out, d)
	}
	return uniqueSortedInts(out), nil
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func uniqueSortedInts(in []int) []int {
	seen := map[int]bool{}
	var out []int
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func asStringDefault(v any, def string) string {
	s := asString(v)
	if s == "" {
		return def
	}
	return s
}

func asBoolDefault(v any, def bool) bool {
	if v == nil {
		return def
	}
	b, ok := v.(bool)
	if !ok {
		return def
	}
	return b
}

func asInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	}
	return 0, false
}

func asIntSlice(v any) []int {
	switch s := v.(type) {
	case []int:
		return s
	case []any:
		var out []int
		for _, item := range s {
			if n, ok := asInt(item); ok {
				out = append(out, n)
			}
		}
		return out
	}
	return nil
}
