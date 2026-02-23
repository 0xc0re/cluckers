package launch

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/0xc0re/cluckers/internal/wine"
)

// deckPatches defines the INI key replacements applied for Steam Deck.
var deckPatches = []struct {
	old string
	new string
}{
	{"Fullscreen=false", "Fullscreen=True"},
	{"FullscreenWindowed=false", "FullscreenWindowed=True"},
	{"ResX=1920", "ResX=1280"},
	{"ResY=1080", "ResY=800"},
}

// PatchDeckConfig detects Steam Deck and patches RealmSystemSettings.ini
// for fullscreen 1280x800. Only patches the first occurrence of each setting
// (the first [SystemSettings] block). Idempotent — skips if already patched.
func PatchDeckConfig(gameDir string) error {
	if !isSteamDeck() {
		return nil
	}

	if err := patchDeckDisplay(gameDir); err != nil {
		return err
	}

	if err := PatchDeckInputConfig(gameDir); err != nil {
		return err
	}

	return nil
}

// patchDeckDisplay patches RealmSystemSettings.ini for fullscreen 1280x800.
func patchDeckDisplay(gameDir string) error {
	iniPath := filepath.Join(gameDir, "Realm-Royale", "RealmGame", "Config", "RealmSystemSettings.ini")

	data, err := os.ReadFile(iniPath)
	if err != nil {
		if os.IsNotExist(err) {
			ui.Verbose("RealmSystemSettings.ini not found, skipping Deck config", true)
			return nil
		}
		return fmt.Errorf("reading RealmSystemSettings.ini: %w", err)
	}

	original := string(data)

	// Check if already patched — all new values present means nothing to do.
	alreadyPatched := true
	for _, p := range deckPatches {
		if !strings.Contains(original, p.new) {
			alreadyPatched = false
			break
		}
	}
	if alreadyPatched {
		ui.Verbose("Steam Deck display config already applied, skipping", true)
		return nil
	}

	output := patchINILines(original, deckPatches)

	if err := os.WriteFile(iniPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("writing RealmSystemSettings.ini: %w", err)
	}

	return nil
}

// deckInputPatches neutralize the UE3 mouse activity counters ("Count bXAxis",
// "Count bYAxis") that trigger input mode auto-detection. These counters cause
// the game to switch from gamepad to keyboard/mouse mode whenever mouse movement
// is detected. On Steam Deck, Wine cursor events and touch screen input generate
// phantom mouse movement, which constantly triggers this switch and disables the
// controller in-match.
//
// The fix removes "Count bXAxis |" and "Count bYAxis |" from mouse bindings
// in [TgGame.TgPlayerInput] and [Engine.PlayerInput]. This preserves mouse
// camera control for KB/M users while preventing the input mode auto-switch.
var deckInputPatches = []struct {
	old string
	new string
}{
	{
		`Bindings=(Name="MouseX",Command="Count bXAxis | Axis aMouseX")`,
		`Bindings=(Name="MouseX",Command="Axis aMouseX")`,
	},
	{
		`Bindings=(Name="MouseY",Command="Count bYAxis | Axis aMouseY")`,
		`Bindings=(Name="MouseY",Command="Axis aMouseY")`,
	},
}

// deckInputTargetSections lists the INI sections where mouse counter patches
// should be applied. Both sections can trigger input mode detection:
//   - [TgGame.TgPlayerInput]: Game-specific input handler (in-match gameplay)
//   - [Engine.PlayerInput]: Base engine input handler (always active)
//
// [TgGame.TgSpectatorInput] is intentionally excluded since it does not affect
// in-match gameplay input and mouse control is needed for spectator camera.
var deckInputTargetSections = []string{
	"TgGame.TgPlayerInput",
	"Engine.PlayerInput",
}

// PatchDeckInputConfig patches DefaultInput.ini and RealmInput.ini to prevent
// input mode auto-switching from gamepad to keyboard/mouse on Steam Deck.
//
// Root cause: UE3's TgPlayerInput has mouse bindings with "Count bXAxis" and
// "Count bYAxis" that track mouse activity. When this counter increments, the
// game's input mode detection switches from gamepad to KB/M mode. On Steam
// Deck under Wine, phantom mouse events from the touch screen and Wine cursor
// warping constantly trigger this counter, causing the controller to stop
// working in-match even though it was detected at startup.
//
// Fix: Remove the Count commands from mouse bindings in [TgGame.TgPlayerInput].
// Also make INI files writable (0644) so the game can persist user controller
// preferences across sessions.
//
// Idempotent: skips files that are already patched or missing.
func PatchDeckInputConfig(gameDir string) error {
	configDir := filepath.Join(gameDir, "Realm-Royale", "RealmGame", "Config")

	// Patch both DefaultInput.ini and RealmInput.ini.
	// DefaultInput.ini has +Bindings= (UE3 append syntax).
	// RealmInput.ini has Bindings= (coalesced result, no prefix).
	inputFiles := []string{"DefaultInput.ini", "RealmInput.ini"}

	for _, filename := range inputFiles {
		iniPath := filepath.Join(configDir, filename)

		data, err := os.ReadFile(iniPath)
		if err != nil {
			if os.IsNotExist(err) {
				ui.Verbose(filename+" not found, skipping input patch", true)
				continue
			}
			return fmt.Errorf("reading %s: %w", filename, err)
		}

		original := string(data)

		// Build the full patch set including +Bindings= variants for DefaultInput.ini.
		patches := make([]struct{ old, new string }, 0, len(deckInputPatches)*2)
		for _, p := range deckInputPatches {
			patches = append(patches, p)
			// Also match the UE3 append syntax (+Bindings=) used in Default*.ini files.
			if strings.HasPrefix(p.old, "Bindings=") {
				patches = append(patches, struct{ old, new string }{
					old: "+" + p.old,
					new: "+" + p.new,
				})
			}
		}

		// Check if already patched.
		alreadyPatched := true
		for _, p := range patches {
			if strings.Contains(original, p.old) {
				alreadyPatched = false
				break
			}
		}
		if alreadyPatched {
			ui.Verbose(filename+" input patches already applied, skipping", true)
			// Still ensure the file is writable.
			ensureWritable(iniPath)
			continue
		}

		// Apply patches within target input sections only.
		output := multiSectionINIPatch(original, deckInputTargetSections, patches)

		if err := os.WriteFile(iniPath, []byte(output), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", filename, err)
		}

		ui.Verbose("Patched "+filename+" for Steam Deck controller input", true)
	}

	// Make all INI files writable so the game can save user controller preferences.
	// The game zip extracts files as read-only (0444), which prevents the game from
	// persisting settings changes (like switching to gamepad mode in Options).
	makeINIsWritable(configDir)

	return nil
}

// multiSectionINIPatch applies patches within any of the specified INI sections.
// Each patch is applied once per target section (first occurrence in that section).
// This prevents accidentally modifying the same key in unrelated sections.
func multiSectionINIPatch(content string, targetSections []string, patches []struct{ old, new string }) string {
	// Build set of target section headers for fast lookup.
	targets := make(map[string]bool, len(targetSections))
	for _, s := range targetSections {
		targets["["+s+"]"] = true
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	var result []string
	inTargetSection := false

	// Track which patches have been applied per section. Key is "section:patchIdx".
	applied := make(map[string]bool)
	currentSection := ""

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Track which INI section we're in.
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			inTargetSection = targets[trimmed]
			if inTargetSection {
				currentSection = trimmed
			}
		}

		// Only apply patches within target sections.
		if inTargetSection {
			for i, p := range patches {
				key := fmt.Sprintf("%s:%d", currentSection, i)
				if !applied[key] && trimmed == p.old {
					line = p.new
					applied[key] = true
					break
				}
			}
		}

		result = append(result, line)
	}

	output := strings.Join(result, "\n")

	// Preserve trailing newline.
	if strings.HasSuffix(content, "\n") && !strings.HasSuffix(output, "\n") {
		output += "\n"
	}

	return output
}

// patchINILines applies first-occurrence line replacements to INI content.
func patchINILines(content string, patches []struct{ old, new string }) string {
	patched := make(map[int]bool)
	scanner := bufio.NewScanner(strings.NewReader(content))
	var result []string

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		for i, p := range patches {
			if !patched[i] && trimmed == p.old {
				line = p.new
				patched[i] = true
				break
			}
		}

		result = append(result, line)
	}

	output := strings.Join(result, "\n")

	// Preserve trailing newline.
	if strings.HasSuffix(content, "\n") && !strings.HasSuffix(output, "\n") {
		output += "\n"
	}

	return output
}

// ensureWritable sets a file to 0644 if it exists and is not already writable.
func ensureWritable(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if info.Mode().Perm()&0200 == 0 {
		os.Chmod(path, 0644)
	}
}

// makeINIsWritable ensures all .ini files in the config directory are writable (0644).
// The game zip extracts files as read-only (0444), which prevents the game from
// saving user preferences (controller settings, graphics options, etc.).
func makeINIsWritable(configDir string) {
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".ini") {
			ensureWritable(filepath.Join(configDir, entry.Name()))
		}
	}
}

// isSteamDeck returns true if running on a Steam Deck.
func isSteamDeck() bool {
	if wine.DetectDistro() == "steamos" {
		return true
	}
	if _, err := os.Stat("/home/deck"); err == nil {
		return true
	}
	return false
}
