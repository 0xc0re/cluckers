//go:build linux

package launch

import (
	"encoding/binary"
	"testing"
)

// buildVDFShortcut constructs a minimal binary VDF shortcut entry with the
// given appID (uint32) and exe path string. The format mirrors Steam's
// shortcuts.vdf binary encoding: \x02appid\x00 + 4-byte LE uint32 + \x01exe\x00 + path\x00.
func buildVDFShortcut(appID uint32, exePath string) []byte {
	var data []byte
	// appid field: type \x02 (int32), key "appid\x00", 4-byte LE value.
	data = append(data, 0x02)
	data = append(data, []byte("appid\x00")...)
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, appID)
	data = append(data, buf...)
	// exe field: type \x01 (string), key "exe\x00", value + \x00.
	data = append(data, 0x01)
	data = append(data, []byte("exe\x00")...)
	data = append(data, []byte(exePath)...)
	data = append(data, 0x00)
	return data
}

func TestFindCluckersAppID_Found(t *testing.T) {
	var appID uint32 = 3928144816
	data := buildVDFShortcut(appID, "/home/deck/.cluckers/cluckers")

	got := FindCluckersAppID(data)
	if got != appID {
		t.Errorf("FindCluckersAppID() = %d, want %d", got, appID)
	}
}

func TestFindCluckersAppID_NotFound(t *testing.T) {
	data := buildVDFShortcut(12345, "/home/deck/othergame/launch.sh")

	got := FindCluckersAppID(data)
	if got != 0 {
		t.Errorf("FindCluckersAppID() = %d, want 0", got)
	}
}

func TestFindCluckersAppID_EmptyData(t *testing.T) {
	got := FindCluckersAppID([]byte{})
	if got != 0 {
		t.Errorf("FindCluckersAppID(empty) = %d, want 0", got)
	}
}
