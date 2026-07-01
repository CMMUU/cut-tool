package ui

import (
	_ "embed"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// notoSansSC is a GB2312-subset of Noto Sans CJK SC (OFL license), embedded
// so Chinese text renders correctly on every platform without relying on
// system fonts.
//
//go:embed fonts/NotoSansSC-subset.ttf
var notoSansSC []byte

var chineseFont = &fyne.StaticResource{
	StaticName:    "NotoSansSC-subset.ttf",
	StaticContent: notoSansSC,
}

// accentPurple is the brand/primary color from the design mockup.
var accentPurple = color.NRGBA{R: 0x5b, G: 0x4d, B: 0xe6, A: 0xff}

// appTheme implements a light/dark theme with a purple accent, matching the
// design mockup. The active variant is switchable at runtime via SetVariant.
type appTheme struct {
	variant fyne.ThemeVariant
}

var _ fyne.Theme = (*appTheme)(nil)

func newAppTheme() *appTheme { return &appTheme{variant: theme.VariantLight} }

// toggle switches between the light and dark variants.
func (t *appTheme) toggle() {
	if t.variant == theme.VariantDark {
		t.variant = theme.VariantLight
	} else {
		t.variant = theme.VariantDark
	}
}

func (t *appTheme) Font(fyne.TextStyle) fyne.Resource { return chineseFont }

func (t *appTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}

func (t *appTheme) Color(n fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	if t.variant == theme.VariantDark {
		return t.dark(n)
	}
	return t.light(n)
}

func (t *appTheme) light(n fyne.ThemeColorName) color.Color {
	switch n {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 0x1a, G: 0x1a, B: 0x1f, A: 0xff}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 0xf4, G: 0xf4, B: 0xf6, A: 0xff}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 0x9a, G: 0x9a, B: 0xa2, A: 0xff}
	case theme.ColorNamePrimary:
		return accentPurple
	case theme.ColorNameSelection:
		return color.NRGBA{R: 0xe8, G: 0xe6, B: 0xfb, A: 0xff}
	case theme.ColorNameHover:
		return color.NRGBA{R: 0xf0, G: 0xef, B: 0xfc, A: 0xff}
	case theme.ColorNameButton:
		return color.NRGBA{R: 0xf4, G: 0xf4, B: 0xf6, A: 0xff}
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 0xe6, G: 0xe6, B: 0xea, A: 0xff}
	}
	return theme.DefaultTheme().Color(n, theme.VariantLight)
}

func (t *appTheme) dark(n fyne.ThemeColorName) color.Color {
	switch n {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 0x1c, G: 0x1c, B: 0x20, A: 0xff}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 0xea, G: 0xea, B: 0xec, A: 0xff}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 0x2a, G: 0x2a, B: 0x30, A: 0xff}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 0x7a, G: 0x7a, B: 0x82, A: 0xff}
	case theme.ColorNamePrimary:
		return accentPurple
	case theme.ColorNameSelection:
		return color.NRGBA{R: 0x35, G: 0x30, B: 0x5a, A: 0xff}
	case theme.ColorNameHover:
		return color.NRGBA{R: 0x2e, G: 0x2b, B: 0x3e, A: 0xff}
	case theme.ColorNameButton:
		return color.NRGBA{R: 0x2a, G: 0x2a, B: 0x30, A: 0xff}
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 0x33, G: 0x33, B: 0x3a, A: 0xff}
	}
	return theme.DefaultTheme().Color(n, theme.VariantDark)
}

func (t *appTheme) Size(n fyne.ThemeSizeName) float32 {
	switch n {
	case theme.SizeNamePadding:
		return 6
	case theme.SizeNameInnerPadding:
		return 8
	case theme.SizeNameText:
		return 13
	case theme.SizeNameInputRadius:
		return 10
	case theme.SizeNameSelectionRadius:
		return 10
	}
	return theme.DefaultTheme().Size(n)
}
