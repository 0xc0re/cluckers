# Deferred Items - Phase 04

## Pre-existing Issues (Out of Scope)

1. **`go vet` warning in internal/gui/widgets/step_list.go:59** - `StepListWidget` does not implement `fyne.Widget` (missing `CreateRenderer` method). This is a pre-existing issue from plan 03 (widgets package), not introduced by plan 04 changes. Needs to be addressed separately.
