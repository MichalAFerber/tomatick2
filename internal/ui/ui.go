package ui

import (
	"log"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/MichalAFerber/tomatick2/internal/assets"
	"github.com/MichalAFerber/tomatick2/internal/autostart"
	"github.com/MichalAFerber/tomatick2/internal/core"
	"github.com/MichalAFerber/tomatick2/internal/focus"
	"github.com/MichalAFerber/tomatick2/internal/hotkey"
	"github.com/MichalAFerber/tomatick2/internal/keepawake"
	"github.com/MichalAFerber/tomatick2/internal/notify"
	"github.com/MichalAFerber/tomatick2/internal/sound"
	"github.com/MichalAFerber/tomatick2/internal/version"
)

// UI is the Fyne front-end for a core.App.
type UI struct {
	fyneApp fyne.App
	desk    desktop.App
	core    *core.App

	alarmPlayer *sound.Player
	cuePlayer   *sound.Player
	keepAwake   *keepawake.KeepAwake
	hotkeys     *hotkey.Manager

	iconCache   map[string]fyne.Resource
	shakeFrames []fyne.Resource
	idleIcon    fyne.Resource
	animIdx     int
	stopped     bool

	settingsWin fyne.Window
}

// Main is the application entry point.
func Main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-version", "--version", "-v":
			println(version.Version)
			return
		}
	}

	a := app.NewWithID(version.BundleID)
	a.SetIcon(fyne.NewStaticResource("about.png", assets.PNG("about.png")))

	coreApp, err := core.Open()
	if err != nil {
		log.Fatalf("tomatick: %v", err)
	}

	u := &UI{
		fyneApp:     a,
		core:        coreApp,
		alarmPlayer: &sound.Player{},
		cuePlayer:   &sound.Player{},
		keepAwake:   &keepawake.KeepAwake{},
		hotkeys:     &hotkey.Manager{},
		iconCache:   map[string]fyne.Resource{},
	}
	if desk, ok := a.(desktop.App); ok {
		u.desk = desk
	}

	u.wireHooks()
	u.loadIcons()
	u.setupTray()

	a.Lifecycle().SetOnStarted(func() {
		hideDockIcon()
		u.configureHotkey()
		u.refresh()
		u.startTickers()
	})
	a.Lifecycle().SetOnStopped(func() {
		u.shutdown()
	})

	a.Run()
}

func (u *UI) wireHooks() {
	u.core.Hooks = core.Hooks{
		Notify:    notify.Send,
		PlayAlarm: u.alarmPlayer.Play,
		StopAlarm: u.alarmPlayer.Stop,
		PlayCue:   func(name string) { u.cuePlayer.Play(name, false) },
		RunFocus:  func(name string) { focus.Run(name) },
	}
}

func (u *UI) resource(name string, data []byte) fyne.Resource {
	if r, ok := u.iconCache[name]; ok {
		return r
	}
	r := fyne.NewStaticResource(name, data)
	u.iconCache[name] = r
	return r
}

func (u *UI) loadIcons() {
	theme := u.core.Settings.GetString("icon_theme", "red")
	u.idleIcon = u.resource("mb_"+theme+".png", assets.IdleIcon(theme))
	u.shakeFrames = nil
	for i, frame := range assets.ShakeFrames(theme) {
		name := theme + "_shake_" + itoa(i)
		u.shakeFrames = append(u.shakeFrames, u.resource(name, frame))
	}
	u.animIdx = 0
}

func (u *UI) startTickers() {
	u.repeat(time.Second, u.onTick)
	u.repeat(90*time.Millisecond, u.onAnim)
}

func (u *UI) repeat(d time.Duration, fn func()) {
	var schedule func()
	schedule = func() {
		time.AfterFunc(d, func() {
			fyne.Do(func() {
				if u.stopped {
					return
				}
				fn()
				schedule()
			})
		})
	}
	schedule()
}

func (u *UI) onTick() {
	res := u.core.Tick()
	if res.Structural {
		u.rebuildMenu()
	}
	u.updateTitle()
}

func (u *UI) onAnim() {
	if !u.core.Alarming() || len(u.shakeFrames) == 0 || u.desk == nil {
		return
	}
	u.desk.SetSystemTrayIcon(u.shakeFrames[u.animIdx])
	u.animIdx = (u.animIdx + 1) % len(u.shakeFrames)
}

func (u *UI) refresh() {
	u.rebuildMenu()
	u.updateTitle()
}

func (u *UI) applySettingsChanges() {
	u.configureHotkey()
	u.loadIcons()
	u.refresh()
}

func (u *UI) configureHotkey() {
	action := u.core.Settings.GetString("hotkey_action", "none")
	combo := u.core.Settings.GetString("hotkey_key", "")
	if action != "none" && combo != "" {
		u.hotkeys.Configure(combo, u.onHotkey)
	} else {
		u.hotkeys.Stop()
	}
}

func (u *UI) onHotkey() {
	fyne.Do(func() {
		switch u.core.Settings.GetString("hotkey_action", "none") {
		case "pomodoro":
			u.core.StartPomodoro("")
			u.refresh()
		case "timer":
			u.askTimer()
		case "keepawake":
			u.toggleKeepAwake()
		}
	})
}

func (u *UI) toggleKeepAwake() {
	u.keepAwake.Toggle()
	u.core.KeepAwakeActive = u.keepAwake.Active()
	action := "disabled"
	if u.keepAwake.Active() {
		action = "enabled"
	}
	_, _ = u.core.History.LogEvent("keepawake", action, "", nil, nil, time.Time{})
	u.rebuildMenu()
}

func (u *UI) applyLaunchAtLogin(want bool) error {
	if err := autostart.SetEnabled(want); err != nil {
		return err
	}
	return u.core.SetLaunchAtLogin(want)
}

func (u *UI) shutdown() {
	u.stopped = true
	u.keepAwake.Off()
	u.core.ClearFocus()
	u.hotkeys.Stop()
	u.alarmPlayer.Stop()
	u.cuePlayer.Stop()
	_ = u.core.Close()
}

func (u *UI) quit() {
	u.shutdown()
	u.fyneApp.Quit()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
