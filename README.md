# Tomatick

[![CI](https://github.com/MichalAFerber/tomatick2/actions/workflows/ci.yml/badge.svg)](https://github.com/MichalAFerber/tomatick2/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.25+-00ADD8.svg)](https://go.dev/)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Windows%20%7C%20Linux-lightgrey.svg)](https://github.com/MichalAFerber/tomatick2)

A **menu bar / system tray** timer, stopwatch, alarm and pomodoro — all in one
icon, with a timestamped history of everything you run.

This is a Go rewrite of the original [macOS-only Python
Tomatick](https://github.com/MichalAFerber/tomatick), built so a single codebase
runs on **macOS, Windows, and Linux**. Config and history use the same JSON /
SQLite schema, so a `config.json` from the Python app loads here.

## Features

- **Timer** — natural-language durations (`25m`, `1h30m`, `90s`, `2:30`), with a
  live `M:SS` countdown in the tray (menu-bar title on macOS; tooltip + menu on
  Windows). Pause / Reset / Stop.
- **Stopwatch** — counts up, with optional laps. Pause / Reset / Stop.
- **Pomodoro** — 25/5/15 by default, auto-advancing phases (🍅 work, ☕ break),
  long break after every 4th work cycle. Pause / Skip phase / Stop.
- **Alarms** — one-shot (a specific date + time) and recurring (a time of day on
  chosen weekdays). Rings with a looping sound + notification until dismissed,
  with Snooze (9 min default).
- **Multiple at once** — run several sessions together; the tray shows your
  "primary" one live, and every active session is listed in the menu with its
  own count. **Pin** any session to make it the primary display.
- **History** — every event (started / paused / completed / phase change / alarm
  fired …) is timestamped into SQLite. See recent events in Settings and
  **Export** the full log to CSV or JSON.
- **Settings** — edit pomodoro durations, default sound, icon theme, and toggle
  **Launch at login**.

Everything is driven from the tray icon's menu: start a session → it appears in
the menu → click its item to pause/stop it (or dismiss a ringing alarm).

## Run it (development)

Requires **Go 1.25+**. On Linux you also need the usual Fyne OpenGL/X11 headers
(`libgl1-mesa-dev`, `xorg-dev`). On Windows, a `gcc` (MinGW) is required because
Fyne uses CGO.

```bash
git clone https://github.com/MichalAFerber/tomatick2.git
cd tomatick2
go test ./...
go run ./cmd/tomatick
```

A tomato icon appears in the menu bar (macOS), notification area (Windows), or
system tray (Linux). Click it to open the menu.

On Linux, a tray host that speaks StatusNotifierItem / AppIndicator is required
(GNOME needs an extension; KDE and most other desktops work out of the box).

## Run the tests

The timing, scheduling, history, and settings logic has no GUI dependencies:

```bash
go test ./internal/sessions ./internal/alarms ./internal/history ./internal/settings ./internal/core
```

`go test ./...` also compiles the UI packages (needs CGO + platform GUI libs).

## Build a standalone app

```bash
# current OS
go build -o tomatick ./cmd/tomatick

# packaged bundles (run on the target OS, or use fyne-cross)
make package-mac      # Tomatick.app, Dock-hidden
make package-windows  # Tomatick.exe
make package-linux    # Tomatick.tar.xz
```

`fyne package` must be installed. Pin it to the `fyne.io/fyne/v2` version in
`go.mod` — the packager decides the bundle layout, so a floating `@latest`
changes it under you between releases. The release workflow pins the same
version:

```bash
go install fyne.io/fyne/v2/cmd/fyne@v2.8.1
```

On macOS the bundle is unsigned; on first launch right-click → Open to get past
Gatekeeper, or:

```bash
xattr -dr com.apple.quarantine /Applications/Tomatick.app
```

Tomatick is a **tray** app — no Dock icon on macOS (runtime
`LSUIElement` / accessory policy). Look for the tomato in the menu bar. To start
it at login, use the icon's menu → **Settings → Launch at login**.

## Where data lives

| OS      | Directory |
|---------|-----------|
| macOS   | `~/Library/Application Support/Tomatick/` |
| Windows | `%AppData%\Tomatick\` |
| Linux   | `~/.local/share/Tomatick/` |

```
config.json     # pomodoro settings, alarms, sound, launch-at-login
history.db      # SQLite event history
```

Honor `TOMATICK_SUPPORT_DIR` to redirect both files (used by tests).

## Platform notes

| Capability        | macOS                         | Windows                          | Linux                          |
|-------------------|-------------------------------|----------------------------------|--------------------------------|
| Tray title text   | yes (next to the icon)        | tooltip + menu                   | title where the tray supports it |
| Sounds            | `/System/Library/Sounds`      | `%WINDIR%\Media`                 | bundled ding + `paplay`/`aplay` |
| Keep awake        | `caffeinate`                  | `SetThreadExecutionState`        | `systemd-inhibit`              |
| Launch at login   | LaunchAgent `us.tomatick`     | HKCU Run key                     | `~/.config/autostart/`         |
| Focus / DND       | Shortcuts app, by name        | ignored                          | ignored                        |
| Global hotkey     | Accessibility permission      | `RegisterHotKey`                 | X11 grab                       |

## Project layout

```
cmd/tomatick/          # main
internal/
  sessions/            # Timer / Stopwatch / Pomodoro (pure logic)
  alarms/              # Alarm model + scheduling (pure logic)
  history/             # SQLite history store
  settings/            # config.json load/save
  core/                # tick, wiring, no GUI
  ui/                  # Fyne tray + settings windows
  sound/ notify/ keepawake/ autostart/ hotkey/ focus/
  assets/              # tomato art, bundled ding.wav
```

## Website

The product site lives at **[tomatick.us](https://tomatick.us)** (source still
in the [Python repo](https://github.com/MichalAFerber/tomatick/tree/main/site)).

## Credits

| Component | License | Used for |
|-----------|---------|----------|
| [Fyne](https://fyne.io) | BSD-3-Clause | windows, dialogs, system tray |
| [fyne.io/systray](https://github.com/fyne-io/systray) | Apache-2.0 | tray title / icon |
| [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) | BSD-3-Clause | history database (pure Go) |
| [golang.design/x/hotkey](https://github.com/golang-design/hotkey) | MIT | global hotkey |

The menu-bar tomato icon (red / white / black themes) and the app icon are
original artwork, generated by [`scripts/make_icon.py`](scripts/make_icon.py).

## License

Released under the [MIT License](LICENSE) — © 2026 Michal Ferber.
