package domain_test

import (
	"testing"
	"time"

	"github.com/oernster/WhatDay/internal/domain"
	"github.com/oernster/WhatDay/internal/testsupport"
)

// Range walked by the exhaustive calendar tests (FR-006).
const (
	firstYear = 2000
	lastYear  = 2399
)

var weekdayNames = [...]string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

var london = testsupport.London

// isLeap is the Gregorian rule, stated here rather than taken from Go.
func isLeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

func daysIn(year, month int) int {
	lengths := [...]int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if month == 2 && isLeap(year) {
		return 29
	}
	return lengths[month-1]
}

// sakamoto answers the weekday (0 is Sunday) by Sakamoto's method, an oracle
// independent of Go's time package.
func sakamoto(year, month, day int) int {
	offsets := [...]int{0, 3, 2, 5, 0, 3, 5, 1, 4, 6, 2, 4}
	if month < 3 {
		year--
	}
	return (year + year/4 - year/100 + year/400 + offsets[month-1] + day) % 7
}

type civilDate struct{ year, month, day int }

func (d civilDate) next() civilDate {
	switch {
	case d.day < daysIn(d.year, d.month):
		return civilDate{d.year, d.month, d.day + 1}
	case d.month < 12:
		return civilDate{d.year, d.month + 1, 1}
	}
	return civilDate{d.year + 1, 1, 1}
}

func TestDayNameAtInstant(t *testing.T) {
	t.Parallel()
	instant := time.Date(2026, 9, 19, 15, 44, 49, 0, time.FixedZone("BST", 3600))
	if got := domain.DayName(instant, london(t)); got != "Saturday" {
		t.Fatalf("got %q, want Saturday", got)
	}
}

func TestDayNameEitherSideOfMidnight(t *testing.T) {
	t.Parallel()
	zone := london(t)
	cases := []struct {
		instant time.Time
		want    string
	}{
		{time.Date(2026, 9, 19, 23, 59, 59, 999999999, zone), "Saturday"},
		{time.Date(2026, 9, 20, 0, 0, 0, 0, zone), "Sunday"},
		{time.Date(2026, 12, 31, 23, 59, 59, 999999999, zone), "Thursday"},
		{time.Date(2027, 1, 1, 0, 0, 0, 0, zone), "Friday"},
	}
	for _, c := range cases {
		if got := domain.DayName(c.instant, zone); got != c.want {
			t.Errorf("%s: got %q, want %q", c.instant, got, c.want)
		}
	}
}

func TestDayIgnoresLocalZone(t *testing.T) {
	t.Parallel()
	hawaii := time.FixedZone("HST", -10*3600)
	// 00:30 BST on Sunday is still 13:30 on Saturday in Hawaii.
	instant := time.Date(2026, 9, 19, 13, 30, 0, 0, hawaii)
	if got := domain.DayName(instant, london(t)); got != "Sunday" {
		t.Fatalf("got %q, want Sunday", got)
	}
}

func TestLeapYearExamples(t *testing.T) {
	t.Parallel()
	zone := london(t)
	cases := []struct {
		date civilDate
		want string
	}{
		{civilDate{2000, 2, 29}, "Tuesday"},
		{civilDate{2024, 2, 29}, "Thursday"},
		{civilDate{2028, 2, 29}, "Tuesday"},
		{civilDate{2100, 2, 28}, "Sunday"},
		{civilDate{2100, 3, 1}, "Monday"},
	}
	for _, c := range cases {
		noon := time.Date(c.date.year, time.Month(c.date.month), c.date.day, 12, 0, 0, 0, zone)
		if got := domain.DayName(noon, zone); got != c.want {
			t.Errorf("%v: got %q, want %q", c.date, got, c.want)
		}
	}
}

func TestEveryDayAgainstIndependentFormula(t *testing.T) {
	t.Parallel()
	zone := london(t)
	checked := 0
	for d := (civilDate{firstYear, 1, 1}); d.year <= lastYear; d = d.next() {
		noon := time.Date(d.year, time.Month(d.month), d.day, 12, 0, 0, 0, zone)
		want := weekdayNames[sakamoto(d.year, d.month, d.day)]
		if got := domain.DayName(noon, zone); got != want {
			t.Fatalf("%v: got %q, want %q", d, got, want)
		}
		checked++
	}
	// 400 Gregorian years always hold 146097 days; anything else means the
	// walk itself is wrong.
	const daysIn400Years = 146097
	if checked != daysIn400Years {
		t.Fatalf("walked %d days, want %d", checked, daysIn400Years)
	}
}
