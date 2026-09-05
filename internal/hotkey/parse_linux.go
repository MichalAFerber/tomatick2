//go:build linux

package hotkey

import "golang.design/x/hotkey"

func parseCombo(combo string) ([]hotkey.Modifier, hotkey.Key, bool) {
	if combo == "" {
		return nil, 0, false
	}
	var mods []hotkey.Modifier
	var key hotkey.Key
	var haveKey bool
	for _, r := range combo {
		switch r {
		case '⌘':
			mods = append(mods, hotkey.Mod4) // Super
		case '⌥':
			mods = append(mods, hotkey.Mod1) // Alt
		case '⌃':
			mods = append(mods, hotkey.ModCtrl)
		case '⇧':
			mods = append(mods, hotkey.ModShift)
		default:
			k, ok := keyFromRune(r)
			if !ok {
				return nil, 0, false
			}
			key, haveKey = k, true
		}
	}
	if !haveKey || len(mods) == 0 {
		return nil, 0, false
	}
	return mods, key, true
}
