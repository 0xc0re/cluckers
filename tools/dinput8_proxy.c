/*
 * dinput8_proxy.c -- DirectInput8 proxy DLL for Realm Royale controller diagnostics.
 *
 * This DLL intercepts IDirectInputDevice8::GetDeviceState to log the actual
 * DIJOYSTATE2 data the game reads each frame. It also logs Acquire/Unacquire
 * cycling and SetCooperativeLevel calls for diagnosis.
 *
 * BUILD:
 *   x86_64-w64-mingw32-gcc -shared -o tools/dinput8.dll tools/dinput8_proxy.c \
 *       -lole32 -luuid
 *
 * DEPLOY:
 *   1. Copy dinput8.dll to the game's executable directory (next to RoyaleGame.exe)
 *   2. Set WINEDLLOVERRIDES=dxgi,dinput8=n in the game's environment
 *      (The Go launcher sets dxgi=n; for proxy testing, extend to dxgi,dinput8=n)
 *   3. Launch the game normally via: cluckers launch --verbose
 *
 * READ LOGS:
 *   - Windows path: C:\cluckers_dinput8.log
 *   - Linux path:   Z:/cluckers_dinput8.log  (actually /cluckers_dinput8.log)
 *   - Or from Wine prefix: $WINEPREFIX/drive_c/cluckers_dinput8.log
 *
 * WHAT TO LOOK FOR:
 *   - Are axis values (lX, lY, lRx, lRy) changing when sticks are moved?
 *     If all zeros despite physical input, the data format mapping is wrong.
 *   - Are buttons non-zero on press? If not, button mapping is wrong.
 *   - Is Acquire/Unacquire cycling every frame? (Known UE3 behavior, not a bug.)
 *   - What cooperative level flags does the game set?
 */

#define COBJMACROS
#define INITGUID
#define DIRECTINPUT_VERSION 0x0800

#include <windows.h>
#include <dinput.h>
#include <stdio.h>

/* ---------------------------------------------------------------------------
 * Globals
 * --------------------------------------------------------------------------- */

static FILE *g_log = NULL;
static HMODULE g_real_dinput8 = NULL;

typedef HRESULT (WINAPI *PFN_DirectInput8Create)(
    HINSTANCE, DWORD, REFIID, LPVOID *, LPUNKNOWN);
static PFN_DirectInput8Create g_real_DI8Create = NULL;

/* Counters for throttled logging. */
static LONG g_getdevicestate_count = 0;
static LONG g_acquire_count = 0;
static LONG g_unacquire_count = 0;

/* Original vtable function pointers (per-device, but we only hook the first). */
typedef HRESULT (WINAPI *PFN_GetDeviceState)(IDirectInputDevice8W *, DWORD, LPVOID);
typedef HRESULT (WINAPI *PFN_Acquire)(IDirectInputDevice8W *);
typedef HRESULT (WINAPI *PFN_Unacquire)(IDirectInputDevice8W *);
typedef HRESULT (WINAPI *PFN_SetCooperativeLevel)(IDirectInputDevice8W *, HWND, DWORD);

static PFN_GetDeviceState g_orig_GetDeviceState = NULL;
static PFN_Acquire g_orig_Acquire = NULL;
static PFN_Unacquire g_orig_Unacquire = NULL;
static PFN_SetCooperativeLevel g_orig_SetCooperativeLevel = NULL;

/* ---------------------------------------------------------------------------
 * Logging helpers
 * --------------------------------------------------------------------------- */

static void log_init(void)
{
    if (!g_log) {
        g_log = fopen("C:\\cluckers_dinput8.log", "w");
        if (g_log) {
            fprintf(g_log, "=== dinput8 proxy loaded ===\n");
            fflush(g_log);
        }
    }
}

static void log_msg(const char *fmt, ...)
{
    if (!g_log) return;
    va_list ap;
    va_start(ap, fmt);
    vfprintf(g_log, fmt, ap);
    va_end(ap);
    fflush(g_log);
}

/* ---------------------------------------------------------------------------
 * IDirectInputDevice8 vtable hooks
 * --------------------------------------------------------------------------- */

static HRESULT WINAPI proxy_GetDeviceState(
    IDirectInputDevice8W *self, DWORD cbData, LPVOID lpvData)
{
    HRESULT hr = g_orig_GetDeviceState(self, cbData, lpvData);
    LONG n = InterlockedIncrement(&g_getdevicestate_count);

    /* Log first 100 calls, then every 300th call (~5s at 60fps). */
    BOOL should_log = (n <= 100) || ((n % 300) == 0);

    /* Always log if there are non-zero button presses. */
    if (SUCCEEDED(hr) && cbData >= sizeof(DIJOYSTATE2) && lpvData) {
        DIJOYSTATE2 *js = (DIJOYSTATE2 *)lpvData;

        BOOL has_buttons = FALSE;
        for (int i = 0; i < 16; i++) {
            if (js->rgbButtons[i] & 0x80) {
                has_buttons = TRUE;
                break;
            }
        }
        if (has_buttons) should_log = TRUE;

        if (should_log) {
            log_msg("[GDS #%ld] hr=0x%08lx size=%lu  "
                    "X=%ld Y=%ld Z=%ld  RX=%ld RY=%ld RZ=%ld  "
                    "S0=%ld S1=%ld  POV=%lu %lu %lu %lu  Btns=",
                    n, hr, cbData,
                    js->lX, js->lY, js->lZ,
                    js->lRx, js->lRy, js->lRz,
                    js->rglSlider[0], js->rglSlider[1],
                    js->rgdwPOV[0], js->rgdwPOV[1],
                    js->rgdwPOV[2], js->rgdwPOV[3]);
            for (int i = 0; i < 16; i++) {
                if (js->rgbButtons[i] & 0x80)
                    log_msg("%d ", i);
            }
            log_msg("\n");
        }
    } else if (should_log) {
        log_msg("[GDS #%ld] hr=0x%08lx size=%lu (no DIJOYSTATE2 parse)\n",
                n, hr, cbData);
    }

    return hr;
}

static HRESULT WINAPI proxy_Acquire(IDirectInputDevice8W *self)
{
    HRESULT hr = g_orig_Acquire(self);
    LONG n = InterlockedIncrement(&g_acquire_count);

    /* Log first 10 and then every 300th. */
    if (n <= 10 || (n % 300) == 0) {
        log_msg("[Acquire #%ld] hr=0x%08lx\n", n, hr);
    }
    return hr;
}

static HRESULT WINAPI proxy_Unacquire(IDirectInputDevice8W *self)
{
    HRESULT hr = g_orig_Unacquire(self);
    LONG n = InterlockedIncrement(&g_unacquire_count);

    /* Log first 10 and then every 300th. */
    if (n <= 10 || (n % 300) == 0) {
        log_msg("[Unacquire #%ld] hr=0x%08lx\n", n, hr);
    }
    return hr;
}

static HRESULT WINAPI proxy_SetCooperativeLevel(
    IDirectInputDevice8W *self, HWND hwnd, DWORD dwFlags)
{
    log_msg("[SetCooperativeLevel] hwnd=%p flags=0x%08lx", hwnd, dwFlags);
    if (dwFlags & DISCL_FOREGROUND) log_msg(" FOREGROUND");
    if (dwFlags & DISCL_BACKGROUND) log_msg(" BACKGROUND");
    if (dwFlags & DISCL_EXCLUSIVE) log_msg(" EXCLUSIVE");
    if (dwFlags & DISCL_NONEXCLUSIVE) log_msg(" NONEXCLUSIVE");
    log_msg("\n");

    return g_orig_SetCooperativeLevel(self, hwnd, dwFlags);
}

/* ---------------------------------------------------------------------------
 * Vtable hooking: patch the device's vtable entries in-place.
 *
 * COM objects have a pointer to a vtable at offset 0. We make the vtable
 * writable, swap function pointers, and restore protection. This avoids
 * needing to allocate a full proxy vtable.
 * --------------------------------------------------------------------------- */

static void hook_device(IDirectInputDevice8W *dev)
{
    /* The vtable is an array of function pointers. We need the offsets for:
     *   Acquire         = index 7
     *   Unacquire       = index 8
     *   GetDeviceState  = index 9
     *   SetCooperativeLevel = index 13
     * These indices are for IDirectInputDevice8W (IUnknown has 3 entries,
     * then IDirectInputDevice8 methods follow).
     */
    void **vtable = *(void ***)dev;
    DWORD old_protect;

    /* Save originals before patching. */
    g_orig_Acquire = (PFN_Acquire)vtable[7];
    g_orig_Unacquire = (PFN_Unacquire)vtable[8];
    g_orig_GetDeviceState = (PFN_GetDeviceState)vtable[9];
    g_orig_SetCooperativeLevel = (PFN_SetCooperativeLevel)vtable[13];

    /* Make vtable writable. Patch a range covering indices 7-13. */
    VirtualProtect(&vtable[7], 7 * sizeof(void *), PAGE_READWRITE, &old_protect);

    vtable[7] = (void *)proxy_Acquire;
    vtable[8] = (void *)proxy_Unacquire;
    vtable[9] = (void *)proxy_GetDeviceState;
    vtable[13] = (void *)proxy_SetCooperativeLevel;

    VirtualProtect(&vtable[7], 7 * sizeof(void *), old_protect, &old_protect);

    log_msg("Hooked IDirectInputDevice8W vtable at %p\n", vtable);
}

/* ---------------------------------------------------------------------------
 * IDirectInput8 CreateDevice intercept.
 *
 * We hook CreateDevice on the IDirectInput8 interface to intercept every
 * device the game creates, then hook that device's vtable.
 * --------------------------------------------------------------------------- */

typedef HRESULT (WINAPI *PFN_CreateDevice)(
    IDirectInput8W *, REFGUID, IDirectInputDevice8W **, LPUNKNOWN);
static PFN_CreateDevice g_orig_CreateDevice = NULL;

static HRESULT WINAPI proxy_CreateDevice(
    IDirectInput8W *self, REFGUID rguid,
    IDirectInputDevice8W **ppDevice, LPUNKNOWN pUnkOuter)
{
    HRESULT hr = g_orig_CreateDevice(self, rguid, ppDevice, pUnkOuter);

    if (SUCCEEDED(hr) && ppDevice && *ppDevice) {
        /* Get device info for logging. */
        DIDEVICEINSTANCEW di;
        di.dwSize = sizeof(di);
        if (SUCCEEDED(IDirectInputDevice8_GetDeviceInfo(*ppDevice, &di))) {
            log_msg("[CreateDevice] name='%ls' type=0x%08lx\n",
                    di.tszProductName, di.dwDevType);

            /* Only hook game controllers, not keyboard/mouse. */
            BYTE devType = LOBYTE(LOWORD(di.dwDevType));
            if (devType == DI8DEVTYPE_GAMEPAD ||
                devType == DI8DEVTYPE_JOYSTICK ||
                devType == DI8DEVTYPE_1STPERSON) {
                log_msg("  -> Hooking gamepad device vtable\n");
                hook_device(*ppDevice);
            } else {
                log_msg("  -> Skipping non-gamepad device (type byte=0x%02x)\n", devType);
            }
        } else {
            /* Can't get device info; hook it anyway to be safe. */
            log_msg("[CreateDevice] (unknown device) -> hooking\n");
            hook_device(*ppDevice);
        }
    }

    return hr;
}

static void hook_dinput8(IDirectInput8W *di8)
{
    /* IDirectInput8::CreateDevice is at vtable index 3
     * (after QueryInterface=0, AddRef=1, Release=2). */
    void **vtable = *(void ***)di8;
    DWORD old_protect;

    g_orig_CreateDevice = (PFN_CreateDevice)vtable[3];

    VirtualProtect(&vtable[3], sizeof(void *), PAGE_READWRITE, &old_protect);
    vtable[3] = (void *)proxy_CreateDevice;
    VirtualProtect(&vtable[3], sizeof(void *), old_protect, &old_protect);

    log_msg("Hooked IDirectInput8W::CreateDevice at vtable %p\n", vtable);
}

/* ---------------------------------------------------------------------------
 * Exported DirectInput8Create -- the entry point that replaces the real one.
 * --------------------------------------------------------------------------- */

__declspec(dllexport) HRESULT WINAPI DirectInput8Create(
    HINSTANCE hinst, DWORD dwVersion, REFIID riidltf,
    LPVOID *ppvOut, LPUNKNOWN punkOuter)
{
    log_init();
    log_msg("[DirectInput8Create] version=0x%08lx\n", dwVersion);

    if (!g_real_DI8Create) {
        log_msg("ERROR: real DirectInput8Create not loaded\n");
        return E_FAIL;
    }

    HRESULT hr = g_real_DI8Create(hinst, dwVersion, riidltf, ppvOut, punkOuter);
    if (SUCCEEDED(hr) && ppvOut && *ppvOut) {
        /* Hook the IDirectInput8 interface to intercept CreateDevice. */
        hook_dinput8((IDirectInput8W *)*ppvOut);
    }

    return hr;
}

/* ---------------------------------------------------------------------------
 * DllMain -- load the real dinput8.dll from system32.
 * --------------------------------------------------------------------------- */

BOOL WINAPI DllMain(HINSTANCE hinstDLL, DWORD fdwReason, LPVOID lpvReserved)
{
    (void)hinstDLL;
    (void)lpvReserved;

    if (fdwReason == DLL_PROCESS_ATTACH) {
        log_init();
        log_msg("dinput8 proxy DLL loaded (pid=%lu)\n", GetCurrentProcessId());

        /* Load the real dinput8.dll from system32. */
        g_real_dinput8 = LoadLibraryA("C:\\windows\\system32\\dinput8.dll");
        if (!g_real_dinput8) {
            log_msg("FATAL: could not load real dinput8.dll (error=%lu)\n",
                    GetLastError());
            return FALSE;
        }

        g_real_DI8Create = (PFN_DirectInput8Create)
            GetProcAddress(g_real_dinput8, "DirectInput8Create");
        if (!g_real_DI8Create) {
            log_msg("FATAL: could not find DirectInput8Create in real dinput8.dll\n");
            return FALSE;
        }

        log_msg("Real dinput8.dll loaded from system32\n");
    } else if (fdwReason == DLL_PROCESS_DETACH) {
        log_msg("dinput8 proxy DLL unloading "
                "(GDS calls=%ld, Acquire=%ld, Unacquire=%ld)\n",
                g_getdevicestate_count, g_acquire_count, g_unacquire_count);
        if (g_log) {
            fclose(g_log);
            g_log = NULL;
        }
        if (g_real_dinput8) {
            FreeLibrary(g_real_dinput8);
            g_real_dinput8 = NULL;
        }
    }

    return TRUE;
}
