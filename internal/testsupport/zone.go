// Package testsupport holds helpers shared by more than one test suite. It is
// imported by tests only.
package testsupport

import (
	"testing"
	"time"
	_ "time/tzdata" // tests see the same zone data the binary embeds (C-4)

	"github.com/oernster/WhatDay/internal/domain"
)

// London loads domain.ZoneName or fails the test.
func London(t *testing.T) *time.Location {
	t.Helper()
	zone, err := time.LoadLocation(domain.ZoneName)
	if err != nil {
		t.Fatalf("load %s: %v", domain.ZoneName, err)
	}
	return zone
}
