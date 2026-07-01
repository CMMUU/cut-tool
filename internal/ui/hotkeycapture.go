package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// hotkeyCapture is a focusable control that records a global-shortcut combo
// from real key presses. Click it to start recording; hold your modifiers and
// press a key, and the combo is captured. It implements desktop.Keyable so it
// receives raw key-down/up events (Fyne's KeyEvent alone does not report
// modifier state on this version).
type hotkeyCapture struct {
	widget.BaseWidget

	cv        fyne.Canvas
	mods      []string        // captured modifier names
	key       string          // captured key name
	held      map[string]bool // modifiers currently held down
	recording bool
	onChange  func(mods []string, key string)

	label *canvas.Text
	bg    *canvas.Rectangle
}

func newHotkeyCapture(cv fyne.Canvas, mods []string, key string, onChange func([]string, string)) *hotkeyCapture {
	h := &hotkeyCapture{
		cv:       cv,
		mods:     mods,
		key:      key,
		held:     map[string]bool{},
		onChange: onChange,
		label:    canvas.NewText("", theme.ForegroundColor()),
	}
	h.label.TextStyle = fyne.TextStyle{Bold: true}
	h.label.Alignment = fyne.TextAlignCenter
	h.updateLabel()
	h.ExtendBaseWidget(h)
	return h
}

// comboText renders the current combo as e.g. "⌘ + ⇧ + V".
func (h *hotkeyCapture) comboText() string {
	if len(h.mods) == 0 && h.key == "" {
		return "未设置 / Not set"
	}
	parts := make([]string, 0, len(h.mods)+1)
	for _, m := range h.mods {
		parts = append(parts, modGlyph(m))
	}
	if h.key != "" {
		parts = append(parts, h.key)
	}
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " + "
		}
		out += p
	}
	return out
}

func (h *hotkeyCapture) updateLabel() {
	if h.recording {
		h.label.Text = "按下组合键… / Press keys…"
	} else {
		h.label.Text = h.comboText()
	}
	h.label.Refresh()
}

// Value returns the currently captured combo.
func (h *hotkeyCapture) Value() (mods []string, key string) {
	return h.mods, h.key
}

// SetValue replaces the displayed combo (used by Restore Defaults).
func (h *hotkeyCapture) SetValue(mods []string, key string) {
	h.mods = mods
	h.key = key
	h.recording = false
	h.updateLabel()
}

func (h *hotkeyCapture) CreateRenderer() fyne.WidgetRenderer {
	rect := canvas.NewRectangle(theme.InputBackgroundColor())
	rect.CornerRadius = 8
	rect.StrokeColor = theme.PrimaryColor()
	rect.StrokeWidth = 0
	h.bg = rect
	content := container.NewStack(rect, container.NewPadded(container.NewCenter(h.label)))
	return widget.NewSimpleRenderer(content)
}

// Tapped starts recording and grabs keyboard focus.
func (h *hotkeyCapture) Tapped(_ *fyne.PointEvent) {
	h.recording = true
	h.held = map[string]bool{}
	h.updateLabel()
	h.highlight(true)
	if h.cv != nil {
		h.cv.Focus(h)
	}
}

// --- fyne.Focusable ---

func (h *hotkeyCapture) FocusGained() {}
func (h *hotkeyCapture) FocusLost() {
	if h.recording {
		h.recording = false
		h.updateLabel()
	}
	h.highlight(false)
}
func (h *hotkeyCapture) TypedRune(_ rune)          {}
func (h *hotkeyCapture) TypedKey(_ *fyne.KeyEvent) {}

// --- desktop.Keyable ---

func (h *hotkeyCapture) KeyDown(e *fyne.KeyEvent) {
	if !h.recording {
		return
	}
	if mod, ok := modNameForKey(e.Name); ok {
		h.held[mod] = true
		return
	}
	key, ok := keyNameForKey(e.Name)
	if !ok {
		return // ignore unsupported keys
	}
	// Snapshot held modifiers in a stable order.
	var mods []string
	for _, name := range []string{"ctrl", "shift", "alt", "cmd"} {
		if h.held[name] {
			mods = append(mods, name)
		}
	}
	if len(mods) == 0 {
		return // require at least one modifier
	}
	h.mods = mods
	h.key = key
	h.recording = false
	h.updateLabel()
	h.highlight(false)
	if h.cv != nil {
		h.cv.Unfocus()
	}
	if h.onChange != nil {
		h.onChange(h.mods, h.key)
	}
}

func (h *hotkeyCapture) KeyUp(e *fyne.KeyEvent) {
	if mod, ok := modNameForKey(e.Name); ok {
		delete(h.held, mod)
	}
}

func (h *hotkeyCapture) highlight(on bool) {
	if h.bg == nil {
		return
	}
	if on {
		h.bg.StrokeWidth = 2
	} else {
		h.bg.StrokeWidth = 0
	}
	h.bg.Refresh()
}

// modNameForKey maps a Fyne modifier key to our modifier name.
func modNameForKey(n fyne.KeyName) (string, bool) {
	switch n {
	case desktop.KeyShiftLeft, desktop.KeyShiftRight:
		return "shift", true
	case desktop.KeyControlLeft, desktop.KeyControlRight:
		return "ctrl", true
	case desktop.KeyAltLeft, desktop.KeyAltRight:
		return "alt", true
	case desktop.KeySuperLeft, desktop.KeySuperRight:
		return "cmd", true
	}
	return "", false
}

// keyNameForKey maps a Fyne main key to our key name (A-Z, 0-9, SPACE).
func keyNameForKey(n fyne.KeyName) (string, bool) {
	if n == fyne.KeySpace {
		return "SPACE", true
	}
	s := string(n)
	if len(s) == 1 && ((s >= "A" && s <= "Z") || (s >= "0" && s <= "9")) {
		return s, true
	}
	return "", false
}

// modGlyph returns a compact symbol for a modifier name.
func modGlyph(name string) string {
	switch name {
	case "cmd":
		return "⌘"
	case "shift":
		return "⇧"
	case "ctrl":
		return "⌃"
	case "alt":
		return "⌥"
	}
	return capitalize(name)
}
