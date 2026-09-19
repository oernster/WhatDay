// Package runlog keeps the log a run leaves (FR-062, FR-063): a line naming
// when the run started, then what the run reports, in WhatDay.log inside
// %LOCALAPPDATA%\WhatDay. Ported from Bridge Talk's runlog, trimmed to what
// WhatDay needs.
//
// A windowed program started from a shortcut or the Run key has no error
// output: Windows hands it a handle of 0, so the Go runtime's own crash report
// would reach nobody. Keep points the error output at the log before anything
// else runs, so a crash leaves a record.
package runlog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"

	"github.com/oernster/WhatDay/internal/application"
	"golang.org/x/sys/windows"
)

const (
	// FileName names the log inside the product's local data folder.
	FileName = application.ProductName + ".log"
	// MaxBytes is the size past which a run starts the log afresh, so the
	// file cannot grow without end: 1 MB, as Bridge Talk measured enough.
	MaxBytes = 1 << 20
)

const (
	stampLayout = "2006-01-02 15:04:05"
	folderPerm  = 0o755
	filePerm    = 0o644
	dataFolder  = "LOCALAPPDATA"
)

// errNoDataFolder says the environment names no local data folder.
var errNoDataFolder = errors.New(dataFolder + " is not set")

// Path answers the log's path without touching the disk.
func Path() (string, error) {
	base := os.Getenv(dataFolder)
	if base == "" {
		return "", errNoDataFolder
	}
	return filepath.Join(base, application.ProductName, FileName), nil
}

// Open opens the log at path for a run started at started, making its folder
// where there is none, then adds the run's start line. The log is kept unless
// it is over MaxBytes, when it is started afresh.
func Open(path string, started time.Time) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), folderPerm); err != nil {
		return nil, fmt.Errorf("making the folder for %s: %w", path, err)
	}
	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if info, err := os.Stat(path); err == nil && info.Size() > MaxBytes {
		flags |= os.O_TRUNC
	}
	file, err := os.OpenFile(path, flags, filePerm)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	if _, err := fmt.Fprintf(file, "%s started %s\n", application.ProductName, started.Format(stampLayout)); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("writing to %s: %w", path, err)
	}
	return file, nil
}

// Keep sends what the run reports as it fails to log, which stays open for
// the rest of the run. Where the run has no error output, all of it goes to
// the log; otherwise it stays where it is and the crash report is copied to
// the log as well.
func Keep(log *os.File) error {
	if !hasErrorOutput() {
		return sendAll(log)
	}
	if err := debug.SetCrashOutput(log, debug.CrashOptions{}); err != nil {
		return fmt.Errorf("copying crash reports to %s: %w", log.Name(), err)
	}
	return nil
}

// hasErrorOutput reports whether the run was given an error output.
func hasErrorOutput() bool {
	handle, err := windows.GetStdHandle(windows.STD_ERROR_HANDLE)
	return err == nil && handle != 0 && handle != windows.InvalidHandle
}

// sendAll points the run's error output at log: first the handle the Go
// runtime looks up for each report it writes, then os.Stderr, which was fixed
// from that handle when the program started.
func sendAll(log *os.File) error {
	if err := windows.SetStdHandle(windows.STD_ERROR_HANDLE, windows.Handle(log.Fd())); err != nil {
		return fmt.Errorf("sending error output to %s: %w", log.Name(), err)
	}
	os.Stderr = log
	return nil
}

// Logger writes timestamped lines to the log; it implements application.Log.
type Logger struct {
	mu   sync.Mutex
	file *os.File
	now  func() time.Time
}

// NewLogger writes to file, stamping each line with now.
func NewLogger(file *os.File, now func() time.Time) *Logger {
	return &Logger{file: file, now: now}
}

// Printf writes one stamped line. A line that cannot be written is dropped:
// the log is where failures are reported, so there is nowhere else to say so.
func (l *Logger) Printf(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = fmt.Fprintf(l.file, "%s  %s\n", l.now().Format(stampLayout), fmt.Sprintf(format, args...))
}
