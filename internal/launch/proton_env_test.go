//go:build linux

package launch

import (
	"errors"
	"strings"
	"testing"
)

// --- filterEnv tests ---

func TestFilterEnv_RemovesSingleKey(t *testing.T) {
	env := []string{"HOME=/home/user", "LD_LIBRARY_PATH=/lib", "PATH=/bin"}
	got := filterEnv(env, "LD_LIBRARY_PATH")
	want := []string{"HOME=/home/user", "PATH=/bin"}
	assertEnvEqual(t, got, want)
}

func TestFilterEnv_RemovesMultipleKeys(t *testing.T) {
	env := []string{"A=1", "B=2", "C=3"}
	got := filterEnv(env, "A", "C")
	want := []string{"B=2"}
	assertEnvEqual(t, got, want)
}

func TestFilterEnv_NoMatch(t *testing.T) {
	env := []string{"A=1"}
	got := filterEnv(env, "X")
	want := []string{"A=1"}
	assertEnvEqual(t, got, want)
}

// --- buildProtonEnvFrom tests ---

func TestBuildProtonEnv_StripsLDLibraryPath(t *testing.T) {
	base := []string{"HOME=/home/user", "LD_LIBRARY_PATH=/appimage/lib", "PATH=/bin"}
	got := buildProtonEnvFrom(base, "/home/user/.cluckers/compatdata", false)
	assertEnvNotContainsKey(t, got, "LD_LIBRARY_PATH")
}

func TestBuildProtonEnv_StripsWINEPREFIX(t *testing.T) {
	base := []string{"WINEPREFIX=/old/prefix", "HOME=/home/user"}
	got := buildProtonEnvFrom(base, "/compat", false)
	assertEnvNotContainsKey(t, got, "WINEPREFIX")
}

func TestBuildProtonEnv_StripsWINE(t *testing.T) {
	base := []string{"WINE=/usr/bin/wine", "HOME=/home/user"}
	got := buildProtonEnvFrom(base, "/compat", false)
	assertEnvNotContainsKey(t, got, "WINE")
}

func TestBuildProtonEnv_StripsWINEFSYNC(t *testing.T) {
	base := []string{"WINEFSYNC=1", "HOME=/home/user"}
	got := buildProtonEnvFrom(base, "/compat", false)
	assertEnvNotContainsKey(t, got, "WINEFSYNC")
}

func TestBuildProtonEnv_StripsWINEESYNC(t *testing.T) {
	base := []string{"WINEESYNC=1", "HOME=/home/user"}
	got := buildProtonEnvFrom(base, "/compat", false)
	assertEnvNotContainsKey(t, got, "WINEESYNC")
}

func TestBuildProtonEnv_ReplacesWINEDLLOVERRIDES(t *testing.T) {
	base := []string{"WINEDLLOVERRIDES=something", "HOME=/home/user"}
	got := buildProtonEnvFrom(base, "/compat", false)
	// Old value should be stripped and replaced with dxgi=n
	assertEnvContains(t, got, "WINEDLLOVERRIDES=dxgi=n")
}

func TestBuildProtonEnv_SetsSTEAM_COMPAT_DATA_PATH(t *testing.T) {
	got := buildProtonEnvFrom([]string{"HOME=/home/user"}, "/home/user/.cluckers/compatdata", false)
	assertEnvContains(t, got, "STEAM_COMPAT_DATA_PATH=/home/user/.cluckers/compatdata")
}

func TestBuildProtonEnv_SetsSteamGameId(t *testing.T) {
	got := buildProtonEnvFrom([]string{"HOME=/home/user"}, "/compat", false)
	assertEnvContains(t, got, "SteamGameId=0")
}

func TestBuildProtonEnv_SetsSteamAppId(t *testing.T) {
	got := buildProtonEnvFrom([]string{"HOME=/home/user"}, "/compat", false)
	assertEnvContains(t, got, "SteamAppId=0")
}

func TestBuildProtonEnv_SetsSTEAM_COMPAT_CLIENT_INSTALL_PATH_Empty(t *testing.T) {
	got := buildProtonEnvFrom([]string{"HOME=/home/user"}, "/compat", false)
	assertEnvContains(t, got, "STEAM_COMPAT_CLIENT_INSTALL_PATH=")
}

func TestBuildProtonEnv_SetsWINEDLLOVERRIDES(t *testing.T) {
	got := buildProtonEnvFrom([]string{"HOME=/home/user"}, "/compat", false)
	assertEnvContains(t, got, "WINEDLLOVERRIDES=dxgi=n")
}

func TestBuildProtonEnv_VerboseTrue_SetsPROTON_LOG(t *testing.T) {
	got := buildProtonEnvFrom([]string{"HOME=/home/user"}, "/compat", true)
	assertEnvContains(t, got, "PROTON_LOG=1")
}

func TestBuildProtonEnv_VerboseFalse_NoPROTON_LOG(t *testing.T) {
	got := buildProtonEnvFrom([]string{"HOME=/home/user"}, "/compat", false)
	assertEnvNotContainsKey(t, got, "PROTON_LOG")
}

func TestBuildProtonEnv_PassesThroughUnrelatedVars(t *testing.T) {
	base := []string{"HOME=/home/user", "PATH=/usr/bin", "USER=testuser"}
	got := buildProtonEnvFrom(base, "/compat", false)
	assertEnvContains(t, got, "HOME=/home/user")
	assertEnvContains(t, got, "PATH=/usr/bin")
	assertEnvContains(t, got, "USER=testuser")
}

// --- buildProtonCommand tests ---

func TestBuildProtonCommand_WithSHM(t *testing.T) {
	program, args := buildProtonCommand(
		"/opt/GE-Proton10-1/proton",
		"/tmp/shm_launcher.exe",
		"/tmp/bootstrap.bin",
		`Local\realm_content_bootstrap_1234`,
		"/home/user/.cluckers/game/Realm-Royale/Binaries/Win64/ShippingPC-RealmGameNoEditor.exe",
		[]string{"-user=foo", "-token=bar", "-hostx=1.2.3.4"},
	)

	if program != "python3" {
		t.Errorf("program = %q, want %q", program, "python3")
	}

	wantArgs := []string{
		"/opt/GE-Proton10-1/proton",
		"run",
		"/tmp/shm_launcher.exe",
		`Z:\tmp\bootstrap.bin`,
		`Local\realm_content_bootstrap_1234`,
		`Z:\home\user\.cluckers\game\Realm-Royale\Binaries\Win64\ShippingPC-RealmGameNoEditor.exe`,
		"-user=foo",
		"-token=bar",
		"-hostx=1.2.3.4",
	}

	if len(args) != len(wantArgs) {
		t.Fatalf("args length = %d, want %d\nargs: %v\nwant: %v", len(args), len(wantArgs), args, wantArgs)
	}

	for i, got := range args {
		if got != wantArgs[i] {
			t.Errorf("args[%d] = %q, want %q", i, got, wantArgs[i])
		}
	}
}

func TestBuildProtonCommand_WithoutSHM(t *testing.T) {
	program, args := buildProtonCommand(
		"/opt/GE-Proton10-1/proton",
		"", "", "",
		"/home/user/.cluckers/game/Realm-Royale/Binaries/Win64/ShippingPC-RealmGameNoEditor.exe",
		[]string{"-user=foo", "-token=bar"},
	)

	if program != "python3" {
		t.Errorf("program = %q, want %q", program, "python3")
	}

	wantArgs := []string{
		"/opt/GE-Proton10-1/proton",
		"run",
		"/home/user/.cluckers/game/Realm-Royale/Binaries/Win64/ShippingPC-RealmGameNoEditor.exe",
		"-user=foo",
		"-token=bar",
	}

	if len(args) != len(wantArgs) {
		t.Fatalf("args length = %d, want %d\nargs: %v\nwant: %v", len(args), len(wantArgs), args, wantArgs)
	}

	for i, got := range args {
		if got != wantArgs[i] {
			t.Errorf("args[%d] = %q, want %q", i, got, wantArgs[i])
		}
	}
}

// --- protonErrorSuggestion tests ---

func TestProtonErrorSuggestion_ContainsDeleteCompatdata(t *testing.T) {
	got := protonErrorSuggestion("/home/user/.cluckers/compatdata")
	if !strings.Contains(got, "Delete /home/user/.cluckers/compatdata/") {
		t.Errorf("suggestion missing compatdata delete instruction:\n%s", got)
	}
}

func TestProtonErrorSuggestion_ContainsUpdateProtonGE(t *testing.T) {
	got := protonErrorSuggestion("/home/user/.cluckers/compatdata")
	if !strings.Contains(got, "Update Proton-GE") {
		t.Errorf("suggestion missing Proton-GE update instruction:\n%s", got)
	}
}

func TestProtonErrorSuggestion_ContainsCluckersUpdate(t *testing.T) {
	got := protonErrorSuggestion("/home/user/.cluckers/compatdata")
	if !strings.Contains(got, "cluckers update") {
		t.Errorf("suggestion missing cluckers update instruction:\n%s", got)
	}
}

// --- shmBridgeError tests ---

func TestShmBridgeError_DetectsCreateFileMapping(t *testing.T) {
	err := shmBridgeError(errors.New("exit status 1"), "CreateFileMapping failed", "/compat")
	if err == nil {
		t.Fatal("expected non-nil UserError for CreateFileMapping stderr")
	}
	if !strings.Contains(err.Message, "Shared memory bridge failed") {
		t.Errorf("Message = %q, want to contain 'Shared memory bridge failed'", err.Message)
	}
}

func TestShmBridgeError_DetectsShmLauncher(t *testing.T) {
	err := shmBridgeError(errors.New("exit status 1"), "shm_launcher error: something went wrong", "/compat")
	if err == nil {
		t.Fatal("expected non-nil UserError for shm_launcher stderr")
	}
	if !strings.Contains(err.Message, "Shared memory bridge failed") {
		t.Errorf("Message = %q, want to contain 'Shared memory bridge failed'", err.Message)
	}
}

func TestShmBridgeError_DetectsOpenFileMapping(t *testing.T) {
	err := shmBridgeError(errors.New("exit status 1"), "OpenFileMapping returned null", "/compat")
	if err == nil {
		t.Fatal("expected non-nil UserError for OpenFileMapping stderr")
	}
}

func TestShmBridgeError_DetectsSharedMemory(t *testing.T) {
	err := shmBridgeError(errors.New("exit status 1"), "shared memory allocation error", "/compat")
	if err == nil {
		t.Fatal("expected non-nil UserError for shared memory stderr")
	}
}

func TestShmBridgeError_NilForNoPatterns(t *testing.T) {
	err := shmBridgeError(errors.New("exit status 1"), "some unrelated wine error", "/compat")
	if err != nil {
		t.Errorf("expected nil for non-shm stderr, got: %v", err)
	}
}

func TestShmBridgeError_NilForNilExitErr(t *testing.T) {
	err := shmBridgeError(nil, "CreateFileMapping failed", "/compat")
	if err != nil {
		t.Errorf("expected nil for nil exitErr, got: %v", err)
	}
}

func TestShmBridgeError_ContainsCompatdataInSuggestion(t *testing.T) {
	ue := shmBridgeError(errors.New("exit status 1"), "CreateFileMapping failed", "/home/user/.cluckers/compatdata")
	if ue == nil {
		t.Fatal("expected non-nil UserError")
	}
	if !strings.Contains(ue.Suggestion, "/home/user/.cluckers/compatdata") {
		t.Errorf("Suggestion should contain compatdata path:\n%s", ue.Suggestion)
	}
}

func TestShmBridgeError_CaseInsensitive(t *testing.T) {
	err := shmBridgeError(errors.New("exit status 1"), "CREATEFILEMAPPING FAILED", "/compat")
	if err == nil {
		t.Fatal("expected non-nil UserError for case-insensitive match")
	}
}

// --- lastNLines tests ---

func TestLastNLines_ReturnsLastN(t *testing.T) {
	got := lastNLines("a\nb\nc\nd\ne", 3)
	want := "c\nd\ne"
	if got != want {
		t.Errorf("lastNLines(5 lines, 3) = %q, want %q", got, want)
	}
}

func TestLastNLines_FewerLinesThanN(t *testing.T) {
	got := lastNLines("ab", 5)
	want := "ab"
	if got != want {
		t.Errorf("lastNLines(1 line, 5) = %q, want %q", got, want)
	}
}

func TestLastNLines_EmptyString(t *testing.T) {
	got := lastNLines("", 3)
	want := ""
	if got != want {
		t.Errorf("lastNLines(empty, 3) = %q, want %q", got, want)
	}
}

// --- test helpers ---

func assertEnvEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("env length = %d, want %d\ngot:  %v\nwant: %v", len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("env[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func assertEnvContains(t *testing.T, env []string, entry string) {
	t.Helper()
	for _, e := range env {
		if e == entry {
			return
		}
	}
	t.Errorf("env missing %q\nenv: %v", entry, env)
}

func assertEnvNotContainsKey(t *testing.T, env []string, key string) {
	t.Helper()
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			t.Errorf("env should not contain %s but found %q", key, e)
		}
	}
}
