package files

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/verstak/verstak-desktop/internal/core/vault"
)

func newTestService(t *testing.T) (*Service, string) {
	t.Helper()
	v := vault.NewVault(nil)
	if err := v.CreateVault(t.TempDir()); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	return NewService(v), v.GetVaultPath()
}

func TestServiceRequiresOpenVault(t *testing.T) {
	v := vault.NewVault(nil)
	s := NewService(v)

	if _, err := s.ListVaultFiles(""); err == nil {
		t.Fatal("ListVaultFiles with closed vault: expected error")
	}
}

func TestListVaultFilesExcludesReservedAndReturnsEntries(t *testing.T) {
	s, root := newTestService(t)
	if err := os.WriteFile(filepath.Join(root, "readme.md"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "Docs"), 0o755); err != nil {
		t.Fatal(err)
	}

	entries, err := s.ListVaultFiles("")
	if err != nil {
		t.Fatalf("ListVaultFiles: %v", err)
	}

	names := map[string]FileEntry{}
	for _, entry := range entries {
		names[entry.Name] = entry
		if strings.HasPrefix(entry.RelativePath, ".verstak") {
			t.Fatalf("reserved entry leaked into list: %+v", entry)
		}
	}
	if names["readme.md"].Type != FileTypeFile {
		t.Fatalf("readme.md type = %q", names["readme.md"].Type)
	}
	if names["Docs"].Type != FileTypeFolder {
		t.Fatalf("Docs type = %q", names["Docs"].Type)
	}
}

func TestPathPolicyRejectsUnsafeOperations(t *testing.T) {
	s, _ := newTestService(t)

	cases := []string{
		"/etc/passwd",
		"C:\\Windows\\system.ini",
		"C:/Windows/system.ini",
		`\\server\share`,
		"//server/share",
		`..\outside`,
		`folder\..\outside`,
		"../outside",
		"folder/../../outside",
		`folder\sub/../../outside`,
		"bad\x00path",
		".verstak",
		".verstak/",
		".verstak/vault.json",
		"./.verstak",
		".Verstak/trash",
	}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			if _, err := s.GetVaultFileMetadata(input); err == nil {
				t.Fatal("metadata: expected error")
			}
			if _, err := s.ReadVaultTextFile(input); err == nil {
				t.Fatal("read: expected error")
			}
			if _, err := s.ReadVaultFileBytes(input); err == nil {
				t.Fatal("read bytes: expected error")
			}
			if err := s.WriteVaultTextFile(input, "x", WriteOptions{CreateIfMissing: true}); err == nil {
				t.Fatal("write: expected error")
			}
			if err := s.MoveVaultPath(input, "safe.txt", MoveOptions{}); err == nil {
				t.Fatal("move: expected error")
			}
			if _, err := s.TrashVaultPath(input); err == nil {
				t.Fatal("trash: expected error")
			}
		})
	}
}

func TestReadVaultTextFileRules(t *testing.T) {
	s, root := newTestService(t)
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "Folder"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "binary.bin"), []byte{0xff, 0xfe, 0xfd}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "huge.txt"), []byte(strings.Repeat("a", int(MaxTextFileBytes)+1)), 0o644); err != nil {
		t.Fatal(err)
	}

	text, err := s.ReadVaultTextFile("note.md")
	if err != nil {
		t.Fatalf("ReadVaultTextFile note: %v", err)
	}
	if text != "hello\n" {
		t.Fatalf("text = %q", text)
	}

	if _, err := s.ReadVaultTextFile("Folder"); err == nil || !strings.Contains(err.Error(), "not-regular-file") {
		t.Fatalf("read folder error = %v, want not-regular-file", err)
	}
	if _, err := s.ReadVaultTextFile("missing.md"); err == nil || !strings.Contains(err.Error(), "not-found") {
		t.Fatalf("read missing error = %v, want not-found", err)
	}
	if _, err := s.ReadVaultTextFile("huge.txt"); err == nil || !strings.Contains(err.Error(), "file-too-large") {
		t.Fatalf("read huge error = %v, want file-too-large", err)
	}
	if _, err := s.ReadVaultTextFile("binary.bin"); err == nil || !strings.Contains(err.Error(), "not-text-file") {
		t.Fatalf("read binary error = %v, want not-text-file", err)
	}
}

func TestReadVaultFileBytesRules(t *testing.T) {
	s, root := newTestService(t)
	imageBytes := []byte{0x89, 0x50, 0x4e, 0x47}
	if err := os.WriteFile(filepath.Join(root, "image.png"), imageBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "Folder"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "huge.bin"), []byte(strings.Repeat("a", int(MaxBinaryReadBytes)+1)), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := s.ReadVaultFileBytes("image.png")
	if err != nil {
		t.Fatalf("ReadVaultFileBytes image: %v", err)
	}
	if result.RelativePath != "image.png" {
		t.Fatalf("relative path = %q, want image.png", result.RelativePath)
	}
	if result.Size != int64(len(imageBytes)) {
		t.Fatalf("size = %d, want %d", result.Size, len(imageBytes))
	}
	if result.MimeHint != "image/png" {
		t.Fatalf("mime hint = %q, want image/png", result.MimeHint)
	}
	if result.DataBase64 != "iVBORw==" {
		t.Fatalf("dataBase64 = %q, want iVBORw==", result.DataBase64)
	}

	if _, err := s.ReadVaultFileBytes("Folder"); err == nil || !strings.Contains(err.Error(), "not-regular-file") {
		t.Fatalf("read folder error = %v, want not-regular-file", err)
	}
	if _, err := s.ReadVaultFileBytes("missing.png"); err == nil || !strings.Contains(err.Error(), "not-found") {
		t.Fatalf("read missing error = %v, want not-found", err)
	}
	if _, err := s.ReadVaultFileBytes("huge.bin"); err == nil || !strings.Contains(err.Error(), "file-too-large") {
		t.Fatalf("read huge error = %v, want file-too-large", err)
	}
}

func TestWriteVaultTextFileAtomicAndConflictBehavior(t *testing.T) {
	s, root := newTestService(t)

	if err := s.WriteVaultTextFile("Notes/one.md", "first", WriteOptions{CreateIfMissing: true}); err == nil {
		t.Fatal("write should fail when parent folder is missing")
	}
	if err := s.CreateVaultFolder("Notes"); err != nil {
		t.Fatalf("CreateVaultFolder: %v", err)
	}
	if err := s.WriteVaultTextFile("Notes/one.md", "first", WriteOptions{CreateIfMissing: true}); err != nil {
		t.Fatalf("write create: %v", err)
	}
	if err := s.WriteVaultTextFile("Notes/one.md", "second", WriteOptions{CreateIfMissing: true}); err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("write conflict error = %v, want conflict", err)
	}
	if err := s.WriteVaultTextFile("Notes/one.md", "second", WriteOptions{CreateIfMissing: true, Overwrite: true}); err != nil {
		t.Fatalf("write overwrite: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "Notes", "one.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "second" {
		t.Fatalf("file content = %q", string(data))
	}

	matches, err := filepath.Glob(filepath.Join(root, "Notes", ".verstak-write-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("atomic write left temp files: %v", matches)
	}

	if err := s.WriteVaultTextFile("", "root", WriteOptions{CreateIfMissing: true}); err == nil || !strings.Contains(err.Error(), "empty path") {
		t.Fatalf("write root error = %v, want empty path", err)
	}
}

func TestWriteVaultFileBytesAtomicAndConflictBehavior(t *testing.T) {
	s, root := newTestService(t)

	if err := s.WriteVaultFileBytes("Images/logo.png", "iVBORw==", WriteOptions{CreateIfMissing: true}); err == nil {
		t.Fatal("write bytes should fail when parent folder is missing")
	}
	if err := s.CreateVaultFolder("Images"); err != nil {
		t.Fatalf("CreateVaultFolder: %v", err)
	}
	if err := s.WriteVaultFileBytes("Images/logo.png", "iVBORw==", WriteOptions{CreateIfMissing: true}); err != nil {
		t.Fatalf("write bytes create: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "Images", "logo.png"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string([]byte{0x89, 0x50, 0x4e, 0x47}) {
		t.Fatalf("file bytes = %v", data)
	}
	if err := s.WriteVaultFileBytes("Images/logo.png", "AQID", WriteOptions{CreateIfMissing: true}); err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("write bytes conflict error = %v, want conflict", err)
	}
	if err := s.WriteVaultFileBytes("Images/logo.png", "AQID", WriteOptions{Overwrite: true}); err != nil {
		t.Fatalf("write bytes overwrite: %v", err)
	}
	data, err = os.ReadFile(filepath.Join(root, "Images", "logo.png"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string([]byte{0x01, 0x02, 0x03}) {
		t.Fatalf("overwritten bytes = %v", data)
	}

	matches, err := filepath.Glob(filepath.Join(root, "Images", ".verstak-write-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("atomic byte write left temp files: %v", matches)
	}
	if err := s.WriteVaultFileBytes("Images/bad.bin", "not-base64!", WriteOptions{CreateIfMissing: true}); err == nil || !strings.Contains(err.Error(), "invalid-base64") {
		t.Fatalf("invalid base64 error = %v, want invalid-base64", err)
	}
	tooLarge := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", int(MaxBinaryReadBytes)+1)))
	if err := s.WriteVaultFileBytes("Images/huge.bin", tooLarge, WriteOptions{CreateIfMissing: true}); err == nil || !strings.Contains(err.Error(), "file-too-large") {
		t.Fatalf("oversized bytes error = %v, want file-too-large", err)
	}
}

func TestCreateVaultFolderConflict(t *testing.T) {
	s, _ := newTestService(t)
	if err := s.CreateVaultFolder("Folder"); err != nil {
		t.Fatalf("CreateVaultFolder first: %v", err)
	}
	if err := s.CreateVaultFolder("Folder"); err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("CreateVaultFolder conflict error = %v, want conflict", err)
	}
}

func TestMoveVaultPathRules(t *testing.T) {
	s, root := newTestService(t)
	if err := os.Mkdir(filepath.Join(root, "A"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "A", "one.txt"), []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "target.txt"), []byte("target"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := s.MoveVaultPath("A/one.txt", "moved.txt", MoveOptions{}); err != nil {
		t.Fatalf("move file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "moved.txt")); err != nil {
		t.Fatalf("moved file missing: %v", err)
	}

	if err := s.MoveVaultPath("A", "B", MoveOptions{}); err != nil {
		t.Fatalf("move folder: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "C"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := s.MoveVaultPath("C", "C/Child", MoveOptions{}); err == nil || !strings.Contains(err.Error(), "move-into-self") {
		t.Fatalf("move into self error = %v, want move-into-self", err)
	}
	if err := s.MoveVaultPath("moved.txt", "target.txt", MoveOptions{}); err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("move conflict error = %v, want conflict", err)
	}
	if err := s.MoveVaultPath("", "root-move", MoveOptions{}); err == nil {
		t.Fatal("move root should fail")
	}
}

func TestCopyVaultPathCopiesBinaryFilesAndFoldersWithoutPartialTargets(t *testing.T) {
	s, root := newTestService(t)
	if err := os.MkdirAll(filepath.Join(root, "Source", "Nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	payload := []byte{0x00, 0xff, 0x10, 0x0a}
	if err := os.WriteFile(filepath.Join(root, "Source", "Nested", "data.bin"), payload, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "Target"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := s.CopyVaultPath("Source", "Target/SourceCopy", CopyOptions{}); err != nil {
		t.Fatal(err)
	}
	copied, err := os.ReadFile(filepath.Join(root, "Target", "SourceCopy", "Nested", "data.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if string(copied) != string(payload) {
		t.Fatalf("copied bytes = %v", copied)
	}
	if err := s.CopyVaultPath("Source", "Target/SourceCopy", CopyOptions{}); err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("copy conflict error = %v", err)
	}
	if err := s.CopyVaultPath("Source", "Missing/SourceCopy", CopyOptions{}); err == nil || !strings.Contains(err.Error(), "parent-not-found") {
		t.Fatalf("missing parent error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "Missing")); !os.IsNotExist(err) {
		t.Fatalf("copy created a missing target parent: %v", err)
	}
}

func TestCopyAndMoveRejectProtectedOrUnsafeDirectoryTrees(t *testing.T) {
	s, root := newTestService(t)
	protected := filepath.Join(root, "Deal")
	if err := os.MkdirAll(filepath.Join(protected, ".verstak"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(protected, ".verstak", "workspace.json"), []byte(`{"workspaceId":"deal-id"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.CopyVaultPath("Deal", "DealCopy", CopyOptions{}); err == nil || !strings.Contains(err.Error(), "protected-root") {
		t.Fatalf("protected copy error = %v", err)
	}
	if err := s.MoveVaultPath("Deal", "DealMoved", MoveOptions{}); err == nil || !strings.Contains(err.Error(), "protected-root") {
		t.Fatalf("protected move error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "DealCopy")); !os.IsNotExist(err) {
		t.Fatalf("protected copy left a target: %v", err)
	}

	if runtime.GOOS != "windows" {
		unsafeDir := filepath.Join(root, "Unsafe")
		if err := os.Mkdir(unsafeDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(root, "Deal"), filepath.Join(unsafeDir, "link")); err != nil {
			t.Fatal(err)
		}
		if err := s.CopyVaultPath("Unsafe", "UnsafeCopy", CopyOptions{}); err == nil || !strings.Contains(err.Error(), "symlink-not-allowed") {
			t.Fatalf("symlink tree copy error = %v", err)
		}
		if _, err := os.Stat(filepath.Join(root, "UnsafeCopy")); !os.IsNotExist(err) {
			t.Fatalf("unsafe copy left a target: %v", err)
		}
	}
}

func TestTrashVaultPathMovesToReservedTrashAndHidesFromList(t *testing.T) {
	s, root := newTestService(t)
	if err := os.WriteFile(filepath.Join(root, "delete-me.txt"), []byte("bye"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "delete-folder"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "same.txt"), []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "Other"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Other", "same.txt"), []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}

	fileResult, err := s.TrashVaultPath("delete-me.txt")
	if err != nil {
		t.Fatalf("trash file: %v", err)
	}
	if fileResult.OriginalPath != "delete-me.txt" || fileResult.TrashID == "" || fileResult.DeletedAt == "" {
		t.Fatalf("unexpected trash result: %+v", fileResult)
	}
	if _, err := os.Stat(filepath.Join(root, fileResult.TrashPath)); err != nil {
		t.Fatalf("trashed file missing: %v", err)
	}
	metaPath := filepath.Join(root, ".verstak", "trash", "files", fileResult.TrashID, "metadata.json")
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("trash metadata missing: %v", err)
	}
	var meta map[string]string
	if err := json.Unmarshal(metaData, &meta); err != nil {
		t.Fatalf("trash metadata invalid JSON: %v", err)
	}
	for _, key := range []string{"originalPath", "deletedAt", "originalType", "trashId", "basename"} {
		if meta[key] == "" {
			t.Fatalf("trash metadata missing %s: %s", key, string(metaData))
		}
	}
	if meta["basename"] != "delete-me.txt" || meta["originalType"] != string(FileTypeFile) {
		t.Fatalf("trash metadata = %+v", meta)
	}

	if _, err := s.TrashVaultPath("delete-folder"); err != nil {
		t.Fatalf("trash folder: %v", err)
	}
	firstSame, err := s.TrashVaultPath("same.txt")
	if err != nil {
		t.Fatalf("trash same root: %v", err)
	}
	secondSame, err := s.TrashVaultPath("Other/same.txt")
	if err != nil {
		t.Fatalf("trash same nested: %v", err)
	}
	if firstSame.TrashID == secondSame.TrashID || firstSame.TrashPath == secondSame.TrashPath {
		t.Fatalf("repeated trash basename collided: first=%+v second=%+v", firstSame, secondSame)
	}
	if _, err := s.TrashVaultPath(""); err == nil {
		t.Fatal("trash root should fail")
	}
	if _, err := s.TrashVaultPath("missing.txt"); err == nil || !strings.Contains(err.Error(), "not-found") {
		t.Fatalf("trash missing error = %v, want not-found", err)
	}

	entries, err := s.ListVaultFiles("")
	if err != nil {
		t.Fatalf("ListVaultFiles: %v", err)
	}
	for _, entry := range entries {
		if entry.Name == "delete-me.txt" || entry.Name == "delete-folder" || entry.Name == ".verstak" {
			t.Fatalf("unexpected entry after trash: %+v", entry)
		}
	}
}

func TestListTrashEntriesReturnsMetadata(t *testing.T) {
	s, root := newTestService(t)
	if err := os.WriteFile(filepath.Join(root, "old.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "new.txt"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldResult, err := s.TrashVaultPath("old.txt")
	if err != nil {
		t.Fatalf("trash old: %v", err)
	}
	newResult, err := s.TrashVaultPath("new.txt")
	if err != nil {
		t.Fatalf("trash new: %v", err)
	}

	entries, err := s.ListTrashEntries()
	if err != nil {
		t.Fatalf("ListTrashEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("trash entries count = %d, want 2: %+v", len(entries), entries)
	}
	if entries[0].TrashID != newResult.TrashID || entries[1].TrashID != oldResult.TrashID {
		t.Fatalf("trash entries order = %+v, want newest first", entries)
	}
	if entries[0].OriginalPath != "new.txt" || entries[0].TrashPath != newResult.TrashPath || entries[0].OriginalType != FileTypeFile || entries[0].Basename != "new.txt" || entries[0].Size != 3 {
		t.Fatalf("new trash entry = %+v", entries[0])
	}
}

func TestRestoreTrashEntryRestoresOriginalPathAndRemovesTrashMetadata(t *testing.T) {
	s, root := newTestService(t)
	if err := os.Mkdir(filepath.Join(root, "Docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Docs", "restore.txt"), []byte("restore me"), 0o644); err != nil {
		t.Fatal(err)
	}
	trash, err := s.TrashVaultPath("Docs/restore.txt")
	if err != nil {
		t.Fatalf("TrashVaultPath: %v", err)
	}

	restored, err := s.RestoreTrashEntry(trash.TrashID, RestoreOptions{})
	if err != nil {
		t.Fatalf("RestoreTrashEntry: %v", err)
	}
	if restored != "Docs/restore.txt" {
		t.Fatalf("restored path = %q, want Docs/restore.txt", restored)
	}
	data, err := os.ReadFile(filepath.Join(root, "Docs", "restore.txt"))
	if err != nil {
		t.Fatalf("restored file missing: %v", err)
	}
	if string(data) != "restore me" {
		t.Fatalf("restored content = %q", string(data))
	}
	if _, err := os.Stat(filepath.Join(root, trash.TrashPath)); !os.IsNotExist(err) {
		t.Fatalf("trash payload should be removed, stat err = %v", err)
	}
	entries, err := s.ListTrashEntries()
	if err != nil {
		t.Fatalf("ListTrashEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("trash entries after restore = %+v, want none", entries)
	}
}

func TestDeleteTrashEntryRemovesPayloadAndMetadata(t *testing.T) {
	s, root := newTestService(t)
	if err := os.WriteFile(filepath.Join(root, "delete-permanently.txt"), []byte("remove me"), 0o644); err != nil {
		t.Fatal(err)
	}
	trash, err := s.TrashVaultPath("delete-permanently.txt")
	if err != nil {
		t.Fatalf("TrashVaultPath: %v", err)
	}

	if err := s.DeleteTrashEntry(trash.TrashID); err != nil {
		t.Fatalf("DeleteTrashEntry: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".verstak", "trash", "files", trash.TrashID)); !os.IsNotExist(err) {
		t.Fatalf("trash directory should be removed, stat err = %v", err)
	}
	entries, err := s.ListTrashEntries()
	if err != nil {
		t.Fatalf("ListTrashEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("trash entries after permanent delete = %+v, want none", entries)
	}
	if err := s.DeleteTrashEntry(trash.TrashID); err == nil || !strings.Contains(err.Error(), "not-found: trash entry") {
		t.Fatalf("second DeleteTrashEntry error = %v, want not found", err)
	}
}

func TestRestoreTrashEntryConflictAndOverwrite(t *testing.T) {
	s, root := newTestService(t)
	if err := os.WriteFile(filepath.Join(root, "conflict.txt"), []byte("trashed"), 0o644); err != nil {
		t.Fatal(err)
	}
	trash, err := s.TrashVaultPath("conflict.txt")
	if err != nil {
		t.Fatalf("TrashVaultPath: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "conflict.txt"), []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := s.RestoreTrashEntry(trash.TrashID, RestoreOptions{}); err == nil || !strings.Contains(err.Error(), "conflict: conflict.txt") {
		t.Fatalf("restore conflict error = %v, want conflict", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "conflict.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "existing" {
		t.Fatalf("conflicting file content = %q, want existing", string(data))
	}

	restored, err := s.RestoreTrashEntry(trash.TrashID, RestoreOptions{Overwrite: true})
	if err != nil {
		t.Fatalf("RestoreTrashEntry overwrite: %v", err)
	}
	if restored != "conflict.txt" {
		t.Fatalf("restored path = %q, want conflict.txt", restored)
	}
	data, err = os.ReadFile(filepath.Join(root, "conflict.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "trashed" {
		t.Fatalf("overwritten content = %q, want trashed", string(data))
	}
}

func TestRestoreTrashEntryRejectsInvalidTrashID(t *testing.T) {
	s, _ := newTestService(t)
	for _, trashID := range []string{"", "../escape", "bad/slash"} {
		t.Run(trashID, func(t *testing.T) {
			if _, err := s.RestoreTrashEntry(trashID, RestoreOptions{}); err == nil {
				t.Fatal("expected invalid trash id error")
			}
		})
	}
}

func TestSymlinkEscapeRejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on many Windows test environments")
	}

	s, root := newTestService(t)
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "outside.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideFile, filepath.Join(root, "escape.txt")); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}

	meta, err := s.GetVaultFileMetadata("escape.txt")
	if err != nil {
		t.Fatalf("metadata symlink: %v", err)
	}
	if meta.Type != FileTypeSymlink {
		t.Fatalf("symlink type = %q", meta.Type)
	}

	if _, err := s.ReadVaultTextFile("escape.txt"); err == nil || !strings.Contains(err.Error(), "symlink-not-allowed") {
		t.Fatalf("read symlink error = %v, want symlink-not-allowed", err)
	}
	if err := s.WriteVaultTextFile("escape.txt", "x", WriteOptions{Overwrite: true}); err == nil || !strings.Contains(err.Error(), "symlink-not-allowed") {
		t.Fatalf("write symlink error = %v, want symlink-not-allowed", err)
	}
	if err := s.MoveVaultPath("escape.txt", "moved-link.txt", MoveOptions{}); err == nil || !strings.Contains(err.Error(), "symlink-not-allowed") {
		t.Fatalf("move symlink error = %v, want symlink-not-allowed", err)
	}
	if _, err := s.TrashVaultPath("escape.txt"); err == nil || !strings.Contains(err.Error(), "symlink-not-allowed") {
		t.Fatalf("trash symlink error = %v, want symlink-not-allowed", err)
	}
}

func TestSymlinkInsideVaultRejectedForMutatingAndReadOperations(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on many Windows test environments")
	}

	s, root := newTestService(t)
	if err := os.WriteFile(filepath.Join(root, "target.txt"), []byte("inside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "target.txt"), filepath.Join(root, "inside-link.txt")); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}

	meta, err := s.GetVaultFileMetadata("inside-link.txt")
	if err != nil {
		t.Fatalf("metadata inside symlink: %v", err)
	}
	if meta.Type != FileTypeSymlink {
		t.Fatalf("symlink type = %q", meta.Type)
	}
	if _, err := s.ReadVaultTextFile("inside-link.txt"); err == nil || !strings.Contains(err.Error(), "symlink-not-allowed") {
		t.Fatalf("read inside symlink error = %v, want symlink-not-allowed", err)
	}
	if err := s.WriteVaultTextFile("inside-link.txt", "x", WriteOptions{Overwrite: true}); err == nil || !strings.Contains(err.Error(), "symlink-not-allowed") {
		t.Fatalf("write inside symlink error = %v, want symlink-not-allowed", err)
	}
	if err := s.MoveVaultPath("inside-link.txt", "moved-link.txt", MoveOptions{}); err == nil || !strings.Contains(err.Error(), "symlink-not-allowed") {
		t.Fatalf("move inside symlink error = %v, want symlink-not-allowed", err)
	}
	if _, err := s.TrashVaultPath("inside-link.txt"); err == nil || !strings.Contains(err.Error(), "symlink-not-allowed") {
		t.Fatalf("trash inside symlink error = %v, want symlink-not-allowed", err)
	}

	matches, err := filepath.Glob(filepath.Join(root, ".verstak-write-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("write symlink left root temp files: %v", matches)
	}
}

func TestListVaultFilesRejectsSymlinkDirectoryEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on many Windows test environments")
	}

	s, root := newTestService(t)
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "outside.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "outside-dir")); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}

	if _, err := s.ListVaultFiles("outside-dir"); err == nil || !strings.Contains(err.Error(), "symlink-not-allowed") {
		t.Fatalf("list symlink dir error = %v, want symlink-not-allowed", err)
	}

	entries, err := s.ListVaultFiles("")
	if err != nil {
		t.Fatalf("list root: %v", err)
	}
	var foundSymlink bool
	for _, entry := range entries {
		if entry.RelativePath == "outside-dir" {
			foundSymlink = true
			if entry.Type != FileTypeSymlink {
				t.Fatalf("root symlink entry type = %q, want symlink", entry.Type)
			}
		}
	}
	if !foundSymlink {
		t.Fatal("root list should expose the symlink as metadata without following it")
	}
}

func TestCreateVaultFolderRejectsSymlinkParentEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on many Windows test environments")
	}

	s, root := newTestService(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "outside-dir")); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}

	if err := s.CreateVaultFolder("outside-dir/new-folder"); err == nil || !strings.Contains(err.Error(), "symlink-not-allowed") {
		t.Fatalf("create folder through symlink parent error = %v, want symlink-not-allowed", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "new-folder")); !os.IsNotExist(err) {
		t.Fatalf("folder should not be created outside vault, stat err=%v", err)
	}
}
