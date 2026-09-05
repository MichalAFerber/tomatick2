package ui

import (
	"os"
	"path/filepath"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/MichalAFerber/tomatick2/internal/alarms"
	"github.com/MichalAFerber/tomatick2/internal/assets"
	"github.com/MichalAFerber/tomatick2/internal/hotkey"
	"github.com/MichalAFerber/tomatick2/internal/sessions"
	"github.com/MichalAFerber/tomatick2/internal/settings"
	"github.com/MichalAFerber/tomatick2/internal/sound"
	"github.com/MichalAFerber/tomatick2/internal/version"
)

func (u *UI) openSettings(tabIndex int) {
	if u.settingsWin != nil {
		showApp()
		u.settingsWin.Show()
		u.settingsWin.RequestFocus()
		return
	}

	w := u.fyneApp.NewWindow("Tomatick Settings")
	u.settingsWin = w
	w.SetOnClosed(func() { u.settingsWin = nil })

	pomo := u.core.Settings.Pomodoro()

	soundSel := widget.NewSelect(sound.Available(), func(name string) {
		u.cuePlayer.Play(name, false)
	})
	soundSel.SetSelected(u.core.Settings.DefaultSound())

	themeSel := widget.NewSelect([]string{"Red", "White", "Black"}, nil)
	themeSel.SetSelected(capitalize(u.core.Settings.GetString("icon_theme", "red")))

	snooze := widget.NewEntry()
	snooze.SetText(strconv.Itoa(u.core.Settings.SnoozeMinutes()))

	actionLabels := make([]string, len(hotkey.Actions))
	for i, a := range hotkey.Actions {
		actionLabels[i] = a.Label
	}
	hkAction := widget.NewSelect(actionLabels, nil)
	curAction := u.core.Settings.GetString("hotkey_action", "none")
	for _, a := range hotkey.Actions {
		if a.ID == curAction {
			hkAction.SetSelected(a.Label)
		}
	}

	keyLabels := make([]string, len(hotkey.Keys))
	for i, k := range hotkey.Keys {
		if k == "" {
			keyLabels[i] = "(none)"
		} else {
			keyLabels[i] = k
		}
	}
	hkKey := widget.NewSelect(keyLabels, nil)
	curKey := u.core.Settings.GetString("hotkey_key", "")
	if curKey == "" {
		hkKey.SetSelected("(none)")
	} else {
		hkKey.SetSelected(curKey)
	}

	launch := widget.NewCheck("Launch Tomatick at login", nil)
	launch.SetChecked(u.core.Settings.LaunchAtLogin())

	exportBtn := widget.NewButton("Export Settings…", func() { u.exportSettings(w) })
	importBtn := widget.NewButton("Import Settings…", func() { u.importSettings(w) })

	general := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Default sound", soundSel),
			widget.NewFormItem("Menu bar icon", themeSel),
			widget.NewFormItem("Snooze minutes", snooze),
			widget.NewFormItem("Global hotkey", hkAction),
			widget.NewFormItem("Hotkey key", hkKey),
		),
		launch,
		container.NewHBox(exportBtn, importBtn),
	)

	work := intEntry(pomo.WorkMinutes)
	short := intEntry(pomo.ShortBreakMinutes)
	long := intEntry(pomo.LongBreakMinutes)
	cycles := intEntry(pomo.CyclesBeforeLongBreak)
	auto := widget.NewCheck("Auto-start next phase", nil)
	auto.SetChecked(pomo.AutoStartNext)
	focusOn := widget.NewEntry()
	focusOn.SetText(u.core.Settings.GetString("focus_shortcut_on", ""))
	focusOff := widget.NewEntry()
	focusOff.SetText(u.core.Settings.GetString("focus_shortcut_off", ""))
	focusDuring := widget.NewCheck("Trigger Focus during work phases", nil)
	focusDuring.SetChecked(u.core.Settings.GetBool("focus_during_work", true))

	pomodoroTab := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Work minutes", work),
			widget.NewFormItem("Short break minutes", short),
			widget.NewFormItem("Long break minutes", long),
			widget.NewFormItem("Cycles before long break", cycles),
		),
		auto,
		widget.NewForm(
			widget.NewFormItem("Focus Shortcut (on)", focusOn),
			widget.NewFormItem("Focus Shortcut (off)", focusOff),
		),
		focusDuring,
		widget.NewLabel("Focus shortcuts run via the macOS Shortcuts app and are ignored on other systems."),
	)

	tabs := container.NewAppTabs(
		container.NewTabItem("General", container.NewPadded(general)),
		container.NewTabItem("Pomodoro", container.NewPadded(pomodoroTab)),
		container.NewTabItem("Presets", container.NewPadded(u.presetsPanel())),
		container.NewTabItem("Alarms", container.NewPadded(u.alarmsPanel())),
		container.NewTabItem("History", container.NewPadded(u.historyPanel())),
		container.NewTabItem("About", container.NewPadded(u.aboutPanel())),
	)
	if tabIndex >= 0 && tabIndex < len(tabs.Items) {
		tabs.SelectIndex(tabIndex)
	}

	cancel := widget.NewButton("Cancel", func() { w.Close() })
	save := widget.NewButton("Save", func() {
		cfg := u.core.Settings.Pomodoro()
		cfg.WorkMinutes = readInt(work, cfg.WorkMinutes)
		cfg.ShortBreakMinutes = readInt(short, cfg.ShortBreakMinutes)
		cfg.LongBreakMinutes = readInt(long, cfg.LongBreakMinutes)
		cfg.CyclesBeforeLongBreak = readInt(cycles, cfg.CyclesBeforeLongBreak)
		cfg.AutoStartNext = auto.Checked
		u.core.Settings.SetPomodoro(cfg)

		if soundSel.Selected != "" {
			u.core.Settings.Data["default_sound"] = soundSel.Selected
		}
		u.core.Settings.Data["icon_theme"] = uncapitalize(themeSel.Selected)
		u.core.Settings.Data["snooze_minutes"] = readInt(snooze, u.core.Settings.SnoozeMinutes())

		u.core.Settings.Data["focus_shortcut_on"] = focusOn.Text
		u.core.Settings.Data["focus_shortcut_off"] = focusOff.Text
		u.core.Settings.Data["focus_during_work"] = focusDuring.Checked

		u.core.Settings.Data["hotkey_action"] = "none"
		for _, a := range hotkey.Actions {
			if a.Label == hkAction.Selected {
				u.core.Settings.Data["hotkey_action"] = a.ID
			}
		}
		if hkKey.Selected == "" || hkKey.Selected == "(none)" {
			u.core.Settings.Data["hotkey_key"] = ""
		} else {
			u.core.Settings.Data["hotkey_key"] = hkKey.Selected
		}

		if launch.Checked != u.core.Settings.LaunchAtLogin() {
			if err := u.applyLaunchAtLogin(launch.Checked); err != nil {
				u.alert("Couldn't update launch-at-login: " + err.Error())
			}
		}
		_ = u.core.Settings.Save()
		u.applySettingsChanges()
		w.Close()
	})
	save.Importance = widget.HighImportance

	w.SetContent(container.NewBorder(
		nil,
		container.NewPadded(container.NewHBox(layout.NewSpacer(), cancel, save)),
		nil, nil, tabs,
	))
	w.Resize(fyne.NewSize(560, 520))
	w.CenterOnScreen()
	showApp()
	w.Show()
}

func (u *UI) presetsPanel() fyne.CanvasObject {
	selected := -1
	list := widget.NewList(
		func() int { return len(u.core.Settings.Presets()) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			p := u.core.Settings.Presets()[i]
			o.(*widget.Label).SetText(p.Label + "  " + sessions.FormatClock(p.Seconds))
		},
	)
	list.OnSelected = func(id widget.ListItemID) { selected = int(id) }
	refresh := func() { list.Refresh() }

	add := widget.NewButton("Add", func() {
		u.askPreset(nil, func(p settings.Preset) {
			u.core.AddPreset(p)
			refresh()
			u.rebuildMenu()
		})
	})
	edit := widget.NewButton("Edit", func() {
		presets := u.core.Settings.Presets()
		if selected < 0 || selected >= len(presets) {
			return
		}
		idx := selected
		cur := presets[idx]
		u.askPreset(&cur, func(p settings.Preset) {
			u.core.UpdatePreset(idx, p)
			refresh()
			u.rebuildMenu()
		})
	})
	del := widget.NewButton("Delete", func() {
		presets := u.core.Settings.Presets()
		if selected < 0 || selected >= len(presets) {
			return
		}
		idx := selected
		u.confirm("Delete this preset?", "Delete", func() {
			u.core.DeletePreset(idx)
			refresh()
			u.rebuildMenu()
		})
	})
	return container.NewBorder(
		nil,
		container.NewHBox(add, edit, del),
		nil, nil, list,
	)
}

func (u *UI) alarmsPanel() fyne.CanvasObject {
	selected := -1
	list := widget.NewList(
		func() int { return len(u.core.Alarms) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(u.core.Alarms[i].MenuText())
		},
	)
	list.OnSelected = func(id widget.ListItemID) { selected = int(id) }
	refresh := func() { list.Refresh() }

	add := widget.NewButton("Add", func() {
		u.editAlarm(nil, func(al *alarms.Alarm) {
			u.core.AddAlarm(al)
			refresh()
			u.rebuildMenu()
		})
	})
	edit := widget.NewButton("Edit", func() {
		if selected < 0 || selected >= len(u.core.Alarms) {
			return
		}
		u.editAlarm(u.core.Alarms[selected], func(al *alarms.Alarm) {
			u.core.UpdateAlarm(al)
			refresh()
			u.rebuildMenu()
		})
	})
	del := widget.NewButton("Delete", func() {
		if selected < 0 || selected >= len(u.core.Alarms) {
			return
		}
		id := u.core.Alarms[selected].ID
		u.confirm("Delete this alarm?", "Delete", func() {
			u.core.DeleteAlarm(id)
			refresh()
			u.rebuildMenu()
		})
	})
	return container.NewBorder(
		nil,
		container.NewHBox(add, edit, del),
		nil, nil, list,
	)
}

func (u *UI) historyPanel() fyne.CanvasObject {
	format := func() []string {
		rows, err := u.core.History.All()
		if err != nil {
			return nil
		}
		out := make([]string, 0, len(rows))
		for i := len(rows) - 1; i >= 0; i-- {
			r := rows[i]
			name := r.Kind
			if r.Label != nil && *r.Label != "" {
				name = *r.Label
			}
			ts := r.TS
			if len(ts) >= 19 {
				ts = ts[:10] + " " + ts[11:19]
			}
			out = append(out, ts+"  "+name+" · "+r.Action)
		}
		return out
	}
	lines := format()
	count := widget.NewLabel("")
	setCount := func(n int) {
		if n == 0 {
			count.SetText("no events yet")
		} else {
			count.SetText(strconv.Itoa(n) + " events")
		}
	}
	setCount(len(lines))

	list := widget.NewList(
		func() int { return len(lines) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i >= 0 && i < len(lines) {
				o.(*widget.Label).SetText(lines[i])
			}
		},
	)
	reload := func() {
		lines = format()
		setCount(len(lines))
		list.Refresh()
	}

	exportBtn := widget.NewButton("Export…", func() { u.exportHistory(u.settingsWin, reload) })
	clearBtn := widget.NewButton("Clear", func() {
		u.confirm("Delete all history events?", "Delete", func() {
			_ = u.core.ClearHistory()
			reload()
		})
	})
	return container.NewBorder(
		nil,
		container.NewHBox(exportBtn, clearBtn, count),
		nil, nil, list,
	)
}

func (u *UI) aboutPanel() fyne.CanvasObject {
	img := canvas.NewImageFromResource(fyne.NewStaticResource("about.png", assets.PNG("about.png")))
	img.SetMinSize(fyne.NewSize(72, 72))
	img.FillMode = canvas.ImageFillContain
	title := widget.NewLabel(version.AppName + " " + version.Version)
	title.TextStyle = fyne.TextStyle{Bold: true}
	body := widget.NewLabel("A menu bar / system tray timer, stopwatch, alarm and pomodoro.")
	body.Wrapping = fyne.TextWrapWord
	guide := widget.NewButton("Quick Start Guide", func() { openURL(version.HelpURL) })
	repo := widget.NewButton("GitHub Repo", func() { openURL(version.RepoURL) })
	bmc := widget.NewButton("Buy Me a Coffee", func() { openURL(version.BMCURL) })
	return container.NewVBox(
		container.NewHBox(img, container.NewVBox(title, body)),
		guide, repo, bmc,
	)
}

func (u *UI) exportSettings(parent fyne.Window) {
	d := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if writer == nil {
			return
		}
		path := writer.URI().Path()
		_ = writer.Close()
		if err := u.core.ExportSettings(path); err != nil {
			u.alert("Couldn't write settings: " + err.Error())
			return
		}
		u.alert("Exported settings to:\n" + path)
	}, parent)
	d.SetFileName("tomatick-settings.json")
	d.Show()
}

func (u *UI) importSettings(parent fyne.Window) {
	dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if reader == nil {
			return
		}
		path := reader.URI().Path()
		_ = reader.Close()
		n, err := u.core.ImportSettings(path)
		if err != nil {
			u.alert("Couldn't read settings: " + err.Error())
			return
		}
		u.applySettingsChanges()
		parent.Close()
		u.alert("Imported " + strconv.Itoa(n) + " setting group(s).")
	}, parent).Show()
}

func (u *UI) exportHistory(parent fyne.Window, reload func()) {
	w := u.fyneApp.NewWindow("Export History")
	csvBtn := widget.NewButton("CSV", func() {
		w.Close()
		u.saveHistory(parent, "csv")
	})
	jsonBtn := widget.NewButton("JSON", func() {
		w.Close()
		u.saveHistory(parent, "json")
	})
	w.SetContent(container.NewPadded(container.NewVBox(
		widget.NewLabel("Export format:"),
		container.NewHBox(csvBtn, jsonBtn),
	)))
	w.Resize(fyne.NewSize(280, 120))
	w.CenterOnScreen()
	w.Show()
	_ = reload
}

func (u *UI) saveHistory(parent fyne.Window, fmt string) {
	home, _ := os.UserHomeDir()
	desktop := filepath.Join(home, "Desktop")
	name := "tomatick_history." + fmt
	d := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if writer == nil {
			return
		}
		path := writer.URI().Path()
		_ = writer.Close()
		var e error
		if fmt == "csv" {
			e = u.core.ExportHistoryCSV(path)
		} else {
			e = u.core.ExportHistoryJSON(path)
		}
		if e != nil {
			u.alert("Couldn't export: " + e.Error())
			return
		}
		n, _ := u.core.History.Count()
		u.alert("Exported " + strconv.Itoa(n) + " events to:\n" + path)
	}, parent)
	d.SetFileName(name)
	if st, err := os.Stat(desktop); err == nil && st.IsDir() {
		// default name is enough; user can pick Desktop
		_ = desktop
	}
	d.Show()
}

func intEntry(v int) *widget.Entry {
	e := widget.NewEntry()
	e.SetText(strconv.Itoa(v))
	return e
}

func readInt(e *widget.Entry, current int) int {
	n, err := strconv.Atoi(e.Text)
	if err != nil || n < 1 {
		return current
	}
	return n
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] -= 32
	}
	return string(r)
}

func uncapitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'A' && r[0] <= 'Z' {
		r[0] = r[0] + 32
	}
	return string(r)
}
