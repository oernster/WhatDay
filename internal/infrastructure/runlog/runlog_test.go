package runlog

// FR-062, FR-063: a run leaves a log. A crash ends the process that has it, so
// the crash tests start this test binary again as a child; TestMain turns the
// child into the crash asked for. Ported from Bridge Talk.

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/oernster/WhatDay/internal/application"
)

const (
	// childEnv names what the child does; logEnv names the log it keeps.
	childEnv = "WHATDAY_RUNLOG_CHILD"
	logEnv   = "WHATDAY_RUNLOG_PATH"

	// panicAct panics on another goroutine; fatalAct sends all error output
	// to the log, writes a warning to it, then fails with a fatal error.
	panicAct = "panic"
	fatalAct = "fatal"

	plantedPanic   = "a planted panic on another goroutine"
	plantedWarning = "warning: a planted warning"
	// fatalHeadline is the line the Go runtime starts a fatal error's report
	// with; SetCrashOutput alone leaves it out of the file (Bridge Talk
	// measured this on Go 1.26.3).
	fatalHeadline = "fatal error: sync: unlock of unlocked mutex"

	// goCrashExitCode is the exit code the Go runtime ends a crashed program with.
	goCrashExitCode = 2
	// childFailedExitCode and childSurvivedExitCode end a child that never
	// reached its crash.
	childFailedExitCode   = 3
	childSurvivedExitCode = 4
	// childWait is how long a child waits for its crash before saying it survived.
	childWait = 10 * time.Second
)

// started is the time every test's run starts at.
var started = time.Date(2026, time.September, 19, 11, 18, 31, 0, time.Local)

func TestMain(m *testing.M) {
	if act := os.Getenv(childEnv); act != "" {
		crash(act, os.Getenv(logEnv))
	}
	os.Exit(m.Run())
}

// crash is the child: it keeps the log as the application does, then fails as
// act asks.
func crash(act, path string) {
	log, err := Open(path, started)
	if err == nil {
		if act == fatalAct {
			err = sendAll(log)
		} else {
			err = Keep(log)
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "the child could not keep its log: %v\n", err)
		os.Exit(childFailedExitCode)
	}
	switch act {
	case panicAct:
		go func() { panic(plantedPanic) }()
	case fatalAct:
		fmt.Fprintln(os.Stderr, plantedWarning)
		var unlocked sync.Mutex
		unlocked.Unlock()
	}
	time.Sleep(childWait)
	os.Exit(childSurvivedExitCode)
}

// runChild starts the child with an error output of its own and answers with
// its log and what it wrote to that output.
func runChild(t *testing.T, act string) (logged, errorOutput string) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("finding the test binary: %v", err)
	}
	path := filepath.Join(t.TempDir(), FileName)
	child := exec.Command(executable, "-test.run=^$")
	child.Env = append(os.Environ(), childEnv+"="+act, logEnv+"="+path)
	var said bytes.Buffer
	child.Stderr = &said

	ran := child.Run()

	var exit *exec.ExitError
	if !errors.As(ran, &exit) || exit.ExitCode() != goCrashExitCode {
		t.Fatalf("the child ended with %v, want a crash; it said %q", ran, said.String())
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the child's log: %v", err)
	}
	return string(raw), said.String()
}

// startLine is the line a run started at when adds to the log.
func startLine(when time.Time) string {
	return application.ProductName + " started " + when.Format(stampLayout) + "\n"
}

func TestAPanicOnAnotherGoroutineIsInTheLog(t *testing.T) {
	t.Parallel()
	logged, errorOutput := runChild(t, panicAct)
	if !strings.HasPrefix(logged, startLine(started)) {
		t.Errorf("the log does not open with the run's start line: %q", logged)
	}
	if want := "panic: " + plantedPanic; !strings.Contains(logged, want) {
		t.Errorf("the log lacks %q: %q", want, logged)
	}
	if want := "panic: " + plantedPanic; !strings.Contains(errorOutput, want) {
		t.Errorf("the error output lacks %q: %q", want, errorOutput)
	}
}

// The child has an error output of its own, so it is sent to the log directly:
// a test binary cannot be started without one.
func TestWhereTheRunHasNoErrorOutputEverythingIsInTheLog(t *testing.T) {
	t.Parallel()
	logged, errorOutput := runChild(t, fatalAct)
	for _, want := range []string{plantedWarning, fatalHeadline} {
		if !strings.Contains(logged, want) {
			t.Errorf("the log lacks %q: %q", want, logged)
		}
		if strings.Contains(errorOutput, want) {
			t.Errorf("the error output still took %q: %q", want, errorOutput)
		}
	}
}

// plantLog writes content to a log file under a new folder.
func plantLog(t *testing.T, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), FileName)
	if err := os.WriteFile(path, content, filePerm); err != nil {
		t.Fatalf("planting %s: %v", path, err)
	}
	return path
}

// openAndClose opens the log for a run started at when, then closes it.
func openAndClose(t *testing.T, path string, when time.Time) {
	t.Helper()
	log, err := Open(path, when)
	if err != nil {
		t.Fatalf("opening %s: %v", path, err)
	}
	if err := log.Close(); err != nil {
		t.Fatalf("closing %s: %v", path, err)
	}
}

func TestEveryRunAddsItsStartLineAfterWhatTheLogHolds(t *testing.T) {
	t.Parallel()
	earlier := "an earlier report\n"
	path := plantLog(t, []byte(earlier))
	later := started.Add(time.Hour)
	openAndClose(t, path, started)
	openAndClose(t, path, later)
	raw, err := os.ReadFile(path)
	if want := earlier + startLine(started) + startLine(later); err != nil || string(raw) != want {
		t.Errorf("the log holds %q, %v; want %q", raw, err, want)
	}
}

func TestALogOverTheLimitIsStartedAfresh(t *testing.T) {
	t.Parallel()
	for _, each := range []struct {
		size int
		kept bool
	}{{MaxBytes, true}, {MaxBytes + 1, false}} {
		planted := bytes.Repeat([]byte("x"), each.size)
		path := plantLog(t, planted)
		openAndClose(t, path, started)
		raw, err := os.ReadFile(path)
		want := startLine(started)
		if each.kept {
			want = string(planted) + want
		}
		if err != nil || string(raw) != want {
			t.Errorf("a log of %d bytes: holds %d bytes, %v; want %d", each.size, len(raw), err, len(want))
		}
	}
}

func TestTheLogIsMadeWithItsFolder(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "Local", application.ProductName, FileName)
	openAndClose(t, path, started)
	if raw, err := os.ReadFile(path); err != nil || string(raw) != startLine(started) {
		t.Errorf("the log holds %q, %v; want the start line", raw, err)
	}
}

func TestALogThatCannotBeKeptIsRefusedNamingIt(t *testing.T) {
	t.Parallel()
	plainFile := plantLog(t, []byte("x"))
	for _, path := range []string{filepath.Join(plainFile, "Local", FileName), t.TempDir()} {
		log, err := Open(path, started)
		if log != nil {
			_ = log.Close()
		}
		if err == nil || !strings.Contains(err.Error(), path) {
			t.Errorf("opening %s: got %v, want a refusal naming the path", path, err)
		}
	}
}

func TestTheLogSitsInTheProductsDataFolder(t *testing.T) {
	base := t.TempDir()
	t.Setenv(dataFolder, base)
	got, err := Path()
	if want := filepath.Join(base, application.ProductName, FileName); err != nil || got != want {
		t.Errorf("got %q, %v; want %q", got, err, want)
	}
}

func TestWithNoDataFolderThereIsNoLog(t *testing.T) {
	t.Setenv(dataFolder, "")
	if got, err := Path(); !errors.Is(err, errNoDataFolder) {
		t.Errorf("got %q, %v; want %v", got, err, errNoDataFolder)
	}
}

func TestLoggerStampsEachLine(t *testing.T) {
	t.Parallel()
	path := plantLog(t, nil)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, filePerm)
	if err != nil {
		t.Fatalf("opening %s: %v", path, err)
	}
	logger := NewLogger(file, func() time.Time { return started })
	logger.Printf("colour %s", "Blue")
	if err := file.Close(); err != nil {
		t.Fatalf("closing: %v", err)
	}
	raw, _ := os.ReadFile(path)
	if want := started.Format(stampLayout) + "  colour Blue\n"; string(raw) != want {
		t.Errorf("got %q, want %q", raw, want)
	}
}

func TestKeepWithAnErrorOutputCopiesCrashes(t *testing.T) {
	// The test runner gives this process an error output, so Keep takes the
	// crash-copy route and leaves os.Stderr alone.
	path := plantLog(t, nil)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, filePerm)
	if err != nil {
		t.Fatalf("opening %s: %v", path, err)
	}
	t.Cleanup(func() { _ = file.Close() })
	before := os.Stderr
	if err := Keep(file); err != nil {
		t.Fatalf("keep: %v", err)
	}
	// Stop copying crashes to the file before it is closed.
	t.Cleanup(func() { _ = debug.SetCrashOutput(os.Stderr, debug.CrashOptions{}) })
	if os.Stderr != before {
		t.Error("a run with an error output had it replaced")
	}
}
