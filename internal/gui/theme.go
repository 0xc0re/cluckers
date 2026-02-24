//go:build gui

package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// cluckersTheme implements fyne.Theme with a custom dark color scheme.
type cluckersTheme struct{}

// NewCluckersTheme returns a custom dark theme for the Cluckers launcher.
func NewCluckersTheme() fyne.Theme {
	return &cluckersTheme{}
}

func (t *cluckersTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 30, G: 30, B: 35, A: 255} // #1E1E23
	case theme.ColorNameButton:
		return color.NRGBA{R: 60, G: 60, B: 70, A: 255} // #3C3C46
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 76, G: 175, B: 80, A: 255} // #4CAF50 Material green
	case theme.ColorNameOverlayBackground:
		return color.NRGBA{R: 40, G: 40, B: 48, A: 255} // Slightly lighter than background
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 45, G: 45, B: 52, A: 255} // Subtle contrast for inputs
	default:
		return theme.DefaultTheme().Color(name, theme.VariantDark)
	}
}

func (t *cluckersTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *cluckersTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *cluckersTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
