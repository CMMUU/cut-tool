//go:build linux

package hotkey

import "golang.design/x/hotkey"

// platformMods is the default modifier set when the user has not configured
// one (Ctrl+Shift on Linux/X11).
var platformMods = []hotkey.Modifier{hotkey.ModCtrl, hotkey.ModShift}

// DefaultModNames is the default modifier set as user-facing names.
var DefaultModNames = []string{"ctrl", "shift"}

// modByName maps user-facing modifier names to X11 modifier constants.
// On X11 Alt is Mod1 and Super/Meta is Mod4.
var modByName = map[string]hotkey.Modifier{
	"ctrl":  hotkey.ModCtrl,
	"shift": hotkey.ModShift,
	"alt":   hotkey.Mod1,
	"cmd":   hotkey.Mod4, // "cmd"/Super
}

// AvailableMods lists modifier names offered in the settings UI, in order.
var AvailableMods = []string{"ctrl", "shift", "alt", "cmd"}
