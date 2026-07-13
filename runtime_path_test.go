package main

import "testing"

func TestBundledWebView2PathUsesExecutableDirectory(t *testing.T) {
	got := bundledWebView2Path("/opt/verstak/verstak-desktop")
	want := "/opt/verstak/webview2"
	if got != want {
		t.Fatalf("bundledWebView2Path() = %q, want %q", got, want)
	}
}
