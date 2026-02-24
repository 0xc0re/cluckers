package launch

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xc0re/cluckers/assets"
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

	deployDeckControllerLayout()

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

	// Ensure file is writable before writing — game zip extracts as 0444.
	ensureWritable(iniPath)

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
// Fix: Remove ALL Count commands from mouse bindings globally. UE3 coalescing
// can create duplicate entries from base templates, so we replace every
// occurrence (not just the first). Additionally, DefaultInput.ini gets UE3
// removal directives (-Bindings=...) to prevent coalescing from re-adding
// Count entries from the engine's BaseInput.ini template.
//
// Also make INI files writable (0644) so the game can persist user controller
// preferences across sessions.
//
// Idempotent: skips files that are already patched or missing.
func PatchDeckInputConfig(gameDir string) error {
	configDir := filepath.Join(gameDir, "Realm-Royale", "RealmGame", "Config")
	engineConfigDir := filepath.Join(gameDir, "Realm-Royale", "Engine", "Config")

	// Patch BaseInput.ini first — this is the engine template that UE3 uses
	// to regenerate RealmInput.ini. Without patching the source, the Count
	// commands get re-added whenever UE3 coalesces INI files.
	baseInputPath := filepath.Join(engineConfigDir, "BaseInput.ini")
	if err := patchCountCommands(baseInputPath); err != nil {
		return err
	}

	// Then patch DefaultInput.ini and RealmInput.ini.
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
		output := original

		// Replace ALL occurrences of Count patterns globally. UE3 coalescing
		// can produce duplicate entries from base templates, and section-aware
		// first-only replacement misses the duplicates.
		for _, p := range deckInputPatches {
			output = strings.ReplaceAll(output, p.old, p.new)
			// Also handle UE3 append syntax (+Bindings=) in Default*.ini.
			if strings.HasPrefix(p.old, "Bindings=") {
				output = strings.ReplaceAll(output, "+"+p.old, "+"+p.new)
			}
		}

		// For DefaultInput.ini, add UE3 removal directives (-Bindings=...) to
		// strip Count entries from the engine's BaseInput.ini during coalescing.
		// This prevents the game from re-adding Count entries when it regenerates
		// RealmInput.ini (e.g., when the user changes settings in-game).
		if filename == "DefaultInput.ini" {
			for _, p := range deckInputPatches {
				removeLine := "-" + p.old
				if !strings.Contains(output, removeLine) {
					// Insert removal directive before the corresponding +Bindings= line.
					addLine := "+" + p.new
					idx := strings.Index(output, addLine)
					if idx > 0 {
						output = output[:idx] + removeLine + "\n" + output[idx:]
					}
				}
			}
		}

		// Force gamepad mode: set bUsingGamepad=True in both PlayerInput
		// sections. UE3 reads config properties when creating a new
		// PlayerInput (including after ServerTravel). Without this, the
		// new PlayerInput may default to KB/M mode on Wine because the
		// one-time HID enumeration at startup doesn't re-fire.
		output = strings.ReplaceAll(output, "bUsingGamepad=False", "bUsingGamepad=True")
		output = strings.ReplaceAll(output, "bUsingGamepad=false", "bUsingGamepad=True")
		if !strings.Contains(output, "bUsingGamepad=True") {
			output += "\n[Engine.PlayerInput]\nbUsingGamepad=True\n\n[TgGame.TgPlayerInput]\nbUsingGamepad=True\n"
		}

		if output == original {
			ui.Verbose(filename+" input patches already applied, skipping", true)
			ensureWritable(iniPath)
			continue
		}

		ensureWritable(iniPath)

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

// patchCountCommands removes all "Count bXAxis |" and "Count bYAxis |" patterns
// from an INI file using global replacement. Idempotent — skips if not found or
// file doesn't exist.
func patchCountCommands(iniPath string) error {
	data, err := os.ReadFile(iniPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading %s: %w", iniPath, err)
	}

	original := string(data)
	output := original
	for _, p := range deckInputPatches {
		output = strings.ReplaceAll(output, p.old, p.new)
	}

	if output == original {
		return nil
	}

	ensureWritable(iniPath)
	return os.WriteFile(iniPath, []byte(output), 0644)
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

// deployDeckControllerLayout deploys the embedded Steam Deck controller layout
// to the user's Steam controller config directory. Only deploys if no config
// exists yet (preserves user customizations). Best-effort — failures are silent.
func deployDeckControllerLayout() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	// Find shortcuts.vdf files in Steam userdata directories.
	pattern := filepath.Join(home, ".local", "share", "Steam", "userdata", "*", "config", "shortcuts.vdf")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return
	}

	for _, shortcutsPath := range matches {
		data, err := os.ReadFile(shortcutsPath)
		if err != nil {
			continue
		}

		appID := findCluckersAppID(data)
		if appID == 0 {
			continue
		}

		// Build deploy path: userdata/<id>/config/controller_configs/apps/<appid>/
		userdataDir := filepath.Dir(filepath.Dir(shortcutsPath))
		deployDir := filepath.Join(userdataDir, "config", "controller_configs", "apps", fmt.Sprintf("%d", appID))
		deployPath := filepath.Join(deployDir, "controller_neptune_config.vdf")

		// Don't overwrite existing config — user may have customized it.
		if _, err := os.Stat(deployPath); err == nil {
			ui.Verbose("Controller layout already exists, skipping", true)
			continue
		}

		if err := os.MkdirAll(deployDir, 0755); err != nil {
			continue
		}

		if err := os.WriteFile(deployPath, assets.ControllerLayout, 0644); err != nil {
			continue
		}

		ui.Verbose("Deployed Steam Deck controller layout for app "+fmt.Sprintf("%d", appID), true)
	}
}

// findCluckersAppID searches a binary VDF shortcuts.vdf for a shortcut whose
// exe field contains "cluckers" and returns its app ID. Returns 0 if not found.
//
// Binary VDF field types: \x01 = string (key\x00 + value\x00),
// \x02 = int32 (key\x00 + 4 bytes LE).
func findCluckersAppID(data []byte) uint32 {
	exeField := []byte("\x01exe\x00")
	appIDField := []byte("\x02appid\x00")

	offset := 0
	for {
		idx := bytes.Index(data[offset:], exeField)
		if idx < 0 {
			return 0
		}

		strStart := offset + idx + len(exeField)
		if strStart >= len(data) {
			return 0
		}

		// Read null-terminated exe path string.
		strEnd := bytes.IndexByte(data[strStart:], 0x00)
		if strEnd < 0 {
			return 0
		}

		exePath := strings.ToLower(string(data[strStart : strStart+strEnd]))

		if strings.Contains(exePath, "cluckers") {
			// Found our shortcut. Search backward for the appid field.
			region := data[:offset+idx]
			aidIdx := bytes.LastIndex(region, appIDField)
			if aidIdx >= 0 {
				valStart := aidIdx + len(appIDField)
				if valStart+4 <= len(data) {
					return binary.LittleEndian.Uint32(data[valStart : valStart+4])
				}
			}
		}

		offset = strStart + strEnd + 1
	}
}
