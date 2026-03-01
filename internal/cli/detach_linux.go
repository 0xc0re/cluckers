//go:build gui && linux

package cli

import "syscall"

// detachSysProcAttr returns process attributes that create a new session,
// detaching the child process from the terminal.
func detachSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
