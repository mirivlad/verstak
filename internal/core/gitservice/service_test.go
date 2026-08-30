package gitservice

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestServiceClonesManagedCheckoutAndReportsWorkingTreeState(t *testing.T) {
	vault := t.TempDir()
	workspaceID := uuid.NewString()
	writeWorkspaceMarker(t, vault, "Deal", workspaceID)
	remote := createBareRemote(t)

	service := NewService(vault)
	checkout, err := service.Clone(CloneRequest{WorkspaceID: workspaceID, WorkspaceRoot: "Deal", RepositoryID: "origin", CheckoutName: "origin", RemoteURL: remote})
	if err != nil {
		t.Fatal(err)
	}
	if checkout != "Deal/Repositories/origin" {
		t.Fatalf("checkout = %q", checkout)
	}
	if _, err := os.Stat(filepath.Join(vault, "Deal", "Repositories", "origin", ".git")); err != nil {
		t.Fatalf("clone did not create managed checkout: %v", err)
	}
	status, err := service.Status(RepositoryRequest{WorkspaceID: workspaceID, WorkspaceRoot: "Deal", RepositoryID: "origin", CheckoutName: "origin"})
	if err != nil {
		t.Fatal(err)
	}
	if status.State != CheckoutStateCloned || status.Branch != "main" || !status.Clean || len(status.RecentCommits) != 1 {
		t.Fatalf("initial status = %+v", status)
	}
	if err := os.WriteFile(filepath.Join(vault, filepath.FromSlash(checkout), "README.md"), []byte("changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vault, filepath.FromSlash(checkout), "new.txt"), []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	status, err = service.Status(RepositoryRequest{WorkspaceID: workspaceID, WorkspaceRoot: "Deal", RepositoryID: "origin", CheckoutName: "origin"})
	if err != nil {
		t.Fatal(err)
	}
	if status.Clean || status.ChangedCount != 1 || status.UntrackedCount != 1 || len(status.ChangedFiles) != 2 {
		t.Fatalf("dirty status = %+v", status)
	}
}

func TestGitFailureDoesNotExposeTransientCredential(t *testing.T) {
	_, err := runGitOutput(t.TempDir(), Credential{Username: "git", Value: "token-must-not-leak"}, "status")
	if err == nil {
		t.Fatal("Git status outside a repository unexpectedly succeeded")
	}
	if strings.Contains(err.Error(), "token-must-not-leak") {
		t.Fatalf("Git error leaked credential: %v", err)
	}
}

func TestServiceRegistersExistingRepositoryIntoManagedCheckout(t *testing.T) {
	vault := t.TempDir()
	workspaceID := uuid.NewString()
	writeWorkspaceMarker(t, vault, "Deal", workspaceID)

	source := filepath.Join(t.TempDir(), "source")
	runTestGit(t, filepath.Dir(source), "init", "-b", "main", source)
	runTestGit(t, source, "config", "user.name", "Test User")
	runTestGit(t, source, "config", "user.email", "test@example.invalid")
	if err := os.WriteFile(filepath.Join(source, "README.md"), []byte("existing\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runTestGit(t, source, "add", "README.md")
	runTestGit(t, source, "commit", "-m", "existing")

	service := NewService(vault)
	checkout, err := service.RegisterExisting(ExistingRepositoryRequest{
		RepositoryRequest: RepositoryRequest{WorkspaceID: workspaceID, WorkspaceRoot: "Deal", RepositoryID: "existing", CheckoutName: "existing"},
		SourcePath:        source,
	})
	if err != nil {
		t.Fatal(err)
	}
	if checkout != "Deal/Repositories/existing" {
		t.Fatalf("checkout = %q", checkout)
	}
	if got := string(mustReadFile(t, filepath.Join(vault, filepath.FromSlash(checkout), "README.md"))); got != "existing\n" {
		t.Fatalf("registered checkout README = %q", got)
	}
}

func TestStatusReportsSyncedDescriptorAsNotClonedOnThisDevice(t *testing.T) {
	vault := t.TempDir()
	workspaceID := uuid.NewString()
	writeWorkspaceMarker(t, vault, "Deal", workspaceID)

	status, err := NewService(vault).Status(RepositoryRequest{WorkspaceID: workspaceID, WorkspaceRoot: "Deal", RepositoryID: "synced", CheckoutName: "synced"})
	if err != nil {
		t.Fatal(err)
	}
	if status.State != CheckoutStateNotCloned || !status.Clean || len(status.ChangedFiles) != 0 || len(status.RecentCommits) != 0 {
		t.Fatalf("status = %+v", status)
	}
}

func TestPrivateKeyEnvironmentIsTransient(t *testing.T) {
	cleanup, env, err := credentialEnvironment(Credential{Username: "git", Value: "private-key-must-not-persist", PrivateKey: true})
	if err != nil {
		t.Fatal(err)
	}
	keyPath := ""
	for _, entry := range env {
		if strings.HasPrefix(entry, "GIT_SSH_COMMAND=") {
			parts := strings.Split(entry, "'")
			if len(parts) >= 2 {
				keyPath = parts[1]
			}
		}
		if strings.Contains(entry, "private-key-must-not-persist") {
			t.Fatalf("credential value leaked through environment: %q", entry)
		}
	}
	if keyPath == "" {
		t.Fatalf("GIT_SSH_COMMAND missing from %v", env)
	}
	if got := string(mustReadFile(t, keyPath)); got != "private-key-must-not-persist" {
		t.Fatalf("temporary key = %q", got)
	}
	cleanup()
	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		t.Fatalf("temporary key remains after cleanup: %v", err)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func writeWorkspaceMarker(t *testing.T, vault, root, workspaceID string) {
	t.Helper()
	path := filepath.Join(vault, root, ".verstak", "workspace.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{"schemaVersion":1,"workspaceId":"` + workspaceID + `"}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func createBareRemote(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	remote := filepath.Join(root, "remote.git")
	runTestGit(t, root, "init", "--bare", remote)
	seed := filepath.Join(root, "seed")
	runTestGit(t, root, "init", "-b", "main", seed)
	runTestGit(t, seed, "config", "user.name", "Test User")
	runTestGit(t, seed, "config", "user.email", "test@example.invalid")
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("initial\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runTestGit(t, seed, "add", "README.md")
	runTestGit(t, seed, "commit", "-m", "initial")
	runTestGit(t, seed, "remote", "add", "origin", remote)
	runTestGit(t, seed, "push", "-u", "origin", "main")
	return remote
}

func runTestGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}
