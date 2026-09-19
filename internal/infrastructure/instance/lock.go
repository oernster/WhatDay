// Package instance keeps WhatDay to one running copy per user session
// (FR-061), with a named Windows mutex.
package instance

import (
	"errors"
	"fmt"

	"github.com/oernster/WhatDay/internal/application"
	"golang.org/x/sys/windows"
)

// Name is the mutex WhatDay holds while it runs. The Local namespace scopes
// it to the user's session.
const Name = `Local\` + application.ProductName + ".SingleInstance"

// Lock is a held single-instance mutex.
type Lock struct{ handle windows.Handle }

// Acquire takes the mutex called name. It answers held false with no error
// when another process already holds it; the new copy should then exit.
func Acquire(name string) (lock *Lock, held bool, err error) {
	wide, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, false, fmt.Errorf("naming the mutex %q: %w", name, err)
	}
	handle, err := windows.CreateMutex(nil, false, wide)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		_ = windows.CloseHandle(handle)
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("creating the mutex %q: %w", name, err)
	}
	return &Lock{handle: handle}, true, nil
}

// Release lets the mutex go, so a later copy may start.
func (l *Lock) Release() error {
	return windows.CloseHandle(l.handle)
}
