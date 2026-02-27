/*
 * shm_launcher.c - Creates a named shared memory section with content bootstrap
 * data, then launches the game executable. The game expects to find the bootstrap
 * via OpenFileMapping() using the name passed in -content_bootstrap_shm=.
 *
 * Build: x86_64-w64-mingw32-gcc -o shm_launcher.exe shm_launcher.c -municode
 * Usage: shm_launcher.exe <bootstrap_file> <shm_name> <game_exe> [game_args...]
 *
 * When no CLI arguments are given, reads arguments from launch-config.txt in the
 * same directory as the executable. Each line in the config file is one argument
 * (line 1 = bootstrap_file, line 2 = shm_name, line 3 = game_exe, rest = game args).
 *
 * The launcher:
 *   1. Reads bootstrap data from <bootstrap_file>
 *   2. Creates a named file mapping called <shm_name>
 *   3. Copies the bootstrap data into the mapping
 *   4. Launches <game_exe> with the remaining arguments
 *   5. Waits for the game to exit
 *   6. Cleans up the mapping
 */

#include <windows.h>
#include <stdio.h>

#define MAX_CONFIG_ARGS 64
#define MAX_LINE_LEN 4096

/*
 * read_config_file - Reads launch-config.txt from the same directory as the
 * running executable. Each non-empty line becomes one argument.
 * Returns the number of arguments read, or 0 on failure.
 */
static int read_config_file(wchar_t *args[], int max_args) {
    /* Get the path of the running executable. */
    wchar_t exe_path[MAX_PATH];
    DWORD len = GetModuleFileNameW(NULL, exe_path, MAX_PATH);
    if (len == 0 || len >= MAX_PATH) {
        fprintf(stderr, "GetModuleFileName failed (err=%lu)\n", GetLastError());
        return 0;
    }

    /* Replace the exe filename with launch-config.txt. */
    wchar_t *last_slash = wcsrchr(exe_path, L'\\');
    if (!last_slash) last_slash = wcsrchr(exe_path, L'/');
    if (last_slash)
        wcscpy(last_slash + 1, L"launch-config.txt");
    else
        wcscpy(exe_path, L"launch-config.txt");

    FILE *f = _wfopen(exe_path, L"r, ccs=UTF-8");
    if (!f) {
        fprintf(stderr, "Could not open config file: %ls (err=%lu)\n", exe_path, GetLastError());
        return 0;
    }

    printf("[shm_launcher] Reading config: %ls\n", exe_path);

    int count = 0;
    wchar_t line_buf[MAX_LINE_LEN];
    while (count < max_args && fgetws(line_buf, MAX_LINE_LEN, f)) {
        /* Strip trailing newline/carriage return. */
        size_t line_len = wcslen(line_buf);
        while (line_len > 0 && (line_buf[line_len - 1] == L'\n' || line_buf[line_len - 1] == L'\r')) {
            line_buf[--line_len] = L'\0';
        }

        /* Skip empty lines. */
        if (line_len == 0) continue;

        args[count] = (wchar_t *)malloc((line_len + 1) * sizeof(wchar_t));
        if (!args[count]) {
            fprintf(stderr, "malloc failed for config line %d\n", count);
            fclose(f);
            return count;
        }
        wcscpy(args[count], line_buf);
        count++;
    }

    fclose(f);
    printf("[shm_launcher] Read %d args from config\n", count);
    return count;
}

int wmain(int argc, wchar_t *argv[]) {
    wchar_t *config_args[MAX_CONFIG_ARGS];
    int config_argc = 0;
    int use_config = 0;

    if (argc < 4) {
        /* No CLI args — try config file. */
        config_argc = read_config_file(config_args, MAX_CONFIG_ARGS);
        if (config_argc < 3) {
            fprintf(stderr, "Usage: shm_launcher.exe <bootstrap_file> <shm_name> <game_exe> [game_args...]\n");
            fprintf(stderr, "  Or place a launch-config.txt next to the executable.\n");
            /* Free any partially read config args. */
            for (int i = 0; i < config_argc; i++) free(config_args[i]);
            return 1;
        }
        use_config = 1;
    }

    /* Select argument source: CLI or config file. */
    int eff_argc = use_config ? config_argc : (argc - 1);
    wchar_t **eff_argv = use_config ? config_args : (argv + 1);

    wchar_t *bootstrap_file = eff_argv[0];
    wchar_t *shm_name = eff_argv[1];
    wchar_t *game_exe = eff_argv[2];

    /* Read bootstrap data from file */
    HANDLE hFile = CreateFileW(bootstrap_file, GENERIC_READ, FILE_SHARE_READ,
                               NULL, OPEN_EXISTING, 0, NULL);
    if (hFile == INVALID_HANDLE_VALUE) {
        fprintf(stderr, "Failed to open bootstrap file (err=%lu)\n", GetLastError());
        return 1;
    }

    DWORD fileSize = GetFileSize(hFile, NULL);
    if (fileSize == INVALID_FILE_SIZE || fileSize == 0) {
        fprintf(stderr, "Invalid bootstrap file size (err=%lu)\n", GetLastError());
        CloseHandle(hFile);
        return 1;
    }

    BYTE *data = (BYTE *)malloc(fileSize);
    if (!data) {
        fprintf(stderr, "malloc failed\n");
        CloseHandle(hFile);
        return 1;
    }

    DWORD bytesRead;
    if (!ReadFile(hFile, data, fileSize, &bytesRead, NULL) || bytesRead != fileSize) {
        fprintf(stderr, "Failed to read bootstrap file (err=%lu)\n", GetLastError());
        free(data);
        CloseHandle(hFile);
        return 1;
    }
    CloseHandle(hFile);

    printf("[shm_launcher] Bootstrap data: %lu bytes\n", fileSize);

    /* Create named shared memory */
    HANDLE hMapping = CreateFileMappingW(INVALID_HANDLE_VALUE, NULL, PAGE_READWRITE,
                                         0, fileSize, shm_name);
    if (!hMapping) {
        fprintf(stderr, "CreateFileMapping failed (err=%lu)\n", GetLastError());
        free(data);
        return 1;
    }

    LPVOID pView = MapViewOfFile(hMapping, FILE_MAP_WRITE, 0, 0, fileSize);
    if (!pView) {
        fprintf(stderr, "MapViewOfFile failed (err=%lu)\n", GetLastError());
        CloseHandle(hMapping);
        free(data);
        return 1;
    }

    memcpy(pView, data, fileSize);
    free(data);

    printf("[shm_launcher] Shared memory '%ls' created (%lu bytes)\n", shm_name, fileSize);

    /* Build command line for the game */
    wchar_t cmdline[32768];
    int pos = 0;

    /* Quote the exe path */
    pos += swprintf(cmdline + pos, sizeof(cmdline)/sizeof(wchar_t) - pos, L"\"%ls\"", game_exe);

    /* Append remaining args (index 3+ in effective argv) */
    for (int i = 3; i < eff_argc; i++) {
        pos += swprintf(cmdline + pos, sizeof(cmdline)/sizeof(wchar_t) - pos, L" %ls", eff_argv[i]);
    }

    printf("[shm_launcher] Launching: %ls\n", cmdline);

    /* Launch the game */
    STARTUPINFOW si = { .cb = sizeof(si) };
    PROCESS_INFORMATION pi = {0};

    if (!CreateProcessW(NULL, cmdline, NULL, NULL, FALSE, 0, NULL, NULL, &si, &pi)) {
        fprintf(stderr, "CreateProcess failed (err=%lu)\n", GetLastError());
        UnmapViewOfFile(pView);
        CloseHandle(hMapping);
        return 1;
    }

    printf("[shm_launcher] Game started (pid=%lu), waiting...\n", pi.dwProcessId);

    /* Wait for game to exit */
    WaitForSingleObject(pi.hProcess, INFINITE);

    DWORD exitCode = 0;
    GetExitCodeProcess(pi.hProcess, &exitCode);
    printf("[shm_launcher] Game exited with code %lu\n", exitCode);

    /* Cleanup */
    CloseHandle(pi.hThread);
    CloseHandle(pi.hProcess);
    UnmapViewOfFile(pView);
    CloseHandle(hMapping);

    /* Free config file args if used. */
    if (use_config) {
        for (int i = 0; i < config_argc; i++) free(config_args[i]);
    }

    return (int)exitCode;
}
