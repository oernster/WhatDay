package clock

import (
	"testing"
	"time"
)

func TestNowIsTheSystemClock(t *testing.T) {
	t.Parallel()
	before := time.Now()
	got := System{}.Now()
	after := time.Now()
	if got.Before(before) || got.After(after) {
		t.Fatalf("got %s, want between %s and %s", got, before, after)
	}
}
