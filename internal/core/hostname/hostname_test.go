package hostname

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type normalizationVector struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

type normalizationVectors struct {
	Bare []normalizationVector `json:"bare"`
	URL  []normalizationVector `json:"url"`
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
