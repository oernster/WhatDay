package instance

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const (
	// childEnv carries the mutex name a child process tries to take.
	childEnv = "WHATDAY_INSTANCE_CHILD"
	// Exit codes the child answers with.
	childHeld    = 10
	childRefused = 11
	childFailed  = 12
)

func TestMain(m *testing.M) {
	if name := os.Getenv(childEnv); name != "" {
		_, held, err := Acquire(name)
		switch {
		case err != nil:
			os.Exit(childFailed)
		case held:
			os.Exit(childHeld)
		}
		os.Exit(childRefused)
	}
	os.Exit(m.Run())
}

// uniqueName keeps parallel tests and the real application apart.
func uniqueName(t *testing.T) string {
	return fmt.Sprintf(`Local\WhatDayTest.%s.%d`, t.Name(), time.Now().UnixNano())
}

func acquire(t *testing.T, name string) (*Lock, bool) {
	t.Helper()
	lock, held, err := Acquire(name)
	if err != nil {
		t.Fatalf("acquire %s: %v", name, err)
	}
	return lock, held
}

func TestSecondInstanceRefused(t *testing.T) {
	t.Parallel()
	name := uniqueName(t)
	first, held := acquire(t, name)
	if !held {
		t.Fatal("the first copy was refused")
	}
	defer func() { _ = first.Release() }()
	if second, held := acquire(t, name); held || second != nil {
		t.Fatal("a second copy in this process was let in")
	}
}

func TestAnotherProcessIsRefused(t *testing.T) {
	t.Parallel()
	name := uniqueName(t)
	first, held := acquire(t, name)
	if !held {
		t.Fatal("the first copy was refused")
	}
	defer func() { _ = first.Release() }()
	if code := runChild(t, name); code != childRefused {
		t.Fatalf("another process ended with %d, want %d (refused)", code, childRefused)
	}
}

func TestReleaseLetsTheNextCopyIn(t *testing.T) {
	t.Parallel()
	name := uniqueName(t)
	first, _ := acquire(t, name)
	if err := first.Release(); err != nil {
		t.Fatalf("release: %v", err)
	}
	if code := runChild(t, name); code != childHeld {
		t.Fatalf("after release another process ended with %d, want %d (held)", code, childHeld)
	}
}

func TestANameWithANulIsRefused(t *testing.T) {
	t.Parallel()
	if _, _, err := Acquire("bad\x00name"); err == nil || !strings.Contains(err.Error(), "naming") {
		t.Fatalf("got %v, want a naming error", err)
	}
}

func TestAnInvalidNamespaceIsAnError(t *testing.T) {
	t.Parallel()
	// Backslashes beyond the namespace prefix are not allowed in a mutex name.
	if _, _, err := Acquire(`NoSuchNamespace\x\y`); err == nil || !strings.Contains(err.Error(), "creating") {
		t.Fatalf("got %v, want a create error", err)
	}
}

func TestNameIsSessionLocal(t *testing.T) {
	t.Parallel()
	if want := `Local\WhatDay.SingleInstance`; Name != want {
		t.Fatalf("got %q, want %q", Name, want)
	}
}

func runChild(t *testing.T, name string) int {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("finding the test binary: %v", err)
	}
	child := exec.Command(executable, "-test.run=^$")
	child.Env = append(os.Environ(), childEnv+"="+name)
	err = child.Run()
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("the child ended with %v, want an exit code", err)
	}
	return exit.ExitCode()
}
