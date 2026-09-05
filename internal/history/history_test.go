package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLogAndRecent(t *testing.T) {
	h := openTemp(t)
	d := 300
	ts1 := time.Date(2026, 6, 26, 10, 0, 0, 0, time.Local)
	ts2 := time.Date(2026, 6, 26, 10, 5, 0, 0, time.Local)
	if _, err := h.LogEvent("timer", "started", "tea", nil, &d, ts1); err != nil {
		t.Fatal(err)
	}
	if _, err := h.LogEvent("timer", "completed", "tea", nil, nil, ts2); err != nil {
		t.Fatal(err)
	}
	rows, err := h.Recent(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("len = %d", len(rows))
	}
	if rows[0].Action != "completed" {
		t.Fatalf("newest action = %s", rows[0].Action)
	}
	if rows[1].Label == nil || *rows[1].Label != "tea" {
		t.Fatalf("label = %v", rows[1].Label)
	}
	if rows[1].DurationS == nil || *rows[1].DurationS != 300 {
		t.Fatalf("duration = %v", rows[1].DurationS)
	}
}

func TestDetailsJSON(t *testing.T) {
	h := openTemp(t)
	if _, err := h.LogEvent("pomodoro", "phase_change", "", map[string]any{"from": "work", "to": "break"}, nil, time.Time{}); err != nil {
		t.Fatal(err)
	}
	rows, err := h.Recent(1)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(*rows[0].Details), &got); err != nil {
		t.Fatal(err)
	}
	if got["from"] != "work" || got["to"] != "break" {
		t.Fatalf("got %+v", got)
	}
}

func TestCountAndClear(t *testing.T) {
	h := openTemp(t)
	for i := 0; i < 3; i++ {
		if _, err := h.LogEvent("stopwatch", "started", "", nil, nil, time.Time{}); err != nil {
			t.Fatal(err)
		}
	}
	n, err := h.Count()
	if err != nil || n != 3 {
		t.Fatalf("count = %d %v", n, err)
	}
	if err := h.Clear(); err != nil {
		t.Fatal(err)
	}
	n, _ = h.Count()
	if n != 0 {
		t.Fatalf("after clear count = %d", n)
	}
}

func TestExportCSVAndJSON(t *testing.T) {
	h := openTemp(t)
	d := 60
	ts := time.Date(2026, 6, 26, 9, 0, 0, 0, time.Local)
	if _, err := h.LogEvent("timer", "started", "x", nil, &d, ts); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "out.csv")
	jsonPath := filepath.Join(dir, "out.json")
	if err := h.ExportCSV(csvPath); err != nil {
		t.Fatal(err)
	}
	if err := h.ExportJSON(jsonPath); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(csvPath)
	if !contains(string(b), "started") {
		t.Fatalf("csv missing started: %s", b)
	}
	var data []Event
	jb, _ := os.ReadFile(jsonPath)
	if err := json.Unmarshal(jb, &data); err != nil {
		t.Fatal(err)
	}
	if data[0].Label == nil || *data[0].Label != "x" {
		t.Fatalf("json label = %v", data[0].Label)
	}
	if data[0].DurationS == nil || *data[0].DurationS != 60 {
		t.Fatalf("json duration = %v", data[0].DurationS)
	}
}

func openTemp(t *testing.T) *History {
	t.Helper()
	h, err := Open(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = h.Close() })
	return h
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}
