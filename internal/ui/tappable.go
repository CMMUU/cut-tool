package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// tappable wraps an arbitrary CanvasObject and fires onTap when clicked,
// giving list rows and sidebar entries click behavior without a button's
// visual chrome. It also reports hover enter/leave via onHover so callers can
// render a pre-selection highlight that follows the mouse.
type tappable struct {
	widget.BaseWidget
	content fyne.CanvasObject
	onTap   func()
	onHover func(bool)
}

func newTappable(content fyne.CanvasObject, onTap func()) *tappable {
	t := &tappable{content: content, onTap: onTap}
	t.ExtendBaseWidget(t)
	return t
}

// newHoverTappable is newTappable plus a hover callback (true on enter, false
// on leave).
func newHoverTappable(content fyne.CanvasObject, onTap func(), onHover func(bool)) *tappable {
	t := &tappable{content: content, onTap: onTap, onHover: onHover}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tappable) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.content)
}

func (t *tappable) Tapped(_ *fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

// --- desktop.Hoverable ---

func (t *tappable) MouseIn(_ *desktop.MouseEvent) {
	if t.onHover != nil {
		t.onHover(true)
	}
}

func (t *tappable) MouseOut() {
	if t.onHover != nil {
		t.onHover(false)
	}
}

func (t *tappable) MouseMoved(_ *desktop.MouseEvent) {}

// Cursor shows a pointer over hoverable rows.
func (t *tappable) Cursor() desktop.Cursor { return desktop.PointerCursor }

