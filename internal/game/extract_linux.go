//go:build linux

package game

// prepareTarget is a no-op on Linux. Unix file permissions allow the owner
// to overwrite files regardless of the read-only bit when using O_TRUNC.
func prepareTarget(_ string) {}
