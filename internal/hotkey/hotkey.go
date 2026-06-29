// Package hotkey registers a global keyboard shortcut that shows the
// clipboard-history window regardless of which app has focus.
package hotkey

import (
	"context"

	"golang.design/x/hotkey"
)

// Register starts listening for the global hotkey (Cmd+Shift+V on macOS,
// Ctrl+Shift+V on Windows/Linux). Calls show on every keydown. Blocks until
// ctx is canceled, then unregisters the hotkey and returns.
func Register(ctx context.Context, show func()) error {
	hk := hotkey.New(platformMods, hotkey.KeyV)
	if err := hk.Register(); err != nil {
		return err
	}
	defer hk.Unregister()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-hk.Keydown():
			show()
		}
	}
}
