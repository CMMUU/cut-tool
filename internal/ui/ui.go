// Package ui owns the Fyne main window: a search box and a scrollable list
// of clipboard history entries. Selecting an entry writes it back to the
// clipboard and hides the window.
package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"
)

// Deps wires the UI to the rest of the app.
type Deps struct {
	// OnSelect is called with the selected item when the user picks an entry.
	OnSelect func(item Item)
	// OnSearch returns display items matching query (called on every keystroke).
	OnSearch func(query string) []Item
	// OnPin toggles the pinned state of an entry. nil = feature disabled.
	OnPin func(hash string, pinned bool)
	// OnDelete removes a single entry by hash. nil = feature disabled.
	OnDelete func(hash string)
	// OnClear removes every entry. nil = feature disabled.
	OnClear func()
	// GetRetentionDays returns the current unpinned-entry retention period.
	GetRetentionDays func() int
	// SetRetentionDays persists a new retention period.
	SetRetentionDays func(days int) error

	// Hotkey configuration. All are optional; if AvailableKeys is empty the
	// hotkey section is hidden. Modifier/key names are plain strings so the
	// ui package need not import the hotkey package.
	AvailableMods []string                  // selectable modifier names, in order
	AvailableKeys []string                  // selectable key names, in order
	GetHotkey     func() ([]string, string) // current mods + key
	SetHotkey     func(mods []string, key string) error

	// Defaults for the "Restore Defaults" action.
	DefaultRetentionDays int
	DefaultHotkeyMods    []string
	DefaultHotkeyKey     string
}

// Item is one row in the history list.
type Item struct {
	Type      string // "text" or "image"
	Preview   string
	Content   string // full text handed to OnSelect (text entries)
	Data      []byte // binary payload, e.g. PNG bytes (image entries)
	Hash      string
	Pinned    bool
	Timestamp string
}

// Category identifiers for the sidebar.
const (
	catAll     = "All"
	catText    = "Text"
	catLinks   = "Links"
	catImages  = "Images"
	catStarred = "Starred"
)

// Window wraps the Fyne application and main window.
type Window struct {
	app   fyne.App
	win   fyne.Window
	deps  Deps
	theme *appTheme

	// Three-column view state.
	master   []Item // full set from the last load
	category string // active sidebar category
	query    string // active search query
	selected *Item  // entry shown in the preview panel

	listBox    *fyne.Container // scrollable column of row cards
	previewBox *fyne.Container // right preview panel content
	sidebarBox *fyne.Container // left category list
	statusLbl  *widget.Label   // bottom-left status text
}

// NewWindow builds the Fyne application and window. Call Show to make it
// visible for the first time.
func NewWindow(deps Deps) *Window {
	a := app.New()
	th := newAppTheme()
	a.Settings().SetTheme(th) // light purple, CJK-capable

	w := a.NewWindow("cut-tool")
	w.Resize(fyne.NewSize(1040, 660))
	w.SetCloseIntercept(w.Hide) // hide to tray instead of quitting

	uw := &Window{app: a, win: w, deps: deps, theme: th, category: catAll}
	uw.buildContent()
	return uw
}

// buildContent and the three-column layout live in layout.go.

// capitalize upper-cases the first letter of s (ASCII), for modifier labels.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// Show makes the window visible and rebuilds the list to reflect latest history.
func (w *Window) Show() {
	w.buildContent()
	w.win.Show()
	w.win.RequestFocus()
}

// Hide hides the window without destroying it.
func (w *Window) Hide() { w.win.Hide() }

// App returns the underlying fyne.App so callers (e.g. tray.Setup) can
// attach system tray menus before the event loop starts.
func (w *Window) App() fyne.App { return w.app }

// Run starts the Fyne event loop. Must be called on the main thread.
func (w *Window) Run() { w.app.Run() }
