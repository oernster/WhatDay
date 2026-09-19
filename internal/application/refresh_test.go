package application_test

import (
	"testing"
	"time"
)

func TestStartShowsDayAndArmsMidnight(t *testing.T) {
	t.Parallel()
	zone := london(t)
	r := newRig(t, time.Date(2026, 9, 19, 15, 44, 49, 0, zone), &fakeStore{})
	r.ind.Start()
	if got := r.view.lastDay(); got != "Saturday" {
		t.Fatalf("day: got %q, want Saturday", got)
	}
	if want := time.Date(2026, 9, 20, 0, 0, 0, 0, zone); !r.scheduler.last().Equal(want) {
		t.Fatalf("wake: got %s, want %s", r.scheduler.last(), want)
	}
}

func TestMidnightWakeShowsNewDay(t *testing.T) {
	t.Parallel()
	zone := london(t)
	r := newRig(t, time.Date(2026, 9, 19, 23, 0, 0, 0, zone), &fakeStore{})
	r.ind.Start()
	r.clock.now = time.Date(2026, 9, 20, 0, 0, 0, 0, zone)
	r.ind.Refresh()
	if got := r.view.lastDay(); got != "Sunday" {
		t.Fatalf("day: got %q, want Sunday", got)
	}
	if want := time.Date(2026, 9, 21, 0, 0, 0, 0, zone); !r.scheduler.last().Equal(want) {
		t.Fatalf("wake: got %s, want %s", r.scheduler.last(), want)
	}
}

func TestEarlyWakeRearmsSameMidnight(t *testing.T) {
	t.Parallel()
	zone := london(t)
	midnight := time.Date(2026, 9, 20, 0, 0, 0, 0, zone)
	r := newRig(t, time.Date(2026, 9, 19, 23, 0, 0, 0, zone), &fakeStore{})
	r.ind.Start()
	r.clock.now = midnight.Add(-400 * time.Millisecond) // timer fired early
	r.ind.Refresh()
	if got := r.view.lastDay(); got != "Saturday" {
		t.Fatalf("day: got %q, want Saturday until midnight", got)
	}
	if !r.scheduler.last().Equal(midnight) {
		t.Fatalf("wake: got %s, want the same midnight %s", r.scheduler.last(), midnight)
	}
}

func TestResumeReevaluates(t *testing.T) {
	t.Parallel()
	zone := london(t)
	r := newRig(t, time.Date(2026, 9, 21, 23, 50, 0, 0, zone), &fakeStore{})
	r.ind.Start()
	// Slept through midnight; resumed Tuesday morning.
	r.clock.now = time.Date(2026, 9, 22, 7, 0, 0, 0, zone)
	r.ind.Refresh()
	if got := r.view.lastDay(); got != "Tuesday" {
		t.Fatalf("day: got %q, want Tuesday", got)
	}
	if want := time.Date(2026, 9, 23, 0, 0, 0, 0, zone); !r.scheduler.last().Equal(want) {
		t.Fatalf("wake: got %s, want %s", r.scheduler.last(), want)
	}
}

func TestClockChangeReevaluates(t *testing.T) {
	t.Parallel()
	zone := london(t)
	monday := time.Date(2026, 9, 21, 12, 0, 0, 0, zone)
	r := newRig(t, monday, &fakeStore{})
	r.ind.Start()
	r.clock.now = time.Date(2026, 9, 23, 12, 0, 0, 0, zone)
	r.ind.Refresh()
	if got := r.view.lastDay(); got != "Wednesday" {
		t.Fatalf("forward: got %q, want Wednesday", got)
	}
	r.clock.now = monday
	r.ind.Refresh()
	if got := r.view.lastDay(); got != "Monday" {
		t.Fatalf("back: got %q, want Monday", got)
	}
	if want := time.Date(2026, 9, 22, 0, 0, 0, 0, zone); !r.scheduler.last().Equal(want) {
		t.Fatalf("wake after moving back: got %s, want %s", r.scheduler.last(), want)
	}
}
