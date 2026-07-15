//go:build linux

package tray

import "testing"

func TestDefaultIconUsesPlatformResource(t *testing.T) {
	icon := DefaultIcon()
	if len(icon) == 0 {
		t.Fatal("DefaultIcon() returned no data")
	}
	if IconFileExtension() != ".png" {
		t.Fatalf("IconFileExtension() = %q, want .png on Linux", IconFileExtension())
	}
	if string(icon[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatal("DefaultIcon() is not PNG data on Linux")
	}
}
