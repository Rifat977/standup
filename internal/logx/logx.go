// Package logx is the standup logger. It writes to ~/.standup/standup.log
// (append-mode) and optionally tees to stderr when verbose mode is enabled.
//
// Use it from any package after Init() has been called once at startup.
// Calls before Init() are no-ops, so it is always safe to use.
package logx

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type level int

const (
	lvlDebug level = iota
	lvlInfo
	lvlWarn
	lvlError
)

var (
	mu      sync.Mutex
	logger  *log.Logger
	logFile *os.File
	minLvl  = lvlInfo
)

// Init opens ~/.standup/standup.log for append. If verbose is true, output is
// also written to stderr and the level threshold drops to debug.
// Safe to call multiple times — subsequent calls reconfigure level/sinks.
func Init(dir string, verbose bool) error {
	mu.Lock()
	defer mu.Unlock()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "standup.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	if logFile != nil {
		_ = logFile.Close()
	}
	logFile = f

	var w io.Writer = f
	if verbose {
		w = io.MultiWriter(f, os.Stderr)
		minLvl = lvlDebug
	} else {
		minLvl = lvlInfo
	}
	logger = log.New(w, "", log.LstdFlags|log.Lmicroseconds)
	return nil
}

// Path returns the absolute log file path, or "" if logging hasn't been initialised.
func Path() string {
	mu.Lock()
	defer mu.Unlock()
	if logFile == nil {
		return ""
	}
	return logFile.Name()
}

// Close flushes and closes the log file. Call from main on shutdown.
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
		logger = nil
	}
}

func emit(lvl level, prefix, format string, args ...any) {
	mu.Lock()
	l := logger
	threshold := minLvl
	mu.Unlock()
	if l == nil || lvl < threshold {
		return
	}
	l.Output(3, prefix+" "+fmt.Sprintf(format, args...))
}

func Debug(format string, args ...any) { emit(lvlDebug, "[DEBUG]", format, args...) }
func Info(format string, args ...any)  { emit(lvlInfo, "[INFO] ", format, args...) }
func Warn(format string, args ...any)  { emit(lvlWarn, "[WARN] ", format, args...) }
func Error(format string, args ...any) { emit(lvlError, "[ERROR]", format, args...) }
