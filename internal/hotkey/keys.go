package hotkey

import (
	"fmt"
	"sort"
	"strings"

	"golang.design/x/hotkey"
)

// keyByName maps user-facing key names to hotkey.Key constants. The constant
// names are identical across platforms (only their underlying values differ),
// so this map is platform-independent.
var keyByName = map[string]hotkey.Key{
	"A": hotkey.KeyA, "B": hotkey.KeyB, "C": hotkey.KeyC, "D": hotkey.KeyD,
	"E": hotkey.KeyE, "F": hotkey.KeyF, "G": hotkey.KeyG, "H": hotkey.KeyH,
	"I": hotkey.KeyI, "J": hotkey.KeyJ, "K": hotkey.KeyK, "L": hotkey.KeyL,
	"M": hotkey.KeyM, "N": hotkey.KeyN, "O": hotkey.KeyO, "P": hotkey.KeyP,
	"Q": hotkey.KeyQ, "R": hotkey.KeyR, "S": hotkey.KeyS, "T": hotkey.KeyT,
	"U": hotkey.KeyU, "V": hotkey.KeyV, "W": hotkey.KeyW, "X": hotkey.KeyX,
	"Y": hotkey.KeyY, "Z": hotkey.KeyZ,
	"0": hotkey.Key0, "1": hotkey.Key1, "2": hotkey.Key2, "3": hotkey.Key3,
	"4": hotkey.Key4, "5": hotkey.Key5, "6": hotkey.Key6, "7": hotkey.Key7,
	"8": hotkey.Key8, "9": hotkey.Key9,
	"SPACE": hotkey.KeySpace,
}

// DefaultKeyName is the key used when the user has not configured one.
const DefaultKeyName = "V"

// AvailableKeys lists key names offered in the settings UI, sorted so letters
// precede digits precede Space.
var AvailableKeys = availableKeys()

func availableKeys() []string {
	out := make([]string, 0, len(keyByName))
	for k := range keyByName {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		return keyRank(out[i]) < keyRank(out[j])
	})
	return out
}

// keyRank orders letters (0..) before digits before multi-char names.
func keyRank(k string) int {
	switch {
	case len(k) == 1 && k >= "A" && k <= "Z":
		return int(k[0] - 'A')
	case len(k) == 1 && k >= "0" && k <= "9":
		return 100 + int(k[0]-'0')
	default:
		return 1000
	}
}

// parseMods converts modifier names to hotkey.Modifier values. Unknown names
// are rejected. An empty input yields the platform default.
func parseMods(names []string) ([]hotkey.Modifier, error) {
	if len(names) == 0 {
		return platformMods, nil
	}
	mods := make([]hotkey.Modifier, 0, len(names))
	for _, n := range names {
		m, ok := modByName[strings.ToLower(strings.TrimSpace(n))]
		if !ok {
			return nil, fmt.Errorf("hotkey: unknown modifier %q", n)
		}
		mods = append(mods, m)
	}
	return mods, nil
}

// parseKey converts a key name to a hotkey.Key. An empty input yields the
// default key.
func parseKey(name string) (hotkey.Key, error) {
	if strings.TrimSpace(name) == "" {
		name = DefaultKeyName
	}
	k, ok := keyByName[strings.ToUpper(strings.TrimSpace(name))]
	if !ok {
		return 0, fmt.Errorf("hotkey: unknown key %q", name)
	}
	return k, nil
}
