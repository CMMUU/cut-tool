// Package hotkey registers a global keyboard shortcut that shows the
// clipboard-history window regardless of which app has focus. The shortcut
// is user-configurable and can be re-registered at runtime.
package hotkey

import (
	"context"
	"sync"

	"golang.design/x/hotkey"
)

// Manager owns the currently registered global hotkey and can rebind it to a
// new combination at runtime. It is safe for concurrent use.
type Manager struct {
	show func()

	mu     sync.Mutex
	hk     *hotkey.Hotkey
	cancel context.CancelFunc // stops the current listen loop
	parent context.Context
}

// NewManager creates a Manager that calls show on every hotkey keydown. The
// parent context bounds the lifetime of all registrations; when it is
// canceled the active hotkey is unregistered.
func NewManager(parent context.Context, show func()) *Manager {
	return &Manager{show: show, parent: parent}
}

// Apply (re)registers the hotkey for the given modifier and key names. Empty
// inputs fall back to the platform defaults. If registration of the new
// combination fails, the previous binding is left untouched and the error is
// returned, so a bad choice never leaves the app with no working shortcut.
func (m *Manager) Apply(modNames []string, keyName string) error {
	mods, err := parseMods(modNames)
	if err != nil {
		return err
	}
	key, err := parseKey(keyName)
	if err != nil {
		return err
	}

	hk := hotkey.New(mods, key)
	if err := hk.Register(); err != nil {
		return err
	}

	m.mu.Lock()
	// Tear down the previous registration only after the new one succeeded.
	if m.cancel != nil {
		m.cancel()
	}
	if m.hk != nil {
		_ = m.hk.Unregister()
	}
	ctx, cancel := context.WithCancel(m.parent)
	m.hk = hk
	m.cancel = cancel
	m.mu.Unlock()

	go m.listen(ctx, hk)
	return nil
}

// listen forwards keydown events to show until ctx is canceled.
func (m *Manager) listen(ctx context.Context, hk *hotkey.Hotkey) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-hk.Keydown():
			if m.show != nil {
				m.show()
			}
		}
	}
}

// Close unregisters the active hotkey. It is invoked implicitly when the
// parent context is canceled, but may be called explicitly too.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	if m.hk != nil {
		_ = m.hk.Unregister()
		m.hk = nil
	}
}
