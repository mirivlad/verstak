//go:build linux

package tray

import (
	"bytes"
	"image"
	"image/color"
	_ "image/png"
	"testing"
)

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
	assertVerstakBrand(t, icon)
}

func assertVerstakBrand(t *testing.T, icon []byte) {
	t.Helper()
	image, _, err := image.Decode(bytes.NewReader(icon))
	if err != nil {
		t.Fatalf("decode tray icon: %v", err)
	}

	want := color.NRGBA{R: 0x1c, G: 0x2f, B: 0x4a, A: 0xff}
	for y := image.Bounds().Min.Y; y < image.Bounds().Max.Y; y++ {
		for x := image.Bounds().Min.X; x < image.Bounds().Max.X; x++ {
			if color.NRGBAModel.Convert(image.At(x, y)) == want {
				return
			}
		}
	}
	t.Fatal("tray icon does not contain the Verstak navy brand color")
}
