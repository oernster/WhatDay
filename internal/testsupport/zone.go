// Package testsupport holds helpers shared by more than one test suite. It is
// imported by tests only.
package testsupport

import (
	"testing"
	"time"
	_ "time/tzdata" // tests see the same zone data the binary embeds (C-4)
)

// HomeZone is the owner's zone, the worked example the tests use. WhatDay
// itself follows whatever zone Windows is set to (Amendment 4).
const HomeZone = "Europe/London"

// London loads HomeZone or fails the test.
func London(t *testing.T) *time.Location {
	t.Helper()
	zone, err := time.LoadLocation(HomeZone)
	if err != nil {
		t.Fatalf("load %s: %v", HomeZone, err)
	}
	return zone
}
