package hostname

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type normalizationVector struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

type normalizationVectors struct {
	Bare []normalizationVector `json:"bare"`
	URL  []normalizationVector `json:"url"`
	Page []normalizationVector `json:"page"`
}

func TestNormalizeHostnameV1Vectors(t *testing.T) {
	vectors := loadVectors(t)
	for _, vector := range vectors.Bare {
		if got := NormalizeHostnameV1(vector.Input); got != vector.Output {
			t.Errorf("NormalizeHostnameV1(%q) = %q, want %q", vector.Input, got, vector.Output)
		}
	}
}

func TestNormalizeURLHostnameV1Vectors(t *testing.T) {
	vectors := loadVectors(t)
	for _, vector := range vectors.URL {
		if got := NormalizeURLHostnameV1(vector.Input); got != vector.Output {
			t.Errorf("NormalizeURLHostnameV1(%q) = %q, want %q", vector.Input, got, vector.Output)
		}
	}
}

func TestNormalizePageURLV1Vectors(t *testing.T) {
	vectors := loadVectors(t)
	if len(vectors.Page) == 0 {
		t.Fatal("page vectors are missing")
	}
	for _, vector := range vectors.Page {
		if got := NormalizePageURLV1(vector.Input); got != vector.Output {
			t.Errorf("NormalizePageURLV1(%q) = %q, want %q", vector.Input, got, vector.Output)
		}
	}
}

// An address longer than the limit loses its query rather than being cut in the
// middle, because a cut address names a page that does not exist.
func TestNormalizePageURLV1DropsAnOversizedQuery(t *testing.T) {
	long := "https://example.com/report?data=" + strings.Repeat("a", maxPageURLLength)
	if got := NormalizePageURLV1(long); got != "https://example.com/report" {
		t.Errorf("NormalizePageURLV1(oversized) = %q, want %q", got, "https://example.com/report")
	}
	longPath := "https://example.com/" + strings.Repeat("b", maxPageURLLength)
	if got := NormalizePageURLV1(longPath); got != "https://example.com/" {
		t.Errorf("NormalizePageURLV1(oversized path) = %q, want %q", got, "https://example.com/")
	}
}

func loadVectors(t *testing.T) normalizationVectors {
	t.Helper()
	path := filepath.Join("testdata", "hostname-normalization-v1.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var vectors normalizationVectors
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return vectors
}
