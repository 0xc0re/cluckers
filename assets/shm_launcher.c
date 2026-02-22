/*
 * shm_launcher.c - Creates a named shared memory section with content bootstrap
 * data, then launches the game executable. The game expects to find the bootstrap
 * via OpenFileMapping() using the name passed in -content_bootstrap_shm=.
 *
 * Build: x86_64-w64-mingw32-gcc -o shm_launcher.exe shm_launcher.c
 * Usage: shm_launcher.exe <bootstrap_file> <shm_name> <game_exe> [game_args...]
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

int wmain(int argc, wchar_t *argv[]) {
    if (argc < 4) {
        fprintf(stderr, "Usage: shm_launcher.exe <bootstrap_file> <shm_name> <game_exe> [game_args...]\n");
        return 1;
    }

    wchar_t *bootstrap_file = argv[1];
    wchar_t *shm_name = argv[2];
    wchar_t *game_exe = argv[3];

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
    pos += swprintf(cmdline + pos, sizeof(cmdline)/sizeof(wchar_t) - pos, L"\"%s\"", game_exe);

    /* Append remaining args */
    for (int i = 4; i < argc; i++) {
        pos += swprintf(cmdline + pos, sizeof(cmdline)/sizeof(wchar_t) - pos, L" %s", argv[i]);
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

    return (int)exitCode;
}
