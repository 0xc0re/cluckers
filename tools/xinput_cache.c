/*
 * XInput caching shim DLL for Cluckers launcher.
 *
 * Wraps Proton's builtin xinput1_3.dll by loading it from Proton's system
 * directory. All XInput API calls are forwarded to the real implementation.
 * A caching layer returns last-known-good state when the real DLL returns
 * ERROR_DEVICE_NOT_CONNECTED during UE3's transient XInput re-enumeration
 * at ServerTravel boundaries.
 *
 * Build:
 *   x86_64-w64-mingw32-gcc -shared -o assets/xinput1_3_cache.dll \
 *       tools/xinput_cache.c tools/xinput_cache.def -municode -lkernel32
 *
 * Install as native override (WINEDLLOVERRIDES=xinput1_3=n,b) in the game's
 * Binaries/Win64 directory. The shim delegates to Proton's real xinput so
 * Steam Input IPC is preserved.
 *
 * CRITICAL: This is NOT the same as the Phase 7.1 xinput proxy which loaded
 * xinput1_4.dll and bypassed Steam Input IPC entirely (btns=0000). This shim
 * loads Proton's BUILTIN xinput1_3.dll by full path via LOAD_LIBRARY_SEARCH_SYSTEM32,
 * preserving the Steam Input IPC chain.
 */

#include <windows.h>
#include <xinput.h>
#include <stdio.h>

#define CACHE_TIMEOUT_MS 10000  /* 10 seconds */
#define MAX_CONTROLLERS  4

/* Function pointer types for the real XInput implementation */
typedef DWORD (WINAPI *PFN_XInputGetState)(DWORD, XINPUT_STATE*);
typedef DWORD (WINAPI *PFN_XInputGetCapabilities)(DWORD, DWORD, XINPUT_CAPABILITIES*);
typedef void  (WINAPI *PFN_XInputEnable)(BOOL);
typedef DWORD (WINAPI *PFN_XInputSetState)(DWORD, XINPUT_VIBRATION*);
typedef DWORD (WINAPI *PFN_XInputGetBatteryInformation)(DWORD, BYTE, XINPUT_BATTERY_INFORMATION*);
typedef DWORD (WINAPI *PFN_XInputGetKeystroke)(DWORD, DWORD, PXINPUT_KEYSTROKE);

/* Cached state per controller slot */
static XINPUT_STATE        cached_state[MAX_CONTROLLERS];
static XINPUT_CAPABILITIES cached_caps[MAX_CONTROLLERS];
static DWORD               state_timestamp[MAX_CONTROLLERS];
static DWORD               caps_timestamp[MAX_CONTROLLERS];
static BOOL                state_valid[MAX_CONTROLLERS];
static BOOL                caps_valid[MAX_CONTROLLERS];

/* Counters for diagnostic logging */
static DWORD cache_hits = 0;
static DWORD cache_misses = 0;

/* Real function pointers resolved from Proton's builtin DLL */
static PFN_XInputGetState              real_GetState;
static PFN_XInputGetCapabilities       real_GetCapabilities;
static PFN_XInputEnable                real_Enable;
static PFN_XInputSetState              real_SetState;
static PFN_XInputGetBatteryInformation real_GetBatteryInfo;
static PFN_XInputGetKeystroke          real_GetKeystroke;

static HMODULE real_dll;
static BOOL    initialized;

/*
 * init_real_dll loads Proton's builtin xinput implementation.
 *
 * Method 1: LoadLibraryExW with LOAD_LIBRARY_SEARCH_SYSTEM32. This asks the
 *   loader to search only the system directory, bypassing the game directory
 *   where our native override DLL lives. Under Proton, the system directory
 *   contains the builtin xinput1_3.dll with Steam Input IPC.
 *
 * Method 2 (fallback): LoadLibraryW("xinput1_4.dll"). In Proton, xinput1_4
 *   is a separate builtin that may share the same winebus/Steam Input path.
 *   This fallback is tried if Method 1 fails (e.g., LOAD_LIBRARY_SEARCH_SYSTEM32
 *   not supported in some Wine versions).
 */
static void init_real_dll(void) {
    if (initialized) return;
    initialized = TRUE;

    /* Method 1: system directory search only (bypasses native override) */
    real_dll = LoadLibraryExW(L"xinput1_3.dll", NULL, LOAD_LIBRARY_SEARCH_SYSTEM32);

    if (!real_dll) {
        /* Method 2: try xinput1_4.dll (separate builtin in Proton) */
        real_dll = LoadLibraryW(L"xinput1_4.dll");
    }

    if (!real_dll) {
        fprintf(stderr, "[xinput_cache] FATAL: Could not load real xinput DLL (err=%lu)\n",
                GetLastError());
        return;
    }

    real_GetState     = (PFN_XInputGetState)GetProcAddress(real_dll, "XInputGetState");
    real_GetCapabilities = (PFN_XInputGetCapabilities)GetProcAddress(real_dll, "XInputGetCapabilities");
    real_Enable       = (PFN_XInputEnable)GetProcAddress(real_dll, "XInputEnable");
    real_SetState     = (PFN_XInputSetState)GetProcAddress(real_dll, "XInputSetState");
    real_GetBatteryInfo = (PFN_XInputGetBatteryInformation)GetProcAddress(real_dll, "XInputGetBatteryInformation");
    real_GetKeystroke = (PFN_XInputGetKeystroke)GetProcAddress(real_dll, "XInputGetKeystroke");

    fprintf(stderr, "[xinput_cache] Loaded real DLL, caching enabled (timeout=%dms)\n",
            CACHE_TIMEOUT_MS);
    fprintf(stderr, "[xinput_cache] GetState=%p GetCaps=%p Enable=%p\n",
            (void*)real_GetState, (void*)real_GetCapabilities, (void*)real_Enable);
}

/*
 * XInputGetState: forward to real implementation, cache on success.
 * On ERROR_DEVICE_NOT_CONNECTED, return cached data if recent enough.
 */
DWORD WINAPI XInputGetState(DWORD dwUserIndex, XINPUT_STATE *pState) {
    init_real_dll();
    if (!real_GetState || dwUserIndex >= MAX_CONTROLLERS)
        return ERROR_DEVICE_NOT_CONNECTED;

    DWORD result = real_GetState(dwUserIndex, pState);

    if (result == ERROR_SUCCESS) {
        /* Cache the successful state */
        cached_state[dwUserIndex] = *pState;
        state_timestamp[dwUserIndex] = GetTickCount();
        state_valid[dwUserIndex] = TRUE;
    } else if (result == ERROR_DEVICE_NOT_CONNECTED && state_valid[dwUserIndex]) {
        /* Device reports disconnected -- check cache age */
        DWORD now = GetTickCount();
        DWORD elapsed = now - state_timestamp[dwUserIndex];

        if (elapsed < CACHE_TIMEOUT_MS) {
            /* Return cached state (device likely in transient re-enumeration) */
            *pState = cached_state[dwUserIndex];
            cache_hits++;
            if ((cache_hits & 0xFF) == 1) {
                fprintf(stderr, "[xinput_cache] Cache HIT for pad %lu (elapsed=%lums, hits=%lu)\n",
                        dwUserIndex, elapsed, cache_hits);
            }
            return ERROR_SUCCESS;
        } else {
            /* Cache expired -- device genuinely disconnected */
            state_valid[dwUserIndex] = FALSE;
            cache_misses++;
            fprintf(stderr, "[xinput_cache] Cache EXPIRED for pad %lu (elapsed=%lums)\n",
                    dwUserIndex, elapsed);
        }
    }

    return result;
}

/*
 * XInputGetCapabilities: forward to real implementation, cache on success.
 * Same caching logic as XInputGetState for transient disconnections.
 */
DWORD WINAPI XInputGetCapabilities(DWORD dwUserIndex, DWORD dwFlags,
                                    XINPUT_CAPABILITIES *pCapabilities) {
    init_real_dll();
    if (!real_GetCapabilities || dwUserIndex >= MAX_CONTROLLERS)
        return ERROR_DEVICE_NOT_CONNECTED;

    DWORD result = real_GetCapabilities(dwUserIndex, dwFlags, pCapabilities);

    if (result == ERROR_SUCCESS) {
        cached_caps[dwUserIndex] = *pCapabilities;
        caps_timestamp[dwUserIndex] = GetTickCount();
        caps_valid[dwUserIndex] = TRUE;
    } else if (result == ERROR_DEVICE_NOT_CONNECTED && caps_valid[dwUserIndex]) {
        DWORD now = GetTickCount();
        DWORD elapsed = now - caps_timestamp[dwUserIndex];

        if (elapsed < CACHE_TIMEOUT_MS) {
            *pCapabilities = cached_caps[dwUserIndex];
            return ERROR_SUCCESS;
        } else {
            caps_valid[dwUserIndex] = FALSE;
        }
    }

    return result;
}

/* Pass-through: XInputEnable */
void WINAPI XInputEnable(BOOL enable) {
    init_real_dll();
    if (real_Enable) real_Enable(enable);
}

/* Pass-through: XInputSetState (vibration) */
DWORD WINAPI XInputSetState(DWORD dwUserIndex, XINPUT_VIBRATION *pVibration) {
    init_real_dll();
    if (!real_SetState) return ERROR_DEVICE_NOT_CONNECTED;
    return real_SetState(dwUserIndex, pVibration);
}

/* Pass-through: XInputGetBatteryInformation */
DWORD WINAPI XInputGetBatteryInformation(DWORD dwUserIndex, BYTE devType,
                                          XINPUT_BATTERY_INFORMATION *pBatteryInfo) {
    init_real_dll();
    if (!real_GetBatteryInfo) return ERROR_DEVICE_NOT_CONNECTED;
    return real_GetBatteryInfo(dwUserIndex, devType, pBatteryInfo);
}

/* Pass-through: XInputGetKeystroke */
DWORD WINAPI XInputGetKeystroke(DWORD dwUserIndex, DWORD dwReserved,
                                 PXINPUT_KEYSTROKE pKeystroke) {
    init_real_dll();
    if (!real_GetKeystroke) return ERROR_DEVICE_NOT_CONNECTED;
    return real_GetKeystroke(dwUserIndex, dwReserved, pKeystroke);
}

BOOL WINAPI DllMain(HINSTANCE hinstDLL, DWORD fdwReason, LPVOID lpReserved) {
    (void)hinstDLL;
    (void)lpReserved;

    if (fdwReason == DLL_PROCESS_DETACH && real_dll) {
        fprintf(stderr, "[xinput_cache] Unloading (cache_hits=%lu, cache_misses=%lu)\n",
                cache_hits, cache_misses);
        FreeLibrary(real_dll);
        real_dll = NULL;
    }
    return TRUE;
}
