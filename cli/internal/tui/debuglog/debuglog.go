// ABOUTME: Simple debug logger for TUI that writes to a log file
// ABOUTME: Avoids interfering with terminal display while capturing errors

package debuglog

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	logFile *os.File
	mu      sync.Mutex
	enabled bool
)

// Init initializes the debug logger with the config directory
// If configDir is empty, logging is disabled
func Init(configDir string) error {
	mu.Lock()
	defer mu.Unlock()

	if configDir == "" {
		enabled = false
		return nil
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0700); err != nil {
		enabled = false
		return err
	}

	logPath := filepath.Join(configDir, "debug.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		enabled = false
		return err
	}

	logFile = f
	enabled = true
	return nil
}

// Close closes the log file
func Close() {
	mu.Lock()
	defer mu.Unlock()

	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
	enabled = false
}

// Log writes a message to the debug log
func Log(format string, args ...interface{}) {
	mu.Lock()
	defer mu.Unlock()

	if !enabled || logFile == nil {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(logFile, "[%s] %s\n", timestamp, msg)
}

// Error logs an error with context
func Error(context string, err error) {
	if err == nil {
		return
	}
	Log("ERROR [%s]: %v", context, err)
}

// Warn logs a warning message
func Warn(format string, args ...interface{}) {
	Log("WARN: "+format, args...)
}
