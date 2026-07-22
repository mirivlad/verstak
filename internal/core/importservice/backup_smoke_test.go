package importservice

import (
	"os"
	"testing"
)

func TestLocalBackupSourcesStayWithinSafetyBoundary(t *testing.T) {
	if os.Getenv("VERSTAK_IMPORT_BACKUP_SMOKE") != "1" {
		t.Skip("set VERSTAK_IMPORT_BACKUP_SMOKE=1 to check local backup archives")
	}
	archives := []struct {
		label string
		env   string
	}{
		{label: "DokuWiki", env: "VERSTAK_IMPORT_WIKI_ARCHIVE"},
		{label: "Obsidian", env: "VERSTAK_IMPORT_OBSIDIAN_ARCHIVE"},
	}
	for _, archive := range archives {
		t.Run(archive.label, func(t *testing.T) {
			archivePath := os.Getenv(archive.env)
			if archivePath == "" {
				t.Fatalf("%s is required", archive.env)
			}
			session, err := New(t.TempDir(), Options{}).OpenArchive("verstak.import", archivePath)
			if err != nil {
				t.Fatalf("archive rejected: %v", err)
			}
			if session.EntryCount == 0 || session.TotalBytes == 0 {
				t.Fatalf("empty aggregate inventory: entries=%d bytes=%d", session.EntryCount, session.TotalBytes)
			}
		})
	}
}
