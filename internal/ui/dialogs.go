package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/MichalAFerber/tomatick2/internal/sessions"
	"github.com/MichalAFerber/tomatick2/internal/settings"
)

func (u *UI) askTimer() {
	u.showForm("Timer", []formField{
		{label: "Duration (e.g. 25m, 1h30m, 90s)", placeholder: "25m"},
		{label: "Label (optional)", placeholder: ""},
	}, "Start", func(vals []string) {
		seconds, err := sessions.ParseDuration(vals[0])
		if err != nil {
			u.alert("Couldn't understand '" + vals[0] + "'.")
			return
		}
		if seconds <= 0 {
			return
		}
		u.core.StartTimer(seconds, vals[1])
		u.refresh()
	})
}

func (u *UI) askStopwatch() {
	u.showForm("Stopwatch", []formField{
		{label: "Label (optional)", placeholder: ""},
	}, "Start", func(vals []string) {
		u.core.StartStopwatch(vals[0])
		u.refresh()
	})
}

func (u *UI) askPomodoro() {
	u.showForm("Pomodoro", []formField{
		{label: "Label (optional)", placeholder: ""},
	}, "Start", func(vals []string) {
		u.core.StartPomodoro(vals[0])
		u.refresh()
	})
}

func (u *UI) askPreset(existing *settings.Preset, onSave func(settings.Preset)) {
	label, dur := "", ""
	if existing != nil {
		label = existing.Label
		dur = sessions.FormatClock(existing.Seconds)
	}
	u.showForm("Preset", []formField{
		{label: "Preset name", placeholder: "Focus", value: label},
		{label: "Duration (e.g. 25m, 1h30m, 90s)", placeholder: "25m", value: dur},
	}, "Save", func(vals []string) {
		seconds, err := sessions.ParseDuration(vals[1])
		if err != nil {
			u.alert("Couldn't understand '" + vals[1] + "'.")
			return
		}
		if seconds <= 0 {
			return
		}
		onSave(settings.Preset{Label: vals[0], Seconds: seconds})
	})
}

type formField struct {
	label, placeholder, value string
}

func (u *UI) showForm(title string, fields []formField, okLabel string, onOK func([]string)) {
	w := u.fyneApp.NewWindow(title)
	entries := make([]*widget.Entry, len(fields))
	formItems := make([]*widget.FormItem, 0, len(fields))
	for i, f := range fields {
		e := widget.NewEntry()
		e.SetPlaceHolder(f.placeholder)
		e.SetText(f.value)
		entries[i] = e
		formItems = append(formItems, widget.NewFormItem(f.label, e))
	}
	submit := func() {
		vals := make([]string, len(entries))
		for i, e := range entries {
			vals[i] = e.Text
		}
		w.Close()
		onOK(vals)
	}
	if len(entries) > 0 {
		entries[len(entries)-1].OnSubmitted = func(string) { submit() }
	}
	form := widget.NewForm(formItems...)
	form.OnSubmit = submit
	form.OnCancel = func() { w.Close() }
	form.SubmitText = okLabel
	form.CancelText = "Cancel"

	w.SetContent(container.NewPadded(form))
	w.Resize(fyne.NewSize(420, float32(80+len(fields)*48)))
	w.CenterOnScreen()
	showApp()
	w.Show()
	w.RequestFocus()
	if len(entries) > 0 {
		w.Canvas().Focus(entries[0])
	}
}

func (u *UI) alert(msg string) {
	w := u.fyneApp.NewWindow("Tomatick")
	ok := widget.NewButton("OK", func() { w.Close() })
	ok.Importance = widget.HighImportance
	w.SetContent(container.NewPadded(container.NewVBox(
		widget.NewLabel(msg),
		container.NewHBox(layout.NewSpacer(), ok),
	)))
	w.Resize(fyne.NewSize(360, 120))
	w.CenterOnScreen()
	showApp()
	w.Show()
}

func (u *UI) confirm(msg, okLabel string, onOK func()) {
	w := u.fyneApp.NewWindow("Tomatick")
	cancel := widget.NewButton("Cancel", func() { w.Close() })
	ok := widget.NewButton(okLabel, func() {
		w.Close()
		onOK()
	})
	ok.Importance = widget.DangerImportance
	w.SetContent(container.NewPadded(container.NewVBox(
		widget.NewLabel(msg),
		container.NewHBox(layout.NewSpacer(), cancel, ok),
	)))
	w.Resize(fyne.NewSize(360, 130))
	w.CenterOnScreen()
	showApp()
	w.Show()
}
