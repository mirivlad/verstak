// Package debug provides a debug logger that writes to a file.
// Enabled with --debug CLI flag.
package debug

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	logger  *log.Logger
	mu      sync.Mutex
	enabled bool
)

// Init initializes the debug logger. If --debug is present in args,
// it writes to ~/.local/share/verstak/debug/verstak-YYYY-MM-DD-HHMMSS.log.
// Returns true if debug mode is enabled.
func Init(args []string) bool {
	mu.Lock()
	defer mu.Unlock()

	for _, a := range args {
		if a == "--debug" {
			enabled = true
			break
		}
	}

	if !enabled {
		return false
	}

	// Create log directory
	logDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "verstak", "debug")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("[debug] failed to create log dir %s: %v", logDir, err)
		// Fallback to /tmp
		logDir = filepath.Join(os.TempDir(), "verstak-debug")
		os.MkdirAll(logDir, 0755)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02-150405")
	logFile := filepath.Join(logDir, fmt.Sprintf("verstak-%s.log", timestamp))

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("[debug] failed to open log file %s: %v", logFile, err)
		return true // Still enabled, but logging to stderr
	}

	// Write to both file and stderr
	mw := io.MultiWriter(f, os.Stderr)
	logger = log.New(mw, "", log.LstdFlags|log.Lmicroseconds)

	log.Printf("[debug] logger initialized: %s", logFile)
	return true
}

// IsEnabled returns whether debug mode is active.
func IsEnabled() bool {
	mu.Lock()
	defer mu.Unlock()
	return enabled
}

// Logf writes a formatted debug message.
func Logf(format string, v ...interface{}) {
	mu.Lock()
	defer mu.Unlock()
	if !enabled {
		return
	}
	if logger != nil {
		logger.Printf(format, v...)
	}
}

// Log writes a debug message.
func Log(v ...interface{}) {
	mu.Lock()
	defer mu.Unlock()
	if !enabled {
		return
	}
	if logger != nil {
		logger.Println(v...)
	}
}
