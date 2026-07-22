package importservice

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type backupAdapterInput struct {
	Session SourceSession     `json:"session"`
	Entries []Entry           `json:"entries"`
	Texts   map[string]string `json:"texts"`
}

type backupAdapterAggregate struct {
	Format          string `json:"format"`
	Nodes           int    `json:"nodes"`
	Notes           int    `json:"notes"`
	Assets          int    `json:"assets"`
	Warnings        int    `json:"warnings"`
	PlanNodes       int    `json:"planNodes"`
	ValidationCount int    `json:"validationCount"`
}

type backupAdapterOutput struct {
	Aggregate backupAdapterAggregate `json:"aggregate"`
	Plan      Plan                   `json:"plan"`
}

func TestSuppliedBackups(t *testing.T) {
	if os.Getenv("VERSTAK_IMPORT_BACKUP_SMOKE") != "1" {
		t.Skip("set VERSTAK_IMPORT_BACKUP_SMOKE=1 to check supplied backup archives")
	}
	root := desktopRepositoryRoot(t)
	pluginRoot := firstExistingPath(t,
		os.Getenv("VERSTAK_OFFICIAL_PLUGINS_DIR"),
		filepath.Join(root, "..", "verstak-official-plugins"),
	)
	helper := filepath.Join(root, "scripts", "smoke-import-adapters.mjs")
	vault := t.TempDir()
	if err := os.MkdirAll(filepath.Join(vault, ".verstak", "workspaces"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vault, ".verstak", "vault.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		format        string
		archive       string
		wantPages     int
		wantAssets    int
		wantService   int
		wantGraphNote int
		wantGraph     int
		textSelected  func(Entry) bool
	}{
		{
			format: "dokuwiki", archive: firstExistingPath(t, os.Getenv("VERSTAK_IMPORT_WIKI_ARCHIVE"), filepath.Join(root, "..", "wiki.tar.gz"), filepath.Join(root, "..", "..", "..", "wiki.tar.gz")),
			wantPages: 126, wantAssets: 27, wantGraphNote: 123, wantGraph: 150,
			textSelected: func(entry Entry) bool {
				path := "/" + entry.Path
				return entry.Kind == "file" && strings.Contains(path, "/data/pages/") && strings.HasSuffix(strings.ToLower(path), ".txt")
			},
		},
		{
			format: "obsidian", archive: firstExistingPath(t, os.Getenv("VERSTAK_IMPORT_OBSIDIAN_ARCHIVE"), filepath.Join(root, "..", "Obsidian.tar.gz"), filepath.Join(root, "..", "..", "..", "Obsidian.tar.gz")),
			wantPages: 147, wantAssets: 36, wantService: 79, wantGraphNote: 147, wantGraph: 183,
			textSelected: func(entry Entry) bool {
				return entry.Kind == "file" && strings.HasSuffix(strings.ToLower(entry.Path), ".md") && !pathHasSegment(entry.Path, ".obsidian")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.format, func(t *testing.T) {
			service := New(vault, Options{})
			session, err := service.OpenArchive("verstak.import", test.archive)
			if err != nil {
				t.Fatalf("supplied archive rejected: %v", err)
			}
			entries := listAllBackupEntries(t, service, session)
			pages, assets, serviceEntries := backupInventory(test.format, entries)
			if pages != test.wantPages || assets != test.wantAssets || serviceEntries != test.wantService {
				t.Fatalf("aggregate inventory mismatch: pages=%d assets=%d service=%d", pages, assets, serviceEntries)
			}
			texts := make(map[string]string, pages)
			for _, entry := range entries {
				if !test.textSelected(entry) {
					continue
				}
				text, readErr := service.ReadText("verstak.import", session.SourceHandle, entry.ID)
				if readErr != nil {
					t.Fatalf("selected text entry rejected: %v", readErr)
				}
				texts[entry.ID] = text
			}

			output := runBackupAdapter(t, helper, pluginRoot, test.format, backupAdapterInput{Session: session, Entries: entries, Texts: texts})
			if output.Aggregate.Format != test.format || output.Aggregate.Nodes != test.wantGraph || output.Aggregate.Notes != test.wantGraphNote || output.Aggregate.Assets != test.wantAssets || output.Aggregate.ValidationCount != 0 {
				t.Fatalf("adapter aggregate mismatch: format=%s nodes=%d notes=%d assets=%d validation=%d", output.Aggregate.Format, output.Aggregate.Nodes, output.Aggregate.Notes, output.Aggregate.Assets, output.Aggregate.ValidationCount)
			}
			if output.Aggregate.PlanNodes < output.Aggregate.Nodes {
				t.Fatalf("adapter plan lost content: graph=%d plan=%d", output.Aggregate.Nodes, output.Aggregate.PlanNodes)
			}
			first, applyErr := service.ApplyPlan(context.Background(), "verstak.import", session.SourceHandle, output.Plan)
			if applyErr != nil {
				t.Fatalf("reviewed adapter plan rejected: %v", applyErr)
			}
			if first.Notes+first.Files != test.wantGraph {
				t.Fatalf("published content mismatch: notes=%d files=%d", first.Notes, first.Files)
			}
			second, applyErr := service.ApplyPlan(context.Background(), "verstak.import", session.SourceHandle, output.Plan)
			if applyErr != nil {
				t.Fatalf("repeated reviewed plan rejected: %v", applyErr)
			}
			if second.RunPath == first.RunPath || !strings.HasSuffix(second.RunPath, " (2)") {
				t.Fatalf("repeated import was not isolated")
			}
		})
	}
}

func desktopRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve repository root")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}

func firstExistingPath(t *testing.T, candidates ...string) string {
	t.Helper()
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			absolute, absErr := filepath.Abs(candidate)
			if absErr != nil {
				t.Fatal(absErr)
			}
			return absolute
		}
	}
	t.Fatal("required local smoke input is unavailable")
	return ""
}

func listAllBackupEntries(t *testing.T, service *Service, session SourceSession) []Entry {
	t.Helper()
	var entries []Entry
	cursor := ""
	for {
		page, err := service.ListEntries("verstak.import", session.SourceHandle, cursor)
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, page.Entries...)
		if page.NextCursor == "" {
			return entries
		}
		cursor = page.NextCursor
	}
}

func backupInventory(format string, entries []Entry) (pages, assets, serviceEntries int) {
	if format == "dokuwiki" {
		for _, entry := range entries {
			path := "/" + entry.Path
			if entry.Kind == "file" && strings.Contains(path, "/data/pages/") && strings.HasSuffix(strings.ToLower(path), ".txt") {
				pages++
			}
			if entry.Kind == "file" && strings.Contains(path, "/data/media/") {
				assets++
			}
		}
		return pages, assets, 0
	}
	for _, entry := range entries {
		if pathHasSegment(entry.Path, ".obsidian") {
			serviceEntries++
			continue
		}
		if entry.Kind != "file" {
			continue
		}
		if strings.HasSuffix(strings.ToLower(entry.Path), ".md") {
			pages++
		} else {
			assets++
		}
	}
	return pages, assets, serviceEntries
}

func pathHasSegment(value, segment string) bool {
	for _, part := range strings.Split(filepath.ToSlash(value), "/") {
		if part == segment {
			return true
		}
	}
	return false
}

func runBackupAdapter(t *testing.T, helper, pluginRoot, format string, input backupAdapterInput) backupAdapterOutput {
	t.Helper()
	directory := t.TempDir()
	inputPath := filepath.Join(directory, "input.json")
	outputPath := filepath.Join(directory, "output.json")
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	command := exec.Command("node", helper, inputPath, outputPath, pluginRoot, format)
	command.Stdout = &stdout
	command.Stderr = io.Discard
	if err := command.Run(); err != nil {
		t.Fatalf("adapter subprocess failed")
	}
	var aggregate backupAdapterAggregate
	if err := json.Unmarshal(stdout.Bytes(), &aggregate); err != nil {
		t.Fatalf("adapter subprocess returned an invalid aggregate")
	}
	resultData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	var output backupAdapterOutput
	if err := json.Unmarshal(resultData, &output); err != nil {
		t.Fatalf("adapter subprocess returned an invalid plan")
	}
	if output.Aggregate != aggregate {
		t.Fatalf("adapter aggregate channels disagree")
	}
	return output
}
