#include <windows.h>
#include <stdio.h>

/*
 * XInput index remapping proxy for UE3 games on Wine.
 *
 * Problem: UE3 reserves XInput index 0 for keyboard, polls indices 1-3.
 *          Wine assigns the controller to XInput index 0.
 * Fix:     Remap game's index N -> real index N-1.
 *
 * Tries xinput1_4, xinput9_1_0, xinput1_2, xinput1_1 as backends.
 */

typedef DWORD (WINAPI *pfn_XInputGetState)(DWORD, void*);
typedef DWORD (WINAPI *pfn_XInputGetCapabilities)(DWORD, DWORD, void*);
typedef DWORD (WINAPI *pfn_XInputSetState)(DWORD, void*);
typedef void  (WINAPI *pfn_XInputEnable)(BOOL);

static HMODULE hReal = NULL;
static pfn_XInputGetState pGetState = NULL;
static pfn_XInputGetCapabilities pGetCaps = NULL;
static pfn_XInputSetState pSetState = NULL;
static pfn_XInputEnable pEnable = NULL;
static BOOL initialized = FALSE;
static CRITICAL_SECTION g_initLock;
static BOOL g_lockReady = FALSE;
static FILE *logf = NULL;
static int n = 0;

static void proxy_init(void) {
    if (initialized) return;
    if (g_lockReady) EnterCriticalSection(&g_initLock);
    if (initialized) { if (g_lockReady) LeaveCriticalSection(&g_initLock); return; }

    logf = fopen("Z:\\tmp\\xinput_remap.log", "w");

    /* Try multiple xinput DLL variants as backends */
    static const wchar_t *dlls[] = {
        L"xinput1_4.dll",
        L"xinput9_1_0.dll",
        L"xinput1_2.dll",
        L"xinput1_1.dll",
        NULL
    };

    for (int i = 0; dlls[i]; i++) {
        hReal = LoadLibraryW(dlls[i]);
        if (hReal) {
            pGetState = (pfn_XInputGetState)GetProcAddress(hReal, "XInputGetState");
            if (pGetState) {
                /* Test if this DLL actually sees a controller at index 0 */
                BYTE state[64];
                ZeroMemory(state, sizeof(state));
                DWORD r = pGetState(0, state);
                if (logf) fprintf(logf, "REMAP: %ls loaded, GetState(0)=%lu\n", dlls[i], r);
                if (r == 0) {
                    /* Found a working backend */
                    pGetCaps = (pfn_XInputGetCapabilities)GetProcAddress(hReal, "XInputGetCapabilities");
                    pSetState = (pfn_XInputSetState)GetProcAddress(hReal, "XInputSetState");
                    pEnable = (pfn_XInputEnable)GetProcAddress(hReal, "XInputEnable");
                    if (logf) { fprintf(logf, "REMAP: Using %ls as backend (controller at index 0)\n", dlls[i]); fflush(logf); }
                    break;
                }
            }
            FreeLibrary(hReal);
            hReal = NULL;
            pGetState = NULL;
        } else {
            if (logf) fprintf(logf, "REMAP: Failed to load %ls (err=%lu)\n", dlls[i], GetLastError());
        }
    }

    if (!hReal && logf) { fprintf(logf, "REMAP: No working backend found!\n"); fflush(logf); }

    initialized = TRUE;
    if (g_lockReady) LeaveCriticalSection(&g_initLock);
}

/* Remap: game's 1->0, 2->1, 3->2. Index 0 stays 0. */
static DWORD remap(DWORD idx) {
    if (idx >= 1 && idx <= 3) return idx - 1;
    return idx;
}

__declspec(dllexport) DWORD WINAPI XInputGetState(DWORD idx, void *state) {
    proxy_init();
    if (!pGetState) return 0x48F;
    DWORD real_idx = remap(idx);
    DWORD r = pGetState(real_idx, state);
    n++;
    if (logf && r == 0 && (n <= 50 || n % 500 == 0)) {
        /* Log actual state data to verify controller values reach the game.
         * XINPUT_STATE layout: DWORD packet, WORD buttons, BYTE LT, BYTE RT,
         *                      SHORT LX, SHORT LY, SHORT RX, SHORT RY */
        BYTE *s = (BYTE*)state;
        WORD btns = *(WORD*)(s+4);
        SHORT lx = *(SHORT*)(s+8);
        SHORT ly = *(SHORT*)(s+10);
        SHORT rx = *(SHORT*)(s+12);
        SHORT ry = *(SHORT*)(s+14);
        fprintf(logf, "GetState(%lu->%lu)=%lu btns=%04X LX=%d LY=%d RX=%d RY=%d [#%d]\n",
                idx, real_idx, r, btns, lx, ly, rx, ry, n);
        fflush(logf);
    }
    return r;
}

__declspec(dllexport) DWORD WINAPI XInputGetCapabilities(DWORD idx, DWORD flags, void *caps) {
    proxy_init();
    if (!pGetCaps) return 0x48F;
    return pGetCaps(remap(idx), flags, caps);
}

__declspec(dllexport) DWORD WINAPI XInputSetState(DWORD idx, void *vib) {
    proxy_init();
    if (!pSetState) return 0x48F;
    return pSetState(remap(idx), vib);
}

/*
 * XInputEnable(FALSE) no-op:
 * UE3 calls XInputEnable(FALSE) on WM_ACTIVATEAPP when the window loses focus
 * during ServerTravel map transitions (lobby -> match). Wine's compliant
 * implementation zeros all XInput state data, causing invisible controller
 * input loss for the entire match. This is the same pattern fixed in Proton
 * 8.0-4 for Overwatch 2. We block FALSE to prevent disabling, but forward
 * TRUE (harmless, keeps state consistent).
 */
__declspec(dllexport) void WINAPI XInputEnable(BOOL e) {
    proxy_init();
    if (logf) { fprintf(logf, "XInputEnable(%d) called at n=%d\n", e, n); fflush(logf); }
    if (e == FALSE) {
        if (logf) { fprintf(logf, "BLOCKED XInputEnable(FALSE) - preventing UE3 ServerTravel input loss\n"); fflush(logf); }
        return;
    }
    if (pEnable) pEnable(e);
}

__declspec(dllexport) DWORD WINAPI XInputGetStateEx(DWORD idx, void *state) {
    return XInputGetState(idx, state);
}

BOOL WINAPI DllMain(HINSTANCE h, DWORD reason, LPVOID res) {
    (void)h; (void)res;
    /* Do NOT call proxy_init here - LoadLibrary inside DllMain causes loader lock deadlock */
    if (reason == 1) {
        /* DLL_PROCESS_ATTACH: initialize critical section for thread-safe proxy_init */
        InitializeCriticalSection(&g_initLock);
        g_lockReady = TRUE;
    }
    if (reason == 0) {
        /* DLL_PROCESS_DETACH: clean up */
        if (logf) { fprintf(logf, "REMAP: unloading after %d calls\n", n); fclose(logf); }
        if (g_lockReady) { DeleteCriticalSection(&g_initLock); g_lockReady = FALSE; }
    }
    return TRUE;
}
