//go:build linux

package inputproxy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// winebusRegContent is the Windows registry file content that configures Wine's
// winebus service to disable hidraw access (preventing Wine from seeing physical
// controllers) and enable SDL (so Wine sees our virtual uinput gamepad via SDL).
//
// Uses \r\n line endings per Windows registry format specification.
var winebusRegContent = "Windows Registry Editor Version 5.00\r\n" +
	"\r\n" +
	"[HKEY_LOCAL_MACHINE\\System\\CurrentControlSet\\Services\\winebus]\r\n" +
	"\"DisableHidraw\"=dword:00000001\r\n" +
	"\"Enable SDL\"=dword:00000001\r\n"

// WriteWinebusRegFile creates a temporary .reg file with winebus configuration.
// Returns the file path, a cleanup function that removes the temp file, and any error.
func WriteWinebusRegFile() (string, func(), error) {
	tmpFile, err := os.CreateTemp("", "winebus_*.reg")
	if err != nil {
		return "", func() {}, fmt.Errorf("creating temp .reg file: %w", err)
	}

	path := tmpFile.Name()
	cleanup := func() { os.Remove(path) }

	if _, err := tmpFile.WriteString(winebusRegContent); err != nil {
		tmpFile.Close()
		cleanup()
		return "", func() {}, fmt.Errorf("writing .reg content: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("closing .reg file: %w", err)
	}

	return path, cleanup, nil
}

// ApplyWinebusRegistry imports the winebus .reg file into the Proton prefix
// using `python3 <protonScript> run regedit /S Z:\<regFile>`.
func ApplyWinebusRegistry(compatdataPath, protonScript string, env []string) error {
	regPath, cleanup, err := WriteWinebusRegFile()
	if err != nil {
		return fmt.Errorf("writing winebus reg file: %w", err)
	}
	defer cleanup()

	// Convert Linux path to Wine path (Z:\path\to\file)
	winePath := "Z:" + strings.ReplaceAll(regPath, "/", "\\")

	cmd := exec.Command("python3", protonScript, "run", "regedit", "/S", winePath)

	// Merge provided environment with STEAM_COMPAT_DATA_PATH
	cmdEnv := make([]string, len(env))
	copy(cmdEnv, env)
	cmdEnv = append(cmdEnv, "STEAM_COMPAT_DATA_PATH="+compatdataPath)
	cmd.Env = cmdEnv

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("regedit import failed: %w\noutput: %s", err, string(output))
	}

	return nil
}

// NeedsWinebusPatching checks whether the winebus registry entries already exist
// in the Proton prefix. Returns true if patching is needed (key absent or file missing).
// This enables idempotent patching -- skip if already done.
func NeedsWinebusPatching(compatdataPath string) bool {
	systemReg := filepath.Join(compatdataPath, "pfx", "system.reg")
	data, err := os.ReadFile(systemReg)
	if err != nil {
		return true // File doesn't exist or can't be read -- needs patching
	}

	// Check for the DisableHidraw key in the registry file
	return !strings.Contains(string(data), "DisableHidraw")
}
