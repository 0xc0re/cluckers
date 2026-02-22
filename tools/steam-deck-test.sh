#!/bin/sh
# steam-deck-test.sh -- Revert debug changes and test controller input on Steam Deck.
#
# Run this script on the Steam Deck (Desktop Mode terminal or SSH).
# It reverts changes from prior debugging sessions and walks through
# a phased testing procedure for controller input.
#
# Usage: sh tools/steam-deck-test.sh

set -e

CLUCKERS_BIN="/home/deck/.local/bin/cluckers"
CLUCKERS_REAL="/home/deck/.local/bin/cluckers.real"

# RealmInput.ini location (inside the game directory).
# Adjust if your game is installed elsewhere.
GAME_DIR="${CLUCKERS_GAME_DIR:-$HOME/.cluckers/game}"
REALM_INPUT_INI="$GAME_DIR/Engine/Config/BaseInput.ini"

# --------------------------------------------------------------------------- #
# Phase 0: Revert prior debugging changes
# --------------------------------------------------------------------------- #

echo "============================================"
echo "Phase 0: Revert prior debugging changes"
echo "============================================"
echo ""

# 0a. Restore original cluckers binary if it was renamed.
if [ -f "$CLUCKERS_REAL" ]; then
    echo "  Found $CLUCKERS_REAL -- restoring original binary..."
    mv "$CLUCKERS_REAL" "$CLUCKERS_BIN"
    echo "  Restored: $CLUCKERS_BIN"
else
    echo "  No backup binary found at $CLUCKERS_REAL -- nothing to restore."
fi

# 0b. Remove any debug wrapper scripts that might shadow the binary.
WRAPPER_SCRIPTS="/home/deck/.local/bin/cluckers-debug /home/deck/.local/bin/cluckers-wrapper"
for script in $WRAPPER_SCRIPTS; do
    if [ -f "$script" ]; then
        echo "  Removing debug wrapper: $script"
        rm -f "$script"
    fi
done

# 0c. Fix RealmInput.ini: revert c_bUseServerBindings=false back to true.
if [ -f "$REALM_INPUT_INI" ]; then
    if grep -q 'c_bUseServerBindings=false' "$REALM_INPUT_INI" 2>/dev/null; then
        echo "  Fixing RealmInput.ini: setting c_bUseServerBindings=true..."
        sed -i 's/c_bUseServerBindings=false/c_bUseServerBindings=true/g' "$REALM_INPUT_INI"
        echo "  Fixed."
    else
        echo "  RealmInput.ini already has c_bUseServerBindings=true (or file not found)."
    fi
else
    echo "  RealmInput.ini not found at $REALM_INPUT_INI -- skipping."
    echo "  (Set CLUCKERS_GAME_DIR if your game directory is elsewhere.)"
fi

echo ""
echo "Phase 0 complete. Prior debug changes reverted."
echo ""

# --------------------------------------------------------------------------- #
# Phase A: Test with STEAM_INPUT_DISABLE=1
# --------------------------------------------------------------------------- #

echo "============================================"
echo "Phase A: Test with Steam Input disabled"
echo "============================================"
echo ""
echo "This disables Steam Input's virtual gamepad layer so the game sees"
echo "the Deck's raw controller hardware via Wine's winebus.sys (evdev/HID)."
echo ""
echo "The updated cluckers binary (quick-8) sets this automatically on"
echo "Steam Deck. If you have the latest binary, just run:"
echo ""
echo "  cluckers launch --verbose"
echo ""
echo "If you want to test manually with additional debug output:"
echo ""
echo "  STEAM_INPUT_DISABLE=1 SDL_JOYSTICK_HIDAPI=0 cluckers launch --verbose"
echo ""
echo "What to check:"
echo "  - Does the game detect a controller at startup?"
echo "  - Can you navigate menus with the joystick?"
echo "  - If it still stays in keyboard mode, proceed to Phase B."
echo ""
printf "Press Enter when ready to see Phase B instructions..."
read dummy
echo ""

# --------------------------------------------------------------------------- #
# Phase B: Deploy dinput8 proxy DLL for diagnosis
# --------------------------------------------------------------------------- #

echo "============================================"
echo "Phase B: Deploy dinput8 proxy DLL"
echo "============================================"
echo ""
echo "If Phase A did not fix controller input, deploy the dinput8 proxy DLL"
echo "to capture the exact DirectInput data the game receives."
echo ""
echo "Steps:"
echo ""
echo "  1. Build the DLL (requires mingw-w64, do this on your dev machine):"
echo "     make dinput8_proxy"
echo ""
echo "  2. Copy dinput8.dll to the game's executable directory:"
echo "     scp tools/dinput8.dll deck@steamdeck:~/.cluckers/game/Binaries/Win64/"
echo ""
echo "  3. Set WINEDLLOVERRIDES to load the proxy:"
echo "     WINEDLLOVERRIDES=dxgi,dinput8=n cluckers launch --verbose"
echo ""
echo "  4. Play for a few seconds, then exit the game."
echo ""
echo "  5. Check the log file. From the Deck:"
DINPUT_LOG_WINE="\$WINEPREFIX/drive_c/cluckers_dinput8.log"
DINPUT_LOG_LINUX="/cluckers_dinput8.log"
echo "     cat $DINPUT_LOG_WINE"
echo "     # or: cat $DINPUT_LOG_LINUX"
echo ""
printf "Press Enter when ready to see Phase C instructions..."
read dummy
echo ""

# --------------------------------------------------------------------------- #
# Phase C: Read the log and report findings
# --------------------------------------------------------------------------- #

echo "============================================"
echo "Phase C: Interpreting the dinput8 log"
echo "============================================"
echo ""
echo "The log file shows DIJOYSTATE2 data for each GetDeviceState call."
echo ""
echo "Look for these patterns:"
echo ""
echo "  1. AXES CHANGING: If lX, lY, lRx, lRy values change when you move"
echo "     the sticks, DirectInput data flow is working correctly."
echo "     The problem is likely in the game's input mode detection."
echo ""
echo "  2. ALL ZEROS: If all axis values are 0 despite moving sticks,"
echo "     the game's custom data format doesn't match the device's axes."
echo "     The proxy DLL may need axis remapping added."
echo ""
echo "  3. BUTTONS: If rgbButtons show non-zero on press, button mapping works."
echo "     If buttons are always zero, the button format mapping is wrong."
echo ""
echo "  4. ACQUIRE CYCLING: High Acquire/Unacquire counts are normal for"
echo "     this UE3 game (it cycles every frame). Not a bug."
echo ""
echo "  5. NO GAMEPAD DEVICE: If the log only shows keyboard/mouse CreateDevice"
echo "     calls and no gamepad, the controller isn't being enumerated at all."
echo "     Check: ls /dev/input/event* and evtest to see raw hardware."
echo ""
echo "Share the log file for further analysis:"
echo "  cat $DINPUT_LOG_WINE"
echo ""
echo "============================================"
echo "Testing procedure complete."
echo "============================================"
