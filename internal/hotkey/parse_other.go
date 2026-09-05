//go:build !darwin && !windows && !linux

package hotkey

import "golang.design/x/hotkey"

func parseCombo(combo string) ([]hotkey.Modifier, hotkey.Key, bool) {
	return nil, 0, false
}
