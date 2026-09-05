// Package hotkey registers a single global key combo.
//
// Combos use the same symbol strings as the Python app (⌥⌘P, ⌃⌥⌘T, …).
// Registration is best-effort: without Accessibility trust on macOS, or
// if the platform helper is missing, Configure degrades to a no-op.
package hotkey

import (
	"sync"

	"golang.design/x/hotkey"
)

// Keys are the combos offered in Settings. Empty string means none.
var Keys = []string{"", "⌥⌘P", "⌥⌘T", "⌥⌘K", "⌃⌥⌘P", "⌃⌥⌘T", "⌃⌥⌘K"}

// Actions are (id, label) pairs for the action dropdown.
var Actions = []struct{ ID, Label string }{
	{"none", "(disabled)"},
	{"pomodoro", "Start Pomodoro"},
	{"timer", "Start Timer"},
	{"keepawake", "Toggle Keep Awake"},
}

// Manager owns at most one registered global hotkey.
type Manager struct {
	mu     sync.Mutex
	hk     *hotkey.Hotkey
	cancel chan struct{}
}

// Configure replaces the current hotkey. An empty combo unregisters.
func (m *Manager) Configure(combo string, callback func()) {
	m.Stop()
	mods, key, ok := parseCombo(combo)
	if !ok || callback == nil {
		return
	}
	hk := hotkey.New(mods, key)
	if err := hk.Register(); err != nil {
		return
	}
	cancel := make(chan struct{})
	m.mu.Lock()
	m.hk = hk
	m.cancel = cancel
	m.mu.Unlock()
	go func() {
		for {
			select {
			case <-cancel:
				return
			case <-hk.Keydown():
				callback()
			}
		}
	}()
}

// Stop unregisters the hotkey.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		close(m.cancel)
		m.cancel = nil
	}
	if m.hk != nil {
		_ = m.hk.Unregister()
		m.hk = nil
	}
}

func keyFromRune(r rune) (hotkey.Key, bool) {
	switch r {
	case 'P', 'p':
		return hotkey.KeyP, true
	case 'T', 't':
		return hotkey.KeyT, true
	case 'K', 'k':
		return hotkey.KeyK, true
	}
	return 0, false
}
