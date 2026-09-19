package domain_test

import (
	"testing"
	"time"

	"github.com/oernster/WhatDay/internal/domain"
)

func TestNextMidnightAcrossTransitions(t *testing.T) {
	t.Parallel()
	zone := london(t)
	cases := []struct {
		name    string
		instant time.Time
		want    time.Time
	}{
		{"first BST day, 23 hours", time.Date(2026, 3, 29, 12, 0, 0, 0, zone), time.Date(2026, 3, 29, 23, 0, 0, 0, time.UTC)},
		{"first GMT day, 25 hours", time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC), time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)},
		{"exactly midnight answers the next one", time.Date(2026, 9, 20, 0, 0, 0, 0, zone), time.Date(2026, 9, 20, 23, 0, 0, 0, time.UTC)},
		{"leap day", time.Date(2028, 2, 29, 9, 0, 0, 0, zone), time.Date(2028, 3, 1, 0, 0, 0, 0, time.UTC)},
		{"year end", time.Date(2026, 12, 31, 23, 59, 59, 0, zone), time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		if got := domain.NextMidnight(c.instant, zone); !got.Equal(c.want) {
			t.Errorf("%s: got %s, want %s", c.name, got.UTC(), c.want)
		}
	}
}

func TestMidnightChain(t *testing.T) {
	t.Parallel()
	zone := london(t)
	const (
		short  = 23 * time.Hour
		normal = 24 * time.Hour
		long   = 25 * time.Hour
	)
	counts := map[time.Duration]int{}
	date := civilDate{firstYear, 1, 1}
	at := time.Date(date.year, time.Month(date.month), date.day, 0, 0, 0, 0, zone)
	for date.year <= lastYear {
		next := domain.NextMidnight(at, zone)
		date = date.next()
		y, m, d := next.In(zone).Date()
		h, mi, s := next.In(zone).Clock()
		if y != date.year || int(m) != date.month || d != date.day || h != 0 || mi != 0 || s != 0 || next.In(zone).Nanosecond() != 0 {
			t.Fatalf("after %s: got %s, want %v 00:00 London", at, next.In(zone), date)
		}
		length := next.Sub(at)
		if length != short && length != normal && length != long {
			t.Fatalf("day starting %s lasted %s", at, length)
		}
		counts[length]++
		at = next
	}
	years := lastYear - firstYear + 1
	if counts[short] != years || counts[long] != years {
		t.Fatalf("got %d short and %d long days over %d years, want one of each per year", counts[short], counts[long], years)
	}
}
