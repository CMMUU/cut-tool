package ui

import (
	"errors"
	"image/color"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var (
	errPositiveInt = errors.New("请输入正整数 / must be a positive integer")
	errNeedCombo   = errors.New("快捷键需至少一个修饰键和一个主键 / need a modifier and a key")
)

// ShowSettings swaps the window content for an in-app settings page styled
// like the main window (top bar + card body + action row). A back arrow
// returns to the history view without saving; Save persists and returns.
func (w *Window) ShowSettings() {
	if w.deps.GetRetentionDays == nil || w.deps.SetRetentionDays == nil {
		return
	}
	w.win.SetContent(w.buildSettingsPage())
	w.win.Show()
	w.win.RequestFocus()
}

func (w *Window) buildSettingsPage() fyne.CanvasObject {
	top := w.settingsTopBar()

	// --- Retention field ---
	daysEntry := widget.NewEntry()
	daysEntry.SetText(strconv.Itoa(w.deps.GetRetentionDays()))
	daysEntry.Validator = func(s string) error {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil || n <= 0 {
			return errPositiveInt
		}
		return nil
	}
	retentionCard := w.settingsCard("历史保留 / Retention",
		"未收藏的记录超过该天数后自动清除；收藏（★）的永久保留。",
		container.NewBorder(nil, nil,
			widget.NewLabel("保留天数 / Days"), nil, daysEntry))

	// --- Hotkey capture (optional) ---
	var capture *hotkeyCapture
	body := container.NewVBox(retentionCard)
	hotkeyEnabled := len(w.deps.AvailableKeys) > 0 && w.deps.GetHotkey != nil && w.deps.SetHotkey != nil
	if hotkeyEnabled {
		curMods, curKey := w.deps.GetHotkey()
		capture = newHotkeyCapture(w.win.Canvas(), curMods, curKey, nil)
		hkCard := w.settingsCard("全局快捷键 / Global Hotkey",
			"点击下方方框，然后按下你想要的组合键（至少一个修饰键 + 一个主键）。",
			fixedHeight(44, capture))
		body.Add(hkCard)
	}

	// --- Action row: Restore Defaults (left) | Cancel Save (right) ---
	restoreBtn := widget.NewButtonWithIcon("恢复默认 / Restore Defaults",
		theme.ViewRefreshIcon(), func() {
			daysEntry.SetText(strconv.Itoa(w.defaultRetention()))
			if capture != nil {
				mods, key := w.defaultHotkey()
				capture.SetValue(mods, key)
			}
		})
	cancelBtn := widget.NewButton("取消 / Cancel", func() { w.buildContent() })
	saveBtn := widget.NewButtonWithIcon("保存 / Save", theme.ConfirmIcon(), func() {
		w.saveSettings(daysEntry, capture)
	})
	saveBtn.Importance = widget.HighImportance

	actions := container.NewBorder(nil, nil, restoreBtn, container.NewHBox(cancelBtn, saveBtn))

	content := container.NewVBox(body, widget.NewSeparator(), container.NewPadded(actions))
	scroll := container.NewVScroll(container.NewPadded(content))
	return container.NewBorder(top, nil, nil, nil, scroll)
}

// settingsTopBar mirrors the main top bar: back arrow + "Settings" title.
func (w *Window) settingsTopBar() fyne.CanvasObject {
	back := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() { w.buildContent() })
	back.Importance = widget.LowImportance
	title := boldLabel("设置 / Settings", 16)
	left := container.NewHBox(back, title)
	bar := container.NewBorder(nil, nil, left, nil)
	return container.NewVBox(container.NewPadded(bar), widget.NewSeparator())
}

// settingsCard wraps a titled section in the rounded card style used across
// the app (matching list rows / preview boxes).
func (w *Window) settingsCard(title, hint string, control fyne.CanvasObject) fyne.CanvasObject {
	head := boldLabel(title, 14)
	desc := canvas.NewText(hint, theme.PlaceHolderColor())
	desc.TextSize = 12
	inner := container.NewVBox(head, desc, control)

	card := canvas.NewRectangle(theme.InputBackgroundColor())
	card.CornerRadius = 10
	return container.NewPadded(container.NewStack(card, container.NewPadded(inner)))
}

// saveSettings validates and persists both fields, then returns to the list.
func (w *Window) saveSettings(daysEntry *widget.Entry, capture *hotkeyCapture) {
	if n, err := strconv.Atoi(strings.TrimSpace(daysEntry.Text)); err == nil && n > 0 {
		if serr := w.deps.SetRetentionDays(n); serr != nil {
			dialog.ShowError(serr, w.win)
			return
		}
	} else {
		dialog.ShowError(errPositiveInt, w.win)
		return
	}
	if capture != nil {
		mods, key := capture.Value()
		if len(mods) == 0 || key == "" {
			dialog.ShowError(errNeedCombo, w.win)
			return
		}
		if serr := w.deps.SetHotkey(mods, key); serr != nil {
			dialog.ShowError(serr, w.win)
			return
		}
	}
	w.buildContent() // back to the history view
}

func (w *Window) defaultRetention() int {
	if w.deps.DefaultRetentionDays > 0 {
		return w.deps.DefaultRetentionDays
	}
	return 15
}

func (w *Window) defaultHotkey() (mods []string, key string) {
	mods = w.deps.DefaultHotkeyMods
	key = w.deps.DefaultHotkeyKey
	return mods, key
}

// fixedHeight wraps obj so it keeps a constant height (used for the capture box).
func fixedHeight(h float32, obj fyne.CanvasObject) fyne.CanvasObject {
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(0, h))
	return container.NewStack(spacer, obj)
}
