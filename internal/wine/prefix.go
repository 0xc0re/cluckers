package wine

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xc0re/cluckers/internal/config"
	"github.com/0xc0re/cluckers/internal/ui"
)

// PrefixPath returns the default Wine prefix path under the data directory.
func PrefixPath() string {
	return filepath.Join(config.DataDir(), "prefix")
}

// CreatePrefix creates a Wine prefix at prefixPath using the appropriate strategy:
//   - Proton-GE: copy default_pfx template + wineboot
//   - System Wine: wineboot + winetricks
//
// IMPORTANT: Do NOT run winetricks on Proton-GE prefixes -- it breaks
// Proton-GE's DLL override chain.
func CreatePrefix(prefixPath string, winePath string, verbose bool) error {
	if IsProtonGE(winePath) {
		protonDir := ProtonBaseDir(winePath)
		return createFromProtonTemplate(prefixPath, protonDir, winePath, verbose)
	}
	return createWithWinetricks(prefixPath, winePath, verbose)
}

// createFromProtonTemplate creates a prefix by copying Proton-GE's default_pfx template,
// then running wineboot --init to finalize.
func createFromProtonTemplate(prefixPath, protonDir, winePath string, verbose bool) error {
	// Check both possible template locations:
	//   - ProtonUp-Qt versioned installs: <protonDir>/default_pfx
	//   - System packages (e.g. AUR proton-ge-custom-bin): <protonDir>/files/share/default_pfx
	templateDir := filepath.Join(protonDir, "default_pfx")
	if _, err := os.Stat(templateDir); err != nil {
		alt := filepath.Join(protonDir, "files", "share", "default_pfx")
		if _, err2 := os.Stat(alt); err2 != nil {
			return &ui.UserError{
				Message:    "Proton-GE default prefix template not found",
				Detail:     fmt.Sprintf("Checked:\n  %s\n  %s", templateDir, alt),
				Suggestion: "Your Proton-GE installation may be incomplete. Try reinstalling via ProtonUp-Qt.",
			}
		}
		templateDir = alt
	}

	ui.Info("Creating Wine prefix from Proton-GE template...")
	ui.Verbose(fmt.Sprintf("Template: %s", templateDir), verbose)
	ui.Verbose(fmt.Sprintf("Destination: %s", prefixPath), verbose)

	// Recursively copy the template directory.
	if err := copyProtonTemplate(templateDir, prefixPath, verbose); err != nil {
		// Clean up partial prefix on failure.
		os.RemoveAll(prefixPath)
		return fmt.Errorf("copy Proton-GE template: %w", err)
	}

	// Run wineboot --init to finalize the prefix.
	ui.Verbose("Running wineboot --init to finalize prefix...", verbose)
	if err := runWineboot(prefixPath, winePath); err != nil {
		return fmt.Errorf("wineboot init after template copy: %w", err)
	}

	ui.Info("Wine prefix created successfully.")
	return nil
}

// createWithWinetricks creates a prefix using system Wine + winetricks.
func createWithWinetricks(prefixPath, winePath string, verbose bool) error {
	// Ensure winetricks is available.
	if _, err := exec.LookPath("winetricks"); err != nil {
		distro := DetectDistro()
		var suggestion string
		switch distro {
		case "arch", "steamos":
			suggestion = "Install winetricks: sudo pacman -S winetricks\n  Or install Proton-GE via ProtonUp-Qt (no winetricks needed)."
		case "ubuntu", "debian", "linuxmint", "pop":
			suggestion = "Install winetricks: sudo apt install winetricks\n  Or install Proton-GE via ProtonUp-Qt (no winetricks needed)."
		case "fedora":
			suggestion = "Install winetricks: sudo dnf install winetricks\n  Or install Proton-GE via ProtonUp-Qt (no winetricks needed)."
		default:
			suggestion = "Install winetricks for your distribution, or install Proton-GE via ProtonUp-Qt (no winetricks needed)."
		}
		return &ui.UserError{
			Message:    "winetricks not found. It is required to set up a Wine prefix with system Wine.",
			Suggestion: suggestion,
		}
	}

	ui.Info("Creating Wine prefix with system Wine...")
	ui.Verbose(fmt.Sprintf("Wine: %s", winePath), verbose)
	ui.Verbose(fmt.Sprintf("Prefix: %s", prefixPath), verbose)

	// Step 1: Initialize prefix with wineboot.
	ui.Verbose("Running wineboot --init...", verbose)
	if err := runWineboot(prefixPath, winePath); err != nil {
		return fmt.Errorf("wineboot init: %w", err)
	}

	// Step 2: Install dependencies via winetricks.
	ui.Info("Installing dependencies via winetricks (this may take several minutes)...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "winetricks", "-q", "vcrun2022", "d3dx11_43", "dxvk")
	cmd.Env = wineEnv(prefixPath, winePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return &ui.UserError{
			Message:    "winetricks failed to install required dependencies",
			Detail:     err.Error(),
			Suggestion: "Try running manually: WINEPREFIX=" + prefixPath + " winetricks -q vcrun2022 d3dx11_43 dxvk",
		}
	}

	ui.Info("Wine prefix created successfully.")
	return nil
}

// runWineboot runs wineboot --init with appropriate environment variables.
// Uses a 2-minute timeout and suppresses GUI (DISPLAY="", mscoree/mshtml disabled).
func runWineboot(prefixPath, winePath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, winePath, "wineboot", "--init")
	cmd.Env = wineEnv(prefixPath, winePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wineboot --init failed: %w", err)
	}
	return nil
}

// wineEnv builds the environment for Wine/winetricks commands.
// Suppresses GUI by clearing DISPLAY and disabling mono/gecko via WINEDLLOVERRIDES.
func wineEnv(prefixPath, winePath string) []string {
	env := os.Environ()
	// Filter out existing DISPLAY, WINEPREFIX, WINE, WINEDLLOVERRIDES.
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		key := strings.SplitN(e, "=", 2)[0]
		switch key {
		case "DISPLAY", "WINEPREFIX", "WINE", "WINEDLLOVERRIDES":
			continue
		default:
			filtered = append(filtered, e)
		}
	}
	return append(filtered,
		"WINEPREFIX="+prefixPath,
		"WINE="+winePath,
		"DISPLAY=",
		"WINEDLLOVERRIDES=mscoree,mshtml=",
	)
}

// copyProtonTemplate recursively copies the Proton-GE default_pfx template to dst.
// Handles directories, regular files, and symlinks (with Wine lib path fixup).
func copyProtonTemplate(srcDir, dstDir string, verbose bool) error {
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(dstDir, rel)

		// Use os.Lstat to detect symlinks (NOT d.Info() which follows them via DirEntry).
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}

		switch {
		case info.Mode()&os.ModeSymlink != 0:
			// Symlink: read target and handle Wine lib path fixup.
			target, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("readlink %s: %w", path, err)
			}

			if isWineLibPath(target) {
				// Resolve relative symlink to absolute path relative to source location.
				absTarget := filepath.Join(filepath.Dir(path), target)
				absTarget = filepath.Clean(absTarget)
				if verbose {
					ui.Verbose(fmt.Sprintf("Symlink fixup: %s -> %s", rel, absTarget), true)
				}
				return os.Symlink(absTarget, dst)
			}
			// Copy symlink as-is (target is not a Wine lib path).
			return os.Symlink(target, dst)

		case info.IsDir():
			return os.MkdirAll(dst, 0755)

		default:
			// Regular file: copy content.
			return copyFile(path, dst)
		}
	})
}

// isWineLibPath checks if a symlink target path contains Wine lib directory patterns.
// These relative symlinks in Proton-GE's default_pfx need to be resolved to absolute
// paths when copying to a different location.
func isWineLibPath(target string) bool {
	wineLibDirs := []string{
		"/lib/wine/i386-unix",
		"/lib/wine/i386-windows",
		"/lib/wine/x86_64-unix",
		"/lib/wine/x86_64-windows",
	}
	for _, suffix := range wineLibDirs {
		if strings.Contains(target, suffix) {
			return true
		}
	}
	return false
}

// copyFile copies a regular file from src to dst, preserving permissions.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
