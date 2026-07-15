//go:build windows

package tray

import (
	"encoding/binary"
	"testing"
)

func TestDefaultIconIsMultiResolutionICO(t *testing.T) {
	icon := DefaultIcon()
	if IconFileExtension() != ".ico" || len(icon) < 6 {
		t.Fatalf("Windows tray icon is not ICO data")
	}
	if binary.LittleEndian.Uint16(icon[0:2]) != 0 || binary.LittleEndian.Uint16(icon[2:4]) != 1 {
		t.Fatal("Windows tray icon has an invalid ICO header")
	}
	count := int(binary.LittleEndian.Uint16(icon[4:6]))
	if count < 6 {
		t.Fatalf("ICO image count = %d, want at least 6", count)
	}
	want := map[int]bool{16: false, 20: false, 24: false, 32: false, 48: false, 256: false}
	for offset := 6; offset+16 <= len(icon) && offset < 6+count*16; offset += 16 {
		size := int(icon[offset])
		if size == 0 {
			size = 256
		}
		if _, ok := want[size]; ok {
			want[size] = true
		}
	}
	for size, found := range want {
		if !found {
			t.Errorf("ICO is missing %dx%d image", size, size)
		}
	}
}
