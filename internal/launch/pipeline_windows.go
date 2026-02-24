//go:build windows

package launch

// platformSteps returns Windows-specific pipeline steps.
// On Windows, no Wine detection, prefix creation, or prefix verification is needed.
func platformSteps(_ *LaunchState) []Step {
	return []Step{}
}

// platformPostSteps returns Windows-specific post-download steps.
// On Windows, no Steam Deck configuration is needed.
func platformPostSteps(_ *LaunchState) []Step {
	return []Step{}
}
