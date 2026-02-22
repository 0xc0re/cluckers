# Makefile for cross-compiled Windows tools (mingw-w64).
#
# These tools run inside the Wine environment alongside the game.
# Build requires: x86_64-w64-mingw32-gcc (mingw-w64 toolchain)
#
# On Arch/SteamOS: sudo pacman -S mingw-w64-gcc
# On Debian/Ubuntu: sudo apt install gcc-mingw-w64-x86-64

CC_WIN64 = x86_64-w64-mingw32-gcc

.PHONY: all shm_launcher dinput8_proxy clean

all: shm_launcher dinput8_proxy

# shm_launcher.exe: Creates Win32 named shared memory for content bootstrap,
# then launches the game as a child process.
shm_launcher: tools/shm_launcher.exe
tools/shm_launcher.exe: tools/shm_launcher.c
	$(CC_WIN64) -o $@ $< -municode

# dinput8.dll: Proxy DLL that intercepts DirectInput8 to log DIJOYSTATE2 data.
# Deploy: copy to game directory, set WINEDLLOVERRIDES=dxgi,dinput8=n
dinput8_proxy: tools/dinput8.dll
tools/dinput8.dll: tools/dinput8_proxy.c
	$(CC_WIN64) -shared -o $@ $< -lole32 -luuid

clean:
	rm -f tools/shm_launcher.exe tools/dinput8.dll
