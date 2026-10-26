// Package processes runs explicitly reviewed local programs. It never uses a shell.
package processes

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const Capability = "verstak/core/processes/v1"

// Request is deliberately an executable and argv, never a command line.
// URL requests use only Kind and URL; local file handlers are not exposed.
type Request struct {
	Kind           string   `json:"kind"`
	Executable     string   `json:"executable,omitempty"`
	Args           []string `json:"args,omitempty"`
	Cwd            string   `json:"cwd,omitempty"`
	Mode           string   `json:"mode,omitempty"`
	TimeoutMS      int      `json:"timeoutMs,omitempty"`
	MaxOutputBytes int      `json:"maxOutputBytes,omitempty"`
	URL            string   `json:"url,omitempty"`
}

type Result struct {
	Started       bool   `json:"started"`
	ExitCode      int    `json:"exitCode"`
	Stdout        string `json:"stdout"`
	Stderr        string `json:"stderr"`
	TimedOut      bool   `json:"timedOut"`
	Cancelled     bool   `json:"cancelled"`
	OutputLimited bool   `json:"outputLimited"`
}

// Normalize resolves PATH before review, and bounds everything sent to native UI.
func Normalize(r Request) (Request, error) {
	encoded, err := json.Marshal(r)
	if err != nil || len(encoded) > 16384 {
		return Request{}, errors.New("invalid_request")
	}
	if r.Kind == "url" {
		u, err := url.Parse(r.URL)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil || strings.ContainsAny(r.URL, "\x00\r\n") {
			return Request{}, errors.New("invalid_url")
		}
		if r.Executable != "" || len(r.Args) != 0 || r.Cwd != "" || r.Mode != "" || r.TimeoutMS != 0 || r.MaxOutputBytes != 0 {
			return Request{}, errors.New("invalid_request")
		}
		return Request{Kind: "url", URL: r.URL}, nil
	}
	if r.Kind != "process" || r.URL != "" || r.Executable == "" || strings.ContainsRune(r.Executable, 0) {
		return Request{}, errors.New("invalid_request")
	}
	if len(r.Args) > 256 {
		return Request{}, errors.New("invalid_arguments")
	}
	for _, arg := range r.Args {
		if strings.ContainsRune(arg, 0) {
			return Request{}, errors.New("invalid_arguments")
		}
	}
	if r.Mode != "wait" && r.Mode != "detached" {
		return Request{}, errors.New("invalid_mode")
	}
	if r.Cwd == "" {
		r.Cwd, err = os.UserHomeDir()
		if err != nil {
			return Request{}, errors.New("invalid_directory")
		}
	}
	if !filepath.IsAbs(r.Cwd) {
		return Request{}, errors.New("invalid_directory")
	}
	r.Cwd = filepath.Clean(r.Cwd)
	info, err := os.Stat(r.Cwd)
	if err != nil || !info.IsDir() {
		return Request{}, errors.New("directory_not_found")
	}
	if strings.ContainsAny(r.Executable, `/\`) && !filepath.IsAbs(r.Executable) {
		return Request{}, errors.New("absolute_executable_required")
	}
	path, err := exec.LookPath(r.Executable)
	if err != nil {
		return Request{}, errors.New("program_not_found")
	}
	r.Executable, err = filepath.Abs(path)
	if err != nil {
		return Request{}, errors.New("program_not_found")
	}
	if r.TimeoutMS == 0 {
		r.TimeoutMS = 30000
	}
	if r.MaxOutputBytes == 0 {
		r.MaxOutputBytes = 1048576
	}
	if r.TimeoutMS < 1 || r.TimeoutMS > 600000 || r.MaxOutputBytes < 1 || r.MaxOutputBytes > 4194304 {
		return Request{}, errors.New("invalid_limits")
	}
	return r, nil
}

func fingerprint(owner string, r Request) string {
	data, _ := json.Marshal(struct {
		Owner   string
		Request Request
	}{owner, r})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

type Service struct {
	mu           sync.Mutex
	approvalPath string
	active       map[string]activeRun
}
type activeRun struct {
	owner  string
	cancel context.CancelFunc
}

// approvalPath must be device-local and outside the vault. Only hashes are saved.
func New(approvalPath string) *Service {
	return &Service{approvalPath: approvalPath, active: make(map[string]activeRun)}
}
func (s *Service) approvals() (map[string]bool, error) {
	data, err := os.ReadFile(s.approvalPath)
	if os.IsNotExist(err) {
		return map[string]bool{}, nil
	}
	if err != nil {
		return nil, errors.New("approval_store_unavailable")
	}
	var grants map[string]bool
	if json.Unmarshal(data, &grants) != nil || grants == nil {
		return nil, errors.New("approval_store_invalid")
	}
	return grants, nil
}
func (s *Service) Approved(owner string, r Request) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	grants, err := s.approvals()
	if err != nil {
		return false, err
	}
	return grants[fingerprint(owner, r)], nil
}

// SetApproval is called only after a host-owned native confirmation, never from plugin data.
func (s *Service) SetApproval(owner string, r Request, approved bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	grants, err := s.approvals()
	if err != nil {
		return err
	}
	key := fingerprint(owner, r)
	if approved {
		grants[key] = true
	} else {
		delete(grants, key)
	}
	if err := os.MkdirAll(filepath.Dir(s.approvalPath), 0700); err != nil {
		return errors.New("approval_store_unavailable")
	}
	data, _ := json.Marshal(grants)
	f, err := os.CreateTemp(filepath.Dir(s.approvalPath), ".process-approvals-*")
	if err != nil {
		return errors.New("approval_store_unavailable")
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		return errors.New("approval_store_unavailable")
	}
	// Windows cannot atomically replace an existing destination with os.Rename.
	if err := replaceApprovalFile(f.Name(), s.approvalPath); err != nil {
		return errors.New("approval_store_unavailable")
	}
	return nil
}

type Run struct{ done chan Result }

func (r *Run) Wait() Result { return <-r.done }

// Start registers cancellation before returning so disable cannot race the launch.
// Caller must serialize its access check + Start against plugin disable/reload.
func (s *Service) Start(parent context.Context, owner, id string, r Request) (*Run, error) {
	if r.Kind != "process" || id == "" || len(id) > 128 {
		return nil, errors.New("invalid_request")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	grants, err := s.approvals()
	if err != nil {
		return nil, err
	}
	if !grants[fingerprint(owner, r)] {
		return nil, errors.New("approval_required")
	}
	key := owner + ":" + id
	if _, exists := s.active[key]; exists {
		return nil, errors.New("operation_in_progress")
	}
	count := 0
	for _, op := range s.active {
		if op.owner == owner {
			count++
		}
	}
	if count >= 4 || len(s.active) >= 16 {
		return nil, errors.New("too_many_processes")
	}
	cmd := exec.Command(r.Executable, r.Args...)
	cmd.Dir = r.Cwd
	run := &Run{done: make(chan Result, 1)}
	if r.Mode == "detached" {
		configureProcess(cmd)
		if err := cmd.Start(); err != nil {
			return nil, errors.New("start_failed")
		}
		// Reap without inheriting stdout/stderr pipes or waiting for the GUI to close.
		go func() { _ = cmd.Wait() }()
		run.done <- Result{Started: true, ExitCode: -1}
		return run, nil
	}
	ctx, cancel := context.WithTimeout(parent, time.Duration(r.TimeoutMS)*time.Millisecond)
	output := &capture{remaining: r.MaxOutputBytes, cancel: cancel}
	cmd.Stdout = stream{output, &output.stdout}
	cmd.Stderr = stream{output, &output.stderr}
	cmd.WaitDelay = time.Second
	configureProcess(cmd)
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, errors.New("start_failed")
	}
	terminate, release, err := watchProcess(cmd)
	if err != nil {
		cancel()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, errors.New("process_control_unavailable")
	}
	s.active[key] = activeRun{owner, func() { cancel(); terminate() }}
	go func() {
		stopped := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				terminate()
			case <-stopped:
			}
		}()
		_ = cmd.Wait()
		// Do not let background descendants retain inherited pipes or outlive a waited job.
		terminate()
		close(stopped)
		release()
		output.mu.Lock()
		result := Result{Started: true, ExitCode: cmd.ProcessState.ExitCode(), Stdout: strings.ToValidUTF8(output.stdout.String(), "?"), Stderr: strings.ToValidUTF8(output.stderr.String(), "?"), OutputLimited: output.limited}
		output.mu.Unlock()
		result.TimedOut = errors.Is(ctx.Err(), context.DeadlineExceeded)
		result.Cancelled = ctx.Err() != nil && !result.TimedOut && !result.OutputLimited
		cancel()
		s.mu.Lock()
		delete(s.active, key)
		s.mu.Unlock()
		run.done <- result
	}()
	return run, nil
}
func (s *Service) Cancel(owner, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if op, ok := s.active[owner+":"+id]; ok {
		op.cancel()
	}
}
func (s *Service) Stop(owner string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, op := range s.active {
		if owner == "" || op.owner == owner {
			op.cancel()
		}
	}
}

type capture struct {
	mu             sync.Mutex
	stdout, stderr bytes.Buffer
	remaining      int
	limited        bool
	cancel         context.CancelFunc
}
type stream struct {
	capture *capture
	buffer  *bytes.Buffer
}

func (w stream) Write(p []byte) (int, error) {
	c := w.capture
	c.mu.Lock()
	defer c.mu.Unlock()
	n := len(p)
	keep := n
	if keep > c.remaining {
		keep = c.remaining
		c.limited = true
		c.cancel()
	}
	_, _ = w.buffer.Write(p[:keep])
	c.remaining -= keep
	return n, nil
}

func Preview(r Request) string {
	data, _ := json.MarshalIndent(r, "", "  ")
	return fmt.Sprintf("%s", data)
}
