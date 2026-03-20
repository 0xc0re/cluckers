//go:build gui

package screens

import (
	"errors"

	"github.com/0xc0re/cluckers/internal/ui"
)

// formatGUIError extracts a user-friendly error message. If the error is a
// UserError with a Suggestion, the suggestion is appended so the user sees
// actionable guidance in the GUI (canvas.Text is single-line, so we join
// with " — ").
func formatGUIError(err error) string {
	var ue *ui.UserError
	if errors.As(err, &ue) && ue.Suggestion != "" {
		return ue.Message + " — " + ue.Suggestion
	}
	return err.Error()
}
