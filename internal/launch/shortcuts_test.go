//go:build linux

package launch

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestWriteShortcutEntry_FieldOrder(t *testing.T) {
	var buf bytes.Buffer
	s := &Shortcut{
		AppName:       "Test Game",
		Exe:           `"/path/to/game"`,
		StartDir:      `"/path/to"`,
		LaunchOptions: "--option1",
		Icon:          "/path/to/icon.png",
	}
	writeShortcutEntry(&buf, 0, s)
	data := buf.Bytes()

	// Verify fields appear in correct order by finding their byte offsets.
	appidOff := bytes.Index(data, []byte("\x02appid\x00"))
	appNameOff := bytes.Index(data, []byte("\x01AppName\x00"))
	exeOff := bytes.Index(data, []byte("\x01Exe\x00"))
	startDirOff := bytes.Index(data, []byte("\x01StartDir\x00"))
	iconOff := bytes.Index(data, []byte("\x01icon\x00"))
	launchOptsOff := bytes.Index(data, []byte("\x01LaunchOptions\x00"))

	if appidOff < 0 {
		t.Fatal("appid field not found")
	}
	if appNameOff < 0 {
		t.Fatal("AppName field not found")
	}
	if exeOff < 0 {
		t.Fatal("Exe field not found")
	}
	if startDirOff < 0 {
		t.Fatal("StartDir field not found")
	}
	if iconOff < 0 {
		t.Fatal("icon field not found")
	}
	if launchOptsOff < 0 {
		t.Fatal("LaunchOptions field not found")
	}

	// appid < AppName < Exe < StartDir < icon < LaunchOptions
	if appidOff >= appNameOff {
		t.Errorf("appid (%d) should come before AppName (%d)", appidOff, appNameOff)
	}
	if appNameOff >= exeOff {
		t.Errorf("AppName (%d) should come before Exe (%d)", appNameOff, exeOff)
	}
	if exeOff >= startDirOff {
		t.Errorf("Exe (%d) should come before StartDir (%d)", exeOff, startDirOff)
	}
	if startDirOff >= iconOff {
		t.Errorf("StartDir (%d) should come before icon (%d)", startDirOff, iconOff)
	}
	if iconOff >= launchOptsOff {
		t.Errorf("icon (%d) should come before LaunchOptions (%d)", iconOff, launchOptsOff)
	}
}

func TestWriteShortcutEntry_StringFields(t *testing.T) {
	var buf bytes.Buffer
	s := &Shortcut{
		AppName:       "My Game",
		Exe:           `"/usr/bin/game"`,
		StartDir:      `"/usr/bin"`,
		LaunchOptions: "--fullscreen",
		Icon:          "",
	}
	writeShortcutEntry(&buf, 0, s)
	data := buf.Bytes()

	// Check that string fields have \x01 type prefix and null-terminated values.
	tests := []struct {
		fieldName string
		value     string
	}{
		{"AppName", "My Game"},
		{"Exe", `"/usr/bin/game"`},
		{"StartDir", `"/usr/bin"`},
		{"LaunchOptions", "--fullscreen"},
	}

	for _, tc := range tests {
		prefix := append([]byte{0x01}, append([]byte(tc.fieldName), 0x00)...)
		idx := bytes.Index(data, prefix)
		if idx < 0 {
			t.Errorf("string field %s with \\x01 prefix not found", tc.fieldName)
			continue
		}
		// Value should start right after the prefix.
		valStart := idx + len(prefix)
		valEnd := bytes.IndexByte(data[valStart:], 0x00)
		if valEnd < 0 {
			t.Errorf("string field %s value not null-terminated", tc.fieldName)
			continue
		}
		got := string(data[valStart : valStart+valEnd])
		if got != tc.value {
			t.Errorf("field %s = %q, want %q", tc.fieldName, got, tc.value)
		}
	}
}

func TestWriteShortcutEntry_Int32Fields(t *testing.T) {
	var buf bytes.Buffer
	s := &Shortcut{
		AppName: "Test",
		Exe:     `"/bin/test"`,
	}
	writeShortcutEntry(&buf, 0, s)
	data := buf.Bytes()

	// Check int32 fields: type \x02, key, null, then 4 bytes LE.
	int32Fields := []struct {
		key  string
		want uint32
	}{
		{"appid", 0},
		{"IsHidden", 0},
		{"AllowDesktopConfig", 1},
		{"AllowOverlay", 1},
		{"OpenVR", 0},
		{"LastPlayTime", 0},
	}

	for _, tc := range int32Fields {
		prefix := append([]byte{0x02}, append([]byte(tc.key), 0x00)...)
		idx := bytes.Index(data, prefix)
		if idx < 0 {
			t.Errorf("int32 field %s with \\x02 prefix not found", tc.key)
			continue
		}
		valStart := idx + len(prefix)
		if valStart+4 > len(data) {
			t.Errorf("int32 field %s: not enough bytes for value", tc.key)
			continue
		}
		got := binary.LittleEndian.Uint32(data[valStart : valStart+4])
		if got != tc.want {
			t.Errorf("int32 field %s = %d, want %d", tc.key, got, tc.want)
		}
	}
}

func TestAddShortcutToVDF_EmptyFile(t *testing.T) {
	s := &Shortcut{
		AppName:       "Test Game",
		Exe:           `"/bin/game"`,
		StartDir:      `"/bin"`,
		LaunchOptions: "",
	}

	data, err := AddShortcutToVDF(nil, s)
	if err != nil {
		t.Fatalf("AddShortcutToVDF(nil) error: %v", err)
	}

	// Should start with \x00shortcuts\x00
	header := append([]byte{0x00}, append([]byte("shortcuts"), 0x00)...)
	if !bytes.HasPrefix(data, header) {
		t.Errorf("output should start with \\x00shortcuts\\x00, got first %d bytes: %x", min(len(data), 20), data[:min(len(data), 20)])
	}

	// Should contain entry at index "0"
	entryHeader := []byte("\x000\x00")
	if !bytes.Contains(data, entryHeader) {
		t.Error("output should contain entry at index 0")
	}

	// Should end with \x08\x08
	if len(data) < 2 || data[len(data)-2] != 0x08 || data[len(data)-1] != 0x08 {
		t.Errorf("output should end with \\x08\\x08, got last 2 bytes: %x", data[len(data)-2:])
	}
}

func TestAddShortcutToVDF_AppendToExisting(t *testing.T) {
	// First, create a VDF with one entry.
	s1 := &Shortcut{
		AppName: "First Game",
		Exe:     `"/bin/first"`,
	}
	data1, err := AddShortcutToVDF(nil, s1)
	if err != nil {
		t.Fatalf("first AddShortcutToVDF error: %v", err)
	}

	// Now append a second entry.
	s2 := &Shortcut{
		AppName: "Second Game",
		Exe:     `"/bin/second"`,
	}
	data2, err := AddShortcutToVDF(data1, s2)
	if err != nil {
		t.Fatalf("second AddShortcutToVDF error: %v", err)
	}

	// Both entries should be present.
	if !bytes.Contains(data2, []byte("First Game")) {
		t.Error("output should contain first entry AppName")
	}
	if !bytes.Contains(data2, []byte("Second Game")) {
		t.Error("output should contain second entry AppName")
	}

	// Second entry should have index "1".
	entryHeader1 := []byte("\x001\x00")
	if !bytes.Contains(data2, entryHeader1) {
		t.Error("output should contain entry at index 1")
	}

	// Should still end with \x08\x08
	if len(data2) < 2 || data2[len(data2)-2] != 0x08 || data2[len(data2)-1] != 0x08 {
		t.Errorf("output should end with \\x08\\x08, got last 2 bytes: %x", data2[len(data2)-2:])
	}
}

func TestAddShortcutToVDF_RoundtripWithFindCluckersAppID(t *testing.T) {
	s := &Shortcut{
		AppName:       "Realm Royale (Cluckers)",
		Exe:           `"/home/deck/.local/bin/cluckers"`,
		StartDir:      `"/home/deck/.local/bin"`,
		LaunchOptions: "prep && %command%",
	}

	data, err := AddShortcutToVDF(nil, s)
	if err != nil {
		t.Fatalf("AddShortcutToVDF error: %v", err)
	}

	// findCluckersAppID should find our entry (appid=0 since Steam assigns later).
	appID := findCluckersAppID(data)
	if appID != 0 {
		t.Errorf("findCluckersAppID = %d, want 0 (Steam assigns on restart)", appID)
	}

	// Verify the exe field is readable by the existing reader by checking
	// that the data contains the expected exe string pattern.
	if !bytes.Contains(data, []byte("cluckers")) {
		t.Error("VDF data should contain 'cluckers' in exe field")
	}
}

func TestCalculateBPID(t *testing.T) {
	tests := []struct {
		appID uint32
		want  uint64
	}{
		{3928144816, uint64(3928144816)<<32 | 0x02000000},
		{0, 0x02000000},
		{1, uint64(1)<<32 | 0x02000000},
	}

	for _, tc := range tests {
		got := CalculateBPID(tc.appID)
		if got != tc.want {
			t.Errorf("CalculateBPID(%d) = %d, want %d", tc.appID, got, tc.want)
		}
	}
}
