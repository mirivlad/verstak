package updates

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v0.2.8", "v0.2.7", true},
		{"v0.2.8", "v0.2.8", false},
		{"v0.2.7", "v0.2.8", false},
		{"v0.10.0", "v0.9.9", true}, // numeric, not lexical
		{"v1.0.0", "v1.0.0-beta.2", true},
		{"v1.0.0-beta.2", "v1.0.0", false},
		{"v0.2.8", "v0.2.8-dirty", false}, // a dirty build of the same release
		{"v0.2.9", "v0.2.8-dirty", true},
		{"v0.2.8", "dev", true}, // a build without a release is always behind
		{"v0.2.8", "", true},
		{"v0.2.8", "v0.2.8-5-g1ff1c5a", false}, // git describe: after v0.2.8
		{"v0.2.9", "v0.2.8-5-g1ff1c5a", true},
	}
	for _, c := range cases {
		got, err := IsNewer(c.latest, c.current)
		if err != nil {
			t.Fatalf("IsNewer(%q, %q): %v", c.latest, c.current, err)
		}
		if got != c.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
	if _, err := IsNewer("latest", "v0.2.8"); err == nil {
		t.Error("an unparseable release tag must be an error, not a silent answer")
	}
}

func serve(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "Verstak/") {
			t.Errorf("request must identify Verstak, got User-Agent %q", r.Header.Get("User-Agent"))
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestCheckReportsANewerRelease(t *testing.T) {
	server := serve(t, 200, `{"tag_name":"v0.2.9","html_url":"https://github.com/mirivlad/verstak/releases/tag/v0.2.9"}`)
	fixed := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	result, err := Checker{Endpoint: server.URL, Now: func() time.Time { return fixed }}.Check(context.Background(), "v0.2.8")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Newer || result.Latest != "v0.2.9" || result.Current != "v0.2.8" {
		t.Fatalf("unexpected result %+v", result)
	}
	if result.CheckedAt != "2026-09-30T12:00:00Z" {
		t.Errorf("CheckedAt = %q", result.CheckedAt)
	}
}

func TestCheckRejectsWhatItCannotTrust(t *testing.T) {
	cases := map[string]struct {
		status int
		body   string
	}{
		"server error":        {500, `oops`},
		"not json":            {200, `<html>`},
		"draft":               {200, `{"tag_name":"v9.0.0","draft":true,"html_url":"https://github.com/x"}`},
		"no tag":              {200, `{"html_url":"https://github.com/x"}`},
		"page off github":     {200, `{"tag_name":"v9.0.0","html_url":"https://evil.example/download"}`},
		"unparseable version": {200, `{"tag_name":"nightly","html_url":"https://github.com/x"}`},
	}
	for name, c := range cases {
		server := serve(t, c.status, c.body)
		if _, err := (Checker{Endpoint: server.URL}).Check(context.Background(), "v0.2.8"); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestCheckDoesNotOfferAPrerelease(t *testing.T) {
	server := serve(t, 200, `{"tag_name":"v0.3.0-beta.1","prerelease":true,"html_url":"https://github.com/mirivlad/verstak/releases/tag/v0.3.0-beta.1"}`)
	result, err := Checker{Endpoint: server.URL}.Check(context.Background(), "v0.2.8")
	if err != nil {
		t.Fatal(err)
	}
	if result.Newer {
		t.Error("a pre-release must not be offered as an update")
	}
}
