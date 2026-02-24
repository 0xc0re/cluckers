//go:build gui

package assets

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed cluckers_logo.png
var LogoData []byte

// LogoResource returns the embedded logo as a Fyne static resource.
func LogoResource() fyne.Resource {
	return fyne.NewStaticResource("cluckers_logo.png", LogoData)
}
