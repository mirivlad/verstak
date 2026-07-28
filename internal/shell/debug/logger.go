// Package debug keeps a record of what the application did, so a report of
// "it crashed" can be answered with something other than a guess.
//
// The session log is always written. It used to exist only behind --debug,
// which meant the one run that mattered -- the one that already failed -- was
// the one nobody had a log for. --debug now only makes that log louder and
// mirrors it to stderr.
package debug

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// keptSessions bounds the log directory. Enough runs to cover "it broke
// yesterday too", few enough that nobody has to clean up after us.
const keptSessions = 10

var (
	logger      *log.Logger
	mu          sync.Mutex
	enabled     bool
	sessionPath string
)

// LogDir is where session logs and crash reports are written.
func LogDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(os.TempDir(), "verstak-logs")
	}
	return filepath.Join(home, ".local", "share", "verstak", "logs")
}

// SessionLogPath is the file this run is writing to, or "" if none could be
// opened.
func SessionLogPath() string {
	mu.Lock()
	defer mu.Unlock()
	return sessionPath
}

// Init opens this run's log and routes the standard logger into it. `--debug`
// in args turns on the verbose calls and mirrors everything to stderr.
// Returns true when debug mode is enabled.
func Init(args []string) bool {
	mu.Lock()
	defer mu.Unlock()

	for _, arg := range args {
		if arg == "--debug" {
			enabled = true
			break
		}
	}

	dir := LogDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("[debug] failed to create log dir %s: %v", dir, err)
		dir = filepath.Join(os.TempDir(), "verstak-logs")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return enabled
		}
	}
	pruneSessions(dir)

	path := filepath.Join(dir, fmt.Sprintf("verstak-%s.log", time.Now().Format("2006-01-02-150405")))
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Printf("[debug] failed to open log file %s: %v", path, err)
		return enabled
	}
	sessionPath = path

	// Everything the application already logs goes to the file. Without a
	// terminal there is nowhere else for it to go, and a packaged build is
	// always started without one.
	log.SetOutput(io.MultiWriter(file, os.Stderr))
	logger = log.New(io.MultiWriter(file, os.Stderr), "", log.LstdFlags|log.Lmicroseconds)
	log.Printf("[debug] session log: %s (verbose=%v)", path, enabled)
	return enabled
}

// pruneSessions keeps the newest logs and deletes the rest. Crash reports are
// left alone: they are the ones somebody will ask for.
func pruneSessions(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "verstak-") || !strings.HasSuffix(name, ".log") {
			continue
		}
		names = append(names, name)
	}
	if len(names) < keptSessions {
		return
	}
	sort.Strings(names)
	for _, name := range names[:len(names)-keptSessions+1] {
		os.Remove(filepath.Join(dir, name))
	}
}

// IsEnabled reports whether verbose debug logging is active.
func IsEnabled() bool {
	mu.Lock()
	defer mu.Unlock()
	return enabled
}

// Logf writes a formatted debug message when verbose logging is on.
func Logf(format string, v ...interface{}) {
	mu.Lock()
	defer mu.Unlock()
	if !enabled || logger == nil {
		return
	}
	logger.Printf(format, v...)
}

// Log writes a debug message when verbose logging is on.
func Log(v ...interface{}) {
	mu.Lock()
	defer mu.Unlock()
	if !enabled || logger == nil {
		return
	}
	logger.Println(v...)
}

// WriteCrashReport records a panic and the stack that produced it, in its own
// file so it survives the log retention that will eventually delete the
// session it happened in. It returns the path it wrote, or "".
func WriteCrashReport(reason interface{}, stack []byte) string {
	dir := LogDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ""
	}
	path := filepath.Join(dir, fmt.Sprintf("crash-%s.log", time.Now().Format("2006-01-02-150405")))
	report := fmt.Sprintf("verstak crash\ntime: %s\nos: %s/%s\ngo: %s\nreason: %v\n\n%s\n",
		time.Now().UTC().Format(time.RFC3339), runtime.GOOS, runtime.GOARCH, runtime.Version(), reason, stack)
	if err := os.WriteFile(path, []byte(report), 0o644); err != nil {
		return ""
	}
	log.Printf("[crash] %v — report written to %s", reason, path)
	return path
}

// Recover writes a crash report and lets the panic continue. Swallowing it
// would leave the window alive with half its state gone, which is worse than
// stopping.
func Recover() {
	reason := recover()
	if reason == nil {
		return
	}
	stack := make([]byte, 64*1024)
	stack = stack[:runtime.Stack(stack, true)]
	WriteCrashReport(reason, stack)
	panic(reason)
}
