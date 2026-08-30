package gitservice

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	CheckoutStateCloned    = "cloned"
	CheckoutStateNotCloned = "not-cloned"
)

type CloneRequest struct {
	WorkspaceID   string
	WorkspaceRoot string
	RepositoryID  string
	CheckoutName  string
	RemoteURL     string
	Branch        string
	Credential    Credential
}

type RepositoryRequest struct {
	WorkspaceID   string
	WorkspaceRoot string
	RepositoryID  string
	CheckoutName  string
	Credential    Credential
}

// ExistingRepositoryRequest copies an existing local checkout into the
// managed Deal checkout root. The source is intentionally not retained.
type ExistingRepositoryRequest struct {
	RepositoryRequest
	SourcePath string
}

// Credential is transient process input resolved by Core. It is never stored
// in checkout metadata or returned from this package.
type Credential struct {
	Username   string
	Value      string
	PrivateKey bool
}

type RecentCommit struct {
	ID        string `json:"id"`
	ShortID   string `json:"shortId"`
	Subject   string `json:"subject"`
	Author    string `json:"author"`
	Committed string `json:"committed"`
}

type Status struct {
	State          string         `json:"state"`
	Branch         string         `json:"branch,omitempty"`
	Clean          bool           `json:"clean"`
	ChangedCount   int            `json:"changedCount"`
	UntrackedCount int            `json:"untrackedCount"`
	ChangedFiles   []string       `json:"changedFiles"`
	Ahead          int            `json:"ahead"`
	Behind         int            `json:"behind"`
	RecentCommits  []RecentCommit `json:"recentCommits"`
}

type Service struct{ vaultRoot string }

func NewService(vaultRoot string) *Service { return &Service{vaultRoot: vaultRoot} }

func (s *Service) Clone(request CloneRequest) (string, error) {
	if strings.TrimSpace(request.RemoteURL) == "" {
		return "", fmt.Errorf("remote URL is required")
	}
	checkout, err := RegisterCheckout(s.vaultRoot, CheckoutRegistration{WorkspaceID: request.WorkspaceID, WorkspaceRoot: request.WorkspaceRoot, RepositoryID: request.RepositoryID, CheckoutName: request.CheckoutName})
	if err != nil {
		return "", err
	}
	path := filepath.Join(s.vaultRoot, filepath.FromSlash(checkout))
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("managed checkout already exists")
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	branch := strings.TrimSpace(request.Branch)
	if branch == "" {
		branch = "main"
	}
	if err := runGitWithCredential(filepath.Dir(path), request.Credential, "clone", "--branch", branch, "--", request.RemoteURL, path); err != nil {
		return "", err
	}
	return checkout, nil
}

func (s *Service) RegisterExisting(request ExistingRepositoryRequest) (string, error) {
	source := filepath.Clean(strings.TrimSpace(request.SourcePath))
	if !filepath.IsAbs(source) {
		return "", fmt.Errorf("existing repository path must be absolute")
	}
	if info, err := os.Stat(source); err != nil || !info.IsDir() {
		return "", fmt.Errorf("existing repository path is not a directory")
	}
	if err := runGit(source, "rev-parse", "--is-inside-work-tree"); err != nil {
		return "", fmt.Errorf("existing repository path is not a Git worktree")
	}
	checkout, err := RegisterCheckout(s.vaultRoot, CheckoutRegistration{
		WorkspaceID: request.WorkspaceID, WorkspaceRoot: request.WorkspaceRoot,
		RepositoryID: request.RepositoryID, CheckoutName: request.CheckoutName,
	})
	if err != nil {
		return "", err
	}
	path := filepath.Join(s.vaultRoot, filepath.FromSlash(checkout))
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("managed checkout already exists")
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	if err := runGit(filepath.Dir(path), "clone", "--local", "--", source, path); err != nil {
		return "", err
	}
	return checkout, nil
}

func (s *Service) Status(request RepositoryRequest) (Status, error) {
	path, err := s.checkoutPath(request)
	if err != nil {
		return Status{}, err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return Status{State: CheckoutStateNotCloned, Clean: true, ChangedFiles: []string{}, RecentCommits: []RecentCommit{}}, nil
	} else if err != nil {
		return Status{}, err
	}
	output, err := runGitOutput(path, request.Credential, "status", "--porcelain=v2", "--branch")
	if err != nil {
		return Status{}, err
	}
	status := parseStatus(output)
	status.State = CheckoutStateCloned
	commits, err := recentCommits(path)
	if err != nil {
		return Status{}, err
	}
	status.RecentCommits = commits
	return status, nil
}

// CheckoutPath returns an existing managed local checkout. It intentionally
// does not create a checkout for a synced descriptor on a new device.
func (s *Service) CheckoutPath(request RepositoryRequest) (string, error) {
	path, err := s.checkoutPath(request)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("repository is not cloned on this device")
	} else if err != nil {
		return "", err
	}
	return path, nil
}

func (s *Service) Fetch(request RepositoryRequest) error { return s.run(request, "fetch", "--prune") }
func (s *Service) Pull(request RepositoryRequest) error  { return s.run(request, "pull", "--ff-only") }
func (s *Service) Push(request RepositoryRequest) error  { return s.run(request, "push") }

func (s *Service) run(request RepositoryRequest, args ...string) error {
	path, err := s.checkoutPath(request)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("repository is not cloned on this device")
	} else if err != nil {
		return err
	}
	return runGitWithCredential(path, request.Credential, args...)
}

func (s *Service) checkoutPath(request RepositoryRequest) (string, error) {
	checkout, err := RegisterCheckout(s.vaultRoot, CheckoutRegistration{WorkspaceID: request.WorkspaceID, WorkspaceRoot: request.WorkspaceRoot, RepositoryID: request.RepositoryID, CheckoutName: request.CheckoutName})
	if err != nil {
		return "", err
	}
	return filepath.Join(s.vaultRoot, filepath.FromSlash(checkout)), nil
}

func runGit(dir string, args ...string) error {
	_, err := runGitOutput(dir, Credential{}, args...)
	return err
}

func runGitWithCredential(dir string, credential Credential, args ...string) error {
	_, err := runGitOutput(dir, credential, args...)
	return err
}

func runGitOutput(dir string, credential Credential, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	cleanup, env, err := credentialEnvironment(credential)
	if err != nil {
		return "", err
	}
	defer cleanup()
	command.Env = append(command.Env, env...)
	output, err := command.CombinedOutput()
	if ctx.Err() != nil {
		return "", fmt.Errorf("Git operation timed out")
	}
	if err != nil {
		return "", fmt.Errorf("Git operation failed")
	}
	if len(output) > 1024*1024 {
		return "", fmt.Errorf("Git operation output exceeded limit")
	}
	return string(output), nil
}

func parseStatus(output string) Status {
	status := Status{Clean: true, ChangedFiles: []string{}, RecentCommits: []RecentCommit{}}
	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.HasPrefix(line, "# branch.head "):
			status.Branch = strings.TrimPrefix(line, "# branch.head ")
		case strings.HasPrefix(line, "# branch.ab +"):
			fmt.Sscanf(strings.TrimPrefix(line, "# branch.ab "), "+%d -%d", &status.Ahead, &status.Behind)
		case strings.HasPrefix(line, "? "):
			status.Clean = false
			status.UntrackedCount++
			status.ChangedFiles = append(status.ChangedFiles, strings.TrimPrefix(line, "? "))
		case strings.HasPrefix(line, "1 ") || strings.HasPrefix(line, "2 ") || strings.HasPrefix(line, "u "):
			status.Clean = false
			status.ChangedCount++
			parts := strings.SplitN(line, " ", 9)
			if len(parts) == 9 {
				status.ChangedFiles = append(status.ChangedFiles, parts[8])
			}
		}
	}
	return status
}

func recentCommits(path string) ([]RecentCommit, error) {
	output, err := runGitOutput(path, Credential{}, "log", "-n", "10", "--format=%H%x1f%h%x1f%s%x1f%an%x1f%aI%x1e")
	if err != nil {
		return nil, err
	}
	commits := []RecentCommit{}
	for _, record := range strings.Split(output, "\x1e") {
		fields := strings.Split(strings.TrimSpace(record), "\x1f")
		if len(fields) != 5 {
			continue
		}
		commits = append(commits, RecentCommit{ID: fields[0], ShortID: fields[1], Subject: fields[2], Author: fields[3], Committed: fields[4]})
	}
	return commits, nil
}

func credentialEnvironment(credential Credential) (func(), []string, error) {
	if credential.Value == "" {
		return func() {}, nil, nil
	}
	if credential.PrivateKey {
		return privateKeyEnvironment(credential.Value)
	}
	return askPassEnvironment(credential)
}

func askPassEnvironment(credential Credential) (func(), []string, error) {
	dir, err := os.MkdirTemp("", "verstak-git-askpass-")
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	script := filepath.Join(dir, "askpass")
	contents := "#!/bin/sh\ncase \"$1\" in *Username*) printf '%s\\n' \"$VERSTAK_GIT_USERNAME\" ;; *) printf '%s\\n' \"$VERSTAK_GIT_PASSWORD\" ;; esac\n"
	if err := os.WriteFile(script, []byte(contents), 0o700); err != nil {
		cleanup()
		return nil, nil, err
	}
	return cleanup, []string{"GIT_ASKPASS=" + script, "VERSTAK_GIT_USERNAME=" + credential.Username, "VERSTAK_GIT_PASSWORD=" + credential.Value}, nil
}

func privateKeyEnvironment(privateKey string) (func(), []string, error) {
	dir, err := os.MkdirTemp("", "verstak-git-key-")
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	path := filepath.Join(dir, "identity")
	if err := os.WriteFile(path, []byte(privateKey), 0o600); err != nil {
		cleanup()
		return nil, nil, err
	}
	return cleanup, []string{"GIT_SSH_COMMAND=ssh -i " + shellSingleQuote(path) + " -o IdentitiesOnly=yes -o BatchMode=yes"}, nil
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\\"'\\\"'") + "'"
}
