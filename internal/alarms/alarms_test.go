package alarms

import (
	"testing"
	"time"
)

func loc() *time.Location { return time.Local }

func TestOneShotNextFire(t *testing.T) {
	a := &Alarm{ID: 1, Label: "dentist", TimeStr: "14:30", Kind: OneShot, DateStr: "2026-06-27", Enabled: true}
	now := time.Date(2026, 6, 26, 9, 0, 0, 0, loc())
	nf := a.ComputeNextFire(now)
	if nf == nil {
		t.Fatal("expected next fire")
	}
	want := time.Date(2026, 6, 27, 14, 30, 0, 0, loc())
	if !nf.Equal(want) {
		t.Fatalf("got %v want %v", nf, want)
	}
}

func TestOneShotDisablesAfterFire(t *testing.T) {
	a := &Alarm{ID: 1, Kind: OneShot, TimeStr: "08:00", DateStr: "2026-06-26", Enabled: true}
	a.ComputeNextFire(time.Date(2026, 6, 26, 7, 0, 0, 0, loc()))
	a.MarkFired(time.Date(2026, 6, 26, 8, 0, 0, 0, loc()))
	if a.Enabled {
		t.Fatal("expected disabled")
	}
	if a.NextFireTime() != nil {
		t.Fatal("expected no next fire")
	}
}

func TestRecurringPicksNextMatchingDay(t *testing.T) {
	// 2026-06-26 is a Friday (Python weekday 4). Alarm on Mondays (0) at 09:00.
	a := &Alarm{ID: 1, Kind: Recurring, TimeStr: "09:00", DaysOfWeek: []int{0}, Enabled: true}
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, loc())
	nf := a.ComputeNextFire(now)
	if nf == nil {
		t.Fatal("expected next fire")
	}
	if PythonWeekday(*nf) != 0 {
		t.Fatalf("weekday = %d", PythonWeekday(*nf))
	}
	if nf.Hour() != 9 || nf.Minute() != 0 {
		t.Fatalf("time = %02d:%02d", nf.Hour(), nf.Minute())
	}
	if !nf.After(now) {
		t.Fatal("next fire not in the future")
	}
}

func TestRecurringTodayIfTimeNotPassed(t *testing.T) {
	a := &Alarm{ID: 1, Kind: Recurring, TimeStr: "09:00", DaysOfWeek: []int{4}, Enabled: true}
	now := time.Date(2026, 6, 26, 8, 0, 0, 0, loc())
	nf := a.ComputeNextFire(now)
	want := time.Date(2026, 6, 26, 9, 0, 0, 0, loc())
	if nf == nil || !nf.Equal(want) {
		t.Fatalf("got %v want %v", nf, want)
	}
}

func TestRecurringReschedulesAfterFire(t *testing.T) {
	a := &Alarm{ID: 1, Kind: Recurring, TimeStr: "09:00", DaysOfWeek: []int{4}, Enabled: true}
	a.ComputeNextFire(time.Date(2026, 6, 26, 8, 0, 0, 0, loc()))
	a.MarkFired(time.Date(2026, 6, 26, 9, 0, 0, 0, loc()))
	if !a.Enabled {
		t.Fatal("should stay enabled")
	}
	nf := a.NextFireTime()
	want := time.Date(2026, 7, 3, 9, 0, 0, 0, loc())
	if nf == nil || !nf.Equal(want) {
		t.Fatalf("got %v want %v", nf, want)
	}
}

func TestDisabledAlarmHasNoNextFire(t *testing.T) {
	a := &Alarm{ID: 1, Kind: Recurring, TimeStr: "09:00", DaysOfWeek: []int{0}, Enabled: false}
	if a.ComputeNextFire(time.Date(2026, 6, 26, 0, 0, 0, 0, loc())) != nil {
		t.Fatal("expected nil")
	}
}

func TestDueAlarms(t *testing.T) {
	a := &Alarm{ID: 1, Kind: Recurring, TimeStr: "09:00", DaysOfWeek: []int{4}, Enabled: true}
	a.ComputeNextFire(time.Date(2026, 6, 26, 8, 0, 0, 0, loc()))
	if due := Due([]*Alarm{a}, time.Date(2026, 6, 26, 8, 59, 0, 0, loc())); len(due) != 0 {
		t.Fatalf("early due = %v", due)
	}
	if due := Due([]*Alarm{a}, time.Date(2026, 6, 26, 9, 0, 0, 0, loc())); len(due) != 1 {
		t.Fatalf("due = %v", due)
	}
}

func TestRoundtripSerialization(t *testing.T) {
	a := &Alarm{ID: 7, Label: "x", TimeStr: "07:15", Kind: Recurring, DaysOfWeek: []int{1, 3}, Sound: "Ping", Enabled: true}
	restored := FromMap(a.ToMap())
	if restored.Label != "x" || restored.Sound != "Ping" {
		t.Fatalf("restored = %+v", restored)
	}
	if len(restored.DaysOfWeek) != 2 || restored.DaysOfWeek[0] != 1 || restored.DaysOfWeek[1] != 3 {
		t.Fatalf("days = %v", restored.DaysOfWeek)
	}
}

func TestDescribe(t *testing.T) {
	weekdays := &Alarm{Kind: Recurring, TimeStr: "09:00", DaysOfWeek: []int{0, 1, 2, 3, 4}}
	if d := weekdays.Describe(); d != "Weekdays 09:00" {
		t.Fatalf("got %q", d)
	}
	once := &Alarm{Kind: OneShot, TimeStr: "09:00", DateStr: "2026-06-27"}
	if d := once.Describe(); d != "Once 2026-06-27 09:00" {
		t.Fatalf("got %q", d)
	}
}

func TestParseDays(t *testing.T) {
	got, err := ParseDays("weekdays")
	if err != nil || !equalInts(got, []int{0, 1, 2, 3, 4}) {
		t.Fatalf("weekdays: %v %v", got, err)
	}
	got, err = ParseDays("Mon,Wed,Fri")
	if err != nil || !equalInts(got, []int{0, 2, 4}) {
		t.Fatalf("named: %v %v", got, err)
	}
	if _, err := ParseDays("xyz"); err == nil {
		t.Fatal("expected error")
	}
}

func TestPythonWeekdayFriday(t *testing.T) {
	// 2026-06-26 is a Friday.
	d := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	if PythonWeekday(d) != 4 {
		t.Fatalf("Friday = %d, want 4", PythonWeekday(d))
	}
	sun := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	if PythonWeekday(sun) != 6 {
		t.Fatalf("Sunday = %d, want 6", PythonWeekday(sun))
	}
	mon := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	if PythonWeekday(mon) != 0 {
		t.Fatalf("Monday = %d, want 0", PythonWeekday(mon))
	}
}
