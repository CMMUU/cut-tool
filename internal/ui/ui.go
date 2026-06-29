// Package ui owns the Fyne main window: a search box and a scrollable list
// of clipboard history entries. Selecting an entry writes it back to the
// clipboard and hides the window.
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Deps wires the UI to the rest of the app.
type Deps struct {
	// OnSelect is called with the full content when the user picks an entry.
	OnSelect func(content string)
	// OnSearch returns display items matching query (called on every keystroke).
	OnSearch func(query string) []Item
	// OnPin toggles the pinned state of an entry. nil = feature disabled.
	OnPin func(hash string, pinned bool)
}

// Item is one row in the history list.
type Item struct {
	Preview   string
	Content   string // full text handed to OnSelect
	Hash      string
	Pinned    bool
	Timestamp string
}

// Window wraps the Fyne application and main window.
type Window struct {
	app  fyne.App
	win  fyne.Window
	deps Deps
}

// NewWindow builds the Fyne application and window. Call Show to make it
// visible for the first time.
func NewWindow(deps Deps) *Window {
	a := app.New()
	w := a.NewWindow("cut-tool — Clipboard History")
	w.Resize(fyne.NewSize(520, 420))
	w.SetCloseIntercept(w.Hide) // hide to tray instead of quitting

	uw := &Window{app: a, win: w, deps: deps}
	uw.buildContent()
	return uw
}

func (w *Window) buildContent() {
	var items []Item

	list := widget.NewList(
		func() int { return len(items) },
		// Template: star button + preview label.
		func() fyne.CanvasObject {
			btn := widget.NewButton("☆", nil)
			lbl := widget.NewLabel("")
			lbl.Truncation = fyne.TextTruncateEllipsis
			return container.NewBorder(nil, nil, btn, nil, lbl)
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			if id >= len(items) {
				return
			}
			c := o.(*fyne.Container)
			btn := c.Objects[1].(*widget.Button) // Border: leading = Objects[1]
			lbl := c.Objects[0].(*widget.Label)  // Border: content = Objects[0]

			item := items[id]
			if item.Pinned {
				btn.SetText("★")
			} else {
				btn.SetText("☆")
			}
			lbl.SetText(item.Preview)

			// Capture hash at update time, not inside the closure, to avoid
			// stale-id issues when the list reuses this CanvasObject.
			hash := item.Hash
			isPinned := item.Pinned
			btn.OnTapped = func() {
				if w.deps.OnPin == nil {
					return
				}
				isPinned = !isPinned
				w.deps.OnPin(hash, isPinned)
				if isPinned {
					btn.SetText("★")
				} else {
					btn.SetText("☆")
				}
			}
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		if id >= len(items) {
			return
		}
		if w.deps.OnSelect != nil {
			w.deps.OnSelect(items[id].Content)
		}
		list.UnselectAll()
		w.win.Hide()
	}

	search := widget.NewEntry()
	search.SetPlaceHolder("Search history…")
	search.OnChanged = func(q string) {
		if w.deps.OnSearch != nil {
			items = w.deps.OnSearch(q)
		}
		list.Refresh()
	}

	// Populate on first load.
	if w.deps.OnSearch != nil {
		items = w.deps.OnSearch("")
	}

	w.win.SetContent(container.NewBorder(search, nil, nil, nil, list))
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
