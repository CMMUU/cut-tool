//go:build darwin

package hotkey

import "golang.design/x/hotkey"

var platformMods = []hotkey.Modifier{hotkey.ModCmd, hotkey.ModShift}
