package ui

import (
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/systray"

	"github.com/MichalAFerber/tomatick2/internal/core"
	"github.com/MichalAFerber/tomatick2/internal/sessions"
	"github.com/MichalAFerber/tomatick2/internal/version"
)

func (u *UI) setupTray() {
	if u.desk == nil {
		return
	}
	if u.idleIcon != nil {
		u.desk.SetSystemTrayIcon(u.idleIcon)
	}
	u.rebuildMenu()
}

func (u *UI) rebuildMenu() {
	if u.desk == nil {
		return
	}
	items := []*fyne.MenuItem{}

	start := fyne.NewMenuItem("Start", nil)
	startChildren := []*fyne.MenuItem{
		fyne.NewMenuItem("Timer…", u.askTimer),
		fyne.NewMenuItem("Stopwatch", u.askStopwatch),
		fyne.NewMenuItem("Pomodoro", u.askPomodoro),
		fyne.NewMenuItem("Alarm…", func() { u.openSettings(3) }),
	}
	presets := u.core.Settings.Presets()
	if len(presets) > 0 {
		startChildren = append(startChildren, fyne.NewMenuItemSeparator())
		for _, p := range presets {
			pp := p
			title := pp.Label + "  " + sessions.FormatClock(pp.Seconds)
			startChildren = append(startChildren, fyne.NewMenuItem(title, func() {
				u.core.StartPreset(pp)
				u.refresh()
			}))
		}
	}
	start.ChildMenu = fyne.NewMenu("", startChildren...)
	items = append(items, start, fyne.NewMenuItemSeparator())

	if len(u.core.Sessions) > 0 || len(u.core.Firing) > 0 {
		header := fyne.NewMenuItem("Active", nil)
		header.Disabled = true
		items = append(items, header)
		for _, sess := range u.core.Sessions {
			items = append(items, u.sessionItem(sess))
		}
		for _, f := range u.core.Firing {
			items = append(items, u.firingItem(f))
		}
		items = append(items, fyne.NewMenuItemSeparator())
	}

	items = append(items, fyne.NewMenuItem("Settings…", func() { u.openSettings(0) }))

	keep := fyne.NewMenuItem("Keep awake", u.toggleKeepAwake)
	keep.Checked = u.keepAwake.Active()
	items = append(items, keep, fyne.NewMenuItemSeparator())

	items = append(items,
		fyne.NewMenuItem("Quick Start Guide", func() { openURL(version.HelpURL) }),
		fyne.NewMenuItem("Quit", u.quit),
	)

	u.desk.SetSystemTrayMenu(fyne.NewMenu(version.AppName, items...))
}

func (u *UI) sessionItem(sess sessions.Session) *fyne.MenuItem {
	id := sess.ID()
	parent := fyne.NewMenuItem(sess.MenuText(), nil)
	toggleTitle := "Pause"
	if sess.State() == sessions.Paused {
		toggleTitle = "Resume"
	}
	children := []*fyne.MenuItem{
		fyne.NewMenuItem(toggleTitle, func() {
			u.core.TogglePause(id)
			u.refresh()
		}),
	}
	if _, ok := sess.(*sessions.Pomodoro); ok {
		children = append(children, fyne.NewMenuItem("Skip phase", func() {
			u.core.SkipPhase(id)
			u.refresh()
		}))
	} else {
		children = append(children, fyne.NewMenuItem("Reset", func() {
			u.core.ResetSession(id)
			u.refresh()
		}))
	}
	if _, ok := sess.(*sessions.Stopwatch); ok {
		children = append(children, fyne.NewMenuItem("Lap", func() {
			u.core.Lap(id)
		}))
	}
	children = append(children, fyne.NewMenuItem("Stop", func() {
		u.core.StopSession(id)
		u.refresh()
	}))
	pin := fyne.NewMenuItem("Pin as primary", func() {
		u.core.Pin(id)
		u.refresh()
	})
	pin.Checked = u.core.PinnedID == id
	children = append(children, pin)
	parent.ChildMenu = fyne.NewMenu("", children...)
	return parent
}

func (u *UI) firingItem(f core.Firing) *fyne.MenuItem {
	al := f.Alarm
	parent := fyne.NewMenuItem(core.FiringTitle(f), nil)
	parent.ChildMenu = fyne.NewMenu("",
		fyne.NewMenuItem("Dismiss", func() {
			u.core.DismissAlarm(al)
			u.refresh()
		}),
		fyne.NewMenuItem("Snooze", func() {
			u.core.SnoozeAlarm(al)
			u.refresh()
		}),
	)
	return parent
}

func (u *UI) updateTitle() {
	text := u.core.TitleText()
	switch runtime.GOOS {
	case "darwin":
		systray.SetTitle(text)
	case "linux", "freebsd", "openbsd", "netbsd":
		if strings.TrimSpace(text) == "" {
			systray.SetTitle(version.AppName)
		} else {
			systray.SetTitle(strings.TrimSpace(text))
		}
	}
	if strings.TrimSpace(text) == "" {
		systray.SetTooltip(version.AppName)
	} else {
		systray.SetTooltip(version.AppName + text)
	}
	if u.desk == nil {
		return
	}
	if u.core.Alarming() {
		return // shake animation owns the icon
	}
	if u.idleIcon != nil {
		u.desk.SetSystemTrayIcon(u.idleIcon)
	}
}
