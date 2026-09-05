// Package keepawake is a caffeine-style "don't sleep" toggle.
package keepawake

import "sync"

// KeepAwake holds a platform sleep-inhibit assertion.
type KeepAwake struct {
	mu     sync.Mutex
	active bool
	stop   func()
}

// Active reports whether the assertion is held.
func (k *KeepAwake) Active() bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.active
}

// On takes the assertion.
func (k *KeepAwake) On() {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.active {
		return
	}
	stop, err := inhibit()
	if err != nil || stop == nil {
		return
	}
	k.stop = stop
	k.active = true
}

// Off releases the assertion.
func (k *KeepAwake) Off() {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.stop != nil {
		k.stop()
		k.stop = nil
	}
	k.active = false
}

// Toggle flips the assertion. Returns the new active state.
func (k *KeepAwake) Toggle() bool {
	if k.Active() {
		k.Off()
	} else {
		k.On()
	}
	return k.Active()
}
