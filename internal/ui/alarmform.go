package ui

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/MichalAFerber/tomatick2/internal/alarms"
	"github.com/MichalAFerber/tomatick2/internal/sound"
)

func (u *UI) editAlarm(existing *alarms.Alarm, onSave func(*alarms.Alarm)) {
	a := existing
	if a == nil {
		a = alarms.New()
		a.Sound = u.core.Settings.DefaultSound()
	}

	w := u.fyneApp.NewWindow("Alarm")
	label := widget.NewEntry()
	label.SetText(a.Label)
	label.SetPlaceHolder("Label")

	kind := widget.NewRadioGroup([]string{"One-shot (specific date)", "Recurring (days of week)"}, nil)
	if a.Kind == alarms.OneShot {
		kind.SetSelected("One-shot (specific date)")
	} else {
		kind.SetSelected("Recurring (days of week)")
	}

	timeEntry := widget.NewEntry()
	timeEntry.SetText(a.TimeStr)
	timeEntry.SetPlaceHolder("HH:MM")

	dateEntry := widget.NewEntry()
	dateEntry.SetPlaceHolder("YYYY-MM-DD")
	if a.DateStr != "" {
		dateEntry.SetText(a.DateStr)
	} else {
		dateEntry.SetText(time.Now().Format("2006-01-02"))
	}

	dayChecks := make([]*widget.Check, 7)
	dayBox := container.NewHBox()
	selected := map[int]bool{}
	for _, d := range a.DaysOfWeek {
		selected[d] = true
	}
	for i, name := range alarms.WeekdayNames {
		i := i
		c := widget.NewCheck(name, nil)
		c.SetChecked(selected[i])
		dayChecks[i] = c
		dayBox.Add(c)
	}

	sounds := sound.Available()
	soundSel := widget.NewSelect(sounds, func(name string) {
		u.cuePlayer.Play(name, false)
	})
	if a.Sound != "" {
		soundSel.SetSelected(a.Sound)
	} else if len(sounds) > 0 {
		soundSel.SetSelected(u.core.Settings.DefaultSound())
	}

	enabled := widget.NewCheck("Enabled", nil)
	enabled.SetChecked(a.Enabled)

	dateItem := widget.NewFormItem("Date", dateEntry)
	daysItem := widget.NewFormItem("Days", dayBox)

	syncKind := func() {
		oneShot := kind.Selected == "One-shot (specific date)"
		dateEntry.Hidden = !oneShot
		dayBox.Hidden = oneShot
		dateItem.Widget.Refresh()
		daysItem.Widget.Refresh()
	}
	kind.OnChanged = func(string) { syncKind() }
	syncKind()

	form := widget.NewForm(
		widget.NewFormItem("Label", label),
		widget.NewFormItem("Type", kind),
		widget.NewFormItem("Time", timeEntry),
		dateItem,
		daysItem,
		widget.NewFormItem("Sound", soundSel),
		widget.NewFormItem("", enabled),
	)
	form.SubmitText = "Save"
	form.CancelText = "Cancel"
	form.OnCancel = func() { w.Close() }
	form.OnSubmit = func() {
		hhmm, err := alarms.ParseHHMM(timeEntry.Text)
		if err != nil {
			u.alert("'" + timeEntry.Text + "' is not a valid HH:MM time.")
			return
		}
		out := a
		if existing == nil {
			out = alarms.New()
		}
		out.Label = label.Text
		out.TimeStr = hhmm
		out.Sound = soundSel.Selected
		out.Enabled = enabled.Checked
		if kind.Selected == "One-shot (specific date)" {
			out.Kind = alarms.OneShot
			out.DateStr = dateEntry.Text
			out.DaysOfWeek = nil
		} else {
			out.Kind = alarms.Recurring
			out.DateStr = ""
			var days []int
			for i, c := range dayChecks {
				if c.Checked {
					days = append(days, i)
				}
			}
			out.DaysOfWeek = days
		}
		out.ComputeNextFire(time.Now())
		w.Close()
		onSave(out)
	}

	w.SetContent(container.NewPadded(form))
	w.Resize(fyne.NewSize(520, 420))
	w.CenterOnScreen()
	showApp()
	w.Show()
}
