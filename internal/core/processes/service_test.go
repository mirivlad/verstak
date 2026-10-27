package processes

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestProcessHelper(t *testing.T) {
	if os.Getenv("VERSTAK_PROCESS_TEST_HELPER") != "1" {
		return
	}
	args := os.Args
	for len(args) > 0 && args[0] != "--" {
		args = args[1:]
	}
	if len(args) < 2 {
		os.Exit(90)
	}
	switch args[1] {
	case "echo":
		cwd, _ := os.Getwd()
		_ = json.NewEncoder(os.Stdout).Encode(map[string]interface{}{"args": args[2:], "cwd": cwd})
		fmt.Fprint(os.Stderr, "test stderr")
		os.Exit(7)
	case "sleep":
		time.Sleep(time.Minute)
		os.Exit(0)
	case "flood":
		for i := 0; i < 100000; i++ {
			fmt.Fprint(os.Stdout, "0123456789")
			fmt.Fprint(os.Stderr, "abcdefghij")
		}
		os.Exit(0)
	case "child":
		child := exec.Command(os.Args[0], "-test.run=TestProcessHelper", "--", "sleep")
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		if child.Start() != nil {
			os.Exit(91)
		}
		_ = os.WriteFile(args[2], []byte(fmt.Sprint(child.Process.Pid)), 0600)
		time.Sleep(time.Minute)
		os.Exit(0)
	}
	os.Exit(92)
}
func testRequest(t *testing.T, mode string, args ...string) Request {
	t.Helper()
	t.Setenv("VERSTAK_PROCESS_TEST_HELPER", "1")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	r, err := Normalize(Request{Kind: "process", Executable: executable, Args: append([]string{"-test.run=TestProcessHelper", "--", mode}, args...), Cwd: t.TempDir(), Mode: "wait"})
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func testService(t *testing.T, r Request) *Service {
	t.Helper()
	s := New(filepath.Join(t.TempDir(), "device", "approvals.json"))
	return s
}
func TestExactArgumentsDirectoryAndExit(t *testing.T) {
	args := []string{"two words", "$(touch NOT_CREATED)", "; echo unsafe", "", "*", "%PATH%"}
	r := testRequest(t, "echo", args...)
	s := testService(t, r)
	run, err := s.Start(context.Background(), "test", "one", r)
	if err != nil {
		t.Fatal(err)
	}
	result := run.Wait()
	if result.ExitCode != 7 || result.Stderr != "test stderr" {
		t.Fatalf("result=%+v", result)
	}
	var got struct {
		Args []string
		Cwd  string
	}
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Args, args) || got.Cwd != r.Cwd {
		t.Fatalf("literal argv/cwd mismatch: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(r.Cwd, "NOT_CREATED")); !os.IsNotExist(err) {
		t.Fatal("shell interpreted arguments")
	}
}
func TestLimitsAndCancellation(t *testing.T) {
	for _, kind := range []string{"timeout", "cancel", "output", "disable"} {
		t.Run(kind, func(t *testing.T) {
			mode := "sleep"
			if kind == "output" {
				mode = "flood"
			}
			r := testRequest(t, mode)
			if kind == "timeout" {
				r.TimeoutMS = 100
			}
			if kind == "output" {
				r.MaxOutputBytes = 777
			}
			s := testService(t, r)
			run, err := s.Start(context.Background(), "test", "one", r)
			if err != nil {
				t.Fatal(err)
			}
			if kind == "cancel" {
				s.Cancel("other", "one")
				s.Cancel("test", "one")
			}
			if kind == "disable" {
				s.Stop("test")
			}
			done := make(chan Result, 1)
			go func() { done <- run.Wait() }()
			select {
			case result := <-done:
				if kind == "timeout" && !result.TimedOut {
					t.Fatalf("missing timeout: %+v", result)
				}
				if (kind == "cancel" || kind == "disable") && !result.Cancelled {
					t.Fatalf("missing cancel: %+v", result)
				}
				if kind == "output" && (!result.OutputLimited || len(result.Stdout)+len(result.Stderr) > 777) {
					t.Fatalf("output not bounded: %+v", result)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("process did not terminate")
			}
		})
	}
}
func TestLegacyReviewStoreIsLocalButNotAProcessPermission(t *testing.T) {
	r := testRequest(t, "echo", "sensitive-test-argument")
	s := testService(t, r)
	if err := s.SetApproval("test", r, true); err != nil {
		t.Fatal(err)
	}
	if approved, err := New(s.approvalPath).Approved("test", r); err != nil || !approved {
		t.Fatal("approval did not reload", err)
	}
	for _, owner := range []string{"other-plugin", "other-vault"} {
		if approved, err := s.Approved(owner, r); err != nil || approved {
			t.Fatal("legacy review crossed owner", err)
		}
	}
	changed := r
	changed.Args = append(append([]string{}, r.Args...), "changed")
	if approved, err := s.Approved("test", changed); err != nil || approved {
		t.Fatal("legacy review crossed request", err)
	}
	fresh := New(filepath.Join(t.TempDir(), "approvals.json"))
	run, err := fresh.Start(context.Background(), "authorized-plugin", "first-run", r)
	if err != nil {
		t.Fatal("process service still requires a request grant", err)
	}
	if got := run.Wait(); got.ExitCode != 7 {
		t.Fatal(got)
	}
	data, err := os.ReadFile(s.approvalPath)
	if err != nil || strings.Contains(string(data), "sensitive-test-argument") {
		t.Fatal("approval store contains arguments")
	}
	if err := s.SetApproval("test", r, false); err != nil {
		t.Fatal(err)
	}
	if ok, _ := s.Approved("test", r); ok {
		t.Fatal("revoke failed")
	}
}
func TestNormalizeRejectsUnsafeInputs(t *testing.T) {
	r := testRequest(t, "echo")
	invalid := []Request{{Kind: "process", Executable: "not-a-real-verstak-command", Mode: "wait"}, {Kind: "url", URL: "file:///tmp/test"}, {Kind: "url", URL: "https://user:password@example.org"}, {Kind: "url", URL: "https://example.org", Executable: r.Executable}}
	for _, change := range []func(*Request){func(r *Request) { r.Args = []string{"nul\x00byte"} }, func(r *Request) { r.Cwd = "relative" }, func(r *Request) { r.TimeoutMS = 600001 }, func(r *Request) { r.MaxOutputBytes = 4194305 }, func(r *Request) { r.Executable = "./relative" }} {
		x := r
		change(&x)
		invalid = append(invalid, x)
	}
	for _, r := range invalid {
		if _, err := Normalize(r); err == nil {
			t.Fatalf("accepted invalid request: %+v", r)
		}
	}
}
func TestDetachedDoesNotWaitForExit(t *testing.T) {
	r := testRequest(t, "echo", "detached")
	r.Mode = "detached"
	s := testService(t, r)
	start := time.Now()
	run, err := s.Start(context.Background(), "test", "x", r)
	if err != nil {
		t.Fatal(err)
	}
	if result := run.Wait(); !result.Started || result.ExitCode != -1 || time.Since(start) > time.Second {
		t.Fatalf("detached result=%+v", result)
	}
}
func TestRealLinuxCommand(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux runtime smoke")
	}
	r, err := Normalize(Request{Kind: "process", Executable: "printf", Args: []string{"%s\n", "Actions 2.0: literal ; $(echo no-shell)"}, Mode: "wait"})
	if err != nil {
		t.Fatal(err)
	}
	run, err := testService(t, r).Start(context.Background(), "test", "linux", r)
	if err != nil {
		t.Fatal(err)
	}
	result := run.Wait()
	if result.ExitCode != 0 || result.Stdout != "Actions 2.0: literal ; $(echo no-shell)\n" {
		t.Fatalf("result=%+v", result)
	}
	t.Logf("real Linux printf: exit=%d stdout=%q", result.ExitCode, result.Stdout)
}

func TestLinuxTimeoutStopsDescendants(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux /proc process state")
	}
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	r := testRequest(t, "child", pidFile)
	r.TimeoutMS = 500
	run, err := testService(t, r).Start(context.Background(), "test", "child", r)
	if err != nil {
		t.Fatal(err)
	}
	result := run.Wait()
	if !result.TimedOut {
		t.Fatalf("expected timeout: %+v", result)
	}
	pid, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal("helper did not spawn child", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile("/proc/" + string(pid) + "/stat")
		if os.IsNotExist(err) {
			return
		}
		if err == nil {
			tail := string(data)[strings.LastIndex(string(data), ")")+1:]
			fields := strings.Fields(tail)
			if len(fields) > 0 && (fields[0] == "Z" || fields[0] == "X") {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("descendant remained alive after timeout")
}
