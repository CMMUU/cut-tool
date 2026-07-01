//go:build darwin

package hotkey

import "golang.design/x/hotkey"

// platformMods is the default modifier set when the user has not configured
// one (Cmd+Shift on macOS).
var platformMods = []hotkey.Modifier{hotkey.ModCmd, hotkey.ModShift}

// DefaultModNames is the default modifier set as user-facing names.
var DefaultModNames = []string{"cmd", "shift"}

// modByName maps user-facing modifier names to macOS modifier constants.
var modByName = map[string]hotkey.Modifier{
	"cmd":   hotkey.ModCmd,
	"ctrl":  hotkey.ModCtrl,
	"shift": hotkey.ModShift,
	"alt":   hotkey.ModOption, // "alt" is Option on macOS
}

// AvailableMods lists modifier names offered in the settings UI, in order.
var AvailableMods = []string{"cmd", "ctrl", "shift", "alt"}
