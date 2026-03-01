//go:build linux

package launch

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
)

// Shortcut represents a non-Steam game shortcut entry for binary VDF serialization.
type Shortcut struct {
	AppName       string
	Exe           string // Quoted path: `"/path/to/exe"`
	StartDir      string // Quoted path: `"/path/to/dir"`
	LaunchOptions string
	Icon          string
}

// writeStringField writes a binary VDF string field: \x01 + key + \x00 + value + \x00.
func writeStringField(w *bytes.Buffer, key, value string) {
	w.WriteByte(0x01)
	w.WriteString(key)
	w.WriteByte(0x00)
	w.WriteString(value)
	w.WriteByte(0x00)
}

// writeInt32Field writes a binary VDF int32 field: \x02 + key + \x00 + 4 bytes LE.
func writeInt32Field(w *bytes.Buffer, key string, value uint32) {
	w.WriteByte(0x02)
	w.WriteString(key)
	w.WriteByte(0x00)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], value)
	w.Write(buf[:])
}

// writeShortcutEntry serializes a single shortcut entry in binary VDF format.
// Field order matches Steam's expected format exactly.
func writeShortcutEntry(w *bytes.Buffer, index int, s *Shortcut) {
	// Entry header: \x00 + index string + \x00
	w.WriteByte(0x00)
	w.WriteString(strconv.Itoa(index))
	w.WriteByte(0x00)

	// Fields in Steam's expected order.
	writeInt32Field(w, "appid", 0) // Steam assigns on restart
	writeStringField(w, "AppName", s.AppName)
	writeStringField(w, "Exe", s.Exe)
	writeStringField(w, "StartDir", s.StartDir)
	writeStringField(w, "icon", s.Icon)
	writeStringField(w, "ShortcutPath", "") // always empty
	writeStringField(w, "LaunchOptions", s.LaunchOptions)
	writeInt32Field(w, "IsHidden", 0)
	writeInt32Field(w, "AllowDesktopConfig", 1)
	writeInt32Field(w, "AllowOverlay", 1)
	writeInt32Field(w, "OpenVR", 0)
	writeInt32Field(w, "LastPlayTime", 0)

	// Empty tags subsection: \x00tags\x00 + \x08 (end of subsection)
	w.WriteByte(0x00)
	w.WriteString("tags")
	w.WriteByte(0x00)
	w.WriteByte(0x08)

	// End of entry
	w.WriteByte(0x08)
}

// AddShortcutToVDF appends a shortcut entry to existing binary VDF data.
// If existing is nil or empty, creates a new file with the shortcuts header.
// Returns the complete VDF data with the new entry appended.
func AddShortcutToVDF(existing []byte, s *Shortcut) ([]byte, error) {
	var result bytes.Buffer

	if len(existing) == 0 {
		// Create new file: \x00shortcuts\x00
		result.WriteByte(0x00)
		result.WriteString("shortcuts")
		result.WriteByte(0x00)

		writeShortcutEntry(&result, 0, s)

		// File terminators: entry end already written, add file-level terminator.
		result.WriteByte(0x08)
		return result.Bytes(), nil
	}

	// Find the highest existing entry index by scanning for \x00<digits>\x00 patterns
	// after the header.
	highestIndex := -1
	header := []byte("\x00shortcuts\x00")
	headerIdx := bytes.Index(existing, header)
	if headerIdx < 0 {
		return nil, fmt.Errorf("invalid shortcuts.vdf: missing header")
	}

	scanStart := headerIdx + len(header)
	pos := scanStart
	for pos < len(existing)-2 {
		if existing[pos] == 0x00 {
			// Look for \x00<digits>\x00 pattern.
			end := bytes.IndexByte(existing[pos+1:], 0x00)
			if end > 0 && end <= 10 { // reasonable index string length
				candidate := string(existing[pos+1 : pos+1+end])
				if n, err := strconv.Atoi(candidate); err == nil {
					if n > highestIndex {
						highestIndex = n
					}
				}
			}
		}
		pos++
	}

	newIndex := highestIndex + 1

	// Strip the final \x08 (file-level terminator) to insert before it.
	// A valid VDF ends with at least \x08\x08 (entry terminator + file terminator).
	trimmed := existing
	if len(trimmed) > 0 && trimmed[len(trimmed)-1] == 0x08 {
		trimmed = trimmed[:len(trimmed)-1]
	}

	result.Write(trimmed)
	writeShortcutEntry(&result, newIndex, s)
	result.WriteByte(0x08) // file-level terminator

	return result.Bytes(), nil
}

// CalculateBPID calculates the Big Picture ID from a shortcut appID.
// Used for steam://rungameid/<BPID> URLs.
func CalculateBPID(appID uint32) uint64 {
	return (uint64(appID) << 32) | 0x02000000
}
