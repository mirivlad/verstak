package importservice

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type archiveFixture struct {
	name     string
	body     []byte
	mode     os.FileMode
	typeflag byte
}

func writeZIPFixture(t *testing.T, fixtures []archiveFixture) string {
	t.Helper()
	archivePath := filepath.Join(t.TempDir(), "source.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(file)
	for _, fixture := range fixtures {
		header := &zip.FileHeader{Name: fixture.name, Method: zip.Deflate}
		if fixture.mode != 0 {
			header.SetMode(fixture.mode)
		}
		writer, createErr := zw.CreateHeader(header)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, writeErr := writer.Write(fixture.body); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return archivePath
}

func writeTARFixture(t *testing.T, gzipEnabled bool, fixtures []archiveFixture) string {
	t.Helper()
	ext := ".tar"
	if gzipEnabled {
		ext = ".tar.gz"
	}
	archivePath := filepath.Join(t.TempDir(), "source"+ext)
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	var output io.Writer = file
	var gz *gzip.Writer
	if gzipEnabled {
		gz = gzip.NewWriter(file)
		output = gz
	}
	tw := tar.NewWriter(output)
	for _, fixture := range fixtures {
		typeflag := fixture.typeflag
		if typeflag == 0 {
			typeflag = tar.TypeReg
		}
		header := &tar.Header{Name: fixture.name, Mode: 0o600, Size: int64(len(fixture.body)), Typeflag: typeflag}
		if typeflag == tar.TypeDir {
			header.Size = 0
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if header.Size > 0 {
			if _, err := tw.Write(fixture.body); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if gz != nil {
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return archivePath
}

func TestArchiveFormatsIndexAndReadRegularFiles(t *testing.T) {
	fixtures := []archiveFixture{
		{name: "pages/", typeflag: tar.TypeDir, mode: os.ModeDir | 0o755},
		{name: "pages/start.txt", body: []byte("====== Start ======")},
		{name: "media/logo.png", body: []byte{0x89, 'P', 'N', 'G'}},
	}
	paths := []string{
		writeZIPFixture(t, fixtures),
		writeTARFixture(t, false, fixtures),
		writeTARFixture(t, true, fixtures),
	}
	for _, archivePath := range paths {
		t.Run(filepath.Ext(archivePath), func(t *testing.T) {
			service := New(t.TempDir(), Options{})
			session, err := service.OpenArchive("verstak.import", archivePath)
			if err != nil {
				t.Fatal(err)
			}
			if session.EntryCount != 3 {
				t.Fatalf("entryCount=%d", session.EntryCount)
			}
			page, err := service.ListEntries("verstak.import", session.SourceHandle, "")
			if err != nil {
				t.Fatal(err)
			}
			var pageID string
			for _, entry := range page.Entries {
				if entry.Path == "pages/start.txt" {
					pageID = entry.ID
				}
			}
			text, err := service.ReadText("verstak.import", session.SourceHandle, pageID)
			if err != nil {
				t.Fatal(err)
			}
			if text != "====== Start ======" {
				t.Fatalf("text=%q", text)
			}
		})
	}
}

func TestArchiveRejectsUnsafeEntries(t *testing.T) {
	for _, name := range []string{"../escape.txt", "/absolute.txt", "C:/drive.txt", "safe/../../escape.txt", "NUL", `safe\file.txt`} {
		t.Run(name, func(t *testing.T) {
			archivePath := writeZIPFixture(t, []archiveFixture{{name: name, body: []byte("x")}})
			_, err := New(t.TempDir(), Options{}).OpenArchive("verstak.import", archivePath)
			if err == nil || !strings.Contains(err.Error(), "unsafe-source-path") {
				t.Fatalf("expected unsafe-source-path, got %v", err)
			}
		})
	}
}

func TestArchiveRejectsLinksAndCaseFoldDuplicates(t *testing.T) {
	linkArchive := writeZIPFixture(t, []archiveFixture{{name: "linked.txt", body: []byte("target"), mode: os.ModeSymlink | 0o777}})
	if _, err := New(t.TempDir(), Options{}).OpenArchive("verstak.import", linkArchive); err == nil || !strings.Contains(err.Error(), "unsupported-source-entry") {
		t.Fatalf("expected unsupported-source-entry, got %v", err)
	}

	duplicateArchive := writeZIPFixture(t, []archiveFixture{
		{name: "Notes/Readme.md", body: []byte("one")},
		{name: "notes/readme.md", body: []byte("two")},
	})
	if _, err := New(t.TempDir(), Options{}).OpenArchive("verstak.import", duplicateArchive); err == nil || !strings.Contains(err.Error(), "source-path-collision") {
		t.Fatalf("expected source-path-collision, got %v", err)
	} else if strings.Contains(strings.ToLower(err.Error()), "readme") {
		t.Fatalf("collision error leaked a source filename: %v", err)
	}
}

func TestZIPExpansionRatioUsesExactBoundary(t *testing.T) {
	body := []byte(strings.Repeat("compressible-data-", 2_000))
	archivePath := writeZIPFixture(t, []archiveFixture{{name: "large.txt", body: body}})
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	compressed := int64(reader.File[0].CompressedSize64)
	uncompressed := int64(reader.File[0].UncompressedSize64)
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	ratioFloor := uncompressed / compressed
	if uncompressed == compressed*ratioFloor {
		t.Fatal("fixture unexpectedly has an integral ratio")
	}
	_, err = New(t.TempDir(), Options{ExpansionRatio: ratioFloor}).OpenArchive("verstak.import", archivePath)
	if err == nil || !strings.Contains(err.Error(), "source-limit-exceeded") {
		t.Fatalf("expected exact ratio rejection, got %v", err)
	}
}

func TestTARRootEntryIsIgnored(t *testing.T) {
	archivePath := writeTARFixture(t, true, []archiveFixture{
		{name: "./", typeflag: tar.TypeDir},
		{name: "./pages/start.txt", body: []byte("start")},
	})
	session, err := New(t.TempDir(), Options{}).OpenArchive("verstak.import", archivePath)
	if err != nil {
		t.Fatal(err)
	}
	if session.EntryCount != 1 {
		t.Fatalf("entryCount=%d", session.EntryCount)
	}
}

func TestTARRejectsLinks(t *testing.T) {
	archivePath := writeTARFixture(t, false, []archiveFixture{{name: "linked.txt", typeflag: tar.TypeSymlink}})
	_, err := New(t.TempDir(), Options{}).OpenArchive("verstak.import", archivePath)
	if err == nil || !strings.Contains(err.Error(), "unsupported-source-entry") {
		t.Fatalf("expected unsupported-source-entry, got %v", err)
	}
}

func TestConfiguredEntryAndTotalLimits(t *testing.T) {
	archivePath := writeZIPFixture(t, []archiveFixture{
		{name: "one.txt", body: []byte("123")},
		{name: "two.txt", body: []byte("456")},
	})
	for name, options := range map[string]Options{
		"entries": {MaxEntries: 1},
		"single":  {MaxEntryBytes: 2},
		"total":   {MaxTotalBytes: 5},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := New(t.TempDir(), options).OpenArchive("verstak.import", archivePath)
			if err == nil || !strings.Contains(err.Error(), "source-limit-exceeded") {
				t.Fatalf("expected source-limit-exceeded, got %v", err)
			}
		})
	}
}
