// Three-column layout for the main window, matching the design mockup:
//
//	┌──────────────────────────────────────────────────────────┐
//	│  logo  Clipboard      [ search ]              ☀  ⚙        │  top bar
//	├────────────┬──────────────────────────┬──────────────────┤
//	│ categories │  Clipboard History       │  Preview         │
//	│  All  128  │  ┌────────────────────┐  │  ┌────────────┐  │
//	│  Text  45  │  │ ▪ title    · badge │  │  │ content    │  │
//	│  Images    │  │   source · time    │  │  └────────────┘  │
//	│  Links 23  │  └────────────────────┘  │  Type   ...      │
//	│ ────────── │           ...            │  [Copy][Pin][🗑] │
//	│ STARRED    │                          │                  │
//	├────────────┴──────────────────────────┴──────────────────┤
//	│ ● Local history · 128 items       ⌘V Paste  ⌘F Search     │  status bar
//	└──────────────────────────────────────────────────────────┘
package ui

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	sidebarWidth = 210
	previewWidth = 300
)

// buildContent assembles the whole window: top bar, three-column body and
// status bar. It is called on construction and on every Show so the list
// reflects the latest history.
func (w *Window) buildContent() {
	w.reload()

	top := w.buildTopBar()
	w.sidebarBox = container.NewVBox()
	w.listBox = container.NewVBox()
	w.previewBox = container.NewVBox()
	w.refreshSidebar()
	w.refreshList()
	w.refreshPreview()

	sidebar := fixedWidth(sidebarWidth, container.NewVScroll(w.sidebarBox))
	preview := fixedWidth(previewWidth, container.NewVScroll(w.previewBox))

	middleHeader := container.NewBorder(nil, nil,
		boldLabel("Clipboard History", 15), w.clearAllButton())
	middle := container.NewBorder(
		container.NewPadded(middleHeader), nil, nil, nil,
		container.NewVScroll(w.listBox))

	body := container.NewBorder(nil, nil,
		container.NewBorder(nil, nil, nil, widget.NewSeparator(), sidebar),
		container.NewBorder(nil, nil, widget.NewSeparator(), nil, preview),
		middle)

	root := container.NewBorder(top, w.buildStatusBar(), nil, nil,
		container.NewPadded(body))
	w.win.SetContent(root)
}

// reload pulls the full item set from the store (empty query returns all).
func (w *Window) reload() {
	if w.deps.OnSearch != nil {
		w.master = w.deps.OnSearch("")
	}
}

// isLink reports whether an item looks like a URL.
func isLink(it Item) bool {
	s := strings.TrimSpace(it.Content)
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// filtered returns the items matching the active category and query.
func (w *Window) filtered() []Item {
	out := make([]Item, 0, len(w.master))
	q := strings.ToLower(strings.TrimSpace(w.query))
	for _, it := range w.master {
		if !w.matchesCategory(it) {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(it.Preview+" "+it.Content), q) {
			continue
		}
		out = append(out, it)
	}
	return out
}

func (w *Window) matchesCategory(it Item) bool {
	switch w.category {
	case catStarred:
		return it.Pinned
	case catImages:
		return it.Type == "image"
	case catLinks:
		return isLink(it)
	case catText:
		return it.Type != "image" && !isLink(it)
	default: // catAll
		return true
	}
}

// counts tallies items per category for the sidebar badges.
func (w *Window) counts() map[string]int {
	c := map[string]int{}
	for _, it := range w.master {
		c[catAll]++
		switch {
		case it.Type == "image":
			c[catImages]++
		case isLink(it):
			c[catLinks]++
		default:
			c[catText]++
		}
		if it.Pinned {
			c[catStarred]++
		}
	}
	return c
}

// --- top bar ------------------------------------------------------------

func (w *Window) buildTopBar() fyne.CanvasObject {
	logo := roundedSwatch(accentPurple, 24)
	title := boldLabel("cut-tool", 16)
	left := container.NewHBox(logo, title)

	search := widget.NewEntry()
	search.SetPlaceHolder("搜索剪贴板历史 / Search clipboard history…")
	search.OnChanged = func(q string) {
		w.query = q
		w.refreshList()
	}
	searchBar := container.NewBorder(nil, nil, widget.NewIcon(theme.SearchIcon()), nil, search)

	themeBtn := widget.NewButtonWithIcon("", theme.ColorChromaticIcon(), w.toggleTheme)
	themeBtn.Importance = widget.LowImportance
	right := container.NewHBox(themeBtn)
	if w.deps.GetHotkey != nil || w.deps.GetRetentionDays != nil {
		settingsBtn := widget.NewButtonWithIcon("", theme.SettingsIcon(), w.ShowSettings)
		settingsBtn.Importance = widget.LowImportance
		right.Add(settingsBtn)
	}

	bar := container.NewBorder(nil, nil, left, right, container.NewPadded(searchBar))
	return container.NewVBox(container.NewPadded(bar), widget.NewSeparator())
}

// toggleTheme flips between light and dark variants at runtime. Canvas
// objects (cards, badges, labels) capture their colors at build time, so we
// rebuild the whole content after switching to re-read the new palette.
func (w *Window) toggleTheme() {
	if w.theme.variant == theme.VariantLight {
		w.theme.variant = theme.VariantDark
	} else {
		w.theme.variant = theme.VariantLight
	}
	w.app.Settings().SetTheme(w.theme) // refreshes standard widgets
	w.buildContent()                   // rebuilds canvas objects with new colors
}

// --- status bar ---------------------------------------------------------

func (w *Window) buildStatusBar() fyne.CanvasObject {
	dot := canvas.NewCircle(color.NRGBA{R: 0x2e, G: 0xc4, B: 0x6b, A: 0xff})
	dot.Resize(fyne.NewSize(8, 8))
	w.statusLbl = widget.NewLabel("")
	w.updateStatus()
	left := container.NewHBox(container.NewCenter(dot), w.statusLbl)

	hint := canvas.NewText("⌘V 粘贴   ⌘F 搜索   ★ 收藏", theme.PlaceHolderColor())
	hint.TextSize = 12

	bar := container.NewBorder(nil, nil, left, hint)
	return container.NewVBox(widget.NewSeparator(), container.NewPadded(bar))
}

func (w *Window) updateStatus() {
	if w.statusLbl == nil {
		return
	}
	w.statusLbl.SetText(fmt.Sprintf("本地历史 · %d 条 / Local history · %d items",
		len(w.master), len(w.master)))
}

// clearAllButton triggers OnClear then refreshes every column.
func (w *Window) clearAllButton() fyne.CanvasObject {
	btn := widget.NewButton("Clear All", func() {
		if w.deps.OnClear != nil {
			w.deps.OnClear()
		}
		w.selected = nil
		w.reload()
		w.refreshSidebar()
		w.refreshList()
		w.refreshPreview()
	})
	btn.Importance = widget.LowImportance
	return btn
}

// --- sidebar ------------------------------------------------------------

func (w *Window) refreshSidebar() {
	if w.sidebarBox == nil {
		return
	}
	c := w.counts()
	w.sidebarBox.Objects = nil

	w.sidebarBox.Add(sectionLabel("CATEGORIES"))
	for _, cat := range []string{catAll, catText, catImages, catLinks} {
		w.sidebarBox.Add(w.categoryRow(cat, c[cat]))
	}
	w.sidebarBox.Add(widget.NewSeparator())
	w.sidebarBox.Add(sectionLabel("STARRED"))
	if c[catStarred] == 0 {
		empty := canvas.NewText("  暂无收藏 / No starred", theme.PlaceHolderColor())
		empty.TextSize = 12
		w.sidebarBox.Add(empty)
	} else {
		w.sidebarBox.Add(w.categoryRow(catStarred, c[catStarred]))
	}
	w.sidebarBox.Refresh()
}

// categoryRow is a tappable sidebar entry; the active one is highlighted.
func (w *Window) categoryRow(cat string, count int) fyne.CanvasObject {
	name := widget.NewLabel(cat)
	badge := canvas.NewText(fmt.Sprintf("%d", count), theme.PlaceHolderColor())
	badge.TextSize = 12
	row := container.NewBorder(nil, nil, name, badge)

	bg := color.Color(color.Transparent)
	if cat == w.category {
		bg = accentPurple
		name.TextStyle = fyne.TextStyle{Bold: true}
	}
	rect := canvas.NewRectangle(bg)
	rect.CornerRadius = 8

	return newTappable(container.NewStack(rect, container.NewPadded(row)), func() {
		w.category = cat
		w.refreshSidebar()
		w.refreshList()
	})
}

// --- history list -------------------------------------------------------

func (w *Window) refreshList() {
	if w.listBox == nil {
		return
	}
	items := w.filtered()
	w.listBox.Objects = nil
	if len(items) == 0 {
		empty := widget.NewLabelWithStyle("暂无记录 / No items",
			fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
		w.listBox.Add(empty)
	}
	for i := range items {
		w.listBox.Add(w.listRow(items[i]))
	}
	w.listBox.Refresh()
	w.updateStatus()
}

// listRow builds one clickable history card with a hover highlight that
// follows the mouse.
func (w *Window) listRow(it Item) fyne.CanvasObject {
	swatch := roundedSwatch(colorForType(it), 34)

	title := widget.NewLabel(it.Preview)
	title.Truncation = fyne.TextTruncateEllipsis
	sub := canvas.NewText(it.Timestamp, theme.PlaceHolderColor())
	sub.TextSize = 11
	text := container.NewVBox(title, sub)

	inner := container.NewBorder(nil, nil, swatch, typeBadge(it), text)

	// selected rows keep the selection color; others start on the input bg
	// and brighten to the hover color while the pointer is over them.
	base := theme.InputBackgroundColor()
	if w.selected != nil && w.selected.Hash == it.Hash {
		base = theme.SelectionColor()
	}
	card := canvas.NewRectangle(base)
	card.CornerRadius = 10
	body := container.NewStack(card, container.NewPadded(inner))

	item := it
	return newHoverTappable(body,
		func() {
			w.selected = &item
			w.refreshList()    // repaint rows so selection highlight moves
			w.refreshPreview() // update the preview panel
		},
		func(in bool) {
			// Don't override the selected row's highlight.
			if w.selected != nil && w.selected.Hash == item.Hash {
				return
			}
			if in {
				card.FillColor = theme.HoverColor()
			} else {
				card.FillColor = theme.InputBackgroundColor()
			}
			card.Refresh()
		})
}

// --- preview panel ------------------------------------------------------

func (w *Window) refreshPreview() {
	if w.previewBox == nil {
		return
	}
	w.previewBox.Objects = nil

	header := container.NewBorder(nil, nil, boldLabel("Preview", 15),
		widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
			w.selected = nil
			w.refreshPreview()
		}))
	w.previewBox.Add(header)
	w.previewBox.Add(widget.NewSeparator())

	if w.selected == nil {
		hint := widget.NewLabelWithStyle("选择一条记录查看详情\nSelect an item to preview",
			fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
		w.previewBox.Add(container.NewPadded(hint))
		w.previewBox.Refresh()
		return
	}

	it := *w.selected

	// Content box.
	if it.Type == "image" && len(it.Data) > 0 {
		img := canvas.NewImageFromResource(fyne.NewStaticResource(it.Hash+".png", it.Data))
		img.FillMode = canvas.ImageFillContain
		img.SetMinSize(fyne.NewSize(0, 160))
		w.previewBox.Add(img)
	} else {
		content := widget.NewLabel(it.Content)
		content.Wrapping = fyne.TextWrapWord
		box := canvas.NewRectangle(theme.InputBackgroundColor())
		box.CornerRadius = 8
		w.previewBox.Add(container.NewStack(box, container.NewPadded(content)))
	}

	// Metadata rows.
	w.previewBox.Add(metaRow("Type", typeLabel(it)))
	w.previewBox.Add(metaRow("Copied", it.Timestamp))

	// Actions: Copy / Pin / Delete.
	copyBtn := widget.NewButtonWithIcon("Copy", theme.ContentCopyIcon(), func() {
		if w.deps.OnSelect != nil {
			w.deps.OnSelect(it)
		}
		w.win.Hide()
	})
	copyBtn.Importance = widget.HighImportance

	pinText := "Pin"
	if it.Pinned {
		pinText = "Unpin"
	}
	pinBtn := widget.NewButtonWithIcon(pinText, theme.ConfirmIcon(), func() {
		if w.deps.OnPin != nil {
			w.deps.OnPin(it.Hash, !it.Pinned)
		}
		w.reload()
		w.refreshSidebar()
		w.refreshList()
		if w.selected != nil {
			w.selected.Pinned = !w.selected.Pinned
		}
		w.refreshPreview()
	})

	delBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		if w.deps.OnDelete != nil {
			w.deps.OnDelete(it.Hash)
		}
		w.selected = nil
		w.reload()
		w.refreshSidebar()
		w.refreshList()
		w.refreshPreview()
	})

	actions := container.NewGridWithColumns(3, copyBtn, pinBtn, delBtn)
	w.previewBox.Add(container.NewPadded(actions))
	w.previewBox.Refresh()
}

// --- small helpers ------------------------------------------------------

// fixedWidth wraps obj so its column keeps a constant width.
func fixedWidth(wdt float32, obj fyne.CanvasObject) fyne.CanvasObject {
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(wdt, 0))
	return container.NewStack(spacer, obj)
}

func boldLabel(text string, size float32) fyne.CanvasObject {
	t := canvas.NewText(text, theme.ForegroundColor())
	t.TextStyle = fyne.TextStyle{Bold: true}
	t.TextSize = size
	return t
}

func sectionLabel(text string) fyne.CanvasObject {
	t := canvas.NewText(text, theme.PlaceHolderColor())
	t.TextSize = 11
	t.TextStyle = fyne.TextStyle{Bold: true}
	return container.NewPadded(t)
}

func metaRow(key, val string) fyne.CanvasObject {
	k := canvas.NewText(key, theme.PlaceHolderColor())
	k.TextSize = 12
	v := canvas.NewText(val, theme.ForegroundColor())
	v.TextSize = 12
	v.TextStyle = fyne.TextStyle{Bold: true}
	return container.NewPadded(container.NewBorder(nil, nil, k, v))
}

// roundedSwatch is a small colored rounded square used as a type icon/logo.
func roundedSwatch(c color.Color, size float32) fyne.CanvasObject {
	r := canvas.NewRectangle(c)
	r.CornerRadius = 7
	r.SetMinSize(fyne.NewSize(size, size))
	return container.NewGridWrap(fyne.NewSize(size, size), r)
}

// colorForType maps an entry type to its swatch color (matches the mockup).
func colorForType(it Item) color.Color {
	switch {
	case it.Type == "image":
		return color.NRGBA{R: 0x22, G: 0xb5, B: 0x73, A: 0xff} // green
	case isLink(it):
		return accentPurple
	default:
		return color.NRGBA{R: 0x5b, G: 0x8c, B: 0xff, A: 0xff} // blue
	}
}

func typeLabel(it Item) string {
	switch {
	case it.Type == "image":
		return "Image"
	case isLink(it):
		return "URL / Link"
	default:
		return "Text"
	}
}

// typeBadge is the small pill on the right of a list row.
func typeBadge(it Item) fyne.CanvasObject {
	t := canvas.NewText(typeLabel(it), theme.PlaceHolderColor())
	t.TextSize = 11
	bg := canvas.NewRectangle(theme.ButtonColor())
	bg.CornerRadius = 6
	return container.NewGridWrap(fyne.NewSize(74, 24),
		container.NewStack(bg, container.NewCenter(t)))
}
