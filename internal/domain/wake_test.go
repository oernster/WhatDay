package domain_test

import (
	"testing"
	"time"

	"github.com/oernster/WhatDay/internal/domain"
)

func TestWakeDelay(t *testing.T) {
	t.Parallel()
	const gap = time.Minute
	midnight := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		now  time.Time
		wait time.Duration
		due  bool
	}{
		{"hours away waits one gap", midnight.Add(-5 * time.Hour), gap, false},
		{"under a gap away waits exactly", midnight.Add(-1500 * time.Millisecond), 1500 * time.Millisecond, false},
		{"exactly a gap away", midnight.Add(-gap), gap, false},
		{"at target is due", midnight, 0, true},
		{"past target is due", midnight.Add(time.Second), 0, true},
	}
	for _, c := range cases {
		wait, due := domain.WakeDelay(c.now, midnight, gap)
		if wait != c.wait || due != c.due {
			t.Errorf("%s: got %s, %v; want %s, %v", c.name, wait, due, c.wait, c.due)
		}
	}
}
