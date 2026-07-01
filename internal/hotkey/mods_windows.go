//go:build windows

package hotkey

import "golang.design/x/hotkey"

// platformMods is the default modifier set when the user has not configured
// one (Ctrl+Shift on Windows).
var platformMods = []hotkey.Modifier{hotkey.ModCtrl, hotkey.ModShift}

// DefaultModNames is the default modifier set as user-facing names.
var DefaultModNames = []string{"ctrl", "shift"}

// modByName maps user-facing modifier names to Windows modifier constants.
var modByName = map[string]hotkey.Modifier{
	"ctrl":  hotkey.ModCtrl,
	"shift": hotkey.ModShift,
	"alt":   hotkey.ModAlt,
	"cmd":   hotkey.ModWin, // "cmd" maps to the Windows key
}

// AvailableMods lists modifier names offered in the settings UI, in order.
var AvailableMods = []string{"ctrl", "shift", "alt", "cmd"}
