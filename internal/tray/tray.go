// Package tray attaches a system-tray menu to a running Fyne desktop app.
// Using Fyne's built-in desktop.App avoids the dual-main-thread conflict
// that would arise from running getlantern/systray alongside Fyne on macOS.
package tray

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

// Options configures the tray menu callbacks.
type Options struct {
	OnShow     func()
	OnSettings func()
	OnClear    func()
	OnExit     func()
}

// Setup attaches the system-tray menu to a Fyne app. Must be called before
// app.Run(). No-ops in non-desktop environments (e.g. mobile, test driver).
func Setup(a fyne.App, opts Options) {
	da, ok := a.(desktop.App)
	if !ok {
		return
	}
	items := []*fyne.MenuItem{
		fyne.NewMenuItem("Show History", opts.OnShow),
	}
	if opts.OnSettings != nil {
		items = append(items, fyne.NewMenuItem("Settings…", opts.OnSettings))
	}
	items = append(items,
		fyne.NewMenuItem("Clear History", opts.OnClear),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quit", func() {
			if opts.OnExit != nil {
				opts.OnExit()
			}
			a.Quit()
		}),
	)
	da.SetSystemTrayMenu(fyne.NewMenu("cut-tool", items...))
}
