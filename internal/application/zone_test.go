package application_test

import (
	"errors"
	"testing"
	"time"
)

func load(t *testing.T, name string) *time.Location {
	t.Helper()
	zone, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("load %s: %v", name, err)
	}
	return zone
}

func TestTravelFollowsNewZone(t *testing.T) {
	t.Parallel()
	// 23:30 BST on Saturday 19 September 2026 is 22:30 UTC.
	instant := time.Date(2026, 9, 19, 22, 30, 0, 0, time.UTC)
	r := newRig(t, instant, &fakeStore{})
	r.ind.Start()
	if got, want := r.scheduler.last(), time.Date(2026, 9, 19, 23, 0, 0, 0, time.UTC); r.view.lastDay() != "Saturday" || !got.Equal(want) {
		t.Fatalf("London: %s, wake %s; want Saturday, wake %s", r.view.lastDay(), got, want)
	}

	// Landed in New York: 18:30 EDT, still Saturday; midnight there is 04:00 UTC.
	r.zones.zone = load(t, "America/New_York")
	r.ind.Refresh()
	if got, want := r.scheduler.last(), time.Date(2026, 9, 20, 4, 0, 0, 0, time.UTC); r.view.lastDay() != "Saturday" || !got.Equal(want) {
		t.Fatalf("New York: %s, wake %s; want Saturday, wake %s", r.view.lastDay(), got, want)
	}

	// Then Sydney: 08:30 AEST on Sunday.
	r.zones.zone = load(t, "Australia/Sydney")
	r.ind.Refresh()
	if got := r.view.lastDay(); got != "Sunday" {
		t.Fatalf("Sydney: got %s, want Sunday", got)
	}
	if got := r.log.about("timezone "); len(got) != 3 {
		t.Fatalf("want each zone logged once, got %q", got)
	}
}

func TestZoneFaultLoggedOncePerFault(t *testing.T) {
	t.Parallel()
	r := newRig(t, someInstant, &fakeStore{})
	r.zones.err = errors.New("zone key unknown")
	r.ind.Start()
	r.ind.Refresh()
	r.ind.Refresh()
	if got := r.log.about("unreadable"); len(got) != 1 {
		t.Fatalf("want one fault line over three refreshes, got %q", got)
	}
	if got := r.view.lastDay(); got != "Saturday" {
		t.Fatalf("with the fallback zone: got %s, want Saturday", got)
	}
	r.zones.err = nil
	r.ind.Refresh()
	r.zones.err = errors.New("zone key unknown")
	r.ind.Refresh()
	if got := r.log.about("unreadable"); len(got) != 2 {
		t.Fatalf("a fault that clears then returns is logged again, got %q", got)
	}
}
