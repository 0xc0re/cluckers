//go:build gui && windows

package cli

import "syscall"

// detachSysProcAttr returns process attributes that create a detached process
// not attached to the parent's console.
func detachSysProcAttr() *syscall.SysProcAttr {
	const createNewProcessGroup = 0x00000200
	const detachedProcess = 0x00000008
	return &syscall.SysProcAttr{
		CreationFlags: createNewProcessGroup | detachedProcess,
	}
}
